// Copyright GoFrame Author(https://goframe.org). All Rights Reserved.
//
// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT was not distributed with this file,
// You can obtain one at https://github.com/gogf/gf.

package g

import "github.com/gogf/gf/v2/storage/gstorage"

// Storage 返回默认或具名对象存储实例。
func Storage(name ...string) *gstorage.Storage {
	return gstorage.Instance(name...)
}
