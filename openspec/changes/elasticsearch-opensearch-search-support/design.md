## Context

GoFrame's current database-related surfaces are intentionally separated by dependency and semantics:

- `database/gdb` is a SQL ORM abstraction backed by `database/sql`, SQL drivers, models, transactions, table metadata, and DAO generation.
- `database/gredis` is a lighter NoSQL abstraction, with the heavy `go-redis/v9` dependency isolated under `contrib/nosql/redis`.
- Other extension points such as `gcfg` and `gsvc` also keep small root interfaces separate from provider-specific SDK dependencies.

Elasticsearch and OpenSearch are closer to the `gredis` pattern than the `gdb` pattern. They are remote document/search engines reached through REST APIs, with product-specific clients, compatibility matrices, authentication details, bulk semantics, and partial-search failure behavior.

This change creates a new `database/gsearch` abstraction and two concrete contrib modules.

## Goals / Non-Goals

**Goals:**

- Add `database/gsearch` without adding Elasticsearch or OpenSearch SDK dependencies to the root `go.mod`.
- Support both Elasticsearch and OpenSearch in the same process through engine-typed adapter registration.
- Provide stable common operations: `Ping`, `Info`, `Perform`, `Search`, `Bulk`, `Close`, and `Client`.
- Preserve raw REST flexibility through `Perform` and official-client access through `Client() any`.
- Add GoFrame-style configuration and facade access through `search.<group>`, `gins.Search`, and `g.Search`.
- Add independent Elasticsearch and OpenSearch contrib modules with focused tests.
- Document compatibility boundaries and avoid claiming cross-product client compatibility.

**Non-Goals:**

- Do not implement Elasticsearch/OpenSearch as `gdb.Driver`.
- Do not add `contrib/drivers/elasticsearch` or `contrib/drivers/opensearch`.
- Do not change DAO generation, SQL model generation, or `database/gdb`.
- Do not implement a full query DSL builder in the first version.
- Do not expose generated Elastic/OpenSearch typed API types from `database/gsearch`.
- Do not implement OpenSearch gRPC or Elasticsearch/OpenSearch SQL/PPL wrappers.
- Do not implement a search engine server or copy server-side search algorithms.

## Architecture

```text
Application
  -> g.Search("default") or gsearch.Instance("default")
  -> frame/gins.Search()
  -> configuration node: search.<group>
  -> database/gsearch.ConfigFromMap()
  -> database/gsearch.New(config)
  -> adapter registry by EngineType
  -> contrib/nosql/elasticsearch or contrib/nosql/opensearch
  -> official Go client
  -> Elasticsearch/OpenSearch REST API
```

The root package owns only common types and adapter interfaces. Each contrib module owns official-client configuration, request construction, product-specific behavior, and compatibility documentation.

## Root Package Design

### Engine type

```go
type EngineType string

const (
    EngineTypeElasticsearch EngineType = "elasticsearch"
    EngineTypeOpenSearch    EngineType = "opensearch"
)
```

Engine-type constants prevent enum-like raw strings from spreading through implementation code.

### Config

`database/gsearch.Config` should contain shared configuration fields:

- `Type EngineType`
- `Addresses []string`
- `Username string`
- `Password string`
- `APIKey string`
- `ServiceToken string`
- `CloudID string`
- `Headers map[string]string`
- `CACert []byte`
- `CertificateFingerprint string`
- `TLS bool`
- `TLSSkipVerify bool`
- `RetryOnStatus []int`
- `MaxRetries int`
- `CompressRequestBody bool`
- `DiscoverNodesOnStart bool`
- `Extra map[string]any`

The root package may parse common values from configuration, but adapter-specific options must remain in `Extra` or adapter-local config helpers.

### Adapter registry

The registry should be keyed by `EngineType`, not a single default adapter function. This allows both official clients to be blank-imported in the same process:

```go
gsearch.RegisterAdapterFunc(gsearch.EngineTypeElasticsearch, newElasticsearchAdapter)
gsearch.RegisterAdapterFunc(gsearch.EngineTypeOpenSearch, newOpenSearchAdapter)
```

When an adapter is missing, the error should mention the expected contrib import paths.

### Adapter interface

The root adapter should expose stable common behavior:

