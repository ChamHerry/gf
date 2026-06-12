# Proposal: httpclient-tls-engine

## Summary

Add pluggable TLS transport engines to `contrib/sdk/httpclient`, enabling the SDK client to impersonate browser/Node.js TLS fingerprints (JA3/JA4) when calling external APIs. Three built-in engines: Go native, uTLS (custom ClientHelloSpec), and Chrome Impersonate (via imroc/req/v3).

## Motivation

The current `contrib/sdk/httpclient` wraps `gclient` with standard Go TLS. When calling APIs protected by Cloudflare or similar bot-detection services, standard Go TLS fingerprints are easily detected and blocked. Projects like AIProxyV3 solve this with a factory + adapter pattern supporting multiple TLS fingerprint backends, but that code lives in application code, not in the framework.

Bringing this capability into `contrib/sdk/httpclient` means:
- GoFrame SDK consumers get anti-detection HTTP transport out of the box
- The existing struct-to-JSON serialization layer stays unchanged
- Engine selection is a one-line config change

## Scope

### In scope
- `Engine` interface and `EngineType` enum in `contrib/sdk/httpclient`
- Three engine implementations: `go_native`, `utls`, `chrome_impersonate`
- `EngineConfig` added to existing `Config` struct (backward compatible — nil Engine = current behavior)
- Preset fingerprint profiles for uTLS (nodejs_v24, chrome_auto, firefox_auto, randomized)
- Custom `utls.ClientHelloSpec` support for advanced users
- HTTP/SOCKS5 proxy support in all engines
- Unit tests for each engine

### Out of scope
- Node.js subprocess-based fingerprinting (the uTLS Go-native approach covers this)
- Fingerprint auto-rotation / refresh logic
- Response decompression or content-encoding changes
- Changes to `gclient` core or `net/ghttp`

## Non-breaking change

`Engine` field in `Config` defaults to `nil`. When nil, `New()` behaves exactly as before — `gclient.New()` with standard transport. Existing users are unaffected.
