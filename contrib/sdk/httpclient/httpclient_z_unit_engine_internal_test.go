// Copyright GoFrame Author(https://goframe.org). All Rights Reserved.
//
// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this file,
// You can obtain one at https://github.com/gogf/gf.

package httpclient

import (
	"context"
	"net/http"
	"testing"

	utls "github.com/refraction-networking/utls"

	"github.com/gogf/gf/v2/test/gtest"
)

// TestNewEngine_GoNative tests that NewEngine creates a valid nativeEngine
// for EngineTypeGoNative and that CreateTransport returns an *http.Transport.
func TestNewEngine_GoNative(t *testing.T) {
	gtest.C(t, func(t *gtest.T) {
		engine, err := NewEngine(EngineConfig{
			Type: EngineTypeGoNative,
		})
		t.AssertNil(err)
		t.AssertNE(engine, nil)

		transport, err := engine.CreateTransport()
		t.AssertNil(err)
		t.AssertNE(transport, nil)

		_, ok := transport.(*http.Transport)
		t.Assert(ok, true)
	})
}

// TestNewEngine_UTLS_NodeJSV24 tests that NewEngine creates a valid utlsEngine
// with the Node.js v24 fingerprint profile.
func TestNewEngine_UTLS_NodeJSV24(t *testing.T) {
	gtest.C(t, func(t *gtest.T) {
		engine, err := NewEngine(EngineConfig{
			Type:        EngineTypeUTLS,
			Fingerprint: FingerprintNodeJSV24,
		})
		t.AssertNil(err)
		t.AssertNE(engine, nil)

		transport, err := engine.CreateTransport()
		t.AssertNil(err)
		t.AssertNE(transport, nil)

		httpTransport, ok := transport.(*http.Transport)
		t.Assert(ok, true)
		t.AssertNE(httpTransport.DialTLSContext, nil)
	})
}

// TestNewEngine_UTLS_DefaultFingerprint tests that an empty Fingerprint
// defaults to FingerprintNodeJSV24.
func TestNewEngine_UTLS_DefaultFingerprint(t *testing.T) {
	gtest.C(t, func(t *gtest.T) {
		engine, err := NewEngine(EngineConfig{
			Type: EngineTypeUTLS,
		})
		t.AssertNil(err)

		utlsEng, ok := engine.(*utlsEngine)
		t.Assert(ok, true)
		t.Assert(utlsEng.fingerprint, FingerprintNodeJSV24)
	})
}

// TestNewEngine_UTLS_ChromeAuto tests the Chrome Auto preset.
func TestNewEngine_UTLS_ChromeAuto(t *testing.T) {
	gtest.C(t, func(t *gtest.T) {
		engine, err := NewEngine(EngineConfig{
			Type:        EngineTypeUTLS,
			Fingerprint: FingerprintChromeAuto,
		})
		t.AssertNil(err)

		utlsEng, ok := engine.(*utlsEngine)
		t.Assert(ok, true)
		t.Assert(utlsEng.fingerprint, FingerprintChromeAuto)
		t.Assert(utlsEng.customSpec == nil, true)
	})
}

// TestNewEngine_UTLS_FirefoxAuto tests the Firefox Auto preset.
func TestNewEngine_UTLS_FirefoxAuto(t *testing.T) {
	gtest.C(t, func(t *gtest.T) {
		engine, err := NewEngine(EngineConfig{
			Type:        EngineTypeUTLS,
			Fingerprint: FingerprintFirefoxAuto,
		})
		t.AssertNil(err)

		utlsEng, ok := engine.(*utlsEngine)
		t.Assert(ok, true)
		t.Assert(utlsEng.fingerprint, FingerprintFirefoxAuto)
	})
}

// TestNewEngine_UTLS_Randomized tests the Randomized preset.
func TestNewEngine_UTLS_Randomized(t *testing.T) {
	gtest.C(t, func(t *gtest.T) {
		engine, err := NewEngine(EngineConfig{
			Type:        EngineTypeUTLS,
			Fingerprint: FingerprintRandomized,
		})
		t.AssertNil(err)

		utlsEng, ok := engine.(*utlsEngine)
		t.Assert(ok, true)
		t.Assert(utlsEng.fingerprint, FingerprintRandomized)
	})
}

