package bctx

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/docker/docker/builder/dockerignore"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)


func ReadIgnore(p string) ([]string, error) {
	f, err := os.Open(p)
	switch {
	case os.IsNotExist(err):
		return nil, nil
	case err != nil:
		return nil, fmt.Errorf("failed to open: %w", err)
	}
	defer f.Close()
	excludes, err := dockerignore.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("failed to read: %w", err)
	}
	return excludes, nil
}

func Writer(dest string) (io.WriteCloser, string, error) {
	ctx := context.Background()
	switch {
	case strings.HasPrefix(dest, "s3://"), strings.HasPrefix(dest, "file://"):
		u, err := url.Parse(dest)
		if err != nil {
			return nil, "", fmt.Errorf("failed to parse destination(dest=%v): %w", dest, err)
		}
		switch u.Scheme {
		case "s3":
			sess, err := session.NewSession()
			if err != nil {
				return nil, "", fmt.Errorf("failed to create aws session: %w", err)
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
				return nil, "", fmt.Errorf("failed to create(path=%v): %w", p, err)
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
			return nil, "", fmt.Errorf("failed to create(dest=%v): %w", dest, err)
		}
		return f, dest, nil
	}
}
