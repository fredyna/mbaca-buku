package storage

import (
	"context"
	"fmt"
	"io"
	"log"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"github.com/fredy/mbaca-buku/internal/config"
)

// URLPrefix is the same-origin path that presigned URLs are rewritten to.
// nginx proxies it on to the R2 endpoint, which keeps the account ID out of
// the browser and avoids needing CORS rules on the bucket.
const URLPrefix = "/r2"

type R2Storage struct {
	client   *minio.Client
	bucket   string
	endpoint string
}

func NewR2Client(cfg *config.Config) (*R2Storage, error) {
	client, err := minio.New(cfg.R2Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.R2AccessKey, cfg.R2SecretKey, ""),
		Secure: true,
		// R2 accepts only "auto". Setting it explicitly also stops minio-go
		// from probing the bucket location, which R2 does not implement.
		Region: "auto",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create r2 client: %w", err)
	}

	// The bucket is created out of band in the Cloudflare dashboard: a
	// bucket-scoped R2 token cannot create one, so checking is all we can
	// usefully do here.
	ctx := context.Background()
	exists, err := client.BucketExists(ctx, cfg.R2Bucket)
	if err != nil {
		return nil, fmt.Errorf("failed to reach r2 bucket %q: %w", cfg.R2Bucket, err)
	}
	if !exists {
		return nil, fmt.Errorf("r2 bucket %q does not exist", cfg.R2Bucket)
	}

	log.Printf("Connected to R2 bucket: %s", cfg.R2Bucket)
	return &R2Storage{client: client, bucket: cfg.R2Bucket, endpoint: cfg.R2Endpoint}, nil
}

func (s *R2Storage) UploadFile(ctx context.Context, objectName string, reader io.Reader, size int64, contentType string) (string, error) {
	_, err := s.client.PutObject(ctx, s.bucket, objectName, reader, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload file: %w", err)
	}
	return objectName, nil
}

func (s *R2Storage) GetPresignedURL(ctx context.Context, objectName string, expiry time.Duration) (string, error) {
	u, err := s.client.PresignedGetObject(ctx, s.bucket, objectName, expiry, nil)
	if err != nil {
		return "", fmt.Errorf("failed to get presigned URL: %w", err)
	}
	return strings.Replace(u.String(), "https://"+s.endpoint, URLPrefix, 1), nil
}

// OpenFile returns the stored object for reading. The returned value seeks, so
// it can be handed to readers that need random access, and must be closed by
// the caller.
func (s *R2Storage) OpenFile(ctx context.Context, objectName string) (io.ReadSeekCloser, error) {
	obj, err := s.client.GetObject(ctx, s.bucket, objectName, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	return obj, nil
}

func (s *R2Storage) DeleteFile(ctx context.Context, objectName string) error {
	return s.client.RemoveObject(ctx, s.bucket, objectName, minio.RemoveObjectOptions{})
}
