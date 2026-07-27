// Copyright GoFrame Author(https://goframe.org). All Rights Reserved.
//
// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT was not distributed with this file,
// You can obtain one at https://github.com/gogf/gf.

package s3

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"io"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/gogf/gf/v2/errors/gcode"
	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/storage/gstorage"
)

const checksumMetadataKey = "checksum-sha256"

// Put 写入或覆盖一个对象。
func (a *Adapter) Put(ctx context.Context, request gstorage.PutRequest) (gstorage.ObjectInfo, error) {
	return a.put(ctx, request, false)
}

// PutIfAbsent 使用 If-None-Match: * 防止覆盖，并用 checksum metadata 判定幂等或冲突。
func (a *Adapter) PutIfAbsent(ctx context.Context, request gstorage.PutRequest) (gstorage.ObjectInfo, error) {
	request.Checksum = strings.ToLower(strings.TrimSpace(request.Checksum))
	if request.Checksum == "" {
		return gstorage.ObjectInfo{}, gerror.NewCode(gcode.CodeInvalidParameter, `checksum is required for PutIfAbsent`)
	}
	info, err := a.put(ctx, request, true)
	if err == nil {
		return info, nil
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) || errors.Is(err, gstorage.ErrClosed) {
		return gstorage.ObjectInfo{}, err
	}
	operationFailed := gerror.HasCode(err, gcode.CodeOperationFailed)
	if !errors.Is(err, gstorage.ErrAlreadyExists) && !errors.Is(err, gstorage.ErrConflict) && !operationFailed {
		return gstorage.ObjectInfo{}, err
	}
	// 条件 PUT 可能在服务端提前返回 412 时只暴露传输层断连。
	// 对所有歧义写失败执行 HEAD 收敛，不重放已经消费过的输入流。
	existing, headErr := a.Head(ctx, gstorage.HeadRequest{Bucket: request.Bucket, Key: request.Key})
	if headErr != nil {
		return gstorage.ObjectInfo{}, err
	}
	if strings.EqualFold(existing.Checksum, request.Checksum) {
		return existing, nil
	}
	return gstorage.ObjectInfo{}, gerror.Wrapf(
		gstorage.ErrConflict,
		`storage object %q/%q already exists with a different checksum`,
		request.Bucket,
		request.Key,
	)
}

func (a *Adapter) put(ctx context.Context, request gstorage.PutRequest, ifAbsent bool) (gstorage.ObjectInfo, error) {
	if err := a.ensureOpen(); err != nil {
		return gstorage.ObjectInfo{}, err
	}
	checksum, checksumBase64, metadata, err := validatePutRequest(request)
	if err != nil {
		return gstorage.ObjectInfo{}, err
	}
	input := &awss3.PutObjectInput{
		Bucket:        aws.String(strings.TrimSpace(request.Bucket)),
		Key:           aws.String(strings.TrimSpace(request.Key)),
		Body:          request.Body,
		ContentLength: aws.Int64(request.Size),
		Metadata:      metadata,
	}
	if request.ContentType = strings.TrimSpace(request.ContentType); request.ContentType != "" {
		input.ContentType = aws.String(request.ContentType)
	}
	if checksumBase64 != "" {
		input.ChecksumAlgorithm = types.ChecksumAlgorithmSha256
		input.ChecksumSHA256 = aws.String(checksumBase64)
	}
	if ifAbsent {
		input.IfNoneMatch = aws.String("*")
	}
	output, err := a.client.PutObject(ctx, input)
	if err != nil {
		return gstorage.ObjectInfo{}, translateError(ctx, "put", request.Bucket, request.Key, err)
	}
	return gstorage.ObjectInfo{
		Bucket:      strings.TrimSpace(request.Bucket),
		Key:         strings.TrimSpace(request.Key),
		Size:        request.Size,
		ETag:        trimETag(aws.ToString(output.ETag)),
		Checksum:    checksum,
		ContentType: request.ContentType,
		Metadata:    cloneMetadata(metadata),
	}, nil
}

// Get 返回对象读取流。调用方必须关闭 body，适配器不会提前缓存整个对象。
func (a *Adapter) Get(ctx context.Context, request gstorage.GetRequest) (io.ReadCloser, gstorage.ObjectInfo, error) {
	if err := a.ensureOpen(); err != nil {
		return nil, gstorage.ObjectInfo{}, err
	}
	if err := validateIdentity(request.Bucket, request.Key); err != nil {
		return nil, gstorage.ObjectInfo{}, err
	}
	output, err := a.client.GetObject(ctx, &awss3.GetObjectInput{
		Bucket: aws.String(strings.TrimSpace(request.Bucket)),
		Key:    aws.String(strings.TrimSpace(request.Key)),
	})
	if err != nil {
		return nil, gstorage.ObjectInfo{}, translateError(ctx, "get", request.Bucket, request.Key, err)
	}
	if output.Body == nil {
		return nil, gstorage.ObjectInfo{}, gerror.NewCode(gcode.CodeOperationFailed, `s3 get returned an empty body`)
	}
	metadata := normalizeMetadata(output.Metadata)
	return output.Body, gstorage.ObjectInfo{
		Bucket:       strings.TrimSpace(request.Bucket),
		Key:          strings.TrimSpace(request.Key),
		Size:         aws.ToInt64(output.ContentLength),
		ETag:         trimETag(aws.ToString(output.ETag)),
		Checksum:     checksumFrom(metadata, aws.ToString(output.ChecksumSHA256)),
		ContentType:  aws.ToString(output.ContentType),
		Metadata:     metadata,
		LastModified: aws.ToTime(output.LastModified),
	}, nil
}

