package storage

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/holihur/openshop/internal/config"
	"github.com/holihur/openshop/internal/port"
)

// S3 implements port.ObjectStorage against any S3-compatible endpoint
// (AWS S3, MinIO, Aliyun OSS with S3 gateway). Because it is chosen by config,
// the rest of the application is unaware of the vendor.
type S3 struct {
	client  *s3.Client
	presign *s3.PresignClient
	bucket  string
	public  string
}

func NewS3(ctx context.Context, cfg config.StorageConfig) (*S3, error) {
	if cfg.Bucket == "" {
		return nil, fmt.Errorf("storage s3: bucket is required")
	}

	loadOpts := []func(*awsconfig.LoadOptions) error{}
	if cfg.AccessKey != "" && cfg.SecretKey != "" {
		loadOpts = append(loadOpts, awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(cfg.AccessKey, cfg.SecretKey, ""),
		))
	}
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, loadOpts...)
	if err != nil {
		return nil, fmt.Errorf("storage s3: load config: %w", err)
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		if cfg.Endpoint != "" {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
			o.UsePathStyle = true // required by MinIO and most S3 gateways
		}
	})

	return &S3{
		client:  client,
		presign: s3.NewPresignClient(client),
		bucket:  cfg.Bucket,
		public:  cfg.PublicURL,
	}, nil
}

func (s *S3) Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) (string, error) {
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(s.bucket),
		Key:           aws.String(key),
		Body:          r,
		ContentLength: aws.Int64(size),
		ContentType:   aws.String(contentType),
	})
	if err != nil {
		return "", fmt.Errorf("storage s3 put: %w", err)
	}
	return s.URL(ctx, key, 0)
}

func (s *S3) Delete(ctx context.Context, key string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("storage s3 delete: %w", err)
	}
	return nil
}

func (s *S3) URL(ctx context.Context, key string, ttl time.Duration) (string, error) {
	if s.public != "" {
		return s.public + "/" + key, nil
	}
	if ttl <= 0 {
		ttl = 15 * time.Minute
	}
	out, err := s.presign.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(ttl))
	if err != nil {
		return "", fmt.Errorf("storage s3 presign: %w", err)
	}
	return out.URL, nil
}

var _ port.ObjectStorage = (*S3)(nil)

// New selects the storage driver declared in configuration.
func New(ctx context.Context, cfg config.StorageConfig) (port.ObjectStorage, error) {
	switch cfg.Driver {
	case "s3":
		return NewS3(ctx, cfg)
	case "local", "":
		return NewLocal(cfg)
	default:
		return nil, fmt.Errorf("storage: unknown driver %q", cfg.Driver)
	}
}
