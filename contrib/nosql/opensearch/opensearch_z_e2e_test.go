// Copyright GoFrame Author(https://goframe.org). All Rights Reserved.
//
// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT was not distributed with this file,
// You can obtain one at https://github.com/gogf/gf.

// Docker-backed end-to-end tests for the OpenSearch gsearch adapter.

package opensearch

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gogf/gf/v2/database/gsearch"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/test/gtest"
	"github.com/gogf/gf/v2/util/gconv"
)

const (
	// envOpenSearchE2EURL is the URL used to enable Docker-backed OpenSearch E2E tests.
	envOpenSearchE2EURL = "GF_SEARCH_E2E_OPENSEARCH_URL"
)

// Test_E2EOpenSearchDocker verifies the adapter against a real OpenSearch service.
func Test_E2EOpenSearchDocker(t *testing.T) {
	endpoint := os.Getenv(envOpenSearchE2EURL)
	if endpoint == "" {
		t.Skipf("set %s to run Docker-backed OpenSearch E2E tests", envOpenSearchE2EURL)
	}

	gtest.C(t, func(t *gtest.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()

		adapter := New(&gsearch.Config{Addresses: []string{endpoint}})
		t.AssertNil(adapter.Ping(ctx))

		info, err := adapter.Info(ctx)
		t.AssertNil(err)
		t.AssertNE(info.Name, "")
		t.Assert(info.Version["distribution"], "opensearch")

		indexName := e2eOpenSearchIndexName()
		defer cleanupOpenSearchIndex(t, adapter, indexName)

		createIndexResponse, err := adapter.Perform(ctx, &gsearch.Request{
			Method: gsearch.RequestMethodPut,
			Path:   "/" + indexName,
			Body: []byte(`{
				"mappings": {
					"properties": {
						"title": {"type": "text"},
						"tag": {"type": "keyword"}
					}
				}
			}`),
		})
		t.AssertNil(err)
		t.AssertIN(createIndexResponse.StatusCode, []int{http.StatusOK, http.StatusCreated})

		indexDocResponse, err := adapter.Perform(ctx, &gsearch.Request{
			Method: gsearch.RequestMethodPut,
			Path:   "/" + indexName + "/_doc/1",
			Query:  map[string]string{"refresh": "true"},
			Body:   []byte(`{"title":"GoFrame OpenSearch E2E","tag":"single"}`),
		})
		t.AssertNil(err)
		t.AssertIN(indexDocResponse.StatusCode, []int{http.StatusOK, http.StatusCreated})

		searchResponse, err := adapter.Search(ctx, &gsearch.SearchRequest{
			Index: []string{indexName},
			Body:  []byte(`{"query":{"match":{"title":"GoFrame"}}}`),
		})
		t.AssertNil(err)
		t.Assert(len(gconv.SliceAny(searchResponse.Hits["hits"])) > 0, true)

		bulkResponse, err := adapter.Bulk(ctx, &gsearch.BulkRequest{
			Index: indexName,
			Query: map[string]string{"refresh": "true"},
			Body: []byte(strings.Join([]string{
				`{"index":{"_id":"2"}}`,
				`{"title":"GoFrame Bulk One","tag":"bulk"}`,
				`{"index":{"_id":"3"}}`,
				`{"title":"GoFrame Bulk Two","tag":"bulk"}`,
				"",
			}, "\n")),
		})
		t.AssertNil(err)
		t.Assert(bulkResponse.Errors, false)
		t.Assert(len(bulkResponse.Items), 2)

		group := indexName + "-group"
		gsearch.SetConfig(&gsearch.Config{
			Type:      gsearch.EngineTypeOpenSearch,
			Addresses: []string{endpoint},
		}, group)
		defer gsearch.RemoveConfig(group)

		searchClient := g.Search(group)
		facadeInfo, err := searchClient.Info(ctx)
		t.AssertNil(err)
		t.Assert(facadeInfo.Version["distribution"], "opensearch")
	})
}

// e2eOpenSearchIndexName creates a unique index name for live tests.
func e2eOpenSearchIndexName() string {
	return fmt.Sprintf("gf-e2e-opensearch-%d", time.Now().UnixNano())
}

// cleanupOpenSearchIndex deletes the live test index and reports cleanup failures.
func cleanupOpenSearchIndex(t *gtest.T, adapter *OpenSearch, indexName string) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	response, err := adapter.Perform(ctx, &gsearch.Request{
		Method: gsearch.RequestMethodDelete,
		Path:   "/" + indexName,
	})
	if err != nil {
		t.Logf("cleanup index %s failed: %+v", indexName, err)
		return
	}
	if response != nil && response.StatusCode != http.StatusOK && response.StatusCode != http.StatusNotFound {
		t.Logf("cleanup index %s returned status %d: %s", indexName, response.StatusCode, string(response.Body))
	}
}
