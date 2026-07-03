// Copyright GoFrame Author(https://goframe.org). All Rights Reserved.
//
// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this file,
// You can obtain one at https://github.com/gogf/gf.

// Operation forwarding methods for the root search client.

package gsearch

import (
	"context"

	"github.com/gogf/gf/v2/errors/gcode"
	"github.com/gogf/gf/v2/errors/gerror"
)

const (
	// errorNilSearchAdapter is returned when a search client has no adapter.
	errorNilSearchAdapter = `search adapter is nil`
)

// Ping verifies whether the backing search engine is reachable.
func (s *Search) Ping(ctx context.Context) error {
	adapter, err := s.adapter()
	if err != nil {
		return err
	}
	return adapter.Ping(ctx)
}

// Info returns basic server and cluster information.
func (s *Search) Info(ctx context.Context) (*InfoResponse, error) {
	adapter, err := s.adapter()
	if err != nil {
		return nil, err
	}
	return adapter.Info(ctx)
}

// Perform executes a raw REST-style request.
func (s *Search) Perform(ctx context.Context, req *Request) (*Response, error) {
	adapter, err := s.adapter()
	if err != nil {
		return nil, err
	}
	return adapter.Perform(ctx, req)
}

// Search executes a search request.
func (s *Search) Search(ctx context.Context, req *SearchRequest) (*SearchResponse, error) {
	adapter, err := s.adapter()
	if err != nil {
		return nil, err
	}
	return adapter.Search(ctx, req)
}

// Bulk executes a bulk request.
func (s *Search) Bulk(ctx context.Context, req *BulkRequest) (*BulkResponse, error) {
	adapter, err := s.adapter()
	if err != nil {
		return nil, err
	}
	return adapter.Bulk(ctx, req)
}

// Close releases adapter resources.
func (s *Search) Close(ctx context.Context) error {
	adapter, err := s.adapter()
	if err != nil {
		return err
	}
	return adapter.Close(ctx)
}

// Client returns the native client object used by the adapter.
func (s *Search) Client() any {
	if s == nil || s.localAdapter == nil {
		return nil
	}
	return s.localAdapter.Client()
}

// adapter returns the concrete adapter or a configuration error.
func (s *Search) adapter() (Adapter, error) {
	if s == nil || s.localAdapter == nil {
		return nil, gerror.NewCode(gcode.CodeInvalidConfiguration, errorNilSearchAdapter)
	}
	return s.localAdapter, nil
}
