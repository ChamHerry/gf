// Copyright GoFrame Author(https://goframe.org). All Rights Reserved.
//
// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT was not distributed with this file,
// You can obtain one at https://github.com/gogf/gf.

// Unit tests for the Elasticsearch gsearch adapter.

package elasticsearch

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gogf/gf/v2/database/gsearch"
	"github.com/gogf/gf/v2/test/gtest"
)

// elasticProductHeader is the product header required by the official Elastic client.
const elasticProductHeader = "X-Elastic-Product"

// writeElasticJSON writes a JSON response with the Elasticsearch product header.
func writeElasticJSON(t *gtest.T, w http.ResponseWriter, body string) {
	w.Header().Set(elasticProductHeader, "Elasticsearch")
	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write([]byte(body)); err != nil {
		t.Fatal(err)
	}
}

// readRequestBody reads a request body and fails the test on error.
func readRequestBody(t *gtest.T, r *http.Request) string {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}

// Test_RegisterAdapter verifies init registration for the Elasticsearch engine type.
func Test_RegisterAdapter(t *testing.T) {
	gtest.C(t, func(t *gtest.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			writeElasticJSON(t, w, `{"name":"node-1","cluster_name":"cluster-a","cluster_uuid":"uuid-a","version":{"number":"8.19.6"},"tagline":"You Know, for Search"}`)
		}))
		defer server.Close()

		search, err := gsearch.New(&gsearch.Config{
			Type:      gsearch.EngineTypeElasticsearch,
			Addresses: []string{server.URL},
		})
		t.AssertNil(err)
		t.AssertNE(search, nil)
		t.AssertNE(search.Client(), nil)
	})
}

// Test_InfoConfigMapping verifies config mapping to the official client.
func Test_InfoConfigMapping(t *testing.T) {
	gtest.C(t, func(t *gtest.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			username, password, ok := r.BasicAuth()
			t.Assert(ok, true)
			t.Assert(username, "elastic")
			t.Assert(password, "secret")
			t.Assert(r.Header.Get("X-Trace"), "trace-1")
			writeElasticJSON(t, w, `{"name":"node-1","cluster_name":"cluster-a","cluster_uuid":"uuid-a","version":{"number":"8.19.6"},"tagline":"You Know, for Search"}`)
		}))
		defer server.Close()

		adapter := New(&gsearch.Config{
			Addresses: []string{server.URL},
			Username:  "elastic",
			Password:  "secret",
			Headers:   map[string]string{"X-Trace": "trace-1"},
		})
		info, err := adapter.Info(context.Background())
		t.AssertNil(err)
		t.Assert(info.Name, "node-1")
		t.Assert(info.ClusterName, "cluster-a")
		t.Assert(info.ClusterUUID, "uuid-a")
		t.Assert(info.Version["number"], "8.19.6")
		t.Assert(info.Tagline, "You Know, for Search")
	})
}

// Test_PerformRawRequest verifies raw request mapping and response preservation.
func Test_PerformRawRequest(t *testing.T) {
	gtest.C(t, func(t *gtest.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Assert(r.Method, "PUT")
			t.Assert(r.URL.Path, "/movies/_doc/1")
			t.Assert(r.URL.Query().Get("refresh"), "true")
			t.Assert(r.Header.Get("X-Request"), "raw")
			t.Assert(readRequestBody(t, r), `{"title":"GoFrame"}`)
			w.Header().Set("X-Elastic-Product", "Elasticsearch")
			w.Header().Set("X-Result", "created")
			w.WriteHeader(http.StatusCreated)
			if _, err := w.Write([]byte(`{"result":"created"}`)); err != nil {
				t.Fatal(err)
			}
		}))
		defer server.Close()

		adapter := New(&gsearch.Config{Addresses: []string{server.URL}})
		response, err := adapter.Perform(context.Background(), &gsearch.Request{
			Method:  gsearch.RequestMethodPut,
			Path:    "/movies/_doc/1",
			Query:   map[string]string{"refresh": "true"},
			Headers: map[string]string{"X-Request": "raw"},
			Body:    []byte(`{"title":"GoFrame"}`),
		})
		t.AssertNil(err)
		t.Assert(response.StatusCode, http.StatusCreated)
		t.Assert(response.Headers["X-Result"][0], "created")
		t.Assert(response.Body, []byte(`{"result":"created"}`))
	})
}

