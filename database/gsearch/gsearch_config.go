// Copyright GoFrame Author(https://goframe.org). All Rights Reserved.
//
// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this file,
// You can obtain one at https://github.com/gogf/gf.

// Configuration types and group management for search clients.

package gsearch

import (
	"context"

	"github.com/gogf/gf/v2/container/gmap"
	"github.com/gogf/gf/v2/errors/gcode"
	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/internal/intlog"
	"github.com/gogf/gf/v2/util/gconv"
)

// EngineType is the configured search engine type.
type EngineType string

const (
	// EngineTypeElasticsearch identifies the Elasticsearch adapter.
	EngineTypeElasticsearch EngineType = "elasticsearch"

	// EngineTypeOpenSearch identifies the OpenSearch adapter.
	EngineTypeOpenSearch EngineType = "opensearch"

	// DefaultGroupName is the default configuration group name.
	DefaultGroupName = "default"
)

// Config is search engine configuration shared by root and contrib adapters.
type Config struct {
	// Type selects the registered search engine adapter.
	Type EngineType `json:"type"`

	// Addresses contains search engine node addresses.
	Addresses []string `json:"addresses"`

	// Username is used for basic authentication.
	Username string `json:"username"`

	// Password is used for basic authentication.
	Password string `json:"password"`

	// APIKey is used for API key authentication.
	APIKey string `json:"apiKey"`

	// ServiceToken is used for service-token authentication.
	ServiceToken string `json:"serviceToken"`

	// CloudID is used by hosted Elasticsearch deployments.
	CloudID string `json:"cloudId"`

	// Headers contains additional HTTP headers for adapter requests.
	Headers map[string]string `json:"headers"`

	// CACert contains PEM encoded CA certificate data.
	CACert []byte `json:"caCert"`

	// CertificateFingerprint contains the expected server certificate fingerprint.
	CertificateFingerprint string `json:"certificateFingerprint"`

	// TLS specifies whether TLS should be used when connecting to the server.
	TLS bool `json:"tls"`

	// TLSSkipVerify disables TLS server verification when connecting over TLS.
	TLSSkipVerify bool `json:"tlsSkipVerify"`

	// RetryOnStatus contains HTTP status codes that should trigger retries.
	RetryOnStatus []int `json:"retryOnStatus"`

	// MaxRetries contains the maximum retry count.
	MaxRetries int `json:"maxRetries"`

	// CompressRequestBody specifies whether request bodies should be compressed.
	CompressRequestBody bool `json:"compressRequestBody"`

	// DiscoverNodesOnStart specifies whether adapters should discover nodes at startup.
	DiscoverNodesOnStart bool `json:"discoverNodesOnStart"`

	// Extra contains adapter-specific configuration values.
	Extra map[string]any `json:"extra"`
}

var (
	// configChecker checks whether the *Config is nil.
	configChecker = func(v *Config) bool { return v == nil }

	// localConfigMap stores configuration groups.
	localConfigMap = gmap.NewKVMapWithChecker[string, *Config](configChecker, true)
)

// SetConfig sets the global configuration for the specified group.
// If name is not passed, it sets configuration for the default group name.
func SetConfig(config *Config, name ...string) {
	group := DefaultGroupName
	if len(name) > 0 {
		group = name[0]
	}
	localConfigMap.Set(group, config)

	intlog.Printf(context.TODO(), `SetConfig for group "%s": %+v`, group, config)
}

// SetConfigByMap sets the global configuration for the specified group with map.
// If name is not passed, it sets configuration for the default group name.
func SetConfigByMap(m map[string]any, name ...string) error {
	group := DefaultGroupName
	if len(name) > 0 {
		group = name[0]
	}
	config, err := ConfigFromMap(m)
	if err != nil {
		return err
	}
	localConfigMap.Set(group, config)
	return nil
}

// ConfigFromMap parses and returns config from given map.
func ConfigFromMap(m map[string]any) (config *Config, err error) {
	config = &Config{}
	if err = gconv.Scan(m, config); err != nil {
		err = gerror.NewCodef(gcode.CodeInvalidConfiguration, `invalid search configuration: %#v`, m)
	}
	return
}

// GetConfig returns the global configuration with specified group name.
// If name is not passed, it returns configuration of the default group name.
func GetConfig(name ...string) (config *Config, ok bool) {
	group := DefaultGroupName
	if len(name) > 0 {
		group = name[0]
	}
	if v := localConfigMap.Get(group); v != nil {
		return v, true
	}
	return &Config{}, false
}

// RemoveConfig removes the global configuration with specified group.
// If name is not passed, it removes configuration of the default group name.
func RemoveConfig(name ...string) {
	group := DefaultGroupName
	if len(name) > 0 {
		group = name[0]
	}
	localConfigMap.Remove(group)

	intlog.Printf(context.TODO(), `RemoveConfig: %s`, group)
}

// ClearConfig removes all configurations of search clients.
func ClearConfig() {
	localConfigMap.Clear()
}
