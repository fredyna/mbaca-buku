package storage

import (
	"context"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"github.com/fredy/mbaca-buku/internal/config"
)

type MinIOStorage struct {
	client *minio.Client
	bucket string
}

func NewMinIOClient(cfg *config.Config) (*MinIOStorage, error) {
	client, err := minio.New(cfg.MinIOEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.MinIOUser, cfg.MinIOPassword, ""),
		Secure: cfg.MinIOUseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create minio client: %w", err)
	}

	ctx := context.Background()
	exists, err := client.BucketExists(ctx, cfg.MinIOBucket)
	if err != nil {
		return nil, fmt.Errorf("failed to check bucket: %w", err)
	}

	if !exists {
		if err := client.MakeBucket(ctx, cfg.MinIOBucket, minio.MakeBucketOptions{}); err != nil {
			return nil, fmt.Errorf("failed to create bucket: %w", err)
		}
		log.Printf("Created MinIO bucket: %s", cfg.MinIOBucket)
	}

	log.Println("Connected to MinIO")
	return &MinIOStorage{client: client, bucket: cfg.MinIOBucket}, nil
}

func (s *MinIOStorage) UploadFile(ctx context.Context, objectName string, reader io.Reader, size int64, contentType string) (string, error) {
	_, err := s.client.PutObject(ctx, s.bucket, objectName, reader, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload file: %w", err)
	}
	return objectName, nil
}

func (s *MinIOStorage) GetPresignedURL(ctx context.Context, objectName string, expiry time.Duration) (string, error) {
	url, err := s.client.PresignedGetObject(ctx, s.bucket, objectName, expiry, nil)
	if err != nil {
		return "", fmt.Errorf("failed to get presigned URL: %w", err)
	}
	return url.String(), nil
}

func (s *MinIOStorage) DeleteFile(ctx context.Context, objectName string) error {
	return s.client.RemoveObject(ctx, s.bucket, objectName, minio.RemoveObjectOptions{})
}
