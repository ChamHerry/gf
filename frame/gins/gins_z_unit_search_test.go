// Copyright GoFrame Author(https://goframe.org). All Rights Reserved.
//
// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT was not distributed with this file,
// You can obtain one at https://github.com/gogf/gf.

// Unit tests for gins search facade configuration loading.

package gins

import (
	"context"
	"testing"

	"github.com/gogf/gf/v2/database/gsearch"
	"github.com/gogf/gf/v2/internal/instance"
	"github.com/gogf/gf/v2/os/gcfg"
	"github.com/gogf/gf/v2/test/gtest"
)

// searchTestAdapter is a fake search adapter for gins tests.
type searchTestAdapter struct {
	id string
}

// newSearchTestAdapterFunc creates a fake search adapter factory.
func newSearchTestAdapterFunc(id string) gsearch.AdapterFunc {
	return func(_ *gsearch.Config) gsearch.Adapter {
		return &searchTestAdapter{id: id}
	}
}

// Ping implements gsearch.Adapter.
func (a *searchTestAdapter) Ping(_ context.Context) error {
	return nil
}

// Info implements gsearch.Adapter.
func (a *searchTestAdapter) Info(_ context.Context) (*gsearch.InfoResponse, error) {
	return &gsearch.InfoResponse{Name: a.id}, nil
}

// Perform implements gsearch.Adapter.
func (a *searchTestAdapter) Perform(_ context.Context, _ *gsearch.Request) (*gsearch.Response, error) {
	return &gsearch.Response{}, nil
}

// Search implements gsearch.Adapter.
func (a *searchTestAdapter) Search(_ context.Context, _ *gsearch.SearchRequest) (*gsearch.SearchResponse, error) {
	return &gsearch.SearchResponse{}, nil
}

// Bulk implements gsearch.Adapter.
func (a *searchTestAdapter) Bulk(_ context.Context, _ *gsearch.BulkRequest) (*gsearch.BulkResponse, error) {
	return &gsearch.BulkResponse{}, nil
}

// Close implements gsearch.Adapter.
func (a *searchTestAdapter) Close(_ context.Context) error {
	return nil
}

// Client implements gsearch.Adapter.
func (a *searchTestAdapter) Client() any {
	return a
}

// resetSearchTestState clears global instance and configuration state used by gins tests.
func resetSearchTestState() {
	instance.Clear()
	gsearch.ClearConfig()
	Config().GetAdapter().(*gcfg.AdapterFile).ClearContent()
}

// didSearchPanic reports whether f panics.
func didSearchPanic(f func()) (panicked bool) {
	defer func() {
		if recover() != nil {
			panicked = true
		}
	}()
	f()
	return false
}

// Test_Search_DefaultGroup verifies that gins.Search reads search.default configuration.
func Test_Search_DefaultGroup(t *testing.T) {
	gtest.C(t, func(t *gtest.T) {
		resetSearchTestState()
		defer resetSearchTestState()

		engineType := gsearch.EngineType("gins-search-default")
		gsearch.RegisterAdapterFunc(engineType, newSearchTestAdapterFunc("default"))
		Config().GetAdapter().(*gcfg.AdapterFile).SetContent(`
search.default.type = "gins-search-default"
search.default.addresses = ["http://127.0.0.1:9200"]
`)

		search := Search()
		t.AssertNE(search, nil)
		t.Assert(search.Config().Type, engineType)
		t.Assert(search.Config().Addresses, []string{"http://127.0.0.1:9200"})
		t.Assert(search.Adapter().(*searchTestAdapter).id, "default")
	})
}

// Test_Search_NamedGroup verifies that gins.Search reads a named search group.
func Test_Search_NamedGroup(t *testing.T) {
	gtest.C(t, func(t *gtest.T) {
		resetSearchTestState()
		defer resetSearchTestState()

		engineType := gsearch.EngineType("gins-search-named")
		gsearch.RegisterAdapterFunc(engineType, newSearchTestAdapterFunc("analytics"))
		Config().GetAdapter().(*gcfg.AdapterFile).SetContent(`
search.analytics.type = "gins-search-named"
search.analytics.apiKey = "test-api-key"
`)

		search := Search("analytics")
		t.AssertNE(search, nil)
		t.Assert(search.Config().Type, engineType)
		t.Assert(search.Config().APIKey, "test-api-key")
		t.Assert(search.Adapter().(*searchTestAdapter).id, "analytics")
	})
}

// Test_Search_MissingConfig verifies that missing search configuration panics.
func Test_Search_MissingConfig(t *testing.T) {
	gtest.C(t, func(t *gtest.T) {
		resetSearchTestState()
		defer resetSearchTestState()

		panicked := didSearchPanic(func() {
			Search("missing")
		})
		t.Assert(panicked, true)
	})
}

// Test_Search_MissingAdapter verifies that an unregistered engine type panics.
func Test_Search_MissingAdapter(t *testing.T) {
	gtest.C(t, func(t *gtest.T) {
		resetSearchTestState()
		defer resetSearchTestState()

		Config().GetAdapter().(*gcfg.AdapterFile).SetContent(`
search.noadapter.type = "gins-search-no-adapter"
`)
		panicked := didSearchPanic(func() {
			Search("noadapter")
		})
		t.Assert(panicked, true)
	})
}
