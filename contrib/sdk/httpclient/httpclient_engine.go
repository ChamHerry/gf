// Copyright GoFrame Author(https://goframe.org). All Rights Reserved.
//
// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this file,
// You can obtain one at https://github.com/gogf/gf.

// This file defines the Engine interface and related types for pluggable
// TLS transport engines, enabling TLS fingerprint simulation and proxy
// support in the SDK HTTP client.

package httpclient

import (
	"fmt"
	"net/http"
	"time"

	utls "github.com/refraction-networking/utls"
)

// Engine creates an http.RoundTripper with specific TLS fingerprinting
// and proxy configuration. Implementations include go_native, utls, and
// chrome_impersonate.
type Engine interface {
	// CreateTransport returns a configured http.RoundTripper that implements
	// the engine's TLS fingerprinting and proxy settings.
	CreateTransport() (http.RoundTripper, error)
}

// EngineType defines the available transport engine types.
type EngineType string

const (
	// EngineTypeGoNative uses the standard Go net/http transport.
	// This is the baseline engine with standard TLS and proxy support.
	EngineTypeGoNative EngineType = "go_native"

	// EngineTypeUTLS uses the refraction-networking/utls library to simulate
	// custom TLS fingerprints (JA3/JA4). Supports preset profiles and custom
	// ClientHelloSpec.
	EngineTypeUTLS EngineType = "utls"

	// EngineTypeChromeImpersonate uses imroc/req/v3 with ImpersonateChrome
	// to produce a full Chrome TLS fingerprint (JA3/JA4 + HTTP/2 settings).
	EngineTypeChromeImpersonate EngineType = "chrome_impersonate"
)

// FingerprintProfile defines preset TLS fingerprint profiles for the uTLS engine.
type FingerprintProfile string

const (
	// FingerprintNodeJSV24 simulates Node.js v24.6.0 TLS fingerprint.
	// JA3: 944d1e1858cd278718f8a46b65d3212f
	FingerprintNodeJSV24 FingerprintProfile = "nodejs_v24"

	// FingerprintChromeAuto uses utls.HelloChrome_Auto for the latest Chrome profile.
	FingerprintChromeAuto FingerprintProfile = "chrome_auto"

	// FingerprintFirefoxAuto uses utls.HelloFirefox_Auto for the latest Firefox profile.
	FingerprintFirefoxAuto FingerprintProfile = "firefox_auto"

	// FingerprintRandomized uses utls.HelloRandomized for a random fingerprint.
	FingerprintRandomized FingerprintProfile = "randomized"

	// FingerprintCustom uses a user-supplied ClientHelloSpec via EngineConfig.CustomSpec.
	FingerprintCustom FingerprintProfile = "custom"
)

// EngineConfig configures the transport engine for the SDK HTTP client.
type EngineConfig struct {
	// Type specifies which engine to use.
	// Required. Must be one of EngineTypeGoNative, EngineTypeUTLS, EngineTypeChromeImpersonate.
	Type EngineType

	// ProxyURL specifies the proxy server URL.
	// Supports HTTP, HTTPS, and SOCKS5 protocols.
	// Empty string means direct connection (go_native also reads environment variables).
	ProxyURL string

	// Timeout specifies the request timeout duration.
	// Zero means no timeout (suitable for streaming/SSE).
	Timeout time.Duration

	// Fingerprint specifies the TLS fingerprint profile for the uTLS engine.
	// Only used when Type is EngineTypeUTLS.
	// Defaults to FingerprintNodeJSV24 if empty.
	Fingerprint FingerprintProfile

	// CustomSpec provides a custom ClientHelloSpec for the uTLS engine.
	// Only used when Type is EngineTypeUTLS and Fingerprint is FingerprintCustom.
	CustomSpec *utls.ClientHelloSpec

	// InsecureSkipVerify controls whether TLS certificate verification is skipped.
	// When true, the engine accepts any TLS certificate (useful for self-signed certs).
	// Defaults to false (secure). Only applies to utls and go_native engines.
	InsecureSkipVerify bool
}

// NewEngine creates an Engine instance based on the given EngineConfig.
// It returns an error if the engine type is unknown or configuration is invalid.
func NewEngine(config EngineConfig) (Engine, error) {
	switch config.Type {
	case EngineTypeGoNative:
		return &nativeEngine{
			proxyURL: config.ProxyURL,
			timeout:  config.Timeout,
		}, nil

	case EngineTypeUTLS:
		fingerprint := config.Fingerprint
		if fingerprint == "" {
			fingerprint = FingerprintNodeJSV24
		}
		return &utlsEngine{
			proxyURL:           config.ProxyURL,
			timeout:            config.Timeout,
			fingerprint:        fingerprint,
			customSpec:         config.CustomSpec,
			insecureSkipVerify: config.InsecureSkipVerify,
		}, nil

	case EngineTypeChromeImpersonate:
		return &chromeEngine{
			proxyURL: config.ProxyURL,
			timeout:  config.Timeout,
		}, nil

	default:
		return nil, fmt.Errorf(`unsupported engine type: %s`, config.Type)
	}
}
