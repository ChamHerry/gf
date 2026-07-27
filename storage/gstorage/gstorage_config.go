// Copyright GoFrame Author(https://goframe.org). All Rights Reserved.
//
// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT was not distributed with this file,
// You can obtain one at https://github.com/gogf/gf.

package gstorage

import (
	"context"
	"strings"
	"time"

	"github.com/gogf/gf/v2/errors/gcode"
	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/os/gcfg"
	"github.com/gogf/gf/v2/util/gconv"
)

const (
	// DefaultGroupName 是默认对象存储配置组名称。
	DefaultGroupName = "default"
	// ConfigNodeName 是 gcfg YAML 中对象存储配置的根节点。
	ConfigNodeName = "storage"
)

// Credentials 保存对象存储的显式静态凭据。
type Credentials struct {
	AccessKey    string `json:"accessKey"`
	SecretKey    string `json:"secretKey"`
	SessionToken string `json:"sessionToken"`
}

// Config 是 core 与 contrib adapter 共享的对象存储配置。
type Config struct {
	Type        string        `json:"type"`
	Endpoint    string        `json:"endpoint"`
	Region      string        `json:"region"`
	PathStyle   bool          `json:"pathStyle"`
	TLS         bool          `json:"tls"`
	Credentials Credentials   `json:"credentials"`
	Timeout     time.Duration `json:"timeout"`
}

// ConfigFromMap 将 gcfg YAML adapter 读取出的 map 转为强类型配置。
func ConfigFromMap(m map[string]any) (*Config, error) {
	config := &Config{}
	if err := gconv.Struct(m, config); err != nil {
		return nil, gerror.NewCodef(gcode.CodeInvalidConfiguration, `invalid storage configuration: %v`, err)
	}
	config.Type = strings.TrimSpace(config.Type)
	config.Endpoint = strings.TrimSpace(config.Endpoint)
	config.Region = strings.TrimSpace(config.Region)
	config.Credentials.AccessKey = strings.TrimSpace(config.Credentials.AccessKey)
	config.Credentials.SecretKey = strings.TrimSpace(config.Credentials.SecretKey)
	config.Credentials.SessionToken = strings.TrimSpace(config.Credentials.SessionToken)
	if config.Type == "" {
		return nil, gerror.NewCode(gcode.CodeInvalidConfiguration, `storage type is required`)
	}
	return config, nil
}

func loadConfig(group string) (*Config, error) {
	configInstance := gcfg.Instance()
	if configInstance == nil {
		return nil, gerror.NewCode(gcode.CodeMissingConfiguration, `gcfg instance is unavailable`)
	}
	data, err := configInstance.Data(context.Background())
	if err != nil {
		return nil, gerror.WrapCode(gcode.CodeInvalidConfiguration, err, `read storage configuration from gcfg failed`)
	}
	storageNode := gconv.Map(data[ConfigNodeName])
	if len(storageNode) == 0 {
		return nil, gerror.NewCodef(gcode.CodeMissingConfiguration, `configuration node %q is required`, ConfigNodeName)
	}
	groupNode := gconv.Map(storageNode[group])
	if len(groupNode) == 0 {
		return nil, gerror.NewCodef(gcode.CodeMissingConfiguration, `storage configuration group %q is required`, group)
	}
	return ConfigFromMap(groupNode)
}
