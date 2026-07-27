// Copyright GoFrame Author(https://goframe.org). All Rights Reserved.
//
// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT was not distributed with this file,
// You can obtain one at https://github.com/gogf/gf.

package s3_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/aws/smithy-go"
	storageS3 "github.com/gogf/gf/v2/contrib/storage/s3"
	"github.com/gogf/gf/v2/storage/gstorage"
)

type fakeS3Server struct {
	mu                 sync.Mutex
	objects            map[string][]byte
	metadata           map[string]map[string]string
	lastIfNoneMatch    string
	lastChecksumSHA256 string
	lastAuthorization  string
}

type trackingReadCloser struct {
	*bytes.Reader
	closed bool
}

func (r *trackingReadCloser) Close() error {
	r.closed = true
	return nil
}

func newFakeS3Server() *fakeS3Server {
	return &fakeS3Server{
		objects:  make(map[string][]byte),
		metadata: make(map[string]map[string]string),
	}
}

func (s *fakeS3Server) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	identity := strings.TrimPrefix(request.URL.Path, "/")
	if strings.HasSuffix(identity, "/cancel") {
		<-request.Context().Done()
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	switch request.Method {
	case http.MethodPut:
		s.lastIfNoneMatch = request.Header.Get("If-None-Match")
		s.lastChecksumSHA256 = request.Header.Get("X-Amz-Checksum-Sha256")
		s.lastAuthorization = request.Header.Get("Authorization")
		if s.lastIfNoneMatch == "*" {
			if _, ok := s.objects[identity]; ok {
				if strings.HasSuffix(identity, "/ambiguous.bin") {
					connection, _, hijackErr := writer.(http.Hijacker).Hijack()
					if hijackErr == nil {
						_ = connection.Close()
					}
					return
				}
				writeS3Error(writer, http.StatusPreconditionFailed, "PreconditionFailed")
				return
			}
		}
		body, err := io.ReadAll(request.Body)
		if err != nil {
			writeS3Error(writer, http.StatusInternalServerError, "ReadFailed")
			return
		}
		s.objects[identity] = body
		metadata := make(map[string]string)
		for key, values := range request.Header {
			if strings.HasPrefix(strings.ToLower(key), "x-amz-meta-") && len(values) > 0 {
				metadata[strings.TrimPrefix(strings.ToLower(key), "x-amz-meta-")] = values[0]
			}
		}
		s.metadata[identity] = metadata
		writer.Header().Set("ETag", `"fake-etag"`)
		writer.WriteHeader(http.StatusOK)
	case http.MethodHead:
		body, ok := s.objects[identity]
		if !ok {
			writeS3Error(writer, http.StatusNotFound, "NoSuchKey")
			return
		}
		writeObjectHeaders(writer.Header(), body, s.metadata[identity])
		writer.WriteHeader(http.StatusOK)
	case http.MethodGet:
		body, ok := s.objects[identity]
		if !ok {
			writeS3Error(writer, http.StatusNotFound, "NoSuchKey")
			return
		}
		writeObjectHeaders(writer.Header(), body, s.metadata[identity])
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write(body)
	case http.MethodDelete:
		delete(s.objects, identity)
		delete(s.metadata, identity)
		writer.WriteHeader(http.StatusNoContent)
	default:
		writeS3Error(writer, http.StatusMethodNotAllowed, "MethodNotAllowed")
	}
}

