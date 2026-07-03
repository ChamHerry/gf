// Copyright GoFrame Author(https://goframe.org). All Rights Reserved.
//
// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this file,
// You can obtain one at https://github.com/gogf/gf.

// Unit tests for gsearch operation forwarding and partial-result response models.

package gsearch

import (
	"context"
	"errors"
	"testing"

	"github.com/gogf/gf/v2/errors/gcode"
	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/test/gtest"
)

// operationAdapter is a fake adapter that records operation requests.
type operationAdapter struct {
	pingCalled    bool
	closeCalled   bool
	performReq    *Request
	searchReq     *SearchRequest
	bulkReq       *BulkRequest
	infoResponse  *InfoResponse
	performResult *Response
	searchResult  *SearchResponse
	bulkResult    *BulkResponse
	clientObject  any
	err           error
}

// Ping records that the ping operation was called.
func (a *operationAdapter) Ping(_ context.Context) error {
	a.pingCalled = true
	return a.err
}

// Info returns the configured info response.
func (a *operationAdapter) Info(_ context.Context) (*InfoResponse, error) {
	return a.infoResponse, a.err
}

// Perform records the raw request and returns the configured response.
func (a *operationAdapter) Perform(_ context.Context, req *Request) (*Response, error) {
	a.performReq = req
	return a.performResult, a.err
}

// Search records the search request and returns the configured response.
func (a *operationAdapter) Search(_ context.Context, req *SearchRequest) (*SearchResponse, error) {
	a.searchReq = req
	return a.searchResult, a.err
}

// Bulk records the bulk request and returns the configured response.
func (a *operationAdapter) Bulk(_ context.Context, req *BulkRequest) (*BulkResponse, error) {
	a.bulkReq = req
	return a.bulkResult, a.err
}

// Close records that the close operation was called.
func (a *operationAdapter) Close(_ context.Context) error {
	a.closeCalled = true
	return a.err
}

// Client returns the configured native client object.
func (a *operationAdapter) Client() any {
	return a.clientObject
}

// Ping implements Adapter for the T-002 fake adapter.
func (a *fakeAdapter) Ping(_ context.Context) error {
	return nil
}

// Info implements Adapter for the T-002 fake adapter.
func (a *fakeAdapter) Info(_ context.Context) (*InfoResponse, error) {
	return &InfoResponse{Name: a.id}, nil
}

// Perform implements Adapter for the T-002 fake adapter.
func (a *fakeAdapter) Perform(_ context.Context, _ *Request) (*Response, error) {
	return &Response{}, nil
}

// Search implements Adapter for the T-002 fake adapter.
func (a *fakeAdapter) Search(_ context.Context, _ *SearchRequest) (*SearchResponse, error) {
	return &SearchResponse{}, nil
}

// Bulk implements Adapter for the T-002 fake adapter.
func (a *fakeAdapter) Bulk(_ context.Context, _ *BulkRequest) (*BulkResponse, error) {
	return &BulkResponse{}, nil
}

// Close implements Adapter for the T-002 fake adapter.
func (a *fakeAdapter) Close(_ context.Context) error {
	return nil
}

// Client implements Adapter for the T-002 fake adapter.
func (a *fakeAdapter) Client() any {
	return a
}

// Test_OperationForwarding verifies that Search forwards all operations to its adapter.
func Test_OperationForwarding(t *testing.T) {
	gtest.C(t, func(t *gtest.T) {
		nativeClient := map[string]string{"client": "native"}
		adapter := &operationAdapter{
			infoResponse: &InfoResponse{
				Name:        "node-1",
				ClusterName: "cluster-1",
				Version:     map[string]any{"number": "1.0.0"},
			},
			performResult: &Response{StatusCode: 200, Body: []byte(`{"ok":true}`)},
			searchResult:  &SearchResponse{Took: 3},
			bulkResult:    &BulkResponse{Took: 5},
			clientObject:  nativeClient,
		}
		search, err := NewWithAdapter(adapter)
		t.AssertNil(err)

		err = search.Ping(context.Background())
		t.AssertNil(err)
		t.Assert(adapter.pingCalled, true)

		info, err := search.Info(context.Background())
		t.AssertNil(err)
		t.Assert(info.Name, "node-1")
		t.Assert(info.ClusterName, "cluster-1")

		rawReq := &Request{
			Method:  RequestMethodGet,
			Path:    "/_cluster/health",
			Query:   map[string]string{"pretty": "true"},
			Headers: map[string]string{"x-test": "1"},
			Body:    []byte(`{}`),
		}
		rawResp, err := search.Perform(context.Background(), rawReq)
		t.AssertNil(err)
		t.Assert(adapter.performReq, rawReq)
		t.Assert(rawResp.StatusCode, 200)
		t.Assert(rawResp.Body, []byte(`{"ok":true}`))

		searchReq := &SearchRequest{
			Index: []string{"index-a", "index-b"},
			Query: map[string]string{"allow_partial_search_results": "true"},
			Body:  []byte(`{"query":{"match_all":{}}}`),
		}
		searchResp, err := search.Search(context.Background(), searchReq)
		t.AssertNil(err)
		t.Assert(adapter.searchReq, searchReq)
		t.Assert(searchResp.Took, 3)

		bulkReq := &BulkRequest{
			Index: "index-a",
			Body:  []byte("{\"index\":{}}\n{\"name\":\"goframe\"}\n"),
		}
		bulkResp, err := search.Bulk(context.Background(), bulkReq)
		t.AssertNil(err)
		t.Assert(adapter.bulkReq, bulkReq)
		t.Assert(bulkResp.Took, 5)
		t.Assert(search.Client(), nativeClient)

		err = search.Close(context.Background())
		t.AssertNil(err)
		t.Assert(adapter.closeCalled, true)
	})
}

