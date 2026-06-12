# Tasks: httpclient-tls-engine

## Task 1: go.mod — add dependencies

Update `contrib/sdk/httpclient/go.mod` to add:
- `github.com/refraction-networking/utls`
- `github.com/imroc/req/v3`

Run `go mod tidy` in `contrib/sdk/httpclient/`.

Verify: `go build ./...` succeeds.

- [x] Done — `utls v1.8.1` and `req/v3 v3.57.0` added, `go mod tidy` clean, build passes.

---

## Task 2: Engine interface + types (`httpclient_engine.go`)

Create `contrib/sdk/httpclient/httpclient_engine.go` with:
- `Engine` interface: `CreateTransport() (http.RoundTripper, error)`
- `EngineType` string enum: `EngineTypeGoNative`, `EngineTypeUTLS`, `EngineTypeChromeImpersonate`
- `EngineConfig` struct: Type, ProxyURL, Timeout, Fingerprint, CustomSpec, InsecureSkipVerify
- `FingerprintProfile` string enum: `FingerprintNodeJSV24`, `FingerprintChromeAuto`, `FingerprintFirefoxAuto`, `FingerprintRandomized`, `FingerprintCustom`
- `NewEngine(config EngineConfig) (Engine, error)` factory function

Verify: compiles with stub implementations returning `nil, nil`.

- [x] Done.

---

## Task 3: go_native engine (`httpclient_engine_native.go`)

Create `contrib/sdk/httpclient/httpclient_engine_native.go` with:
- `nativeEngine` struct implementing `Engine`
- `CreateTransport()` returns `*http.Transport`:
  - Clone `http.DefaultTransport.(*http.Transport)`
  - TLS config: `MinVersion: tls.VersionTLS12`, `InsecureSkipVerify: false`
  - Proxy support: HTTP via `transport.Proxy`, SOCKS5 via `DialContext`
  - `MaxIdleConnsPerHost: 10`

Reference: AIProxyV3 `native_client.go` `NewProxyAwareHTTPClient` / `NewProxyHTTPClient`.

Verify: `CreateTransport()` returns non-nil `*http.Transport` with correct TLS config.

- [x] Done.

---

## Task 4: Fingerprint profiles (`httpclient_fingerprint.go`)

Create `contrib/sdk/httpclient/httpclient_fingerprint.go` with:
- `getNodeJSV24ClientHelloSpec()` returning `*utls.ClientHelloSpec` — port from AIProxyV3 `nodejs_v24_fingerprint.go`
- `resolveFingerprint(profile FingerprintProfile, custom *utls.ClientHelloSpec) (utls.ClientHelloID, *utls.ClientHelloSpec, error)` — maps profile to either a utls built-in `ClientHelloID` (for auto/randomized) or a concrete `ClientHelloSpec` (for nodejs_v24/custom)

Verify: each profile resolves without error.

- [x] Done — full Node.js v24.6.0 ClientHelloSpec ported (238 lines, JA3: 944d1e1858cd278718f8a46b65d3212f).

---

## Task 5: uTLS engine (`httpclient_engine_utls.go`)

Create `contrib/sdk/httpclient/httpclient_engine_utls.go` with:
- `utlsEngine` struct: proxyURL, timeout, fingerprint profile, optional custom spec, insecureSkipVerify
- `CreateTransport()` returns `*http.Transport` with custom `DialTLSContext`:
  1. Dial TCP (directly or through proxy)
  2. Create `utls.UClient` with resolved fingerprint
  3. Set SNI from target host
  4. `HandshakeContext()`
  5. Return connection (HTTP/2 auto-negotiated via ALPN + http2.ConfigureTransport)

Reference: AIProxyV3 `nodejs_v24_client.go` dial logic.

Verify: `CreateTransport()` returns non-nil with custom `DialTLSContext`.

- [x] Done — includes HTTP CONNECT and SOCKS5 proxy support, InsecureSkipVerify option.

---

## Task 6: Chrome Impersonate engine (`httpclient_engine_chrome.go`)

Create `contrib/sdk/httpclient/httpclient_engine_chrome.go` with:
- `chromeEngine` struct: proxyURL, timeout
- `chromeTransport` struct implementing `http.RoundTripper`:
  - `RoundTrip(req)` creates fresh `req.C().ImpersonateChrome().SetCookieJar(nil)`
  - Applies proxy and timeout
  - Converts `*http.Request` → `req` request → `*http.Response`

Reference: AIProxyV3 `chrome_impersonate_client.go`.

Verify: `CreateTransport()` returns non-nil `http.RoundTripper`.

- [x] Done.

---

## Task 7: Config + New() integration

Modify `contrib/sdk/httpclient/httpclient_config.go`:
- Add `Engine *EngineConfig` field to `Config`

Modify `contrib/sdk/httpclient/httpclient.go`:
- In `New()`, when `config.Engine != nil`:
  1. `engine, err := NewEngine(*config.Engine)`
  2. `transport, err := engine.CreateTransport()`
  3. `client.Transport = transport`
- When `config.Engine == nil`: existing behavior unchanged

Verify: existing tests still pass; new Engine path sets transport correctly.

- [x] Done — backward compatible, panics on engine creation failure (non-recoverable config error).

---

## Task 8: Unit tests

Create test files with tests covering:
- `NewEngine` for all 3 engine types + unknown type error
- `resolveFingerprint` for all profiles + error cases
- `CreateTransport` for each engine
- Proxy configuration (HTTP, SOCKS5, invalid scheme)
- Integration tests with `httptest` for TLS handshake verification
- HTTP CONNECT proxy integration test with local proxy server
- Error path tests for unreachable proxies

Verify: `go test -count=1 -race ./...` passes with ≥80% coverage on new code.

- [x] Done — 30 tests, 80.4% new code coverage, all pass with race detector.

---

## Task 9: Tidy + lint + build

```bash
cd contrib/sdk/httpclient && go mod tidy
golangci-lint run -c ../../../.golangci.yml ./...
go build ./...
```

Verify: 0 lint issues, clean build.

- [x] Done — 0 lint issues (fixed QF1008 and usestdlibvars), clean build.

---

## Task 10: README documentation

Create `contrib/sdk/httpclient/README.md` and `README.zh_CN.md` with:
- Engine overview and available types
- Usage examples for each engine
- Proxy configuration guide
- Fingerprint profile guide

Verify: both language versions present and consistent.

- [x] Done — both EN and zh_CN READMEs created.
