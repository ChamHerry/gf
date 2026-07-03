// Copyright GoFrame Author(https://goframe.org). All Rights Reserved.
//
// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this file,
// You can obtain one at https://github.com/gogf/gf.

// Package gsearch provides driver-agnostic search and document engine access.
package gsearch

import (
	"github.com/gogf/gf/v2/errors/gcode"
	"github.com/gogf/gf/v2/errors/gerror"
)

// Search is a root search client backed by a registered adapter.
type Search struct {
	// config stores the configuration used to create this client.
	config *Config

	// localAdapter stores the concrete adapter used by this client.
	localAdapter Adapter
}

// New creates and returns a search client.
// It uses the passed configuration first, or the default group configuration if
// no configuration is passed.
func New(config ...*Config) (*Search, error) {
	var (
		usedConfig  *Config
		usedAdapter Adapter
	)
	if len(config) > 0 && config[0] != nil {
		usedConfig = config[0]
	} else if configFromGlobal, ok := GetConfig(); ok {
		usedConfig = configFromGlobal
	}
	if usedConfig == nil {
		return nil, gerror.NewCode(
			gcode.CodeInvalidConfiguration,
			`no configuration found for creating Search client`,
		)
	}
	if usedConfig.Type == "" {
		return nil, gerror.NewCode(
			gcode.CodeInvalidConfiguration,
			`missing search engine type`,
		)
	}
	if adapterFunc := getAdapterFunc(usedConfig.Type); adapterFunc != nil {
		usedAdapter = adapterFunc(usedConfig)
	}
	if usedAdapter == nil {
		return nil, gerror.NewCode(
			gcode.CodeNecessaryPackageNotImport,
			errorNilAdapter,
		)
	}
	return &Search{
		config:       usedConfig,
		localAdapter: usedAdapter,
	}, nil
}

// NewWithAdapter creates and returns a search client with the given adapter.
func NewWithAdapter(adapter Adapter) (*Search, error) {
	if adapter == nil {
		return nil, gerror.NewCode(gcode.CodeInvalidParameter, `adapter cannot be nil`)
	}
	return &Search{localAdapter: adapter}, nil
}

// Config returns the configuration used by the search client.
func (s *Search) Config() *Config {
	if s == nil {
		return nil
	}
	return s.config
}

// Adapter returns the concrete adapter used by the search client.
func (s *Search) Adapter() Adapter {
	if s == nil {
		return nil
	}
	return s.localAdapter
}
