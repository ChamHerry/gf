# httpclient

Package `httpclient` provides an HTTP client SDK with pluggable TLS transport engines, enabling TLS fingerprint simulation, custom proxy support, and Chrome impersonation.

# Installation

```
go get -u github.com/gogf/gf/contrib/sdk/httpclient/v2
```

# Features

- **Three transport engines**: `go_native`, `utls`, `chrome_impersonate`
- **TLS fingerprint simulation**: Node.js v24, Chrome, Firefox, Randomized, or custom `ClientHelloSpec`
- **Proxy support**: HTTP CONNECT, HTTPS, and SOCKS5 proxies
- **Backward compatible**: when no engine is configured, the default `gclient` transport is used

# Usage

## Basic usage (no engine)

When no `Engine` is specified, the client behaves exactly as before.

```go
package main

import (
    "context"

    "github.com/gogf/gf/v2/frame/g"

    httpclient "github.com/gogf/gf/contrib/sdk/httpclient/v2"
)

func main() {
    client := httpclient.New(httpclient.Config{
        URL: "http://example.com",
    })

    type Req struct {
        g.Meta `path:"/api/get" method:"get"`
    }
    type Res struct {
        Message string `json:"message"`
    }

    var (
        ctx = context.Background()
        req = &Req{}
        res = &Res{}
    )
    if err := client.Request(ctx, req, res); err != nil {
        panic(err)
    }
    g.Dump(res)
}
```

## Using the uTLS engine (Node.js v24 fingerprint)

Simulate Node.js v24.6.0 TLS fingerprint to bypass JA3/JA4 fingerprinting.

```go
client := httpclient.New(httpclient.Config{
    URL: "https://api.example.com",
    Engine: &httpclient.EngineConfig{
        Type:        httpclient.EngineTypeUTLS,
        Fingerprint: httpclient.FingerprintNodeJSV24,
    },
})
```

## Using the uTLS engine with a SOCKS5 proxy

```go
client := httpclient.New(httpclient.Config{
    URL: "https://api.example.com",
    Engine: &httpclient.EngineConfig{
        Type:        httpclient.EngineTypeUTLS,
        Fingerprint: httpclient.FingerprintChromeAuto,
        ProxyURL:    "socks5://user:pass@127.0.0.1:1080",
        Timeout:     30 * time.Second,
    },
})
```

## Using the Chrome impersonate engine

Full Chrome TLS fingerprint including HTTP/2 settings via `imroc/req/v3`.

```go
client := httpclient.New(httpclient.Config{
    URL: "https://api.example.com",
    Engine: &httpclient.EngineConfig{
        Type: httpclient.EngineTypeChromeImpersonate,
    },
})
```

## Using a custom ClientHelloSpec

```go
client := httpclient.New(httpclient.Config{
    URL: "https://api.example.com",
    Engine: &httpclient.EngineConfig{
        Type:        httpclient.EngineTypeUTLS,
        Fingerprint: httpclient.FingerprintCustom,
        CustomSpec:  myCustomSpec, // *utls.ClientHelloSpec
    },
})
```

# Engine types

| Engine | Description | Use Case |
|---|---|---|
| `go_native` | Standard Go `net/http` transport with proxy support | Default behavior, no fingerprint simulation |
| `utls` | `refraction-networking/utls` for custom TLS fingerprints | Bypass JA3/JA4 fingerprinting with preset or custom profiles |
| `chrome_impersonate` | `imroc/req/v3` with `ImpersonateChrome` | Full Chrome simulation (TLS + HTTP/2 settings) |

# Fingerprint profiles

| Profile | Description | JA3 Hash |
|---|---|---|
| `nodejs_v24` | Node.js v24.6.0 TLS fingerprint | `944d1e1858cd278718f8a46b65d3212f` |
| `chrome_auto` | Latest Chrome profile (managed by uTLS) | N/A |
| `firefox_auto` | Latest Firefox profile (managed by uTLS) | N/A |
| `randomized` | Random TLS fingerprint (managed by uTLS) | N/A |
| `custom` | User-supplied `ClientHelloSpec` | User-defined |

## License

`GoFrame httpclient` is licensed under the [MIT License](../../../LICENSE), 100% free and open-source, forever.