```go
type Adapter interface {
    Ping(ctx context.Context) error
    Info(ctx context.Context) (*InfoResponse, error)
    Perform(ctx context.Context, req *Request) (*Response, error)
    Search(ctx context.Context, req *SearchRequest) (*SearchResponse, error)
    Bulk(ctx context.Context, req *BulkRequest) (*BulkResponse, error)
    Close(ctx context.Context) error
    Client() any
}
```

`Client() any` is the explicit escape hatch for official-client-specific APIs.

## Request / Response Semantics

### Raw request

`Request` should support method, path, query parameters, headers, and body. It is the compatibility fallback for APIs not covered by first-class helpers.

### Search response

Search responses must expose:

- HTTP status and headers from the raw response.
- Raw body bytes or a parsed document.
- `TimedOut` state.
- Shard totals and shard failures.
- A `Partial` or equivalent state so callers can detect incomplete search results.

### Bulk response

Bulk responses must expose:

- HTTP status and headers from the raw response.
- Raw body bytes or parsed body.
- Top-level `Errors`.
- Per-item operation status and item-level errors.

An HTTP 2xx status does not imply that every bulk item succeeded.

## Facade / Config Design

Add a `search` top-level configuration node:

```yaml
search:
  default:
    type: elasticsearch
    addresses:
      - "https://localhost:9200"
    username: "elastic"
    password: "changeme"
  logs:
    type: opensearch
    addresses:
      - "https://localhost:9200"
```

`frame/gins.Search(name...)` should mirror the existing instance/facade style used by `gins.Redis`, but it must call `gsearch` instead of `gdb` or `gredis`.

`frame/g.Search(name...)` should delegate to `gins.Search(name...)`.

## Elasticsearch Adapter

The Elasticsearch adapter lives in `contrib/nosql/elasticsearch` and should:

- Be an independent Go module.
- Depend on Elastic's official Go client only in the contrib module.
- Register `EngineTypeElasticsearch` in `init()`.
- Map common `gsearch.Config` fields to Elastic client configuration.
- Support Elastic-specific auth and deployment options such as API key, service token, Cloud ID, certificate fingerprint, retry, compression, and transport customization where practical.
- Validate product behavior through `Info` or `Ping` when useful, returning clear product-mismatch errors.
- Implement tests with `httptest.Server`, not a mandatory live Elasticsearch instance.

SDK version selection should preserve the root module's Go 1.23 compatibility. If Elastic v9 requires Go 1.24+, use a compatible stable major in this change or explicitly document why the contrib module requires a newer Go version.

## OpenSearch Adapter

The OpenSearch adapter lives in `contrib/nosql/opensearch` and should:

- Be an independent Go module.
- Depend on OpenSearch's official Go client only in the contrib module.
- Register `EngineTypeOpenSearch` in `init()`.
- Map common `gsearch.Config` fields to OpenSearch client configuration.
- Keep AWS SigV4 and OpenSearch-specific options adapter-local.
- Validate product behavior through `Info` or `Ping` when useful, returning clear product-mismatch errors.
- Implement tests with `httptest.Server`, not a mandatory live OpenSearch instance.

Use a stable released OpenSearch client major, not unreleased APIs from the main branch.

## Compatibility

- Existing `gdb`, `gredis`, `gcfg`, and `gsvc` behavior must remain unchanged.
- Root `go.mod` must not gain Elastic or OpenSearch SDK dependencies.
- Elasticsearch and OpenSearch must be explicitly selected by `search.<group>.type`.
- This change must not claim that Elasticsearch clients are fully compatible with OpenSearch 2.x+ or vice versa.

## Testing Strategy

- Root `database/gsearch` tests use fake adapters.
- Facade tests use local configuration and fake adapter registration.
- Contrib adapter tests use `httptest.Server` for request construction, auth headers, error mapping, search partial failures, and bulk per-item errors.
- Live-service tests may be added later behind opt-in environment variables, but they are not required for default CI.

## Risks / Trade-offs

- **Public API stability:** `database/gsearch` is a new root public package. Keep the first surface small and stable.
- **SDK Go-version constraints:** Elastic and OpenSearch clients may require different Go versions. Keep root dependency-free and document contrib requirements.
- **Partial failure semantics:** Bulk and search APIs can partially fail with HTTP 2xx. Tests must cover item-level and shard-level errors.
- **Adapter divergence:** The two engines share REST concepts but differ in compatibility and auth details. Separate modules reduce hidden branching.
- **Patent and licensing:** This change uses public REST APIs and official Apache-2.0 clients. It does not implement server-side search algorithms. Patent review is still required before adding advanced server-like features.
