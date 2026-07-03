// Copyright GoFrame Author(https://goframe.org). All Rights Reserved.
//
// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT was not distributed with this file,
// You can obtain one at https://github.com/gogf/gf.

// Package elasticsearch provides gsearch.Adapter implementation using the official Elasticsearch Go client.
package elasticsearch

import (
	elasticv8 "github.com/elastic/go-elasticsearch/v8"

	"github.com/gogf/gf/v2/database/gsearch"
)

// Elasticsearch is a gsearch adapter backed by the official Elasticsearch Go client.
type Elasticsearch struct {
	// client stores the official Elasticsearch client.
	client *elasticv8.Client

	// config stores the source gsearch configuration.
	config *gsearch.Config

	// initErr stores client initialization errors for later operation calls.
	initErr error
}

func init() {
	gsearch.RegisterAdapterFunc(gsearch.EngineTypeElasticsearch, func(config *gsearch.Config) gsearch.Adapter {
		return New(config)
	})
}

// New creates and returns an Elasticsearch adapter.
func New(config *gsearch.Config) *Elasticsearch {
	if config == nil {
		config = &gsearch.Config{}
	}
	client, err := elasticv8.NewClient(newClientConfig(config))
	return &Elasticsearch{
		client:  client,
		config:  config,
		initErr: err,
	}
}

// Client returns the native Elasticsearch client object.
func (e *Elasticsearch) Client() any {
	if e == nil {
		return nil
	}
	return e.client
}
