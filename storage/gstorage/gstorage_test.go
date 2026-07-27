// Copyright GoFrame Author(https://goframe.org). All Rights Reserved.
//
// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT was not distributed with this file,
// You can obtain one at https://github.com/gogf/gf.

package gstorage_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gcfg"
	"github.com/gogf/gf/v2/storage/gstorage"
)

type fakeAdapter struct {
	mu      sync.Mutex
	objects map[string][]byte
	closed  bool
}

func newFakeAdapter() *fakeAdapter {
	return &fakeAdapter{objects: make(map[string][]byte)}
}

func (a *fakeAdapter) Put(_ context.Context, request gstorage.PutRequest) (gstorage.ObjectInfo, error) {
	body, err := io.ReadAll(request.Body)
	if err != nil {
		return gstorage.ObjectInfo{}, err
	}
	a.mu.Lock()
	a.objects[request.Bucket+"/"+request.Key] = body
	a.mu.Unlock()
	return objectInfo(request), nil
}

func (a *fakeAdapter) PutIfAbsent(_ context.Context, request gstorage.PutRequest) (gstorage.ObjectInfo, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	identity := request.Bucket + "/" + request.Key
	if _, ok := a.objects[identity]; ok {
		return gstorage.ObjectInfo{}, gstorage.ErrAlreadyExists
	}
	body, err := io.ReadAll(request.Body)
	if err != nil {
		return gstorage.ObjectInfo{}, err
	}
	a.objects[identity] = body
	return objectInfo(request), nil
}

func (a *fakeAdapter) Get(_ context.Context, request gstorage.GetRequest) (io.ReadCloser, gstorage.ObjectInfo, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	body, ok := a.objects[request.Bucket+"/"+request.Key]
	if !ok {
		return nil, gstorage.ObjectInfo{}, gstorage.ErrNotFound
	}
	return io.NopCloser(bytes.NewReader(body)), gstorage.ObjectInfo{
		Bucket: request.Bucket,
		Key:    request.Key,
		Size:   int64(len(body)),
	}, nil
}

func (a *fakeAdapter) Head(_ context.Context, request gstorage.HeadRequest) (gstorage.ObjectInfo, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	body, ok := a.objects[request.Bucket+"/"+request.Key]
	if !ok {
		return gstorage.ObjectInfo{}, gstorage.ErrNotFound
	}
	return gstorage.ObjectInfo{Bucket: request.Bucket, Key: request.Key, Size: int64(len(body))}, nil
}

func (a *fakeAdapter) Delete(_ context.Context, request gstorage.DeleteRequest) error {
	a.mu.Lock()
	delete(a.objects, request.Bucket+"/"+request.Key)
	a.mu.Unlock()
	return nil
}

func (a *fakeAdapter) Close(_ context.Context) error {
	a.mu.Lock()
	a.closed = true
	a.mu.Unlock()
	return nil
}

func TestStorageForwardsNarrowStreamingOperations(t *testing.T) {
	adapter := newFakeAdapter()
	storage, err := gstorage.New(adapter)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	body := []byte("streaming-body")
	checksum := sha256.Sum256(body)
	request := gstorage.PutRequest{
		Bucket:   "raw",
		Key:      "tenant/run.bin",
		Body:     bytes.NewReader(body),
		Size:     int64(len(body)),
		Checksum: hex.EncodeToString(checksum[:]),
	}
	if _, err = storage.Put(context.Background(), request); err != nil {
		t.Fatalf("Put() error = %v", err)
	}
	reader, info, err := storage.Get(context.Background(), gstorage.GetRequest{Bucket: "raw", Key: "tenant/run.bin"})
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	readBody, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	if err = reader.Close(); err != nil {
		t.Fatalf("Close body error = %v", err)
	}
	if !bytes.Equal(readBody, body) || info.Size != int64(len(body)) {
		t.Fatalf("Get() = (%q, %+v), want %q", readBody, info, body)
	}
	if _, err = storage.Head(context.Background(), gstorage.HeadRequest{Bucket: "raw", Key: "tenant/run.bin"}); err != nil {
		t.Fatalf("Head() error = %v", err)
	}
	if err = storage.Delete(context.Background(), gstorage.DeleteRequest{Bucket: "raw", Key: "tenant/run.bin"}); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if _, err = storage.Head(context.Background(), gstorage.HeadRequest{Bucket: "raw", Key: "tenant/run.bin"}); !errors.Is(err, gstorage.ErrNotFound) {
		t.Fatalf("Head() error = %v, want ErrNotFound", err)
	}
	if err = storage.Close(context.Background()); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}

func TestNamedInstanceLoadsOnlyFromGcfgYAMLAdapter(t *testing.T) {
	const yamlConfig = `
storage:
  archive:
    type: fake-yaml
    endpoint: 127.0.0.1:19000
    region: us-east-1
    pathStyle: true
    tls: false
    credentials:
      accessKey: test-access
      secretKey: test-secret
    timeout: 5s
`
	adapterContent, err := gcfg.NewAdapterContent(yamlConfig)
	if err != nil {
		t.Fatalf("NewAdapterContent() error = %v", err)
	}
	configInstance := gcfg.Instance()
	originalAdapter := configInstance.GetAdapter()
	configInstance.SetAdapter(adapterContent)
	gstorage.ClearInstances()
	t.Cleanup(func() {
		gstorage.ClearInstances()
		configInstance.SetAdapter(originalAdapter)
	})
	var received *gstorage.Config
	if err = gstorage.RegisterAdapterFunc("fake-yaml", func(config *gstorage.Config) (gstorage.Adapter, error) {
		received = config
		return newFakeAdapter(), nil
	}); err != nil {
		t.Fatalf("RegisterAdapterFunc() error = %v", err)
	}
	first := g.Storage("archive")
	second, err := gstorage.Open("archive")
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	if first == nil || first != second {
		t.Fatalf("named instance was not cached: first=%p second=%p", first, second)
	}
	if received == nil || received.Endpoint != "127.0.0.1:19000" || received.Timeout != 5*time.Second {
		t.Fatalf("loaded config = %+v", received)
	}
}

func TestNewRejectsNilAdapter(t *testing.T) {
	if _, err := gstorage.New(nil); err == nil {
		t.Fatal("New(nil) error = nil")
	}
}

func objectInfo(request gstorage.PutRequest) gstorage.ObjectInfo {
	return gstorage.ObjectInfo{
		Bucket:   request.Bucket,
		Key:      request.Key,
		Size:     request.Size,
		Checksum: request.Checksum,
	}
}
