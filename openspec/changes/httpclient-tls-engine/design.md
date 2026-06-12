# Design: httpclient-tls-engine

## Architecture

```
┌──────────────────────────────────────────────────────────────┐
│                    httpclient.Client                          │
│                                                              │
│  ┌────────────────────────────────────────────────────────┐  │
│  │  Struct serialization layer (UNCHANGED)                │  │
│  │  Request() / Get() / HandleResponse()                  │  │
│  └───────────────────┬────────────────────────────────────┘  │
│                      │ delegates to                          │
│  ┌───────────────────▼────────────────────────────────────┐  │
│  │  gclient.Client (embeds http.Client)                   │  │
│  │  .Transport  ← Engine.CreateTransport() replaces this  │  │
│  └────────────────────────────────────────────────────────┘  │
└──────────────────────────────────────────────────────────────┘

                    Engine.CreateTransport()
                           │
           ┌───────────────┼───────────────┐
           ▼               ▼               ▼
    ┌────────────┐  ┌────────────┐  ┌──────────────┐
    │ go_native  │  │   utls     │  │chrome_impersonate│
    │            │  │            │  │              │
    │ clone      │  │ Transport  │  │ req.C()      │
    │ Default    │  │ + custom   │  │ .Impersonate │
    │ Transport  │  │ DialTLSCtx │  │ Chrome()     │
    │ + proxy    │  │ + utls.    │  │ → extract    │
    │            │  │   UClient  │  │   Transport  │
    └────────────┘  └────────────┘  └──────────────┘
```

## Integration point

`gclient.Client` embeds `http.Client` (not wraps). The `Transport` field is directly accessible:

```go
type Client struct {
    http.Client  // embedded — Transport field is writable
    ...
}
```

When `Config.Engine` is non-nil, `New()` calls `engine.CreateTransport()` and sets the result on `client.Transport`. Everything downstream (request building, JSON marshaling, response handling) is unchanged.

## Engine interface

```go
// Engine creates an http.RoundTripper with specific TLS fingerprinting
// and proxy configuration.
type Engine interface {
    CreateTransport() (http.RoundTripper, error)
}
```

`http.RoundTripper` is a single-method interface. All three engines return a `*http.Transport` (which implements RoundTripper), configured differently.

## Engine implementations

### 1. go_native (`httpclient_engine_native.go`)

Clones `http.DefaultTransport`, configures TLS 1.2+ minimum, and sets up proxy support (HTTP/HTTPS/SOCKS5). This is the baseline engine — equivalent to what AIProxyV3 calls "Go原生客户端".

Key details:
- HTTP proxy: `transport.Proxy = http.ProxyURL(parsed)`
- SOCKS5 proxy: `transport.DialContext` via `golang.org/x/net/proxy.SOCKS5`
- TLS config: `InsecureSkipVerify: false, MinVersion: tls.VersionTLS12`
- Connection pool: `MaxIdleConnsPerHost: 10`

### 2. utls (`httpclient_engine_utls.go`)

Creates a `*http.Transport` with a custom `DialTLSContext` that replaces the standard TLS handshake with `utls.UClient`. The UClient uses a `ClientHelloSpec` (either preset or user-supplied) to produce the desired JA3/JA4 fingerprint.

Key details:
- `DialTLSContext` performs: TCP connect → wrap with `utls.UClient` → `HandshakeContext()` → return connection
- ALPN negotiation: after handshake, check `uconn.ConnectionState().NegotiatedProtocol` to determine HTTP/1.1 vs HTTP/2
- HTTP/2: if ALPN negotiated "h2", wrap the connection with `golang.org/x/net/http2` transport
- Proxy support: dial through proxy first, then apply uTLS wrapping

Fingerprint profiles:
- `nodejs_v24`: custom ClientHelloSpec matching Node.js v24.6.0 (JA3: 944d1e1858cd278718f8a46b65d3212f)
- `chrome_auto`: `utls.HelloChrome_Auto` (utls built-in, auto-updates)
- `firefox_auto`: `utls.HelloFirefox_Auto` (utls built-in)
- `randomized`: `utls.HelloRandomized` (utls built-in)
- `custom`: user provides their own `*utls.ClientHelloSpec` via `EngineConfig.CustomSpec`

