# Tasks: elasticsearch-opensearch-search-support

## 0. OpenSpec Artifacts

- [x] 0.1 Create `.openspec.yaml`, `proposal.md`, `design.md`, `tasks.md`, and delta spec artifacts for this change.
- [x] 0.2 Validate the change with `openspec validate elasticsearch-opensearch-search-support --strict` when the OpenSpec CLI is available.

## 1. Root Abstraction — Configuration, Registry, Instances (`database/gsearch/`)

- [x] 1.1 Create `database/gsearch/gsearch.go` with package-level constants, package comment, and root search client type.
- [x] 1.2 Create `database/gsearch/gsearch_config.go` with `EngineType`, shared `Config`, config parsing, `SetConfig`, and `GetConfig`.
- [x] 1.3 Create `database/gsearch/gsearch_adapter.go` with `Adapter`, `AdapterFunc`, and typed `RegisterAdapterFunc`.
- [x] 1.4 Create `database/gsearch/gsearch_instance.go` with `New`, `NewWithAdapter`, and `Instance(name...)`.
- [x] 1.5 Add fake adapter unit tests covering adapter registration, missing adapter errors, duplicate typed registration behavior, multiple engine types, and instance reuse.
- [x] 1.6 Verify: `go test ./database/gsearch/... -count=1 -race`.

## 2. Root Abstraction — Operations and Responses (`database/gsearch/`)

- [x] 2.1 Create `gsearch_request.go` with raw REST request fields: method, path, query, headers, and body.
- [x] 2.2 Create `gsearch_response.go` with raw `Response`, `InfoResponse`, `SearchResponse`, shard metadata, `BulkResponse`, bulk item metadata, and error details.
- [x] 2.3 Create `gsearch_operation.go` to expose `Ping`, `Info`, `Perform`, `Search`, `Bulk`, `Close`, and `Client` through the root client.
- [x] 2.4 Add tests for operation forwarding, error forwarding, bulk per-item errors, search timeout, and shard failures.
- [x] 2.5 Verify: `go test ./database/gsearch/... -count=1 -race -cover`.

## 3. Facade and Configuration Chain (`frame/g`, `frame/gins`, `internal/consts`)

- [x] 3.1 Add `ConfigNodeNameSearch` to `internal/consts`.
- [x] 3.2 Create `frame/gins/gins_search.go` to read `search.<group>` configuration and create `gsearch` instances.
- [x] 3.3 Add `Search(name ...string)` facade to `frame/g/g_object.go`.
- [x] 3.4 Add tests for default group, named group, missing config, and missing adapter behavior.
- [x] 3.5 Verify: `go test ./frame/g/... ./frame/gins/... -count=1 -race`.

## 4. Elasticsearch Contrib Adapter (`contrib/nosql/elasticsearch/`)

- [x] 4.1 Create an independent Go module with `replace github.com/gogf/gf/v2 => ../../../`.
- [x] 4.2 Select a stable Elastic official Go client major that fits the module's Go-version policy and document the decision.
- [x] 4.3 Create adapter config mapping from `gsearch.Config` to Elastic client config.
- [x] 4.4 Register `gsearch.EngineTypeElasticsearch` in `init()`.
- [x] 4.5 Implement `Ping`, `Info`, `Perform`, `Search`, `Bulk`, `Close`, and `Client`.
- [x] 4.6 Add `httptest.Server` tests for request mapping, authentication headers, raw response handling, product mismatch, search partial failures, and bulk per-item errors.
- [x] 4.7 Verify: `cd contrib/nosql/elasticsearch && go test ./... -count=1 -race`.

## 5. OpenSearch Contrib Adapter (`contrib/nosql/opensearch/`)

- [x] 5.1 Create an independent Go module with `replace github.com/gogf/gf/v2 => ../../../`.
- [x] 5.2 Select a stable OpenSearch official Go client major and document the compatibility matrix.
- [x] 5.3 Create adapter config mapping from `gsearch.Config` to OpenSearch client config, keeping AWS SigV4 and OpenSearch-specific fields adapter-local.
- [x] 5.4 Register `gsearch.EngineTypeOpenSearch` in `init()`.
- [x] 5.5 Implement `Ping`, `Info`, `Perform`, `Search`, `Bulk`, `Close`, and `Client`.
- [x] 5.6 Add `httptest.Server` tests for request mapping, authentication headers, raw response handling, product mismatch, search partial failures, and bulk per-item errors.
- [x] 5.7 Verify: `cd contrib/nosql/opensearch && go test ./... -count=1 -race`.

## 6. Documentation

- [x] 6.1 Add `contrib/nosql/elasticsearch/README.md`.
- [x] 6.2 Add `contrib/nosql/elasticsearch/README.zh_CN.md`.
- [x] 6.3 Add `contrib/nosql/opensearch/README.md`.
- [x] 6.4 Add `contrib/nosql/opensearch/README.zh_CN.md`.
- [x] 6.5 Document blank imports, `search` configuration, basic search, bulk usage, raw `Perform`, official-client escape hatch, version compatibility, licensing, and patent-risk boundaries.
- [x] 6.6 Verify documentation does not contain real credentials or production endpoints.

## 7. Final Verification and Review

- [x] 7.1 Run `go test ./database/gsearch/... -count=1 -race`.
- [x] 7.2 Run `go test ./frame/g/... ./frame/gins/... -count=1 -race`.
- [x] 7.3 Run `cd contrib/nosql/elasticsearch && go test ./... -count=1 -race`.
- [x] 7.4 Run `cd contrib/nosql/opensearch && go test ./... -count=1 -race`.
- [x] 7.5 Run `make tidy`.
- [x] 7.6 Run `make lint`.
- [x] 7.7 Run `/gf-review` or the equivalent repository review checklist.
- [x] 7.8 Sync the implementation console docs directory to final status.

## 8. Docker End-to-End Verification

- [x] 8.1 Add opt-in Docker-backed Elasticsearch E2E coverage that is skipped unless `GF_SEARCH_E2E_ELASTICSEARCH_URL` is set.
- [x] 8.2 Add opt-in Docker-backed OpenSearch E2E coverage that is skipped unless `GF_SEARCH_E2E_OPENSEARCH_URL` is set.
- [x] 8.3 Start `docker.elastic.co/elasticsearch/elasticsearch:8.19.6` on `127.0.0.1:19200` and verify the real Ping/Info/Index/Search/Bulk/Facade chain.
- [x] 8.4 Start `opensearchproject/opensearch:2.11.0` on `127.0.0.1:19201` and verify the real Ping/Info/Index/Search/Bulk/Facade chain.
- [x] 8.5 Run default contrib tests without E2E environment variables to confirm the Docker tests remain opt-in.
- [x] 8.6 Run `make tidy` and `make lint` after adding the E2E tests.
