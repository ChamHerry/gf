// Copyright GoFrame Author(https://goframe.org). All Rights Reserved.
//
// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT was not distributed with this file,
// You can obtain one at https://github.com/gogf/gf.

// Package opensearch provides gsearch.Adapter implementation using the official OpenSearch Go client.
package opensearch

import (
	opensearchv3 "github.com/opensearch-project/opensearch-go/v3"

	"github.com/gogf/gf/v2/database/gsearch"
)

// OpenSearch is a gsearch adapter backed by the official OpenSearch Go client.
type OpenSearch struct {
	// client stores the official OpenSearch client.
	client *opensearchv3.Client

	// config stores the source gsearch configuration.
	config *gsearch.Config

	// initErr stores client initialization errors for later operation calls.
	initErr error
}

func init() {
	gsearch.RegisterAdapterFunc(gsearch.EngineTypeOpenSearch, func(config *gsearch.Config) gsearch.Adapter {
		return New(config)
	})
}

// New creates and returns an OpenSearch adapter.
func New(config *gsearch.Config) *OpenSearch {
	if config == nil {
		config = &gsearch.Config{}
	}
	client, err := opensearchv3.NewClient(newClientConfig(config))
	return &OpenSearch{
		client:  client,
		config:  config,
		initErr: err,
	}
}

// Client returns the native OpenSearch client object.
func (o *OpenSearch) Client() any {
	if o == nil {
		return nil
	}
	return o.client
}
