// Copyright GoFrame Author(https://goframe.org). All Rights Reserved.
//
// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT was not distributed with this file,
// You can obtain one at https://github.com/gogf/gf.

package gstorage

import (
	"strings"
	"sync"

	"github.com/gogf/gf/v2/errors/gcode"
	"github.com/gogf/gf/v2/errors/gerror"
)

// AdapterFunc 根据强类型配置创建对象存储适配器。
type AdapterFunc func(config *Config) (Adapter, error)

var registry = struct {
	sync.Mutex
	adapterFactories map[string]AdapterFunc
	instances        map[string]*Storage
}{
	adapterFactories: make(map[string]AdapterFunc),
	instances:        make(map[string]*Storage),
}

// RegisterAdapterFunc 注册具名适配器工厂。
func RegisterAdapterFunc(adapterType string, factory AdapterFunc) error {
	adapterType = strings.TrimSpace(adapterType)
	if adapterType == "" || factory == nil {
		return gerror.NewCode(gcode.CodeInvalidParameter, `storage adapter type and factory are required`)
	}
	registry.Lock()
	registry.adapterFactories[adapterType] = factory
	registry.Unlock()
	return nil
}

// RegisterInstance 注册默认或具名对象存储实例。
func RegisterInstance(storage *Storage, name ...string) error {
	if err := storage.validate(); err != nil {
		return err
	}
	group := groupName(name...)
	registry.Lock()
	registry.instances[group] = storage
	registry.Unlock()
	return nil
}

// GetInstance 返回已经注册或创建的默认/具名实例。
func GetInstance(name ...string) (*Storage, bool) {
	group := groupName(name...)
	registry.Lock()
	storage, ok := registry.instances[group]
	registry.Unlock()
	return storage, ok
}

// Open 从 gcfg YAML adapter 加载默认或具名配置并缓存实例。
func Open(name ...string) (*Storage, error) {
	group := groupName(name...)
	registry.Lock()
	defer registry.Unlock()
	if storage := registry.instances[group]; storage != nil {
		return storage, nil
	}
	config, err := loadConfig(group)
	if err != nil {
		return nil, err
	}
	factory := registry.adapterFactories[config.Type]
	if factory == nil {
		return nil, gerror.NewCodef(
			gcode.CodeNecessaryPackageNotImport,
			`storage adapter %q is not registered; import its contrib package`,
			config.Type,
		)
	}
	adapter, err := factory(config)
	if err != nil {
		return nil, err
	}
	storage, err := New(adapter)
	if err != nil {
		return nil, err
	}
	registry.instances[group] = storage
	return storage, nil
}

// Instance 返回默认或具名实例；配置或构造失败时会 panic，与 frame/g 其他组件一致。
func Instance(name ...string) *Storage {
	storage, err := Open(name...)
	if err != nil {
		panic(err)
	}
	return storage
}

// RemoveInstance 删除缓存中的默认或具名实例，不隐式关闭适配器。
func RemoveInstance(name ...string) {
	group := groupName(name...)
	registry.Lock()
	delete(registry.instances, group)
	registry.Unlock()
}

// ClearInstances 删除全部缓存实例，不隐式关闭适配器。
func ClearInstances() {
	registry.Lock()
	registry.instances = make(map[string]*Storage)
	registry.Unlock()
}

func groupName(name ...string) string {
	if len(name) > 0 && strings.TrimSpace(name[0]) != "" {
		return strings.TrimSpace(name[0])
	}
	return DefaultGroupName
}
