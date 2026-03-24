package store

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// mockS3Client implements the s3API interface used by S3Backend.
type mockS3Client struct {
	objects      map[string][]byte
	headBucketFn func() error
}

func newMockS3() *mockS3Client {
	return &mockS3Client{objects: make(map[string][]byte)}
}

func (m *mockS3Client) GetObject(_ context.Context, params *s3.GetObjectInput, _ ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	key := aws.ToString(params.Key)
	data, ok := m.objects[key]
	if !ok {
		return nil, &types.NoSuchKey{}
	}
	return &s3.GetObjectOutput{
		Body: io.NopCloser(strings.NewReader(string(data))),
	}, nil
}

func (m *mockS3Client) PutObject(_ context.Context, params *s3.PutObjectInput, _ ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	key := aws.ToString(params.Key)
	data, err := io.ReadAll(params.Body)
	if err != nil {
		return nil, err
	}
	m.objects[key] = data
	return &s3.PutObjectOutput{}, nil
}

func (m *mockS3Client) HeadBucket(_ context.Context, _ *s3.HeadBucketInput, _ ...func(*s3.Options)) (*s3.HeadBucketOutput, error) {
	if m.headBucketFn != nil {
		return nil, m.headBucketFn()
	}
	return &s3.HeadBucketOutput{}, nil
}

func newTestS3Backend(t *testing.T) (*S3Backend, *mockS3Client) {
	t.Helper()
	mock := newMockS3()
	cfg := S3Config{Bucket: "test-bucket"}
	b := NewS3BackendWithClient(mock, cfg, 0)
	return b, mock
}

func TestS3Backend_ReadTasksDataEmpty(t *testing.T) {
	b, _ := newTestS3Backend(t)
	data, err := b.ReadTasksData()
	if err != nil {
		t.Fatalf("ReadTasksData on empty store: %v", err)
	}
	if data != nil {
		t.Errorf("expected nil data for empty store, got %q", data)
	}
}

func TestS3Backend_WriteAndReadTasksData(t *testing.T) {
	b, _ := newTestS3Backend(t)
	payload := []byte(`{"id":"T-1"}` + "\n")

	if err := b.WriteTasksData(payload); err != nil {
		t.Fatalf("WriteTasksData: %v", err)
	}
	got, err := b.ReadTasksData()
	if err != nil {
		t.Fatalf("ReadTasksData: %v", err)
	}
	if string(got) != string(payload) {
		t.Errorf("round-trip mismatch: got %q, want %q", got, payload)
	}
}

func TestS3Backend_WriteAndReadDetail(t *testing.T) {
	b, _ := newTestS3Backend(t)
	content := []byte("# Detail\n\nContent.")

	if err := b.WriteDetail("abc123", content); err != nil {
		t.Fatalf("WriteDetail: %v", err)
	}
	got, err := b.ReadDetail("abc123")
	if err != nil {
		t.Fatalf("ReadDetail: %v", err)
	}
	if string(got) != string(content) {
		t.Errorf("detail round-trip mismatch: got %q, want %q", got, content)
	}
}

func TestS3Backend_ReadDetailNotFound(t *testing.T) {
	b, _ := newTestS3Backend(t)
	_, err := b.ReadDetail("missing")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestS3Backend_PrefixIsApplied(t *testing.T) {
	mock := newMockS3()
	cfg := S3Config{Bucket: "bucket", Prefix: "tssk"}
	b := NewS3BackendWithClient(mock, cfg, 0)

	if err := b.WriteTasksData([]byte("data\n")); err != nil {
		t.Fatalf("WriteTasksData: %v", err)
	}

	// The mock should have stored the data under the prefixed key.
	if _, ok := mock.objects["tssk/tasks.jsonl"]; !ok {
		t.Errorf("expected key 'tssk/tasks.jsonl' in mock, got %v", mock.objects)
	}
}

func TestS3Backend_PrefixWithTrailingSlash(t *testing.T) {
	mock := newMockS3()
	cfg := S3Config{Bucket: "bucket", Prefix: "tssk/"}
	b := NewS3BackendWithClient(mock, cfg, 0)

	if err := b.WriteTasksData([]byte("x\n")); err != nil {
		t.Fatalf("WriteTasksData: %v", err)
	}
	if _, ok := mock.objects["tssk/tasks.jsonl"]; !ok {
		t.Errorf("expected key 'tssk/tasks.jsonl', got %v", mock.objects)
	}
}

func TestS3Backend_HealthCheckOK(t *testing.T) {
	b, _ := newTestS3Backend(t)
	if err := b.HealthCheck(); err != nil {
		t.Errorf("HealthCheck should succeed: %v", err)
	}
}

func TestS3Backend_HealthCheckFails(t *testing.T) {
	mock := newMockS3()
	mock.headBucketFn = func() error { return errors.New("access denied") }
	cfg := S3Config{Bucket: "bucket"}
	b := NewS3BackendWithClient(mock, cfg, 0)

	if err := b.HealthCheck(); err == nil {
		t.Error("expected HealthCheck to fail")
	}
}

// TestS3BackendIntegration exercises a full Store workflow via the S3 backend
// using a mock S3 client.
func TestS3BackendIntegration(t *testing.T) {
	mock := newMockS3()
	cfg := S3Config{Bucket: "bucket"}
	backend := NewS3BackendWithClient(mock, cfg, 0)
	s := NewWithBackend(backend)

	tk, err := s.Add("S3 task", "detail content", nil)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	got, err := s.Get(tk.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Title != "S3 task" {
		t.Errorf("unexpected title: %q", got.Title)
	}

	detail, err := s.ReadDetail(tk)
	if err != nil {
		t.Fatalf("ReadDetail: %v", err)
	}
	if detail != "detail content" {
		t.Errorf("unexpected detail: %q", detail)
	}
}
