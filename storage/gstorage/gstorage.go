// Copyright GoFrame Author(https://goframe.org). All Rights Reserved.
//
// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT was not distributed with this file,
// You can obtain one at https://github.com/gogf/gf.

// Package gstorage 提供与具体对象存储实现无关的窄接口。
package gstorage

import (
	"context"
	"io"
	"time"

	"github.com/gogf/gf/v2/errors/gcode"
	"github.com/gogf/gf/v2/errors/gerror"
)

// Adapter 定义对象存储适配器必须实现的流式操作。
type Adapter interface {
	Put(ctx context.Context, request PutRequest) (ObjectInfo, error)
	PutIfAbsent(ctx context.Context, request PutRequest) (ObjectInfo, error)
	Get(ctx context.Context, request GetRequest) (io.ReadCloser, ObjectInfo, error)
	Head(ctx context.Context, request HeadRequest) (ObjectInfo, error)
	Delete(ctx context.Context, request DeleteRequest) error
	Close(ctx context.Context) error
}

// Storage 是对象存储 facade，只转发核心操作，不暴露底层 SDK。
type Storage struct {
	adapter Adapter
}

// New 使用显式适配器创建 Storage。
func New(adapter Adapter) (*Storage, error) {
	if adapter == nil {
		return nil, gerror.NewCode(gcode.CodeInvalidParameter, `storage adapter cannot be nil`)
	}
	return &Storage{adapter: adapter}, nil
}

// Put 写入或覆盖一个对象。输入流的关闭责任属于调用方。
func (s *Storage) Put(ctx context.Context, request PutRequest) (ObjectInfo, error) {
	if err := s.validate(); err != nil {
		return ObjectInfo{}, err
	}
	return s.adapter.Put(ctx, request)
}

// PutIfAbsent 仅在对象不存在时写入；同 checksum 的已有对象视为幂等成功。
func (s *Storage) PutIfAbsent(ctx context.Context, request PutRequest) (ObjectInfo, error) {
	if err := s.validate(); err != nil {
		return ObjectInfo{}, err
	}
	return s.adapter.PutIfAbsent(ctx, request)
}

// Get 返回对象读取流和对象信息。调用方必须关闭返回的读取流。
func (s *Storage) Get(ctx context.Context, request GetRequest) (io.ReadCloser, ObjectInfo, error) {
	if err := s.validate(); err != nil {
		return nil, ObjectInfo{}, err
	}
	return s.adapter.Get(ctx, request)
}

// Head 返回对象元信息而不读取对象内容。
func (s *Storage) Head(ctx context.Context, request HeadRequest) (ObjectInfo, error) {
	if err := s.validate(); err != nil {
		return ObjectInfo{}, err
	}
	return s.adapter.Head(ctx, request)
}

// Delete 删除一个对象。
func (s *Storage) Delete(ctx context.Context, request DeleteRequest) error {
	if err := s.validate(); err != nil {
		return err
	}
	return s.adapter.Delete(ctx, request)
}

// Close 释放适配器持有的资源。
func (s *Storage) Close(ctx context.Context) error {
	if err := s.validate(); err != nil {
		return err
	}
	return s.adapter.Close(ctx)
}

func (s *Storage) validate() error {
	if s == nil || s.adapter == nil {
		return gerror.NewCode(gcode.CodeInvalidConfiguration, `storage adapter is not configured`)
	}
	return nil
}

// PutRequest 描述一次流式对象写入。
type PutRequest struct {
	Bucket string
	Key    string
	// Body 由调用方持有并负责关闭，适配器只在请求期间读取。
	Body io.Reader
	// Size 是 Body 的精确字节数，必须大于或等于零。
	Size int64
	// Checksum 是小写或大写均可的 64 位 SHA-256 十六进制值；PutIfAbsent 必填。
	Checksum    string
	ContentType string
	// Metadata 是 S3 user metadata；checksum-sha256 为组件保留键。
	Metadata map[string]string
}

// GetRequest 描述一次对象读取。
type GetRequest struct {
	Bucket string
	Key    string
}

// HeadRequest 描述一次对象元信息读取。
type HeadRequest struct {
	Bucket string
	Key    string
}

// DeleteRequest 描述一次对象删除。
type DeleteRequest struct {
	Bucket string
	Key    string
}

// ObjectInfo 是不包含底层 SDK 类型的对象元信息。
type ObjectInfo struct {
	Bucket string
	Key    string
	Size   int64
	ETag   string
	// Checksum 始终使用小写 SHA-256 十六进制格式；未知时为空。
	Checksum     string
	ContentType  string
	Metadata     map[string]string
	LastModified time.Time
}
