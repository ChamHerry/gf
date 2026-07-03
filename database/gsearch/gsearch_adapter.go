// Copyright GoFrame Author(https://goframe.org). All Rights Reserved.
//
// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this file,
// You can obtain one at https://github.com/gogf/gf.

// Adapter registry definitions for search engine implementations.

package gsearch

import (
	"context"

	"github.com/gogf/gf/v2/container/gmap"
)

// Adapter is the root interface for a concrete search engine adapter.
type Adapter interface {
	// Ping verifies whether the backing search engine is reachable.
	Ping(ctx context.Context) error

	// Info returns basic server and cluster information.
	Info(ctx context.Context) (*InfoResponse, error)

	// Perform executes a raw REST-style request.
	Perform(ctx context.Context, req *Request) (*Response, error)

	// Search executes a search request.
	Search(ctx context.Context, req *SearchRequest) (*SearchResponse, error)

	// Bulk executes a bulk request.
	Bulk(ctx context.Context, req *BulkRequest) (*BulkResponse, error)

	// Close releases adapter resources.
	Close(ctx context.Context) error

	// Client returns the native client object used by the adapter.
	Client() any
}

// AdapterFunc is the function that creates a search adapter from configuration.
type AdapterFunc func(config *Config) Adapter

const (
	// errorNilAdapter is returned when no adapter is registered for the configured engine type.
	errorNilAdapter = `search adapter is not set, missing configuration or adapter register? possible references: https://github.com/gogf/gf/tree/master/contrib/nosql/elasticsearch, https://github.com/gogf/gf/tree/master/contrib/nosql/opensearch`
)

var (
	// adapterFuncChecker checks whether the AdapterFunc is nil.
	adapterFuncChecker = func(v AdapterFunc) bool { return v == nil }

	// localAdapterFuncMap stores adapter factory functions by engine type.
	localAdapterFuncMap = gmap.NewKVMapWithChecker[EngineType, AdapterFunc](adapterFuncChecker, true)
)

// RegisterAdapterFunc registers an adapter factory for the given engine type.
func RegisterAdapterFunc(engineType EngineType, adapterFunc AdapterFunc) {
	localAdapterFuncMap.Set(engineType, adapterFunc)
}

// getAdapterFunc returns the registered adapter factory for the given engine type.
func getAdapterFunc(engineType EngineType) AdapterFunc {
	return localAdapterFuncMap.Get(engineType)
}