func TestAdapterPutIfAbsentUsesConditionAndChecksumMetadata(t *testing.T) {
	backend := newFakeS3Server()
	server := httptest.NewServer(backend)
	defer server.Close()
	adapter := newTestAdapter(t, server.URL)
	storage, err := gstorage.New(adapter)
	if err != nil {
		t.Fatalf("gstorage.New() error = %v", err)
	}
	body := []byte("first payload")
	checksum := sha256Hex(body)
	request := putRequest("raw", "tenant/run.bin", body, checksum)
	first, err := storage.PutIfAbsent(context.Background(), request)
	if err != nil {
		t.Fatalf("first PutIfAbsent() error = %v", err)
	}
	if first.Checksum != checksum || first.Size != int64(len(body)) {
		t.Fatalf("first PutIfAbsent() info = %+v", first)
	}
	if backend.lastIfNoneMatch != "*" || backend.lastChecksumSHA256 == "" {
		t.Fatalf("conditional headers = (%q, %q)", backend.lastIfNoneMatch, backend.lastChecksumSHA256)
	}
	if !strings.HasPrefix(backend.lastAuthorization, "AWS4-HMAC-SHA256 ") {
		t.Fatalf("Authorization header = %q, want SDK SigV4", backend.lastAuthorization)
	}
	second, err := storage.PutIfAbsent(context.Background(), putRequest("raw", "tenant/run.bin", body, checksum))
	if err != nil {
		t.Fatalf("idempotent PutIfAbsent() error = %v", err)
	}
	if second.Checksum != checksum {
		t.Fatalf("idempotent PutIfAbsent() checksum = %q", second.Checksum)
	}
	conflictingBody := []byte("different payload")
	_, err = storage.PutIfAbsent(
		context.Background(),
		putRequest("raw", "tenant/run.bin", conflictingBody, sha256Hex(conflictingBody)),
	)
	if !errors.Is(err, gstorage.ErrConflict) {
		t.Fatalf("conflicting PutIfAbsent() error = %v, want ErrConflict", err)
	}
}

func TestAdapterReconcilesAmbiguousConditionalPutFailure(t *testing.T) {
	backend := newFakeS3Server()
	server := httptest.NewServer(backend)
	defer server.Close()
	adapter := newTestAdapter(t, server.URL)
	body := []byte("ambiguous payload")
	checksum := sha256Hex(body)
	if _, err := adapter.PutIfAbsent(context.Background(), putRequest("raw", "ambiguous.bin", body, checksum)); err != nil {
		t.Fatalf("first PutIfAbsent() error = %v", err)
	}
	info, err := adapter.PutIfAbsent(
		context.Background(),
		putRequest("raw", "ambiguous.bin", body, checksum),
	)
	if err != nil {
		t.Fatalf("reconciled PutIfAbsent() error = %v", err)
	}
	if info.Checksum != checksum {
		t.Fatalf("reconciled PutIfAbsent() checksum = %q", info.Checksum)
	}
}

func TestAdapterGetHeadDeleteAndClose(t *testing.T) {
	backend := newFakeS3Server()
	server := httptest.NewServer(backend)
	defer server.Close()
	adapter := newTestAdapter(t, server.URL)
	body := []byte("streaming response")
	checksum := sha256Hex(body)
	if _, err := adapter.Put(context.Background(), putRequest("raw", "stream.bin", body, checksum)); err != nil {
		t.Fatalf("Put() error = %v", err)
	}
	info, err := adapter.Head(context.Background(), gstorage.HeadRequest{Bucket: "raw", Key: "stream.bin"})
	if err != nil {
		t.Fatalf("Head() error = %v", err)
	}
	if info.Size != int64(len(body)) || info.Checksum != checksum {
		t.Fatalf("Head() = %+v", info)
	}
	reader, getInfo, err := adapter.Get(context.Background(), gstorage.GetRequest{Bucket: "raw", Key: "stream.bin"})
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	readBody, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	if err = reader.Close(); err != nil {
		t.Fatalf("response body Close() error = %v", err)
	}
	if !bytes.Equal(readBody, body) || getInfo.Checksum != checksum {
		t.Fatalf("Get() = (%q, %+v)", readBody, getInfo)
	}
	if err = adapter.Delete(context.Background(), gstorage.DeleteRequest{Bucket: "raw", Key: "stream.bin"}); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if _, err = adapter.Head(context.Background(), gstorage.HeadRequest{Bucket: "raw", Key: "stream.bin"}); !errors.Is(err, gstorage.ErrNotFound) {
		t.Fatalf("Head() after delete error = %v, want ErrNotFound", err)
	}
	if err = adapter.Close(context.Background()); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if _, err = adapter.Head(context.Background(), gstorage.HeadRequest{Bucket: "raw", Key: "stream.bin"}); !errors.Is(err, gstorage.ErrClosed) {
		t.Fatalf("Head() after close error = %v, want ErrClosed", err)
	}
	if err = adapter.Close(context.Background()); err != nil {
		t.Fatalf("second Close() error = %v", err)
	}
}

func TestAdapterHonorsContextCancellation(t *testing.T) {
	backend := newFakeS3Server()
	server := httptest.NewServer(backend)
	defer server.Close()
	adapter := newTestAdapter(t, server.URL)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := adapter.Head(ctx, gstorage.HeadRequest{Bucket: "raw", Key: "cancel"})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Head() error = %v, want context.Canceled", err)
	}
}

