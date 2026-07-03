// Copyright GoFrame Author(https://goframe.org). All Rights Reserved.
//
// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT was not distributed with this file,
// You can obtain one at https://github.com/gogf/gf.

// Elasticsearch adapter operations.

package elasticsearch

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	elasticv8 "github.com/elastic/go-elasticsearch/v8"

	"github.com/gogf/gf/v2/database/gsearch"
	"github.com/gogf/gf/v2/errors/gcode"
	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/util/gconv"
)

const (
	// headerContentType is the HTTP Content-Type header name.
	headerContentType = "Content-Type"

	// contentTypeJSON is the JSON request content type.
	contentTypeJSON = "application/json"

	// contentTypeNDJSON is the NDJSON request content type.
	contentTypeNDJSON = "application/x-ndjson"
)

// Ping verifies whether the Elasticsearch cluster is reachable.
func (e *Elasticsearch) Ping(ctx context.Context) error {
	response, err := e.performHTTP(ctx, gsearch.RequestMethodHead, "/", nil, nil, nil, "")
	if err != nil {
		return err
	}
	return errorForStatus(response)
}

// Info returns basic Elasticsearch cluster information.
func (e *Elasticsearch) Info(ctx context.Context) (*gsearch.InfoResponse, error) {
	response, err := e.performHTTP(ctx, gsearch.RequestMethodGet, "/", nil, nil, nil, "")
	if err != nil {
		return nil, err
	}
	if err = errorForStatus(response); err != nil {
		return nil, err
	}
	payload, err := decodeResponseBody(response)
	if err != nil {
		return nil, err
	}
	return &gsearch.InfoResponse{
		Name:        gconv.String(payload["name"]),
		ClusterName: gconv.String(payload["cluster_name"]),
		ClusterUUID: gconv.String(payload["cluster_uuid"]),
		Version:     gconv.Map(payload["version"]),
		Tagline:     gconv.String(payload["tagline"]),
		Raw:         response,
	}, nil
}

// Perform executes a raw Elasticsearch request.
func (e *Elasticsearch) Perform(ctx context.Context, req *gsearch.Request) (*gsearch.Response, error) {
	if req == nil {
		req = &gsearch.Request{}
	}
	method := req.Method
	if method == "" {
		method = gsearch.RequestMethodGet
	}
	return e.performHTTP(ctx, method, req.Path, req.Query, req.Headers, req.Body, contentTypeJSON)
}

// Search executes an Elasticsearch search request.
func (e *Elasticsearch) Search(ctx context.Context, req *gsearch.SearchRequest) (*gsearch.SearchResponse, error) {
	if req == nil {
		req = &gsearch.SearchRequest{}
	}
	response, err := e.performHTTP(
		ctx,
		gsearch.RequestMethodPost,
		searchPath(req.Index),
		req.Query,
		req.Headers,
		req.Body,
		contentTypeJSON,
	)
	if err != nil {
		return nil, err
	}
	payload, err := decodeResponseBody(response)
	if err != nil {
		return nil, err
	}
	return &gsearch.SearchResponse{
		Took:         gconv.Int64(payload["took"]),
		TimedOut:     gconv.Bool(payload["timed_out"]),
		Shards:       parseShards(gconv.Map(payload["_shards"])),
		Hits:         gconv.Map(payload["hits"]),
		Aggregations: gconv.Map(payload["aggregations"]),
		Raw:          payload,
		RawResponse:  response,
	}, nil
}

// Bulk executes an Elasticsearch bulk request.
func (e *Elasticsearch) Bulk(ctx context.Context, req *gsearch.BulkRequest) (*gsearch.BulkResponse, error) {
	if req == nil {
		req = &gsearch.BulkRequest{}
	}
	response, err := e.performHTTP(
		ctx,
		gsearch.RequestMethodPost,
		bulkPath(req.Index),
		req.Query,
		req.Headers,
		req.Body,
		contentTypeNDJSON,
	)
	if err != nil {
		return nil, err
	}
	payload, err := decodeResponseBody(response)
	if err != nil {
		return nil, err
	}
	return &gsearch.BulkResponse{
		Took:        gconv.Int64(payload["took"]),
		Errors:      gconv.Bool(payload["errors"]),
		Items:       parseBulkItems(gconv.SliceAny(payload["items"])),
		Raw:         payload,
		RawResponse: response,
	}, nil
}

// Close releases Elasticsearch client resources.
func (e *Elasticsearch) Close(ctx context.Context) error {
	client, err := e.ensureClient()
	if err != nil {
		return err
	}
	return client.Close(ctx)
}

// ensureClient returns the initialized official client.
func (e *Elasticsearch) ensureClient() (*elasticv8.Client, error) {
	if e == nil {
		return nil, gerror.NewCode(gcode.CodeInvalidConfiguration, "elasticsearch adapter is nil")
	}
	if e.initErr != nil {
		return nil, e.initErr
	}
	if e.client == nil {
		return nil, gerror.NewCode(gcode.CodeInvalidConfiguration, "elasticsearch client is nil")
	}
	return e.client, nil
}

