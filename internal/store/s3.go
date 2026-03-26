package store

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// s3API is the subset of the S3 client API used by S3Backend, extracted as an
// interface so that tests can substitute a mock without a real AWS connection.
type s3API interface {
	GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)
	PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error)
	DeleteObject(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error)
	HeadBucket(ctx context.Context, params *s3.HeadBucketInput, optFns ...func(*s3.Options)) (*s3.HeadBucketOutput, error)
}

// S3Config holds the configuration for the S3 backend.
type S3Config struct {
	// Bucket is the S3 bucket name (required).
	Bucket string
	// Prefix is an optional key prefix (e.g. "tssk/").  A trailing slash is
	// added automatically when non-empty.
	Prefix string
	// Endpoint overrides the default AWS endpoint (useful for MinIO, etc.).
	Endpoint string
	// Region is the AWS region (defaults to us-east-1 when empty).
	Region string
	// RequestTimeout is the per-request context deadline (default: 30s).
	RequestTimeout time.Duration
}

// S3Backend implements Backend using an S3-compatible object store.
// It reuses a single *s3.Client (and its underlying http.Client connection
// pool) for all requests.
type S3Backend struct {
	client    s3API
	cfg       S3Config
	timeout   time.Duration
	tasksFile string
	docsDir   string
}

// NewS3Backend creates an S3Backend using default AWS credential loading
// (environment variables, shared credentials file, EC2/ECS instance role,
// etc.).  Pass a non-empty Endpoint to target MinIO or another S3-compatible
// service.
func NewS3Backend(cfg S3Config) (*S3Backend, error) {
	if cfg.Bucket == "" {
		return nil, errors.New("s3 backend: bucket name must not be empty")
	}
	region := cfg.Region
	if region == "" {
		region = "us-east-1"
	}
	timeout := cfg.RequestTimeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	optFns := []func(*awsconfig.LoadOptions) error{
		awsconfig.WithRegion(region),
		// Clone http.DefaultTransport to preserve important defaults (proxy
		// settings, dialer configuration, TLS timeouts, HTTP/2) and only
		// override connection-pooling–related fields.
		// In practice, http.DefaultTransport is always *http.Transport, so
		// the fallback branch is a defensive safety net for unusual setups
		// where the global transport has been replaced.
		awsconfig.WithHTTPClient(func() *http.Client {
			base, ok := http.DefaultTransport.(*http.Transport)
			if !ok {
				// http.DefaultTransport has been replaced with a non-standard
				// implementation; fall back to a safe set of defaults.
				base = &http.Transport{
					MaxIdleConns:        100,
					MaxIdleConnsPerHost: 10,
					IdleConnTimeout:     90 * time.Second,
				}
				return &http.Client{Transport: base, Timeout: timeout}
			}
			t := base.Clone()
			t.MaxIdleConns = 100
			t.MaxIdleConnsPerHost = 10
			t.IdleConnTimeout = 90 * time.Second
			return &http.Client{Transport: t, Timeout: timeout}
		}()),
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(), optFns...)
	if err != nil {
		return nil, fmt.Errorf("s3 backend: loading aws config: %w", err)
	}

	s3Opts := []func(*s3.Options){}
	if cfg.Endpoint != "" {
		s3Opts = append(s3Opts, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
			// Path-style addressing is required for MinIO and other compatible
			// services that do not support virtual-hosted bucket URLs.
			o.UsePathStyle = true
		})
	}

	client := s3.NewFromConfig(awsCfg, s3Opts...)
	backend, err := NewS3BackendWithClient(client, cfg, timeout)
	if err != nil {
		return nil, err
	}
	return backend, nil
}

// NewS3BackendWithClient creates an S3Backend using the supplied client.
// This is primarily intended for tests that inject a mock S3 client.
// Returns an error if required configuration (e.g. Bucket) is missing.
func NewS3BackendWithClient(client s3API, cfg S3Config, timeout time.Duration) (*S3Backend, error) {
	if cfg.Bucket == "" {
		return nil, errors.New("s3 backend: bucket name must not be empty")
	}
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return &S3Backend{client: client, cfg: cfg, timeout: timeout, tasksFile: defaultTasksFile, docsDir: defaultDocsDir}, nil
}

