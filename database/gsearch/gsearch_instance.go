// Copyright GoFrame Author(https://goframe.org). All Rights Reserved.
//
// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this file,
// You can obtain one at https://github.com/gogf/gf.

// Instance cache management for search clients.

package gsearch

import (
	"context"

	"github.com/gogf/gf/v2/container/gmap"
	"github.com/gogf/gf/v2/internal/intlog"
)

var (
	// searchChecker checks whether the *Search is nil.
	searchChecker = func(v *Search) bool { return v == nil }

	// localInstances stores search client instances by group name.
	localInstances = gmap.NewKVMapWithChecker[string, *Search](searchChecker, true)
)

// Instance returns a search client instance with the specified group.
// If name is not passed, it returns a search instance with default configuration group.
func Instance(name ...string) *Search {
	group := DefaultGroupName
	if len(name) > 0 && name[0] != "" {
		group = name[0]
	}
	return localInstances.GetOrSetFuncLock(group, func() *Search {
		if config, ok := GetConfig(group); ok {
			search, err := New(config)
			if err != nil {
				intlog.Errorf(context.TODO(), `%+v`, err)
				return nil
			}
			return search
		}
		return nil
	})
}
