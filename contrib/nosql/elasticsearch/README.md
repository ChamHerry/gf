# elasticsearch

Package `elasticsearch` registers a `database/gsearch` adapter backed by the official Elasticsearch Go client.

# Installation

```bash
go get -u github.com/gogf/gf/contrib/nosql/elasticsearch/v2
```

# Usage

## Enable the adapter

Import the module for side-effect registration before calling `g.Search()` or `gsearch.New()`.

```go
package main

import (
	_ "github.com/gogf/gf/contrib/nosql/elasticsearch/v2"

	"github.com/gogf/gf/v2/database/gsearch"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gctx"
)

func main() {
	ctx := gctx.New()
	response, err := g.Search().Search(ctx, &gsearch.SearchRequest{
		Index: []string{"movies"},
		Body:  []byte(`{"query":{"match_all":{}}}`),
	})
	if err != nil {
		panic(err)
	}
	g.Dump(response.Hits)
}
```

## Configuration

```yaml
search:
  default:
    type: elasticsearch
    addresses:
      - "http://127.0.0.1:9200"
  secured:
    type: elasticsearch
    addresses:
      - "https://127.0.0.1:9200"
    username: "search-user"
    password: "search-password"
```

Use placeholder credentials only in examples. Put real credentials in your application's secret management flow.

# Operations

## Search

```go
response, err := g.Search().Search(ctx, &gsearch.SearchRequest{
	Index: []string{"movies"},
	Query: map[string]string{"allow_partial_search_results": "true"},
	Body:  []byte(`{"query":{"match":{"title":"goframe"}}}`),
})
if err != nil {
	panic(err)
}
if response.TimedOut || response.Shards.Failed > 0 {
	g.Dump(response.Shards.Failures)
}
```

## Bulk

```go
response, err := g.Search().Bulk(ctx, &gsearch.BulkRequest{
	Index: "movies",
	Body: []byte(
		"{\"index\":{\"_id\":\"1\"}}\n" +
			"{\"title\":\"GoFrame\"}\n",
	),
})
if err != nil {
	panic(err)
}
if response.Errors {
	g.Dump(response.Items)
}
```

## Raw request

```go
response, err := g.Search().Perform(ctx, &gsearch.Request{
	Method: gsearch.RequestMethodGet,
	Path:   "/_cluster/health",
})
if err != nil {
	panic(err)
}
g.Dump(response.StatusCode, response.Body)
```

## Native client escape hatch

`Client()` returns the official `*elasticsearch.Client` value as `any`. Use it when you need an Elasticsearch API that is not modeled by `database/gsearch`.

# Configuration Fields

| Field | Description |
|---|---|
| `type` | Must be `elasticsearch`. |
| `addresses` | Elasticsearch node URLs. |
| `username` / `password` | Basic authentication. |
| `apiKey` | Elasticsearch API key authentication. |
| `serviceToken` | Elasticsearch service-token authentication. |
| `cloudId` | Elastic Cloud deployment ID. |
| `headers` | Additional HTTP headers. |
| `caCert` / `certificateFingerprint` | TLS trust material. |
| `tls` / `tlsSkipVerify` | TLS transport behavior. |
| `retryOnStatus` / `maxRetries` | Retry policy. |
| `compressRequestBody` | Request body compression. |
| `discoverNodesOnStart` | Initial node discovery. |

# Compatibility

This module uses `github.com/elastic/go-elasticsearch/v8 v8.19.6`. The dependency is isolated in this contrib module and is not added to the root `go.mod`.

Use this adapter for Elasticsearch clusters. Do not use it as an OpenSearch compatibility layer; import `contrib/nosql/opensearch/v2` for OpenSearch.

# Error Semantics

Search and bulk requests can partially fail with an HTTP success status. Always inspect:

| Response | Fields |
|---|---|
| `SearchResponse` | `TimedOut`, `Shards.Failed`, `Shards.Failures` |
| `BulkResponse` | `Errors`, `Items[].Error` |

# License and Patent Boundary

`GoFrame elasticsearch` is licensed under the [MIT License](../../../LICENSE). The official Elasticsearch Go client is licensed by its upstream project.

This adapter calls public REST APIs and does not implement server-side indexing, ranking, query planning, vector search, or storage algorithms. Patent review is still required before adding advanced server-like search features.
