// Copyright GoFrame Author(https://goframe.org). All Rights Reserved.
//
// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this file,
// You can obtain one at https://github.com/gogf/gf.

// Unit tests for gsearch configuration, engine-typed adapter registry, and instances.

package gsearch

import (
	"testing"

	"github.com/gogf/gf/v2/errors/gcode"
	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/test/gtest"
)

// fakeAdapter is a test adapter used by the root gsearch package tests.
type fakeAdapter struct {
	id string
}

// newFakeAdapterFunc returns an adapter factory that records the given id.
func newFakeAdapterFunc(id string) AdapterFunc {
	return func(_ *Config) Adapter {
		return &fakeAdapter{id: id}
	}
}

// Test_ConfigFromMap verifies that map configuration is converted into Config.
func Test_ConfigFromMap(t *testing.T) {
	gtest.C(t, func(t *gtest.T) {
		config, err := ConfigFromMap(map[string]any{
			"type":                   string(EngineTypeElasticsearch),
			"addresses":              []string{"http://127.0.0.1:9200"},
			"username":               "elastic",
			"password":               "secret",
			"apiKey":                 "api-key",
			"retryOnStatus":          []int{429, 502},
			"maxRetries":             3,
			"compressRequestBody":    true,
			"discoverNodesOnStart":   true,
			"certificateFingerprint": "fingerprint",
			"extra": map[string]any{
				"product": "elastic",
			},
		})
		t.AssertNil(err)
		t.Assert(config.Type, EngineTypeElasticsearch)
		t.Assert(config.Addresses, []string{"http://127.0.0.1:9200"})
		t.Assert(config.Username, "elastic")
		t.Assert(config.Password, "secret")
		t.Assert(config.APIKey, "api-key")
		t.Assert(config.RetryOnStatus, []int{429, 502})
		t.Assert(config.MaxRetries, 3)
		t.Assert(config.CompressRequestBody, true)
		t.Assert(config.DiscoverNodesOnStart, true)
		t.Assert(config.CertificateFingerprint, "fingerprint")
		t.Assert(config.Extra["product"], "elastic")
	})
}

// Test_New_MissingConfig verifies that creating a search client without config fails.
func Test_New_MissingConfig(t *testing.T) {
	gtest.C(t, func(t *gtest.T) {
		search, err := New()
		t.AssertNil(search)
		t.AssertNE(err, nil)
		t.Assert(gerror.Code(err), gcode.CodeInvalidConfiguration)
	})
}

// Test_New_MissingAdapter verifies that an unregistered engine returns a package-import error.
func Test_New_MissingAdapter(t *testing.T) {
	gtest.C(t, func(t *gtest.T) {
		search, err := New(&Config{Type: EngineTypeElasticsearch})
		t.AssertNil(search)
		t.AssertNE(err, nil)
		t.Assert(gerror.Code(err), gcode.CodeNecessaryPackageNotImport)
		t.AssertIN("contrib/nosql/elasticsearch", err.Error())
		t.AssertIN("contrib/nosql/opensearch", err.Error())
	})
}

// Test_New_MissingType verifies that search configuration requires an engine type.
func Test_New_MissingType(t *testing.T) {
	gtest.C(t, func(t *gtest.T) {
		search, err := New(&Config{})
		t.AssertNil(search)
		t.AssertNE(err, nil)
		t.Assert(gerror.Code(err), gcode.CodeInvalidConfiguration)
	})
}

// Test_RegisterAdapterFunc_ByEngineType verifies that engine-typed registration
// dispatches to the matching adapter factory.
func Test_RegisterAdapterFunc_ByEngineType(t *testing.T) {
	gtest.C(t, func(t *gtest.T) {
		engineTypeA := EngineType("test-register-a")
		engineTypeB := EngineType("test-register-b")
		RegisterAdapterFunc(engineTypeA, newFakeAdapterFunc("a"))
		RegisterAdapterFunc(engineTypeB, newFakeAdapterFunc("b"))

		searchA, err := New(&Config{Type: engineTypeA})
		t.AssertNil(err)
		searchB, err := New(&Config{Type: engineTypeB})
		t.AssertNil(err)
		t.Assert(searchA.Adapter().(*fakeAdapter).id, "a")
		t.Assert(searchB.Adapter().(*fakeAdapter).id, "b")
	})
}