// Test_OperationErrorForwarding verifies that adapter errors are returned unchanged.
func Test_OperationErrorForwarding(t *testing.T) {
	gtest.C(t, func(t *gtest.T) {
		expectedErr := errors.New("adapter failed")
		search, err := NewWithAdapter(&operationAdapter{err: expectedErr})
		t.AssertNil(err)

		err = search.Ping(context.Background())
		t.Assert(err, expectedErr)
		_, err = search.Info(context.Background())
		t.Assert(err, expectedErr)
		_, err = search.Perform(context.Background(), &Request{})
		t.Assert(err, expectedErr)
		_, err = search.Search(context.Background(), &SearchRequest{})
		t.Assert(err, expectedErr)
		_, err = search.Bulk(context.Background(), &BulkRequest{})
		t.Assert(err, expectedErr)
		err = search.Close(context.Background())
		t.Assert(err, expectedErr)
	})
}

// Test_SearchResponsePartialResult verifies timeout and shard failure metadata.
func Test_SearchResponsePartialResult(t *testing.T) {
	gtest.C(t, func(t *gtest.T) {
		search, err := NewWithAdapter(&operationAdapter{
			searchResult: &SearchResponse{
				Took:     10,
				TimedOut: true,
				Shards: ShardsInfo{
					Total:      3,
					Successful: 2,
					Skipped:    0,
					Failed:     1,
					Failures: []ShardFailure{
						{
							Index:  "logs-2026",
							Shard:  "1",
							Status: "500",
							Reason: &ErrorDetails{
								Type:   "query_shard_exception",
								Reason: "failed to create query",
							},
						},
					},
				},
				Hits: map[string]any{"total": map[string]any{"value": 1.0, "relation": "eq"}},
			},
		})
		t.AssertNil(err)

		response, err := search.Search(context.Background(), &SearchRequest{Index: []string{"logs-2026"}})
		t.AssertNil(err)
		t.Assert(response.TimedOut, true)
		t.Assert(response.Shards.Failed, 1)
		t.Assert(response.Shards.Failures[0].Reason.Type, "query_shard_exception")
		t.Assert(response.Hits["total"].(map[string]any)["relation"], "eq")
	})
}

// Test_BulkResponsePerItemError verifies bulk item-level error metadata.
func Test_BulkResponsePerItemError(t *testing.T) {
	gtest.C(t, func(t *gtest.T) {
		search, err := NewWithAdapter(&operationAdapter{
			bulkResult: &BulkResponse{
				Took:   35,
				Errors: true,
				Items: []BulkItem{
					{
						Operation: BulkOperationIndex,
						Index:     "movies",
						ID:        "1",
						Status:    201,
						Result:    "created",
						Shards:    ShardsInfo{Total: 1, Successful: 1},
					},
					{
						Operation: BulkOperationUpdate,
						Index:     "movies",
						ID:        "missing",
						Status:    404,
						Error: &ErrorDetails{
							Type:      "document_missing_exception",
							Reason:    "[missing]: document missing",
							Index:     "movies",
							Shard:     "0",
							IndexUUID: "uuid-1",
						},
					},
				},
			},
		})
		t.AssertNil(err)

		response, err := search.Bulk(context.Background(), &BulkRequest{Body: []byte("{}\n")})
		t.AssertNil(err)
		t.Assert(response.Errors, true)
		t.Assert(len(response.Items), 2)
		t.Assert(response.Items[0].Operation, BulkOperationIndex)
		t.Assert(response.Items[1].Operation, BulkOperationUpdate)
		t.Assert(response.Items[1].Error.Type, "document_missing_exception")
		t.Assert(response.Items[1].Error.IndexUUID, "uuid-1")
	})
}

// Test_OperationNilAdapter verifies operation calls fail with a clear configuration error.
func Test_OperationNilAdapter(t *testing.T) {
	gtest.C(t, func(t *gtest.T) {
		var search *Search
		err := search.Ping(context.Background())
		t.AssertNE(err, nil)
		t.Assert(gerror.Code(err), gcode.CodeInvalidConfiguration)
		t.Assert(search.Client(), nil)
	})
}
