package main

import (
	"compress/gzip"
	"flag"
	"fmt"
	"github.com/docker/cli/cli/command/image/build"
	"github.com/docker/docker/pkg/archive"
	"github.com/docker/docker/pkg/idtools"
	"github.com/ryan-gerstenkorn-sp/bctx/bctx"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
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
	excludes, err := bctx.ReadIgnore(ignore)
	if err != nil {
		return fmt.Errorf("failed to read ignore(path=%v): %w", ignore, err)
	}

	w, outputPath, err := bctx.Writer(flags.Destination)
	if err != nil {
		return fmt.Errorf("failed to prepare writer: %w", err)
	}
	defer w.Close()
	if strings.HasSuffix(flags.Destination, ".gz") {
		w = gzip.NewWriter(w)
		defer w.Close()
	}

	if outputPath != "" {
		rel, err := filepath.Rel(flags.Source, outputPath)
		if err != nil {
			return fmt.Errorf("failed to resolve rel path(src=%v, target=%v): %w", flags.Source, outputPath, err)
		}
		if !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			excludes = append(excludes, rel)
		}
	}

	err = build.ValidateContextDirectory(flags.Source, excludes)
	if err != nil {
		return fmt.Errorf("failed to validate directory (path=%v): %w", flags.Source, err)
	}

	tarOptions := &archive.TarOptions{
		ExcludePatterns: excludes,
		ChownOpts:       &idtools.Identity{UID: 0, GID: 0},
		IncludeFiles:    []string{"."},
	}
	if flags.Dockerfile != "" {
		d, err := ioutil.ReadFile(flags.Dockerfile)
		if err != nil {
			return fmt.Errorf("failed to read Dockerfile (path=%v): %w", flags.Dockerfile, err)
		}
		name, err := ioutil.TempDir(flags.Source, ".")
		if err != nil {
			return fmt.Errorf("failed to create temp dir: %w", err)
		}
		defer os.RemoveAll(filepath.Join(flags.Source, name))

		dockerfile := filepath.Join(name, "Dockerfile")
		tempDockerfile := filepath.Join(flags.Source, dockerfile)
		if err := ioutil.WriteFile(tempDockerfile, d, 0666); err != nil {
			return fmt.Errorf("failed to write temporary Dockerfile (path=%v): %w", tempDockerfile, err)
		}

		tarOptions.ExcludePatterns = append(tarOptions.ExcludePatterns, "Dockerfile", name)
		tarOptions.IncludeFiles = append(tarOptions.IncludeFiles, dockerfile)
		tarOptions.RebaseNames = map[string]string{
			dockerfile: "Dockerfile",
		}
	}
	buildCtx, err := archive.TarWithOptions(flags.Source, tarOptions)
	if err != nil {
		return fmt.Errorf("failed to prepare archive: %w", err)
	}

	_, err = io.Copy(w, buildCtx)
	if err != nil {
		return fmt.Errorf("failed to write build context: %w", err)
	}
	err = w.Close()
	if err != nil {
		return fmt.Errorf("failed to close writer: %w", err)
	}
	return nil
}