// Test_SearchPartialFailures verifies search response timeout and shard failure parsing.
func Test_SearchPartialFailures(t *testing.T) {
	gtest.C(t, func(t *gtest.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Assert(r.Method, "POST")
			t.Assert(r.URL.Path, "/logs/_search")
			t.Assert(r.URL.Query().Get("allow_partial_search_results"), "true")
			t.Assert(readRequestBody(t, r), `{"query":{"match_all":{}}}`)
			writeElasticJSON(t, w, `{
				"took": 15,
				"timed_out": true,
				"_shards": {
					"total": 3,
					"successful": 2,
					"skipped": 0,
					"failed": 1,
					"failures": [
						{
							"index": "logs",
							"shard": 1,
							"status": "500",
							"reason": {"type": "query_shard_exception", "reason": "failed to create query"}
						}
					]
				},
				"hits": {"total": {"value": 1, "relation": "eq"}, "hits": []}
			}`)
		}))
		defer server.Close()

		adapter := New(&gsearch.Config{Addresses: []string{server.URL}})
		response, err := adapter.Search(context.Background(), &gsearch.SearchRequest{
			Index: []string{"logs"},
			Query: map[string]string{"allow_partial_search_results": "true"},
			Body:  []byte(`{"query":{"match_all":{}}}`),
		})
		t.AssertNil(err)
		t.Assert(response.TimedOut, true)
		t.Assert(response.Shards.Failed, 1)
		t.Assert(response.Shards.Failures[0].Shard, "1")
		t.Assert(response.Shards.Failures[0].Reason.Type, "query_shard_exception")
	})
}

// Test_BulkPerItemErrors verifies bulk item error parsing.
func Test_BulkPerItemErrors(t *testing.T) {
	gtest.C(t, func(t *gtest.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Assert(r.Method, "POST")
			t.Assert(r.URL.Path, "/movies/_bulk")
			t.Assert(r.Header.Get("Content-Type"), "application/x-ndjson")
			t.Assert(readRequestBody(t, r), "{\"index\":{}}\n{\"title\":\"GoFrame\"}\n")
			writeElasticJSON(t, w, `{
				"took": 35,
				"errors": true,
				"items": [
					{"index": {"_index": "movies", "_id": "1", "status": 201, "result": "created", "_shards": {"total": 1, "successful": 1, "failed": 0}}},
					{"update": {"_index": "movies", "_id": "missing", "status": 404, "error": {"type": "document_missing_exception", "reason": "[missing]: document missing", "index": "movies", "shard": "0", "index_uuid": "uuid-1"}}}
				]
			}`)
		}))
		defer server.Close()

		adapter := New(&gsearch.Config{Addresses: []string{server.URL}})
		response, err := adapter.Bulk(context.Background(), &gsearch.BulkRequest{
			Index: "movies",
			Body:  []byte("{\"index\":{}}\n{\"title\":\"GoFrame\"}\n"),
		})
		t.AssertNil(err)
		t.Assert(response.Errors, true)
		t.Assert(len(response.Items), 2)
		t.Assert(response.Items[0].Operation, gsearch.BulkOperationIndex)
		t.Assert(response.Items[0].Shards.Successful, 1)
		t.Assert(response.Items[1].Operation, gsearch.BulkOperationUpdate)
		t.Assert(response.Items[1].Error.Type, "document_missing_exception")
		t.Assert(response.Items[1].Error.IndexUUID, "uuid-1")
	})
}

// Test_ProductMismatch verifies official client product checking is preserved.
func Test_ProductMismatch(t *testing.T) {
	gtest.C(t, func(t *gtest.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			if _, err := w.Write([]byte(`{"name":"not-elastic"}`)); err != nil {
				t.Fatal(err)
			}
		}))
		defer server.Close()

		adapter := New(&gsearch.Config{Addresses: []string{server.URL}})
		info, err := adapter.Info(context.Background())
		t.AssertNil(info)
		t.AssertNE(err, nil)
	})
}