func TestAdapterValidatesConfigAndPutRequest(t *testing.T) {
	valid := testConfig("http://127.0.0.1:9000")
	valid.Timeout = 0
	if _, err := storageS3.New(valid); err == nil {
		t.Fatal("New() with zero timeout error = nil")
	}
	valid = testConfig("https://127.0.0.1:9000")
	if _, err := storageS3.New(valid); err == nil {
		t.Fatal("New() with tls mismatch error = nil")
	}
	backend := newFakeS3Server()
	server := httptest.NewServer(backend)
	defer server.Close()
	adapter := newTestAdapter(t, server.URL)
	_, err := adapter.PutIfAbsent(context.Background(), gstorage.PutRequest{
		Bucket: "raw",
		Key:    "missing-checksum",
		Body:   bytes.NewReader(nil),
		Size:   0,
	})
	if err == nil {
		t.Fatal("PutIfAbsent() without checksum error = nil")
	}
}

func TestAdapterDoesNotExposeSDKErrorType(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writeS3Error(writer, http.StatusInternalServerError, "InternalError")
	}))
	defer server.Close()
	adapter := newTestAdapter(t, server.URL)
	_, err := adapter.Head(context.Background(), gstorage.HeadRequest{Bucket: "raw", Key: "failure"})
	if err == nil {
		t.Fatal("Head() error = nil")
	}
	var apiError smithy.APIError
	if errors.As(err, &apiError) {
		t.Fatalf("Head() exposed SDK error type %T", apiError)
	}
}

func TestAdapterLeavesInputStreamOwnershipToCaller(t *testing.T) {
	backend := newFakeS3Server()
	server := httptest.NewServer(backend)
	defer server.Close()
	adapter := newTestAdapter(t, server.URL)
	body := []byte("caller-owned-stream")
	reader := &trackingReadCloser{Reader: bytes.NewReader(body)}
	_, err := adapter.Put(context.Background(), gstorage.PutRequest{
		Bucket:   "raw",
		Key:      "owned.bin",
		Body:     reader,
		Size:     int64(len(body)),
		Checksum: sha256Hex(body),
	})
	if err != nil {
		t.Fatalf("Put() error = %v", err)
	}
	if reader.closed {
		t.Fatal("Put() closed caller-owned input stream")
	}
}

func newTestAdapter(t *testing.T, endpoint string) *storageS3.Adapter {
	t.Helper()
	adapter, err := storageS3.New(testConfig(endpoint))
	if err != nil {
		t.Fatalf("s3.New() error = %v", err)
	}
	return adapter
}

func testConfig(endpoint string) *gstorage.Config {
	return &gstorage.Config{
		Type:      "s3",
		Endpoint:  endpoint,
		Region:    "us-east-1",
		PathStyle: true,
		TLS:       false,
		Credentials: gstorage.Credentials{
			AccessKey: "test-access",
			SecretKey: "test-secret",
		},
		Timeout: 5 * time.Second,
	}
}

func putRequest(bucket string, key string, body []byte, checksum string) gstorage.PutRequest {
	return gstorage.PutRequest{
		Bucket:      bucket,
		Key:         key,
		Body:        bytes.NewReader(body),
		Size:        int64(len(body)),
		Checksum:    checksum,
		ContentType: "application/octet-stream",
		Metadata:    map[string]string{"provenance": "unit-test"},
	}
}

func sha256Hex(body []byte) string {
	digest := sha256.Sum256(body)
	return hex.EncodeToString(digest[:])
}

func writeObjectHeaders(header http.Header, body []byte, metadata map[string]string) {
	header.Set("Content-Length", fmt.Sprintf("%d", len(body)))
	header.Set("Content-Type", "application/octet-stream")
	header.Set("ETag", `"fake-etag"`)
	header.Set("Last-Modified", time.Now().UTC().Format(http.TimeFormat))
	for key, value := range metadata {
		header.Set("X-Amz-Meta-"+key, value)
	}
}

func writeS3Error(writer http.ResponseWriter, status int, code string) {
	writer.Header().Set("Content-Type", "application/xml")
	writer.WriteHeader(status)
	_, _ = fmt.Fprintf(writer, "<Error><Code>%s</Code><Message>%s</Message></Error>", code, code)
}
