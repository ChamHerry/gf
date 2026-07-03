## ADDED Requirements

### Requirement: Search abstraction root package
The framework SHALL provide a new `database/gsearch` root package for search and document engine integrations without importing Elasticsearch or OpenSearch SDK packages into the root module.

#### Scenario: Root package builds without search SDK dependencies
- **WHEN** the root module is tested or listed
- **THEN** `database/gsearch` SHALL compile without importing Elastic or OpenSearch official client packages
- **AND** the root `go.mod` SHALL NOT gain Elastic or OpenSearch official SDK requirements

#### Scenario: User creates a search client through the root package
- **WHEN** a user calls `gsearch.New(config)` with a registered engine type
- **THEN** the root package SHALL create a search client backed by the matching adapter

### Requirement: Engine-typed adapter registry
The `database/gsearch` package SHALL provide adapter registration keyed by `EngineType`, allowing Elasticsearch and OpenSearch adapters to be registered in the same process.

#### Scenario: Elasticsearch and OpenSearch adapters coexist
- **WHEN** both contrib adapters are imported
- **AND** two configs use `type: elasticsearch` and `type: opensearch`
- **THEN** each client SHALL be created through the matching adapter factory

#### Scenario: Adapter is missing
- **WHEN** a config references an engine type without a registered adapter
- **THEN** client creation SHALL return an error that clearly identifies the missing adapter and the expected contrib import path

### Requirement: Common operation interface
The root search client SHALL expose common operations for `Ping`, `Info`, `Perform`, `Search`, `Bulk`, `Close`, and `Client`.

#### Scenario: Raw REST request through Perform
- **WHEN** a user creates a `gsearch.Request` with method, path, query, headers, and body
- **THEN** `Perform` SHALL pass that request to the adapter and return a raw `gsearch.Response`

#### Scenario: Official client escape hatch
- **WHEN** a user needs engine-specific API coverage that the root abstraction does not expose
- **THEN** `Client()` SHALL return the underlying official client as `any`
- **AND** the root package SHALL NOT expose official generated typed API types in its public signatures

### Requirement: Search partial failure visibility
Search responses SHALL expose timeout, shard, and partial-failure information so callers can distinguish complete search results from partial results.

#### Scenario: Search response has shard failures
- **WHEN** an adapter receives a search response with failed shards
- **THEN** `SearchResponse` SHALL expose the shard failure details
- **AND** the response SHALL indicate partial or incomplete search state

#### Scenario: Search response times out
- **WHEN** an adapter receives a search response with a timeout flag
- **THEN** `SearchResponse` SHALL expose the timeout state

### Requirement: Bulk per-item failure visibility
Bulk responses SHALL expose top-level and per-item errors instead of treating HTTP success as total operation success.

#### Scenario: Bulk response has item errors
- **WHEN** the server returns HTTP 2xx with individual bulk item failures
- **THEN** `BulkResponse` SHALL expose the top-level error state
- **AND** each failed item SHALL expose operation type, status, document metadata, and error details

### Requirement: Search configuration node and facade
The framework SHALL provide a `search` configuration node and GoFrame-style facade access through `gins.Search(name...)` and `g.Search(name...)`.

#### Scenario: Default search group
- **WHEN** the configuration contains `search.default`
- **AND** a user calls `g.Search()`
- **THEN** the framework SHALL create or return the default search client

#### Scenario: Named search group
- **WHEN** the configuration contains `search.logs`
- **AND** a user calls `g.Search("logs")`
- **THEN** the framework SHALL create or return the named search client

#### Scenario: Search config is missing
- **WHEN** a user calls `g.Search("missing")` without a matching config group
- **THEN** the framework SHALL return a clear configuration error or nil result consistent with existing GoFrame facade conventions

### Requirement: Elasticsearch contrib adapter
The repository SHALL provide an independent `contrib/nosql/elasticsearch` module that implements `gsearch.Adapter` using Elastic's official Go client.

#### Scenario: Elasticsearch adapter registers itself
- **WHEN** a user blank-imports the Elasticsearch contrib module
- **THEN** the module SHALL register an adapter factory for `gsearch.EngineTypeElasticsearch`

#### Scenario: Elasticsearch adapter maps common config
- **WHEN** `gsearch.Config` contains addresses and authentication options
- **THEN** the Elasticsearch adapter SHALL map supported common fields to Elastic official client configuration

### Requirement: OpenSearch contrib adapter
The repository SHALL provide an independent `contrib/nosql/opensearch` module that implements `gsearch.Adapter` using OpenSearch's official Go client.

#### Scenario: OpenSearch adapter registers itself
- **WHEN** a user blank-imports the OpenSearch contrib module
- **THEN** the module SHALL register an adapter factory for `gsearch.EngineTypeOpenSearch`

#### Scenario: OpenSearch adapter keeps product-specific options local
- **WHEN** OpenSearch-specific options such as AWS SigV4 are required
- **THEN** those options SHALL be handled in the OpenSearch contrib adapter without polluting the root `gsearch.Config` public API

### Requirement: Product compatibility boundaries
The documentation and adapter behavior SHALL avoid promising Elasticsearch/OpenSearch cross-client compatibility.

#### Scenario: User reads contrib documentation
- **WHEN** a user reads either contrib module README
- **THEN** the README SHALL document supported product/client versions and state that Elasticsearch and OpenSearch adapters are selected explicitly by engine type

#### Scenario: Adapter detects product mismatch
- **WHEN** an adapter can determine that it is connected to the wrong product
- **THEN** it SHALL return a clear product-mismatch error

### Requirement: Documentation mirrors
Each new contrib module SHALL provide English `README.md` and Chinese `README.zh_CN.md` documentation.

#### Scenario: New contrib module is documented
- **WHEN** the Elasticsearch or OpenSearch contrib module is added
- **THEN** both `README.md` and `README.zh_CN.md` SHALL exist in that module directory
- **AND** both documents SHALL include configuration, usage, compatibility, licensing, and risk notes