// key returns the full S3 object key for the given relative path.
func (b *S3Backend) key(rel string) string {
	if b.cfg.Prefix == "" {
		return rel
	}
	// Ensure exactly one slash between prefix and path.
	return strings.TrimRight(b.cfg.Prefix, "/") + "/" + rel
}

func (b *S3Backend) ctx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), b.timeout)
}

// getObject fetches an S3 object and returns its body as bytes.
// Returns (nil, nil) when the key does not exist (404 / NoSuchKey).
func (b *S3Backend) getObject(key string) ([]byte, error) {
	ctx, cancel := b.ctx()
	defer cancel()

	out, err := b.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(b.cfg.Bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		var noSuchKey *types.NoSuchKey
		if errors.As(err, &noSuchKey) {
			return nil, nil
		}
		return nil, err
	}
	defer func() { _ = out.Body.Close() }()

	data, err := io.ReadAll(out.Body)
	if err != nil {
		return nil, fmt.Errorf("reading s3 object body: %w", err)
	}
	return data, nil
}

// putObject stores data at the given S3 key.
func (b *S3Backend) putObject(key string, data []byte) error {
	ctx, cancel := b.ctx()
	defer cancel()

	_, err := b.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(b.cfg.Bucket),
		Key:           aws.String(key),
		Body:          bytes.NewReader(data),
		ContentLength: aws.Int64(int64(len(data))),
	})
	if err != nil {
		return fmt.Errorf("s3 PutObject %q: %w", key, err)
	}
	return nil
}

// ReadTasksData returns the raw JSONL content of the tasks metadata object.
// Returns (nil, nil) when the object does not exist yet.
func (b *S3Backend) ReadTasksData() ([]byte, error) {
	data, err := b.getObject(b.key(b.tasksFile))
	if err != nil {
		return nil, fmt.Errorf("s3 backend ReadTasksData: %w", err)
	}
	return data, nil
}

// WriteTasksData replaces the tasks metadata object.
// S3 PutObject is atomic from the reader's perspective.
func (b *S3Backend) WriteTasksData(data []byte) error {
	if err := b.putObject(b.key(b.tasksFile), data); err != nil {
		return fmt.Errorf("s3 backend WriteTasksData: %w", err)
	}
	return nil
}

// ReadDetail returns the markdown detail content for the given docHash.
func (b *S3Backend) ReadDetail(docHash string) ([]byte, error) {
	rel := path.Join(b.docsDir, docHash+".md")
	data, err := b.getObject(b.key(rel))
	if err != nil {
		return nil, fmt.Errorf("s3 backend ReadDetail: %w", err)
	}
	if data == nil {
		return nil, fmt.Errorf("%w: %s", ErrNotFound, docHash)
	}
	return data, nil
}

// WriteDetail stores the markdown detail content for the given docHash.
func (b *S3Backend) WriteDetail(docHash string, data []byte) error {
	rel := b.docsDir + "/" + docHash + ".md"
	if err := b.putObject(b.key(rel), data); err != nil {
		return fmt.Errorf("s3 backend WriteDetail: %w", err)
	}
	return nil
}

// DeleteDetail removes the detail object for the given docHash.
// A missing object is treated as a no-op.
func (b *S3Backend) DeleteDetail(docHash string) error {
	rel := b.docsDir + "/" + docHash + ".md"
	ctx, cancel := b.ctx()
	defer cancel()

	_, err := b.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(b.cfg.Bucket),
		Key:    aws.String(b.key(rel)),
	})
	if err != nil {
		return fmt.Errorf("s3 backend DeleteDetail: %w", err)
	}
	return nil
}

// HealthCheck verifies that the configured bucket is accessible.
func (b *S3Backend) HealthCheck() error {
	ctx, cancel := b.ctx()
	defer cancel()

	if _, err := b.client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(b.cfg.Bucket),
	}); err != nil {
		return fmt.Errorf("s3 backend health check (bucket %q): %w", b.cfg.Bucket, err)
	}
	return nil
}
