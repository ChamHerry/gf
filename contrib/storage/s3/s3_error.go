// Copyright GoFrame Author(https://goframe.org). All Rights Reserved.
//
// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT was not distributed with this file,
// You can obtain one at https://github.com/gogf/gf.

package s3

import (
	"context"
	"errors"
	"strings"

	"github.com/aws/smithy-go"
	smithyhttp "github.com/aws/smithy-go/transport/http"
	"github.com/gogf/gf/v2/errors/gcode"
	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/storage/gstorage"
)

func translateError(ctx context.Context, operation string, bucket string, key string, source error) error {
	if ctxErr := ctx.Err(); ctxErr != nil {
		return ctxErr
	}
	if errors.Is(source, context.Canceled) {
		return context.Canceled
	}
	if errors.Is(source, context.DeadlineExceeded) {
		return context.DeadlineExceeded
	}
	var (
		apiError      smithy.APIError
		responseError *smithyhttp.ResponseError
		code          string
		status        int
	)
	if errors.As(source, &apiError) {
		code = strings.ToLower(strings.TrimSpace(apiError.ErrorCode()))
	}
	if errors.As(source, &responseError) {
		status = responseError.HTTPStatusCode()
	}
	identity := strings.TrimSpace(bucket) + "/" + strings.TrimSpace(key)
	switch {
	case status == 404 || code == "nosuchkey" || code == "notfound":
		return gerror.Wrapf(gstorage.ErrNotFound, `s3 %s object %q failed`, operation, identity)
	case status == 412 || code == "preconditionfailed":
		return gerror.Wrapf(gstorage.ErrAlreadyExists, `s3 %s object %q failed`, operation, identity)
	case status == 409 || code == "conditionalrequestconflict" || code == "conflict":
		return gerror.Wrapf(gstorage.ErrConflict, `s3 %s object %q failed`, operation, identity)
	default:
		// 只保留稳定文字，不把 SDK error 放入 unwrap 链。
		return gerror.NewCodef(
			gcode.CodeOperationFailed,
			`s3 %s object %q failed (status=%d code=%s): %s`,
			operation,
			identity,
			status,
			code,
			source.Error(),
		)
	}
}