// TestNewEngine_UTLS_Custom tests the custom fingerprint profile with a
// user-supplied ClientHelloSpec.
func TestNewEngine_UTLS_Custom(t *testing.T) {
	gtest.C(t, func(t *gtest.T) {
		customSpec := &utls.ClientHelloSpec{
			CipherSuites: []uint16{
				utls.TLS_AES_128_GCM_SHA256,
				utls.TLS_AES_256_GCM_SHA384,
			},
			CompressionMethods: []uint8{0},
			Extensions:         []utls.TLSExtension{},
			TLSVersMin:         utls.VersionTLS12,
			TLSVersMax:         utls.VersionTLS13,
		}

		engine, err := NewEngine(EngineConfig{
			Type:        EngineTypeUTLS,
			Fingerprint: FingerprintCustom,
			CustomSpec:  customSpec,
		})
		t.AssertNil(err)

		utlsEng, ok := engine.(*utlsEngine)
		t.Assert(ok, true)
		t.Assert(utlsEng.fingerprint, FingerprintCustom)
		t.AssertNE(utlsEng.customSpec, nil)
	})
}

// TestNewEngine_ChromeImpersonate tests that NewEngine creates a valid
// chromeEngine for EngineTypeChromeImpersonate.
func TestNewEngine_ChromeImpersonate(t *testing.T) {
	gtest.C(t, func(t *gtest.T) {
		engine, err := NewEngine(EngineConfig{
			Type: EngineTypeChromeImpersonate,
		})
		t.AssertNil(err)
		t.AssertNE(engine, nil)

		transport, err := engine.CreateTransport()
		t.AssertNil(err)
		t.AssertNE(transport, nil)
	})
}

// TestNewEngine_UnknownType tests that NewEngine returns an error for
// an unknown engine type.
func TestNewEngine_UnknownType(t *testing.T) {
	gtest.C(t, func(t *gtest.T) {
		engine, err := NewEngine(EngineConfig{
			Type: EngineType("unknown"),
		})
		t.AssertNE(err, nil)
		t.Assert(engine == nil, true)
	})
}

// TestNewEngine_GoNative_WithProxy tests that the native engine accepts
// an HTTP proxy URL without error.
func TestNewEngine_GoNative_WithProxy(t *testing.T) {
	gtest.C(t, func(t *gtest.T) {
		engine, err := NewEngine(EngineConfig{
			Type:     EngineTypeGoNative,
			ProxyURL: "http://127.0.0.1:8080",
		})
		t.AssertNil(err)

		transport, err := engine.CreateTransport()
		t.AssertNil(err)
		t.AssertNE(transport, nil)
	})
}

// TestNewEngine_GoNative_WithSOCKS5Proxy tests that the native engine
// accepts a SOCKS5 proxy URL without error.
func TestNewEngine_GoNative_WithSOCKS5Proxy(t *testing.T) {
	gtest.C(t, func(t *gtest.T) {
		engine, err := NewEngine(EngineConfig{
			Type:     EngineTypeGoNative,
			ProxyURL: "socks5://127.0.0.1:1080",
		})
		t.AssertNil(err)

		transport, err := engine.CreateTransport()
		t.AssertNil(err)
		t.AssertNE(transport, nil)
	})
}

// TestNewEngine_GoNative_InvalidProxyScheme tests that an unsupported proxy
// scheme returns an error.
func TestNewEngine_GoNative_InvalidProxyScheme(t *testing.T) {
	gtest.C(t, func(t *gtest.T) {
		engine, err := NewEngine(EngineConfig{
			Type:     EngineTypeGoNative,
			ProxyURL: "ftp://127.0.0.1:21",
		})
		t.AssertNil(err)

		_, err = engine.CreateTransport()
		t.AssertNE(err, nil)
	})
}

// TestResolveFingerprint_NodeJSV24 tests that NodeJSV24 returns HelloCustom
// with a non-nil spec.
func TestResolveFingerprint_NodeJSV24(t *testing.T) {
	gtest.C(t, func(t *gtest.T) {
		helloID, spec, err := resolveFingerprint(FingerprintNodeJSV24, nil)
		t.AssertNil(err)
		t.Assert(helloID, utls.HelloCustom)
		t.AssertNE(spec, nil)
	})
}

// TestResolveFingerprint_EmptyString tests that empty string defaults to NodeJSV24.
func TestResolveFingerprint_EmptyString(t *testing.T) {
	gtest.C(t, func(t *gtest.T) {
		helloID, spec, err := resolveFingerprint("", nil)
		t.AssertNil(err)
		t.Assert(helloID, utls.HelloCustom)
		t.AssertNE(spec, nil)
	})
}

// TestResolveFingerprint_ChromeAuto tests that ChromeAuto returns HelloChrome_Auto
// with a nil spec (utls manages it internally).
func TestResolveFingerprint_ChromeAuto(t *testing.T) {
	gtest.C(t, func(t *gtest.T) {
		helloID, spec, err := resolveFingerprint(FingerprintChromeAuto, nil)
		t.AssertNil(err)
		t.Assert(helloID, utls.HelloChrome_Auto)
		t.Assert(spec == nil, true)
	})
}