// performHTTP executes an HTTP-style request through the official client transport.
func (e *Elasticsearch) performHTTP(
	ctx context.Context,
	method gsearch.RequestMethod,
	path string,
	query map[string]string,
	headers map[string]string,
	body []byte,
	defaultContentType string,
) (*gsearch.Response, error) {
	client, err := e.ensureClient()
	if err != nil {
		return nil, err
	}
	if path == "" {
		path = "/"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	var reader io.Reader
	if len(body) > 0 {
		reader = bytes.NewReader(body)
	}
	httpRequest, err := http.NewRequestWithContext(ctx, string(method), "http://"+path, reader)
	if err != nil {
		return nil, err
	}
	for key, value := range query {
		values := httpRequest.URL.Query()
		values.Set(key, value)
		httpRequest.URL.RawQuery = values.Encode()
	}
	for key, value := range headers {
		httpRequest.Header.Set(key, value)
	}
	if len(body) > 0 && defaultContentType != "" && httpRequest.Header.Get(headerContentType) == "" {
		httpRequest.Header.Set(headerContentType, defaultContentType)
	}
	httpResponse, err := client.Perform(httpRequest)
	if err != nil {
		return nil, err
	}
	return responseFromHTTP(httpResponse)
}

// responseFromHTTP converts an HTTP response to gsearch.Response and closes its body.
func responseFromHTTP(response *http.Response) (*gsearch.Response, error) {
	if response == nil {
		return nil, gerror.NewCode(gcode.CodeInvalidConfiguration, "elasticsearch response is nil")
	}
	var body []byte
	if response.Body != nil {
		readBody, readErr := io.ReadAll(response.Body)
		closeErr := response.Body.Close()
		if readErr != nil {
			return nil, readErr
		}
		if closeErr != nil {
			return nil, closeErr
		}
		body = readBody
	}
	return &gsearch.Response{
		StatusCode: response.StatusCode,
		Headers:    response.Header,
		Body:       body,
	}, nil
}

// decodeResponseBody decodes a JSON response body.
func decodeResponseBody(response *gsearch.Response) (map[string]any, error) {
	payload := map[string]any{}
	if response == nil || len(response.Body) == 0 {
		return payload, nil
	}
	if err := json.Unmarshal(response.Body, &payload); err != nil {
		return nil, err
	}
	return payload, nil
}

// errorForStatus returns an error for non-success HTTP status codes.
func errorForStatus(response *gsearch.Response) error {
	if response != nil && response.StatusCode > 299 {
		return gerror.NewCodef(
			gcode.CodeInternalError,
			"elasticsearch response status %d: %s",
			response.StatusCode,
			string(response.Body),
		)
	}
	return nil
}

// searchPath returns the REST path for a search request.
func searchPath(indexes []string) string {
	if len(indexes) == 0 {
		return "/_search"
	}
	return "/" + strings.Join(indexes, ",") + "/_search"
}

// bulkPath returns the REST path for a bulk request.
func bulkPath(index string) string {
	if index == "" {
		return "/_bulk"
	}
	return "/" + index + "/_bulk"
}

// parseShards converts decoded shard metadata.
func parseShards(m map[string]any) gsearch.ShardsInfo {
	failures := make([]gsearch.ShardFailure, 0)
	for _, item := range gconv.SliceAny(m["failures"]) {
		failureMap := gconv.Map(item)
		failures = append(failures, gsearch.ShardFailure{
			Index:   gconv.String(failureMap["index"]),
			Node:    gconv.String(failureMap["node"]),
			Shard:   gconv.String(failureMap["shard"]),
			Status:  gconv.String(failureMap["status"]),
			Primary: gconv.Bool(failureMap["primary"]),
			Reason:  parseErrorDetails(gconv.Map(failureMap["reason"])),
		})
	}
	return gsearch.ShardsInfo{
		Total:      gconv.Int(m["total"]),
		Successful: gconv.Int(m["successful"]),
		Skipped:    gconv.Int(m["skipped"]),
		Failed:     gconv.Int(m["failed"]),
		Failures:   failures,
	}
}

// parseBulkItems converts decoded bulk items.
func parseBulkItems(items []any) []gsearch.BulkItem {
	bulkItems := make([]gsearch.BulkItem, 0, len(items))
	for _, item := range items {
		for operation, value := range gconv.Map(item) {
			valueMap := gconv.Map(value)
			bulkItems = append(bulkItems, gsearch.BulkItem{
				Operation:   gsearch.BulkOperation(operation),
				Index:       gconv.String(valueMap["_index"]),
				ID:          gconv.String(valueMap["_id"]),
				Status:      gconv.Int(valueMap["status"]),
				Result:      gconv.String(valueMap["result"]),
				Version:     gconv.Int64(valueMap["_version"]),
				SeqNo:       gconv.Int64(valueMap["_seq_no"]),
				PrimaryTerm: gconv.Int64(valueMap["_primary_term"]),
				Shards:      parseShards(gconv.Map(valueMap["_shards"])),
				Error:       parseErrorDetails(gconv.Map(valueMap["error"])),
			})
		}
	}
	return bulkItems
}

// parseErrorDetails converts decoded error metadata.
func parseErrorDetails(m map[string]any) *gsearch.ErrorDetails {
	if len(m) == 0 {
		return nil
	}
	errorDetails := &gsearch.ErrorDetails{
		Type:      gconv.String(m["type"]),
		Reason:    gconv.String(m["reason"]),
		Index:     gconv.String(m["index"]),
		IndexUUID: gconv.String(m["index_uuid"]),
		Shard:     gconv.String(m["shard"]),
		Status:    gconv.Int(m["status"]),
		Metadata:  map[string]any{},
	}
	if causedBy := parseErrorDetails(gconv.Map(m["caused_by"])); causedBy != nil {
		errorDetails.CausedBy = causedBy
	}
	for _, item := range gconv.SliceAny(m["root_cause"]) {
		if rootCause := parseErrorDetails(gconv.Map(item)); rootCause != nil {
			errorDetails.RootCause = append(errorDetails.RootCause, *rootCause)
		}
	}
	for key, value := range m {
		switch key {
		case "type", "reason", "index", "index_uuid", "shard", "status", "caused_by", "root_cause":
			continue
		default:
			errorDetails.Metadata[key] = value
		}
	}
	if len(errorDetails.Metadata) == 0 {
		errorDetails.Metadata = nil
	}
	return errorDetails
}
