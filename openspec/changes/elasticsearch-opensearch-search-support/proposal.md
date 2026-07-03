## Why

GoFrame currently provides SQL database support through `database/gdb` and concrete SQL drivers under `contrib/drivers/*`. It also provides Redis support through a lighter root abstraction in `database/gredis` with the concrete implementation in `contrib/nosql/redis`.

Elasticsearch and OpenSearch are search and document engines. They expose HTTP JSON REST APIs and support document indexing, search, bulk ingestion, and index management, but they are not `database/sql` drivers, do not provide SQL transaction semantics, and do not fit GoFrame's ORM, DAO generation, or table-metadata model. Users currently need to import the official clients directly and cannot use a GoFrame-style configuration and facade.

Elastic and OpenSearch clients have also diverged in compatibility, authentication, generated APIs, and product behavior. A single mixed adapter would hide those differences and make compatibility errors harder to diagnose.

## What Changes

### Root search abstraction

- Add a new `database/gsearch` package that defines a lightweight, driver-agnostic search/document-engine abstraction.
- Define `EngineType` constants for `elasticsearch` and `opensearch`.
- Define a root `Config` type for shared settings such as addresses, basic auth, API key, TLS options, retry options, request compression, and an `Extra` field for adapter-specific options.
- Add typed adapter registration by engine type so Elasticsearch and OpenSearch adapters can coexist in the same process.
- Add instance/config helpers similar to GoFrame's existing component style.
- Add tests for missing adapters, typed adapter dispatch, and instance behavior.

### Common operation model

- Add driver-agnostic request and response models for `Perform`, `Ping`, `Info`, `Search`, and `Bulk`.
- Keep raw REST escape hatches without exposing Elastic or OpenSearch generated typed API types from the root package.
- Expose search partial-state details such as shard failures and timeouts.
- Expose bulk per-item failures instead of treating HTTP 2xx as total success.
- Keep `Client() any` for applications that need official-client functionality outside the stable root abstraction.

### GoFrame facade and configuration

- Add a `search` configuration node.
- Add `gins.Search(name...)` and `g.Search(name...)` to create and retrieve search clients from configuration groups.
- Keep existing `database`, `redis`, SQL driver, and DAO flows unchanged.

### Concrete contrib adapters

- Add `contrib/nosql/elasticsearch` as an independent Go module using Elastic's official Go client.
- Add `contrib/nosql/opensearch` as an independent Go module using OpenSearch's official Go client.
- Register each adapter through `init()` under its own engine type.
- Use `httptest.Server` tests for request mapping, auth headers, raw requests, search partial failures, and bulk per-item errors.

### Documentation

- Add English and Chinese README files for each new contrib module.
- Document configuration, blank imports, basic search/bulk examples, version compatibility, licensing, and patent-risk boundaries.

## Capabilities

### New Capabilities

- `elasticsearch-opensearch-search-support`: GoFrame applications can configure and use Elasticsearch or OpenSearch through a common `gsearch` abstraction and optional `g.Search()` facade while keeping concrete SDK dependencies in contrib modules.

### Modified Capabilities

- None.

## Impact

- `database/gsearch/` — new root package with no official Elasticsearch/OpenSearch SDK dependency.
- `frame/gins/gins_search.go` — new search instance facade that reads `search.<group>` configuration.
- `frame/g/g_object.go` — add the public `g.Search(name...)` facade.
- `internal/consts/consts.go` — add `ConfigNodeNameSearch`.
- `contrib/nosql/elasticsearch/` — new independent module and tests.
- `contrib/nosql/opensearch/` — new independent module and tests.
- New documentation files in both contrib modules.
- No changes to `database/gdb/`, `contrib/drivers/*`, or DAO generation commands.