// TestResolveFingerprint_FirefoxAuto tests that FirefoxAuto returns HelloFirefox_Auto.
func TestResolveFingerprint_FirefoxAuto(t *testing.T) {
	gtest.C(t, func(t *gtest.T) {
		helloID, spec, err := resolveFingerprint(FingerprintFirefoxAuto, nil)
		t.AssertNil(err)
		t.Assert(helloID, utls.HelloFirefox_Auto)
		t.Assert(spec == nil, true)
	})
}

// TestResolveFingerprint_Randomized tests that Randomized returns HelloRandomized.
func TestResolveFingerprint_Randomized(t *testing.T) {
	gtest.C(t, func(t *gtest.T) {
		helloID, spec, err := resolveFingerprint(FingerprintRandomized, nil)
		t.AssertNil(err)
		t.Assert(helloID, utls.HelloRandomized)
		t.Assert(spec == nil, true)
	})
}

// TestResolveFingerprint_Custom_Valid tests that Custom with a non-nil spec
// returns HelloCustom with the same spec.
func TestResolveFingerprint_Custom_Valid(t *testing.T) {
	gtest.C(t, func(t *gtest.T) {
		customSpec := &utls.ClientHelloSpec{
			CipherSuites: []uint16{utls.TLS_AES_128_GCM_SHA256},
		}
		helloID, spec, err := resolveFingerprint(FingerprintCustom, customSpec)
		t.AssertNil(err)
		t.Assert(helloID, utls.HelloCustom)
		t.Assert(spec, customSpec)
	})
}

// TestResolveFingerprint_Custom_NilSpec tests that Custom with nil spec returns error.
func TestResolveFingerprint_Custom_NilSpec(t *testing.T) {
	gtest.C(t, func(t *gtest.T) {
		_, _, err := resolveFingerprint(FingerprintCustom, nil)
		t.AssertNE(err, nil)
	})
}

// TestResolveFingerprint_Unknown tests that an unrecognized profile returns error.
func TestResolveFingerprint_Unknown(t *testing.T) {
	gtest.C(t, func(t *gtest.T) {
		_, _, err := resolveFingerprint(FingerprintProfile("nonexistent"), nil)
		t.AssertNE(err, nil)
	})
}

// TestNewEngine_UTLS_CreateTransport_AllProfiles tests that CreateTransport
// succeeds for every supported fingerprint profile.
func TestNewEngine_UTLS_CreateTransport_AllProfiles(t *testing.T) {
	profiles := []FingerprintProfile{
		FingerprintNodeJSV24,
		FingerprintChromeAuto,
		FingerprintFirefoxAuto,
		FingerprintRandomized,
	}
	for _, profile := range profiles {
		gtest.C(t, func(t *gtest.T) {
			engine, err := NewEngine(EngineConfig{
				Type:        EngineTypeUTLS,
				Fingerprint: profile,
			})
			t.AssertNil(err)
			transport, err := engine.CreateTransport()
			t.AssertNil(err)
			t.AssertNE(transport, nil)
		})
	}
}

// TestNewEngine_UTLS_InvalidProxyScheme tests that the utls engine returns
// an error when CreateTransport tries to dial through an unsupported proxy scheme.
func TestNewEngine_UTLS_InvalidProxyScheme(t *testing.T) {
	gtest.C(t, func(t *gtest.T) {
		engine, err := NewEngine(EngineConfig{
			Type:     EngineTypeUTLS,
			ProxyURL: "ftp://127.0.0.1:21",
		})
		t.AssertNil(err)

		// CreateTransport should succeed (transport is lazily configured).
		transport, err := engine.CreateTransport()
		t.AssertNil(err)
		t.AssertNE(transport, nil)

		// The DialTLSContext will fail when called with the bad proxy scheme.
		httpTransport, ok := transport.(*http.Transport)
		t.Assert(ok, true)

		ctx := context.Background()
		_, dialErr := httpTransport.DialTLSContext(ctx, "tcp", "example.com:443")
		t.AssertNE(dialErr, nil)
	})
}

// TestNewEngine_GoNative_WithTimeout tests that a timeout is correctly applied.
func TestNewEngine_GoNative_WithTimeout(t *testing.T) {
	gtest.C(t, func(t *gtest.T) {
		engine, err := NewEngine(EngineConfig{
			Type:    EngineTypeGoNative,
			Timeout: 5_000_000_000,
		})
		t.AssertNil(err)

		nativeEng, ok := engine.(*nativeEngine)
		t.Assert(ok, true)
		t.Assert(nativeEng.timeout > 0, true)
	})
}