// Test_RegisterAdapterFunc_Overwrite verifies that registering the same engine
// type again replaces the previous factory.
func Test_RegisterAdapterFunc_Overwrite(t *testing.T) {
	gtest.C(t, func(t *gtest.T) {
		engineType := EngineType("test-register-overwrite")
		RegisterAdapterFunc(engineType, newFakeAdapterFunc("old"))
		RegisterAdapterFunc(engineType, newFakeAdapterFunc("new"))

		search, err := New(&Config{Type: engineType})
		t.AssertNil(err)
		t.Assert(search.Adapter().(*fakeAdapter).id, "new")
	})
}

// Test_NewWithAdapter verifies that callers can create a search client directly
// from an adapter for tests or custom integrations.
func Test_NewWithAdapter(t *testing.T) {
	gtest.C(t, func(t *gtest.T) {
		search, err := NewWithAdapter(&fakeAdapter{id: "direct"})
		t.AssertNil(err)
		t.Assert(search.Adapter().(*fakeAdapter).id, "direct")
	})
}

// Test_NewWithAdapter_Nil verifies that nil adapter creation is rejected.
func Test_NewWithAdapter_Nil(t *testing.T) {
	gtest.C(t, func(t *gtest.T) {
		search, err := NewWithAdapter(nil)
		t.AssertNil(search)
		t.AssertNE(err, nil)
		t.Assert(gerror.Code(err), gcode.CodeInvalidParameter)
	})
}

// Test_SearchAccessors verifies Config and Adapter accessors.
func Test_SearchAccessors(t *testing.T) {
	gtest.C(t, func(t *gtest.T) {
		engineType := EngineType("test-accessors")
		config := &Config{Type: engineType}
		RegisterAdapterFunc(engineType, newFakeAdapterFunc("accessor"))

		search, err := New(config)
		t.AssertNil(err)
		t.Assert(search.Config(), config)
		t.Assert(search.Adapter().(*fakeAdapter).id, "accessor")

		var nilSearch *Search
		t.AssertNil(nilSearch.Config())
		t.AssertNil(nilSearch.Adapter())
	})
}

// Test_ConfigGroupAndInstance verifies group configuration and instance reuse.
func Test_ConfigGroupAndInstance(t *testing.T) {
	gtest.C(t, func(t *gtest.T) {
		group := "test-instance-group"
		engineType := EngineType("test-instance-engine")
		RegisterAdapterFunc(engineType, newFakeAdapterFunc("instance"))
		SetConfig(&Config{Type: engineType}, group)
		defer RemoveConfig(group)

		searchA := Instance(group)
		searchB := Instance(group)
		t.AssertNE(searchA, nil)
		t.Assert(searchA, searchB)
		t.Assert(searchA.Adapter().(*fakeAdapter).id, "instance")
	})
}

// Test_SetConfigByMapAndClearConfig verifies map-based config groups and clearing.
func Test_SetConfigByMapAndClearConfig(t *testing.T) {
	gtest.C(t, func(t *gtest.T) {
		group := "test-map-config"
		err := SetConfigByMap(map[string]any{
			"type":      string(EngineTypeOpenSearch),
			"addresses": []string{"http://127.0.0.1:9200"},
		}, group)
		t.AssertNil(err)

		config, ok := GetConfig(group)
		t.Assert(ok, true)
		t.Assert(config.Type, EngineTypeOpenSearch)
		t.Assert(config.Addresses, []string{"http://127.0.0.1:9200"})

		ClearConfig()
		_, ok = GetConfig(group)
		t.Assert(ok, false)
	})
}

// Test_InstanceWithoutConfig verifies that missing instance configuration returns nil.
func Test_InstanceWithoutConfig(t *testing.T) {
	gtest.C(t, func(t *gtest.T) {
		search := Instance("test-missing-instance-config")
		t.AssertNil(search)
	})
}
