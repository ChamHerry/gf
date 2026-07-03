// Copyright GoFrame Author(https://goframe.org). All Rights Reserved.
//
// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this file,
// You can obtain one at https://github.com/gogf/gf.

// Request types for raw, search, and bulk search engine operations.

package gsearch

// RequestMethod is the HTTP-style method used by raw search engine requests.
type RequestMethod string

const (
	// RequestMethodGet identifies a GET request.
	RequestMethodGet RequestMethod = "GET"

	// RequestMethodPost identifies a POST request.
	RequestMethodPost RequestMethod = "POST"

	// RequestMethodPut identifies a PUT request.
	RequestMethodPut RequestMethod = "PUT"

	// RequestMethodDelete identifies a DELETE request.
	RequestMethodDelete RequestMethod = "DELETE"

	// RequestMethodHead identifies a HEAD request.
	RequestMethodHead RequestMethod = "HEAD"
)

// BulkOperation is the operation name returned by bulk APIs.
type BulkOperation string

const (
	// BulkOperationIndex identifies an index operation.
	BulkOperationIndex BulkOperation = "index"

	// BulkOperationCreate identifies a create operation.
	BulkOperationCreate BulkOperation = "create"

	// BulkOperationUpdate identifies an update operation.
	BulkOperationUpdate BulkOperation = "update"

	// BulkOperationDelete identifies a delete operation.
	BulkOperationDelete BulkOperation = "delete"
)

// Request is a raw REST-style request passed to a concrete adapter.
type Request struct {
	// Method contains the HTTP-style request method.
	Method RequestMethod `json:"method"`

	// Path contains the request path, such as "/_cluster/health".
	Path string `json:"path"`

	// Query contains query string parameters.
	Query map[string]string `json:"query"`

	// Headers contains request headers.
	Headers map[string]string `json:"headers"`

	// Body contains an already-encoded request body.
	Body []byte `json:"body"`
}

// SearchRequest is a generic request for the search endpoint.
type SearchRequest struct {
	// Index contains target indexes. Empty means adapter default or all indexes.
	Index []string `json:"index"`

	// Query contains query string parameters.
	Query map[string]string `json:"query"`

	// Headers contains request headers.
	Headers map[string]string `json:"headers"`

	// Body contains an already-encoded JSON search body.
	Body []byte `json:"body"`
}

// BulkRequest is a generic request for bulk operations.
type BulkRequest struct {
	// Index contains an optional target index for the bulk endpoint.
	Index string `json:"index"`

	// Query contains query string parameters.
	Query map[string]string `json:"query"`

	// Headers contains request headers.
	Headers map[string]string `json:"headers"`

	// Body contains an already-encoded NDJSON bulk body.
	Body []byte `json:"body"`
}
