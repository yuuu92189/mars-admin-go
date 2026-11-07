package s3

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"strings"

	"mars-admin-go/internal/modules/storage/domain"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type Backend struct {
	client     *minio.Client
	bucket     string
	publicBase string
}

type Config struct {
	Endpoint   string
	Region     string
	Bucket     string
	AccessKey  string
	SecretKey  string
	PublicBase string
}

func New(cfg Config) (*Backend, error) {
	parsed, err := url.Parse(cfg.Endpoint)
	if err != nil || parsed.Host == "" {
		return nil, fmt.Errorf("parse S3 endpoint: %w", err)
	}
	secure := parsed.Scheme == "https"
	client, err := minio.New(parsed.Host, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: secure,
		Region: cfg.Region,
	})
	if err != nil {
		return nil, fmt.Errorf("create S3 client: %w", err)
	}
	return &Backend{
		client: client, bucket: cfg.Bucket,
		publicBase: strings.TrimRight(cfg.PublicBase, "/"),
	}, nil
}

func (b *Backend) Name() string { return "s3" }

func (b *Backend) Put(ctx context.Context, key string, body io.Reader, size int64, contentType string) (domain.ObjectInfo, error) {
	info, err := b.client.PutObject(ctx, b.bucket, key, body, size, minio.PutObjectOptions{ContentType: contentType})
	if err != nil {
		return domain.ObjectInfo{}, fmt.Errorf("put S3 object: %w", err)
	}
	return domain.ObjectInfo{Size: info.Size, ContentType: contentType, ETag: info.ETag}, nil
}

func (b *Backend) Get(ctx context.Context, key string) (io.ReadCloser, domain.ObjectInfo, error) {
	object, err := b.client.GetObject(ctx, b.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, domain.ObjectInfo{}, fmt.Errorf("get S3 object: %w", err)
	}
	info, err := object.Stat()
	if err != nil {
		_ = object.Close()
		return nil, domain.ObjectInfo{}, fmt.Errorf("stat S3 object: %w", err)
	}
	return object, domain.ObjectInfo{Size: info.Size, ContentType: info.ContentType, ETag: info.ETag}, nil
}

func (b *Backend) Delete(ctx context.Context, key string) error {
	if err := b.client.RemoveObject(ctx, b.bucket, key, minio.RemoveObjectOptions{}); err != nil {
		return fmt.Errorf("delete S3 object: %w", err)
	}
	return nil
}

func (b *Backend) Exists(ctx context.Context, key string) (bool, error) {
	_, err := b.client.StatObject(ctx, b.bucket, key, minio.StatObjectOptions{})
	if err != nil {
		response := minio.ToErrorResponse(err)
		if response.Code == "NoSuchKey" || response.Code == "NotFound" {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (b *Backend) URL(key string) string {
	if b.publicBase != "" {
		return b.publicBase + "/" + strings.TrimLeft(key, "/")
	}
	return b.bucket + "/" + strings.TrimLeft(key, "/")
}