// TestNewEngine_UTLS_WithProxyAndTimeout tests utls engine configuration with proxy and timeout.
func TestNewEngine_UTLS_WithProxyAndTimeout(t *testing.T) {
	gtest.C(t, func(t *gtest.T) {
		engine, err := NewEngine(EngineConfig{
			Type:        EngineTypeUTLS,
			Fingerprint: FingerprintChromeAuto,
			ProxyURL:    "socks5://127.0.0.1:1080",
			Timeout:     10_000_000_000,
		})
		t.AssertNil(err)

		utlsEng, ok := engine.(*utlsEngine)
		t.Assert(ok, true)
		t.Assert(utlsEng.proxyURL, "socks5://127.0.0.1:1080")
		t.Assert(utlsEng.timeout > 0, true)
	})
}

// TestNewEngine_ChromeImpersonate_WithProxy tests chrome engine configuration with proxy.
func TestNewEngine_ChromeImpersonate_WithProxy(t *testing.T) {
	gtest.C(t, func(t *gtest.T) {
		engine, err := NewEngine(EngineConfig{
			Type:     EngineTypeChromeImpersonate,
			ProxyURL: "http://127.0.0.1:8080",
			Timeout:  5_000_000_000,
		})
		t.AssertNil(err)

		chromeEng, ok := engine.(*chromeEngine)
		t.Assert(ok, true)
		t.Assert(chromeEng.proxyURL, "http://127.0.0.1:8080")
		t.Assert(chromeEng.timeout > 0, true)
	})
}

// TestUTLSEngine_DialThroughHTTPConnect_Unreachable tests that the utls engine
// correctly attempts to connect through an HTTP proxy and fails when the proxy
// is unreachable, exercising the HTTP CONNECT proxy code path.
func TestUTLSEngine_DialthroughHTTPConnect_Unreachable(t *testing.T) {
	gtest.C(t, func(t *gtest.T) {
		engine, err := NewEngine(EngineConfig{
			Type:               EngineTypeUTLS,
			Fingerprint:        FingerprintChromeAuto,
			ProxyURL:           "http://127.0.0.1:1",
			InsecureSkipVerify: true,
		})
		t.AssertNil(err)

		transport, err := engine.CreateTransport()
		t.AssertNil(err)

		httpTransport, ok := transport.(*http.Transport)
		t.Assert(ok, true)

		ctx := context.Background()
		_, dialErr := httpTransport.DialTLSContext(ctx, "tcp", "example.com:443")
		t.AssertNE(dialErr, nil)
	})
}

// TestUTLSEngine_DialthroughSOCKS5_Unreachable tests that the utls engine
// correctly attempts to connect through a SOCKS5 proxy and fails when the
// proxy is unreachable, exercising the SOCKS5 proxy code path.
func TestUTLSEngine_DialthroughSOCKS5_Unreachable(t *testing.T) {
	gtest.C(t, func(t *gtest.T) {
		engine, err := NewEngine(EngineConfig{
			Type:               EngineTypeUTLS,
			Fingerprint:        FingerprintChromeAuto,
			ProxyURL:           "socks5://127.0.0.1:1",
			InsecureSkipVerify: true,
		})
		t.AssertNil(err)

		transport, err := engine.CreateTransport()
		t.AssertNil(err)

		httpTransport, ok := transport.(*http.Transport)
		t.Assert(ok, true)

		ctx := context.Background()
		_, dialErr := httpTransport.DialTLSContext(ctx, "tcp", "example.com:443")
		t.AssertNE(dialErr, nil)
	})
}

// TestUTLSEngine_DialTLS_InvalidAddr tests dialTLS with an address that
// cannot be split into host:port, exercising the fallback branch.
func TestUTLSEngine_DialTLS_InvalidAddr(t *testing.T) {
	gtest.C(t, func(t *gtest.T) {
		engine, err := NewEngine(EngineConfig{
			Type:               EngineTypeUTLS,
			Fingerprint:        FingerprintChromeAuto,
			InsecureSkipVerify: true,
		})
		t.AssertNil(err)

		transport, err := engine.CreateTransport()
		t.AssertNil(err)

		httpTransport, ok := transport.(*http.Transport)
		t.Assert(ok, true)

		ctx := context.Background()
		// "invalid" has no port — SplitHostPort will fail, exercising the fallback.
		_, dialErr := httpTransport.DialTLSContext(ctx, "tcp", "invalid")
		t.AssertNE(dialErr, nil)
	})
}
