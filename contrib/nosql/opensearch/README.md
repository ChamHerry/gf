# opensearch

Package `opensearch` registers a `database/gsearch` adapter backed by the official OpenSearch Go client.

# Installation

```bash
go get -u github.com/gogf/gf/contrib/nosql/opensearch/v2
```

# Usage

## Enable the adapter

Import the module for side-effect registration before calling `g.Search()` or `gsearch.New()`.

```go
package main

import (
	_ "github.com/gogf/gf/contrib/nosql/opensearch/v2"

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
    type: opensearch
    addresses:
      - "http://127.0.0.1:9200"
  secured:
    type: opensearch
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

## Adapter-local signer

OpenSearch request signing stays out of the root `database/gsearch` config. Pass an official OpenSearch `signer.Signer` through `Config.Extra` when creating the adapter manually. Import this module normally when you need `ExtraKeySigner`.

```go
search, err := gsearch.New(&gsearch.Config{
	Type:      gsearch.EngineTypeOpenSearch,
	Addresses: []string{"https://search-domain.example.com"},
	Extra: map[string]any{
		opensearch.ExtraKeySigner: mySigner,
	},
})
if err != nil {
	panic(err)
}
if err = search.Ping(ctx); err != nil {
	panic(err)
}
```

## Native client escape hatch

`Client()` returns the official `*opensearch.Client` value as `any`. Use it when you need an OpenSearch API that is not modeled by `database/gsearch`.

# Configuration Fields

| Field | Description |
|---|---|
| `type` | Must be `opensearch`. |
| `addresses` | OpenSearch node URLs. |
| `username` / `password` | Basic authentication. |
| `headers` | Additional HTTP headers. |
| `caCert` | TLS trust material. |
| `tls` / `tlsSkipVerify` | TLS transport behavior. |
| `retryOnStatus` / `maxRetries` | Retry policy. |
| `compressRequestBody` | Request body compression. |
| `discoverNodesOnStart` | Initial node discovery. |
| `extra.signer` | Adapter-local official OpenSearch signer, set with `ExtraKeySigner` in Go code. |

# Compatibility

This module uses `github.com/opensearch-project/opensearch-go/v3 v3.1.0`. At implementation time, `opensearch-go/v4 v4.6.0` required Go 1.24 and `opensearch-go/v5@latest` was an unreleased pseudo-version requiring Go 1.25.9, so the first adapter version uses stable v3 to fit this repository's Go 1.23 module policy.

Use this adapter for OpenSearch clusters. The adapter validates `Info` responses and rejects non-OpenSearch distributions. Do not use it as an Elasticsearch compatibility layer; import `contrib/nosql/elasticsearch/v2` for Elasticsearch.

# Error Semantics

Search and bulk requests can partially fail with an HTTP success status. Always inspect:

| Response | Fields |
|---|---|
| `SearchResponse` | `TimedOut`, `Shards.Failed`, `Shards.Failures` |
| `BulkResponse` | `Errors`, `Items[].Error` |

# License and Patent Boundary

`GoFrame opensearch` is licensed under the [MIT License](../../../LICENSE). The official OpenSearch Go client is Apache-2.0 licensed by its upstream project.

This adapter calls public REST APIs and does not implement server-side indexing, ranking, query planning, vector search, or storage algorithms. Patent review is still required before adding advanced server-like search features.