// Head 返回对象元信息。
func (a *Adapter) Head(ctx context.Context, request gstorage.HeadRequest) (gstorage.ObjectInfo, error) {
	if err := a.ensureOpen(); err != nil {
		return gstorage.ObjectInfo{}, err
	}
	if err := validateIdentity(request.Bucket, request.Key); err != nil {
		return gstorage.ObjectInfo{}, err
	}
	output, err := a.client.HeadObject(ctx, &awss3.HeadObjectInput{
		Bucket: aws.String(strings.TrimSpace(request.Bucket)),
		Key:    aws.String(strings.TrimSpace(request.Key)),
	})
	if err != nil {
		return gstorage.ObjectInfo{}, translateError(ctx, "head", request.Bucket, request.Key, err)
	}
	metadata := normalizeMetadata(output.Metadata)
	return gstorage.ObjectInfo{
		Bucket:       strings.TrimSpace(request.Bucket),
		Key:          strings.TrimSpace(request.Key),
		Size:         aws.ToInt64(output.ContentLength),
		ETag:         trimETag(aws.ToString(output.ETag)),
		Checksum:     checksumFrom(metadata, aws.ToString(output.ChecksumSHA256)),
		ContentType:  aws.ToString(output.ContentType),
		Metadata:     metadata,
		LastModified: aws.ToTime(output.LastModified),
	}, nil
}

// Delete 删除对象；S3 对不存在对象的删除保持幂等。
func (a *Adapter) Delete(ctx context.Context, request gstorage.DeleteRequest) error {
	if err := a.ensureOpen(); err != nil {
		return err
	}
	if err := validateIdentity(request.Bucket, request.Key); err != nil {
		return err
	}
	_, err := a.client.DeleteObject(ctx, &awss3.DeleteObjectInput{
		Bucket: aws.String(strings.TrimSpace(request.Bucket)),
		Key:    aws.String(strings.TrimSpace(request.Key)),
	})
	if err != nil {
		return translateError(ctx, "delete", request.Bucket, request.Key, err)
	}
	return nil
}

// Close 关闭空闲连接并拒绝后续操作；重复调用是幂等的。
func (a *Adapter) Close(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if a == nil || a.closed.Swap(true) {
		return nil
	}
	if a.transport != nil {
		a.transport.CloseIdleConnections()
	}
	return nil
}

func (a *Adapter) ensureOpen() error {
	if a == nil || a.client == nil {
		return gerror.NewCode(gcode.CodeInvalidConfiguration, `s3 adapter is not configured`)
	}
	if a.closed.Load() {
		return gstorage.ErrClosed
	}
	return nil
}

func validatePutRequest(request gstorage.PutRequest) (string, string, map[string]string, error) {
	if err := validateIdentity(request.Bucket, request.Key); err != nil {
		return "", "", nil, err
	}
	if request.Body == nil {
		return "", "", nil, gerror.NewCode(gcode.CodeInvalidParameter, `storage body is required`)
	}
	if request.Size < 0 {
		return "", "", nil, gerror.NewCode(gcode.CodeInvalidParameter, `storage size must be zero or greater`)
	}
	metadata := make(map[string]string, len(request.Metadata)+1)
	for key, value := range request.Metadata {
		normalizedKey := strings.ToLower(strings.TrimSpace(key))
		if normalizedKey == "" {
			return "", "", nil, gerror.NewCode(gcode.CodeInvalidParameter, `storage metadata key cannot be empty`)
		}
		if normalizedKey == checksumMetadataKey {
			return "", "", nil, gerror.NewCodef(gcode.CodeInvalidParameter, `metadata key %q is reserved`, checksumMetadataKey)
		}
		metadata[normalizedKey] = value
	}
	checksum := strings.ToLower(strings.TrimSpace(request.Checksum))
	if checksum == "" {
		return "", "", metadata, nil
	}
	rawChecksum, err := hex.DecodeString(checksum)
	if err != nil || len(rawChecksum) != sha256.Size {
		return "", "", nil, gerror.NewCode(gcode.CodeInvalidParameter, `checksum must be a 64-character SHA-256 hex string`)
	}
	metadata[checksumMetadataKey] = checksum
	return checksum, base64.StdEncoding.EncodeToString(rawChecksum), metadata, nil
}

func validateIdentity(bucket string, key string) error {
	if strings.TrimSpace(bucket) == "" {
		return gerror.NewCode(gcode.CodeInvalidParameter, `storage bucket is required`)
	}
	if strings.TrimSpace(key) == "" {
		return gerror.NewCode(gcode.CodeInvalidParameter, `storage key is required`)
	}
	return nil
}

func checksumFrom(metadata map[string]string, checksumBase64 string) string {
	if checksum := strings.ToLower(strings.TrimSpace(metadata[checksumMetadataKey])); checksum != "" {
		return checksum
	}
	decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(checksumBase64))
	if err != nil || len(decoded) != sha256.Size {
		return ""
	}
	return hex.EncodeToString(decoded)
}

func normalizeMetadata(metadata map[string]string) map[string]string {
	normalized := make(map[string]string, len(metadata))
	for key, value := range metadata {
		normalized[strings.ToLower(strings.TrimSpace(key))] = value
	}
	return normalized
}

func cloneMetadata(metadata map[string]string) map[string]string {
	cloned := make(map[string]string, len(metadata))
	for key, value := range metadata {
		cloned[key] = value
	}
	return cloned
}

func trimETag(etag string) string {
	return strings.Trim(strings.TrimSpace(etag), `"`)
}
