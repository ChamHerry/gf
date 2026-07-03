// Copyright GoFrame Author(https://goframe.org). All Rights Reserved.
//
// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT was not distributed with this file,
// You can obtain one at https://github.com/gogf/gf.

// Unit test adapter implementation for the g.Search facade.

package g_test

import (
	"context"

	"github.com/gogf/gf/v2/database/gsearch"
)

// gSearchTestAdapter is a fake search adapter for g facade tests.
type gSearchTestAdapter struct {
	id string
}

// Ping implements gsearch.Adapter.
func (a *gSearchTestAdapter) Ping(_ context.Context) error {
	return nil
}

// Info implements gsearch.Adapter.
func (a *gSearchTestAdapter) Info(_ context.Context) (*gsearch.InfoResponse, error) {
	return &gsearch.InfoResponse{Name: a.id}, nil
}

// Perform implements gsearch.Adapter.
func (a *gSearchTestAdapter) Perform(_ context.Context, _ *gsearch.Request) (*gsearch.Response, error) {
	return &gsearch.Response{}, nil
}

// Search implements gsearch.Adapter.
func (a *gSearchTestAdapter) Search(_ context.Context, _ *gsearch.SearchRequest) (*gsearch.SearchResponse, error) {
	return &gsearch.SearchResponse{}, nil
}

// Bulk implements gsearch.Adapter.
func (a *gSearchTestAdapter) Bulk(_ context.Context, _ *gsearch.BulkRequest) (*gsearch.BulkResponse, error) {
	return &gsearch.BulkResponse{}, nil
}

// Close implements gsearch.Adapter.
func (a *gSearchTestAdapter) Close(_ context.Context) error {
	return nil
}

// Client implements gsearch.Adapter.
func (a *gSearchTestAdapter) Client() any {
	return a
}