### 3. chrome_impersonate (`httpclient_engine_chrome.go`)

Uses `github.com/imroc/req/v3` with `ImpersonateChrome()` to produce a transport that mimics the full Chrome TLS fingerprint (JA3/JA4 + HTTP/2 settings + header order).

Key details:
- Create `req.C().ImpersonateChrome().SetCookieJar(nil)`
- Extract `req.Client.GetClient().Transport` as the RoundTripper
- Per-request: each `RoundTrip` creates a fresh `req.Client` to ensure clean session (no cookie leakage)
- Proxy: `client.SetProxyURL(proxyURL)`

**Design decision**: Chrome Impersonate wraps the entire request in a `req.Client` adapter that implements `http.RoundTripper`, rather than extracting a raw transport. This ensures all impersonation magic (header ordering, HTTP/2 frame settings) is preserved.

```go
type chromeTransport struct {
    proxyURL string
    timeout  time.Duration
}

func (t *chromeTransport) RoundTrip(req *http.Request) (*http.Response, error) {
    client := req.C().
        ImpersonateChrome().
        SetCookieJar(nil)
    if t.timeout > 0 {
        client.SetTimeout(t.timeout)
    }
    if t.proxyURL != "" {
        client.SetProxyURL(t.proxyURL)
    }
    // Convert http.Request → req request → http.Response
    resp, err := client.R().DoRaw(req.Method, req.URL.String(), req.Body)
    ...
}
```

## Config changes

```go
type Config struct {
    URL     string          // existing
    Client  *gclient.Client // existing
    Handler Handler         // existing
    Logger  *glog.Logger    // existing
    RawDump bool            // existing
    
    // NEW: nil = default gclient behavior (backward compatible)
    Engine  *EngineConfig
}

type EngineConfig struct {
    Type        EngineType              // go_native / utls / chrome_impersonate
    ProxyURL    string                  // HTTP/SOCKS5 proxy URL, empty = direct/env
    Timeout     time.Duration           // 0 = no timeout (for streaming)
    Fingerprint FingerprintProfile      // utls only: nodejs_v24 / chrome_auto / firefox_auto / randomized / custom
    CustomSpec  *utls.ClientHelloSpec   // utls + Fingerprint=custom only
}
```

## New dependencies (go.mod for contrib/sdk/httpclient)

| Dependency | Used by | Purpose |
|---|---|---|
| `github.com/refraction-networking/utls` | utls engine | TLS fingerprint simulation |
| `github.com/imroc/req/v3` | chrome engine | Chrome impersonation |
| `golang.org/x/net/proxy` | all engines (SOCKS5) | Already in root module |

## File plan

| File | Action | Lines (est.) |
|---|---|---|
| `httpclient_engine.go` | NEW | ~80 |
| `httpclient_engine_native.go` | NEW | ~120 |
| `httpclient_engine_utls.go` | NEW | ~200 |
| `httpclient_engine_chrome.go` | NEW | ~100 |
| `httpclient_fingerprint.go` | NEW | ~250 (mostly nodejs_v24 ClientHelloSpec data) |
| `httpclient_config.go` | MODIFY | +15 |
| `httpclient.go` | MODIFY | +10 |
| `httpclient_z_unit_engine_test.go` | NEW | ~200 |
| **Total** | | ~975 new lines |

## Risks

1. **uTLS + HTTP/2**: ALPN negotiation must correctly detect "h2" and wrap with http2 transport. If missed, HTTP/2 requests fail silently on HTTP/1.1.
2. **Chrome Impersonate adapter overhead**: Creating a new `req.Client` per request adds ~1ms overhead. Acceptable for API calls, not for high-throughput scenarios.
3. **Fingerprint staleness**: The nodejs_v24 fingerprint is a point-in-time snapshot. It will need updates as Node.js releases new versions. Mitigated by supporting custom ClientHelloSpec.
4. **req/v3 transport extraction**: If `req.Client` changes its internal API in future versions, the chrome engine may break. Pinned via go.mod.
