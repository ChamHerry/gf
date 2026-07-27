//go:build integration

// Copyright GoFrame Author(https://goframe.org). All Rights Reserved.
//
// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT was not distributed with this file,
// You can obtain one at https://github.com/gogf/gf.

package s3

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/gogf/gf/v2/os/gcfg"
	"github.com/gogf/gf/v2/storage/gstorage"
)

func TestMinIOStreamingLifecycle(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	config := loadMinIOTestConfig(t, ctx)
	adapter, err := New(config)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	storage, err := gstorage.New(adapter)
	if err != nil {
		t.Fatalf("gstorage.New() error = %v", err)
	}
	bucket := fmt.Sprintf("gf-gstorage-%d", time.Now().UnixNano())
	if _, err = adapter.client.CreateBucket(ctx, &awss3.CreateBucketInput{Bucket: aws.String(bucket)}); err != nil {
		t.Fatalf("CreateBucket() error = %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		_, _ = adapter.client.DeleteBucket(cleanupCtx, &awss3.DeleteBucketInput{Bucket: aws.String(bucket)})
		_ = storage.Close(cleanupCtx)
	})

	largeBody := bytes.Repeat([]byte("gstorage-minio-stream-"), 400000)
	checksum := sha256HexIntegration(largeBody)
	request := gstorage.PutRequest{
		Bucket:      bucket,
		Key:         "tenant/run/large.bin",
		Body:        bytes.NewReader(largeBody),
		Size:        int64(len(largeBody)),
		Checksum:    checksum,
		ContentType: "application/octet-stream",
		Metadata:    map[string]string{"provenance": "minio-integration"},
	}
	created, err := storage.PutIfAbsent(ctx, request)
	if err != nil {
		t.Fatalf("PutIfAbsent() error = %v", err)
	}
	if created.Size != int64(len(largeBody)) || created.Checksum != checksum {
		t.Fatalf("PutIfAbsent() = %+v", created)
	}

	idempotentRequest := request
	idempotentRequest.Body = bytes.NewReader(largeBody)
	if _, err = storage.PutIfAbsent(ctx, idempotentRequest); err != nil {
		t.Fatalf("idempotent PutIfAbsent() error = %v", err)
	}
	conflictingBody := []byte("different")
	_, err = storage.PutIfAbsent(ctx, gstorage.PutRequest{
		Bucket:   bucket,
		Key:      request.Key,
		Body:     bytes.NewReader(conflictingBody),
		Size:     int64(len(conflictingBody)),
		Checksum: sha256HexIntegration(conflictingBody),
	})
	if !errors.Is(err, gstorage.ErrConflict) {
		t.Fatalf("conflicting PutIfAbsent() error = %v, want ErrConflict", err)
	}

	head, err := storage.Head(ctx, gstorage.HeadRequest{Bucket: bucket, Key: request.Key})
	if err != nil {
		t.Fatalf("Head() error = %v", err)
	}
	if head.Size != int64(len(largeBody)) || head.Checksum != checksum {
		t.Fatalf("Head() = %+v", head)
	}
	reader, getInfo, err := storage.Get(ctx, gstorage.GetRequest{Bucket: bucket, Key: request.Key})
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	actualBody, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	if err = reader.Close(); err != nil {
		t.Fatalf("response body Close() error = %v", err)
	}
	if !bytes.Equal(actualBody, largeBody) || getInfo.Checksum != checksum {
		t.Fatalf("Get() size/checksum = (%d, %q)", len(actualBody), getInfo.Checksum)
	}

	cancelledCtx, cancelNow := context.WithCancel(context.Background())
	cancelNow()
	if _, err = storage.Head(cancelledCtx, gstorage.HeadRequest{Bucket: bucket, Key: request.Key}); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled Head() error = %v", err)
	}
	if err = storage.Delete(ctx, gstorage.DeleteRequest{Bucket: bucket, Key: request.Key}); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if _, err = storage.Head(ctx, gstorage.HeadRequest{Bucket: bucket, Key: request.Key}); !errors.Is(err, gstorage.ErrNotFound) {
		t.Fatalf("Head() after delete error = %v, want ErrNotFound", err)
	}
}

func loadMinIOTestConfig(t *testing.T, ctx context.Context) *gstorage.Config {
	t.Helper()
	fixture := filepath.Join("testdata", "minio.yaml")
	fileAdapter, err := gcfg.NewAdapterFile(fixture)
	if err != nil {
		t.Fatalf("NewAdapterFile() error = %v", err)
	}
	configManager := gcfg.NewWithAdapter(fileAdapter)
	value, err := configManager.Get(ctx, "storage.integration")
	if err != nil {
		t.Fatalf("gcfg.Get() error = %v", err)
	}
	if value == nil {
		t.Fatalf("gcfg fixture %q has no storage.integration", fixture)
	}
	config, err := gstorage.ConfigFromMap(value.Map())
	if err != nil {
		t.Fatalf("ConfigFromMap() error = %v", err)
	}
	return config
}

func sha256HexIntegration(body []byte) string {
	digest := sha256.Sum256(body)
	return hex.EncodeToString(digest[:])
}
