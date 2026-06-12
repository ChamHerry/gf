# Spec: httpclient-tls-engine

## Overview

Add pluggable TLS transport engines to `contrib/sdk/httpclient`, allowing SDK consumers to select between standard Go TLS, uTLS fingerprint simulation, and Chrome impersonation.

## Requirements

### REQ-001: Engine interface

The module shall define an `Engine` interface with a single method `CreateTransport() (http.RoundTripper, error)` that produces a configured `http.RoundTripper` implementing TLS fingerprinting and proxy support.

### REQ-002: EngineType enum

The module shall define an `EngineType` string enum with three values:
- `EngineTypeGoNative` = `"go_native"`
- `EngineTypeUTLS` = `"utls"`
- `EngineTypeChromeImpersonate` = `"chrome_impersonate"`

### REQ-003: go_native engine

The `go_native` engine shall:
- Clone `http.DefaultTransport` as the base
- Enforce `tls.VersionTLS12` as minimum TLS version
- Support HTTP/HTTPS proxy via `transport.Proxy`
- Support SOCKS5 proxy via `transport.DialContext` with `golang.org/x/net/proxy`
- Set `MaxIdleConnsPerHost` to 10
- Return `*http.Transport` as `http.RoundTripper`

### REQ-004: utls engine

The `utls` engine shall:
- Create a `*http.Transport` with custom `DialTLSContext`
- Use `utls.UClient` to perform TLS handshake with a specified `ClientHelloSpec`
- Negotiate ALPN: if "h2" is negotiated, configure HTTP/2 transport via `golang.org/x/net/http2`
- Support HTTP/SOCKS5 proxy by dialing through proxy before applying uTLS
- Support both preset fingerprint profiles and user-supplied custom `ClientHelloSpec`

### REQ-005: Fingerprint profiles

The module shall define `FingerprintProfile` string enum with values:
- `FingerprintNodeJSV24` = `"nodejs_v24"` — custom ClientHelloSpec matching Node.js v24.6.0
- `FingerprintChromeAuto` = `"chrome_auto"` — `utls.HelloChrome_Auto`
- `FingerprintFirefoxAuto` = `"firefox_auto"` — `utls.HelloFirefox_Auto`
- `FingerprintRandomized` = `"randomized"` — `utls.HelloRandomized`
- `FingerprintCustom` = `"custom"` — uses `EngineConfig.CustomSpec`

### REQ-006: chrome_impersonate engine

The `chrome_impersonate` engine shall:
- Implement `http.RoundTripper` by wrapping `github.com/imroc/req/v3` client
- Use `req.C().ImpersonateChrome()` to configure Chrome TLS fingerprint
- Disable CookieJar (`SetCookieJar(nil)`) to ensure clean sessions
- Create a fresh `req.Client` per `RoundTrip` call
- Support HTTP/SOCKS5 proxy via `client.SetProxyURL()`
- Support configurable timeout via `client.SetTimeout()`

### REQ-007: EngineConfig

The module shall define an `EngineConfig` struct with fields:
- `Type EngineType` — selects engine
- `ProxyURL string` — proxy URL (HTTP/HTTPS/SOCKS5), empty for direct/environment
- `Timeout time.Duration` — request timeout, 0 for no timeout
- `Fingerprint FingerprintProfile` — uTLS only
- `CustomSpec *utls.ClientHelloSpec` — uTLS + Fingerprint=custom only

### REQ-008: Config integration

The existing `Config` struct shall gain an `Engine *EngineConfig` field. When `Engine` is nil, `New()` shall behave exactly as before (backward compatible).

### REQ-009: Engine factory

The module shall provide a factory function `NewEngine(config EngineConfig) (Engine, error)` that creates the appropriate engine based on `config.Type`.

### REQ-010: New() modification

`New(config Config)` shall, when `config.Engine` is non-nil:
1. Call `NewEngine(*config.Engine)` to create the engine
2. Call `engine.CreateTransport()` to get the RoundTripper
3. Set the RoundTripper on the `gclient.Client`'s embedded `http.Client.Transport`

### REQ-011: Proxy URL parsing

All engines shall support proxy URL formats:
- HTTP: `http://user:pass@host:port` or `http://host:port`
- HTTPS: `https://user:pass@host:port`
- SOCKS5: `socks5://user:pass@host:port` or `socks5://host:port`
- Empty string: use environment variables (`HTTP_PROXY`/`HTTPS_PROXY`/`ALL_PROXY`) for go_native; direct connection for utls/chrome

### REQ-012: Unit tests

Each engine shall have unit tests verifying:
- Transport creation succeeds without error
- Transport type matches expectation (`*http.Transport` for native/utls, custom type for chrome)
- Proxy configuration is applied (via inspecting transport fields or mock requests)
- uTLS engine uses correct fingerprint profile
