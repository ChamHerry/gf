// Copyright GoFrame Author(https://goframe.org). All Rights Reserved.
//
// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT was not distributed with this file,
// You can obtain one at https://github.com/gogf/gf.

// Docker-backed end-to-end tests for the Elasticsearch gsearch adapter.

package elasticsearch

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
	// envElasticsearchE2EURL is the URL used to enable Docker-backed Elasticsearch E2E tests.
	envElasticsearchE2EURL = "GF_SEARCH_E2E_ELASTICSEARCH_URL"
)

// Test_E2EElasticsearchDocker verifies the adapter against a real Elasticsearch service.
func Test_E2EElasticsearchDocker(t *testing.T) {
	endpoint := os.Getenv(envElasticsearchE2EURL)
	if endpoint == "" {
		t.Skipf("set %s to run Docker-backed Elasticsearch E2E tests", envElasticsearchE2EURL)
	}

	gtest.C(t, func(t *gtest.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()

		adapter := New(&gsearch.Config{Addresses: []string{endpoint}})
		t.AssertNil(adapter.Ping(ctx))

		info, err := adapter.Info(ctx)
		t.AssertNil(err)
		t.AssertNE(info.Name, "")
		t.AssertNE(info.Version["number"], "")

		indexName := e2eElasticsearchIndexName()
		defer cleanupElasticsearchIndex(t, adapter, indexName)

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
			Body:   []byte(`{"title":"GoFrame Elasticsearch E2E","tag":"single"}`),
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
			Type:      gsearch.EngineTypeElasticsearch,
			Addresses: []string{endpoint},
		}, group)
		defer gsearch.RemoveConfig(group)

		searchClient := g.Search(group)
		facadeInfo, err := searchClient.Info(ctx)
		t.AssertNil(err)
		t.AssertNE(facadeInfo.Version["number"], "")
	})
}

// e2eElasticsearchIndexName creates a unique index name for live tests.
func e2eElasticsearchIndexName() string {
	return fmt.Sprintf("gf-e2e-elasticsearch-%d", time.Now().UnixNano())
}

// cleanupElasticsearchIndex deletes the live test index and reports cleanup failures.
func cleanupElasticsearchIndex(t *gtest.T, adapter *Elasticsearch, indexName string) {
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
