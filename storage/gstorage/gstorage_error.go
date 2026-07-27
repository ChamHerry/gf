// Copyright GoFrame Author(https://goframe.org). All Rights Reserved.
//
// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT was not distributed with this file,
// You can obtain one at https://github.com/gogf/gf.

package gstorage

import "errors"

var (
	// ErrAlreadyExists 表示条件写命中了已有对象，但无法确认幂等。
	ErrAlreadyExists = errors.New("storage object already exists")
	// ErrNotFound 表示目标对象不存在。
	ErrNotFound = errors.New("storage object not found")
	// ErrConflict 表示已有对象与本次请求的 checksum 冲突。
	ErrConflict = errors.New("storage object conflict")
	// ErrClosed 表示适配器已经关闭。
	ErrClosed = errors.New("storage adapter is closed")
)
