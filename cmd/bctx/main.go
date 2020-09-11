package main

import (
	"context"
	"flag"
	"io"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"cloud.google.com/go/storage"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/docker/cli/cli/command/image/build"
	"github.com/docker/docker/builder/dockerignore"
	"github.com/docker/docker/pkg/archive"
	"github.com/docker/docker/pkg/idtools"
	"github.com/klauspost/pgzip"
	"golang.org/x/xerrors"
	"google.golang.org/api/option"
)

func main() {
	log.SetPrefix("bctx: ")
	log.SetFlags(0)
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	var flags struct {
		Ignore      string
		Destination string
		Source      string
		Dockerfile  string
	}
	flag.StringVar(&flags.Ignore, "ignore", "", `.dockerignore path (default "$src/.dockerignore")`)
	flag.StringVar(&flags.Destination, "dest", "", "destination path, supported gs://, s3:// and dir (required)")
	flag.StringVar(&flags.Source, "src", ".", "source directory")
	flag.StringVar(&flags.Dockerfile, "f", "", "override Dockerfile")
	flag.Parse()

	if flags.Destination == "" {
		flag.Usage()
		log.Print("-dest is required")
		os.Exit(2)
	}

	ignore := flags.Ignore
	if ignore == "" {
		ignore = filepath.Join(flags.Source, ".dockerignore")
	}
	excludes, err := readIgnore(ignore)
	if err != nil {
		return xerrors.Errorf("failed to read ignore(path=%v): %w", ignore, err)
	}

	w, outputPath, err := writer(flags.Destination)
	if err != nil {
		return xerrors.Errorf("failed to prepare writer: %w", err)
	}
	defer w.Close()
	if strings.HasSuffix(flags.Destination, ".gz") {
		w = pgzip.NewWriter(w)
		defer w.Close()
	}

	if outputPath != "" {
		rel, err := filepath.Rel(flags.Source, outputPath)
		if err != nil {
			return xerrors.Errorf("failed to resolve rel path(src=%v, target=%v): %w", flags.Source, outputPath, err)
		}
		if !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			excludes = append(excludes, rel)
		}
	}

	err = build.ValidateContextDirectory(flags.Source, excludes)
	if err != nil {
		return xerrors.Errorf("failed to validate directory (path=%v): %w", flags.Source, err)
	}

	tarOptions := &archive.TarOptions{
		ExcludePatterns: excludes,
		ChownOpts:       &idtools.Identity{UID: 0, GID: 0},
		IncludeFiles:    []string{"."},
	}
	if flags.Dockerfile != "" {
		d, err := ioutil.ReadFile(flags.Dockerfile)
		if err != nil {
			return xerrors.Errorf("failed to read Dockerfile (path=%v): %w", flags.Dockerfile, err)
		}
		name, err := ioutil.TempDir(flags.Source, ".")
		if err != nil {
			return xerrors.Errorf("failed to create temp dir: %w", err)
		}
		defer os.RemoveAll(filepath.Join(flags.Source, name))

		dockerfile := filepath.Join(name, "Dockerfile")
		tempDockerfile := filepath.Join(flags.Source, dockerfile)
		if err := ioutil.WriteFile(tempDockerfile, d, 0666); err != nil {
			return xerrors.Errorf("failed to write temporary Dockerfile (path=%v): %w", tempDockerfile, err)
		}

		tarOptions.ExcludePatterns = append(tarOptions.ExcludePatterns, "Dockerfile", name)
		tarOptions.IncludeFiles = append(tarOptions.IncludeFiles, dockerfile)
		tarOptions.RebaseNames = map[string]string{
			dockerfile: "Dockerfile",
		}
	}
	buildCtx, err := archive.TarWithOptions(flags.Source, tarOptions)
	if err != nil {
		return xerrors.Errorf("failed to prepare archive: %w", err)
	}

	_, err = io.Copy(w, buildCtx)
	if err != nil {
		return xerrors.Errorf("failed to write build context: %w", err)
	}
	err = w.Close()
	if err != nil {
		return xerrors.Errorf("failed to close writer: %w", err)
	}
	return nil
}

func readIgnore(p string) ([]string, error) {
	f, err := os.Open(p)
	switch {
	case os.IsNotExist(err):
		return nil, nil
	case err != nil:
		return nil, xerrors.Errorf("failed to open: %w", err)
	}
	defer f.Close()
	excludes, err := dockerignore.ReadAll(f)
	if err != nil {
		return nil, xerrors.Errorf("failed to read: %w", err)
	}
	return excludes, nil
}

func writer(dest string) (io.WriteCloser, string, error) {
	ctx := context.Background()
	switch {
	case strings.HasPrefix(dest, "gs://"), strings.HasPrefix(dest, "s3://"), strings.HasPrefix(dest, "file://"):
		u, err := url.Parse(dest)
		if err != nil {
			return nil, "", xerrors.Errorf("failed to parse destination(dest=%v): %w", dest, err)
		}
		switch u.Scheme {
		case "gs":
			var opts []option.ClientOption
			if cred := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS_JSON"); cred != "" {
				opts = append(opts, option.WithCredentialsJSON([]byte(cred)))
			}
			storageClient, err := storage.NewClient(ctx, opts...)
			if err != nil {
				return nil, "", xerrors.Errorf("failed to create gcs client: %w", err)
			}
			return storageClient.Bucket(u.Host).Object(strings.TrimPrefix(u.Path, "/")).NewWriter(ctx), "", nil
		case "s3":
			sess, err := session.NewSession()
			if err != nil {
				return nil, "", xerrors.Errorf("failed to create aws session: %w", err)
			}
			r, w := io.Pipe()
			go func() {
				uploader := s3manager.NewUploader(sess)
				path := strings.TrimPrefix(u.Path, "/")
				_, err := uploader.UploadWithContext(ctx, &s3manager.UploadInput{
					Bucket: &u.Host,
					Key:    &path,
					Body:   r,
				})
				_ = r.CloseWithError(err)
			}()
			return w, "", nil
		case "file":
			p := filepath.FromSlash(u.Path)
			f, err := os.Create(p)
			if err != nil {
				return nil, "", xerrors.Errorf("failed to create(path=%v): %w", p, err)
			}
			return f, p, nil
		}
		panic("unreachable")
	default:
		if dest == "-" {
			return os.Stdout, "", nil
		}
		f, err := os.Create(dest)
		if err != nil {
			return nil, "", xerrors.Errorf("failed to create(dest=%v): %w", dest, err)
		}
		return f, dest, nil
	}
}
