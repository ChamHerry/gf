// Copyright GoFrame Author(https://goframe.org). All Rights Reserved.
//
// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this file,
// You can obtain one at https://github.com/gogf/gf.

// Response models for raw, search, and bulk search engine operations.

package gsearch

// Response is a raw adapter response.
type Response struct {
	// StatusCode contains the HTTP-style response status code.
	StatusCode int `json:"statusCode"`

	// Headers contains response headers.
	Headers map[string][]string `json:"headers"`

	// Body contains the raw response body.
	Body []byte `json:"body"`
}

// InfoResponse contains basic server and cluster information.
type InfoResponse struct {
	// Name contains the server node name.
	Name string `json:"name"`

	// ClusterName contains the cluster name.
	ClusterName string `json:"clusterName"`

	// ClusterUUID contains the cluster UUID.
	ClusterUUID string `json:"clusterUuid"`

	// Version contains engine-specific version fields.
	Version map[string]any `json:"version"`

	// Tagline contains an engine-specific tagline.
	Tagline string `json:"tagline"`

	// Raw contains the raw response used to build this object.
	Raw *Response `json:"raw"`
}

// ErrorDetails contains search engine error metadata.
type ErrorDetails struct {
	// Type contains the engine error type.
	Type string `json:"type"`

	// Reason contains a human-readable error reason.
	Reason string `json:"reason"`

	// Index contains the related index name.
	Index string `json:"index"`

	// IndexUUID contains the related index UUID.
	IndexUUID string `json:"indexUuid"`

	// Shard contains the related shard identifier.
	Shard string `json:"shard"`

	// Status contains the item-level HTTP-style status code when available.
	Status int `json:"status"`

	// CausedBy contains the nested engine error cause.
	CausedBy *ErrorDetails `json:"causedBy"`

	// RootCause contains root-cause engine errors.
	RootCause []ErrorDetails `json:"rootCause"`

	// Metadata contains adapter-specific error fields that are not modeled above.
	Metadata map[string]any `json:"metadata"`
}

// ShardsInfo contains shard execution counters and failures.
type ShardsInfo struct {
	// Total contains the total shard count for the request.
	Total int `json:"total"`

	// Successful contains the number of successful shards.
	Successful int `json:"successful"`

	// Skipped contains the number of skipped shards.
	Skipped int `json:"skipped"`

	// Failed contains the number of failed shards.
	Failed int `json:"failed"`

	// Failures contains shard failure details.
	Failures []ShardFailure `json:"failures"`
}

// ShardFailure contains a single shard failure.
type ShardFailure struct {
	// Index contains the failed index name.
	Index string `json:"index"`

	// Node contains the failed node identifier.
	Node string `json:"node"`

	// Shard contains the failed shard identifier.
	Shard string `json:"shard"`

	// Status contains the shard failure status.
	Status string `json:"status"`

	// Primary reports whether the failed shard was a primary shard.
	Primary bool `json:"primary"`

	// Reason contains the engine error details.
	Reason *ErrorDetails `json:"reason"`
}

// SearchResponse contains search results and partial-result metadata.
type SearchResponse struct {
	// Took contains the server-side search duration in milliseconds.
	Took int64 `json:"took"`

	// TimedOut reports whether the search timed out before completion.
	TimedOut bool `json:"timedOut"`

	// Shards contains shard execution counters and failures.
	Shards ShardsInfo `json:"shards"`

	// Hits contains the raw hits section.
	Hits map[string]any `json:"hits"`

	// Aggregations contains the raw aggregations section.
	Aggregations map[string]any `json:"aggregations"`

	// Raw contains the decoded raw response.
	Raw map[string]any `json:"raw"`

	// RawResponse contains the raw response used to build this object.
	RawResponse *Response `json:"rawResponse"`
}

// BulkResponse contains bulk operation results and per-item errors.
type BulkResponse struct {
	// Took contains the server-side bulk duration in milliseconds.
	Took int64 `json:"took"`

	// Errors reports whether one or more bulk items failed.
	Errors bool `json:"errors"`

	// Items contains per-operation results in request order.
	Items []BulkItem `json:"items"`

	// Raw contains the decoded raw response.
	Raw map[string]any `json:"raw"`

	// RawResponse contains the raw response used to build this object.
	RawResponse *Response `json:"rawResponse"`
}

// BulkItem contains one bulk operation result.
type BulkItem struct {
	// Operation contains the bulk operation name.
	Operation BulkOperation `json:"operation"`

	// Index contains the target index.
	Index string `json:"index"`

	// ID contains the target document identifier.
	ID string `json:"id"`

	// Status contains the item-level HTTP-style status code.
	Status int `json:"status"`

	// Result contains the item result, such as created or updated.
	Result string `json:"result"`

	// Version contains the document version when returned.
	Version int64 `json:"version"`

	// SeqNo contains the sequence number assigned by the engine.
	SeqNo int64 `json:"seqNo"`

	// PrimaryTerm contains the primary term assigned by the engine.
	PrimaryTerm int64 `json:"primaryTerm"`

	// Shards contains shard execution counters for the item.
	Shards ShardsInfo `json:"shards"`

	// Error contains item-level error details.
	Error *ErrorDetails `json:"error"`
}
