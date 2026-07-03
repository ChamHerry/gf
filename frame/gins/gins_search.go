// Copyright GoFrame Author(https://goframe.org). All Rights Reserved.
//
// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT was not distributed with this file,
// You can obtain one at https://github.com/gogf/gf.

// Search instance creation and configuration loading.

package gins

import (
	"context"
	"fmt"

	"github.com/gogf/gf/v2/database/gsearch"
	"github.com/gogf/gf/v2/errors/gcode"
	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/internal/consts"
	"github.com/gogf/gf/v2/internal/instance"
	"github.com/gogf/gf/v2/internal/intlog"
	"github.com/gogf/gf/v2/util/gconv"
	"github.com/gogf/gf/v2/util/gutil"
)

// Search returns an instance of search client with specified configuration group name.
// Note that it panics if any error occurs duration instance creating.
func Search(name ...string) *gsearch.Search {
	var (
		err   error
		ctx   = context.Background()
		group = gsearch.DefaultGroupName
	)
	if len(name) > 0 && name[0] != "" {
		group = name[0]
	}
	instanceKey := fmt.Sprintf("%s.%s", frameCoreComponentNameSearch, group)
	result := instance.GetOrSetFuncLock(instanceKey, func() any {
		// If already configured, it returns the search instance.
		if _, ok := gsearch.GetConfig(group); ok {
			return gsearch.Instance(group)
		}
		if Config().Available(ctx) {
			var (
				configMap    map[string]any
				searchConfig *gsearch.Config
				searchClient *gsearch.Search
			)
			if configMap, err = Config().Data(ctx); err != nil {
				intlog.Errorf(ctx, `retrieve config data map failed: %+v`, err)
			}
			if _, v := gutil.MapPossibleItemByKey(configMap, consts.ConfigNodeNameSearch); v != nil {
				configMap = gconv.Map(v)
			}
			if len(configMap) > 0 {
				if v, ok := configMap[group]; ok {
					if searchConfig, err = gsearch.ConfigFromMap(gconv.Map(v)); err != nil {
						panic(err)
					}
				} else {
					intlog.Printf(ctx, `missing configuration for search group "%s"`, group)
				}
			} else {
				intlog.Print(ctx, `missing configuration for search: "search" node not found`)
			}
			if searchClient, err = gsearch.New(searchConfig); err != nil {
				panic(err)
			}
			return searchClient
		}
		panic(gerror.NewCode(
			gcode.CodeMissingConfiguration,
			`no configuration found for creating search client`,
		))
	})
	if result != nil {
		return result.(*gsearch.Search)
	}
	return nil
}
