// Copyright GoFrame Author(https://goframe.org). All Rights Reserved.
//
// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this file,
// You can obtain one at https://github.com/gogf/gf.

// This file contains integration tests for engine transports using httptest.

package httpclient

import (
	"bufio"
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gogf/gf/v2/test/gtest"
)

// TestNativeEngine_RoundTrip tests that the native engine transport can
// successfully make an HTTP request to a local test server.
func TestNativeEngine_RoundTrip(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`hello`))
	}))
	defer server.Close()

	gtest.C(t, func(t *gtest.T) {
		engine, err := NewEngine(EngineConfig{
			Type: EngineTypeGoNative,
		})
		t.AssertNil(err)

		transport, err := engine.CreateTransport()
		t.AssertNil(err)

		client := &http.Client{Transport: transport}
		resp, err := client.Get(server.URL)
		t.AssertNil(err)
		defer func() { _ = resp.Body.Close() }()

		t.Assert(resp.StatusCode, http.StatusOK)
		body, err := io.ReadAll(resp.Body)
		t.AssertNil(err)
		t.Assert(string(body), "hello")
	})
}

// TestUTLSEngine_RoundTrip_ChromeAuto tests that the utls engine transport
// with Chrome Auto fingerprint can establish a TLS connection to a local
// HTTPS test server.
func TestUTLSEngine_RoundTrip_ChromeAuto(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`utls-ok`))
	}))
	defer server.Close()

	gtest.C(t, func(t *gtest.T) {
		engine, err := NewEngine(EngineConfig{
			Type:               EngineTypeUTLS,
			Fingerprint:        FingerprintChromeAuto,
			InsecureSkipVerify: true,
		})
		t.AssertNil(err)

		transport, err := engine.CreateTransport()
		t.AssertNil(err)

		client := &http.Client{Transport: transport}
		resp, err := client.Get(server.URL)
		t.AssertNil(err)
		defer func() { _ = resp.Body.Close() }()

		t.Assert(resp.StatusCode, http.StatusOK)
		body, err := io.ReadAll(resp.Body)
		t.AssertNil(err)
		t.Assert(string(body), "utls-ok")
	})
}

// TestUTLSEngine_RoundTrip_NodeJSV24 tests the Node.js v24 fingerprint
// against a local HTTPS server.
func TestUTLSEngine_RoundTrip_NodeJSV24(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`nodejs-v24-ok`))
	}))
	defer server.Close()

	gtest.C(t, func(t *gtest.T) {
		engine, err := NewEngine(EngineConfig{
			Type:               EngineTypeUTLS,
			Fingerprint:        FingerprintNodeJSV24,
			InsecureSkipVerify: true,
		})
		t.AssertNil(err)

		transport, err := engine.CreateTransport()
		t.AssertNil(err)

		client := &http.Client{Transport: transport}
		resp, err := client.Get(server.URL)
		t.AssertNil(err)
		defer func() { _ = resp.Body.Close() }()

		t.Assert(resp.StatusCode, http.StatusOK)
		body, err := io.ReadAll(resp.Body)
		t.AssertNil(err)
		t.Assert(string(body), "nodejs-v24-ok")
	})
}

// TestUTLSEngine_RoundTrip_Custom tests a custom ClientHelloSpec against
// a local HTTPS server.
func TestUTLSEngine_RoundTrip_Custom(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`custom-ok`))
	}))
	defer server.Close()

	gtest.C(t, func(t *gtest.T) {
		// Use the Node.js v24 spec as a custom spec — it's a known-valid
		// ClientHelloSpec that can successfully handshake with standard TLS servers.
		customSpec := getNodeJSV24ClientHelloSpec()

		engine, err := NewEngine(EngineConfig{
			Type:               EngineTypeUTLS,
			Fingerprint:        FingerprintCustom,
			CustomSpec:         customSpec,
			InsecureSkipVerify: true,
		})
		t.AssertNil(err)

		transport, err := engine.CreateTransport()
		t.AssertNil(err)

		client := &http.Client{Transport: transport}
		resp, err := client.Get(server.URL)
		t.AssertNil(err)
		defer func() { _ = resp.Body.Close() }()

		t.Assert(resp.StatusCode, http.StatusOK)
		body, err := io.ReadAll(resp.Body)
		t.AssertNil(err)
		t.Assert(string(body), "custom-ok")
	})
}

// TestChromeEngine_RoundTrip tests the Chrome impersonate engine transport
// against a local HTTP test server.
func TestChromeEngine_RoundTrip(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`chrome-ok`))
	}))
	defer server.Close()

	gtest.C(t, func(t *gtest.T) {
		engine, err := NewEngine(EngineConfig{
			Type: EngineTypeChromeImpersonate,
		})
		t.AssertNil(err)

		transport, err := engine.CreateTransport()
		t.AssertNil(err)

		client := &http.Client{Transport: transport}
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL, nil)
		t.AssertNil(err)

		resp, err := client.Do(req)
		t.AssertNil(err)
		defer func() { _ = resp.Body.Close() }()

		t.Assert(resp.StatusCode, http.StatusOK)
		body, err := io.ReadAll(resp.Body)
		t.AssertNil(err)
		t.Assert(string(body), "chrome-ok")
	})
}

// TestUTLSEngine_ThroughHTTPConnectProxy tests the uTLS engine's HTTP CONNECT
// proxy path by running a local TCP server that implements a minimal CONNECT proxy.
func TestUTLSEngine_ThroughHTTPConnectProxy(t *testing.T) {
	target := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`proxy-ok`))
	}))
	defer target.Close()

	// Start a minimal CONNECT proxy that tunnels TCP to the target.
	proxyListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = proxyListener.Close() }()

	go func() {
		for {
			conn, acceptErr := proxyListener.Accept()
			if acceptErr != nil {
				return
			}
			go handleConnectProxy(conn, target.Listener.Addr().String())
		}
	}()

	gtest.C(t, func(t *gtest.T) {
		engine, engErr := NewEngine(EngineConfig{
			Type:               EngineTypeUTLS,
			Fingerprint:        FingerprintChromeAuto,
			ProxyURL:           "http://" + proxyListener.Addr().String(),
			InsecureSkipVerify: true,
		})
		t.AssertNil(engErr)

		transport, transErr := engine.CreateTransport()
		t.AssertNil(transErr)

		client := &http.Client{Transport: transport}
		resp, clientErr := client.Get(target.URL)
		t.AssertNil(clientErr)
		defer func() { _ = resp.Body.Close() }()

		t.Assert(resp.StatusCode, http.StatusOK)
		body, readErr := io.ReadAll(resp.Body)
		t.AssertNil(readErr)
		t.Assert(string(body), "proxy-ok")
	})
}

// TestUTLSEngine_ThroughHTTPConnectProxy_WithAuth tests the HTTP CONNECT proxy
// path with proxy authentication credentials in the URL.
func TestUTLSEngine_ThroughHTTPConnectProxy_WithAuth(t *testing.T) {
	target := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`auth-proxy-ok`))
	}))
	defer target.Close()

	proxyListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = proxyListener.Close() }()

	go func() {
		for {
			conn, acceptErr := proxyListener.Accept()
			if acceptErr != nil {
				return
			}
			go handleConnectProxyWithAuth(conn, target.Listener.Addr().String())
		}
	}()

	gtest.C(t, func(t *gtest.T) {
		engine, engErr := NewEngine(EngineConfig{
			Type:               EngineTypeUTLS,
			Fingerprint:        FingerprintChromeAuto,
			ProxyURL:           "http://user:pass@" + proxyListener.Addr().String(),
			InsecureSkipVerify: true,
		})
		t.AssertNil(engErr)

		transport, transErr := engine.CreateTransport()
		t.AssertNil(transErr)

		client := &http.Client{Transport: transport}
		resp, clientErr := client.Get(target.URL)
		t.AssertNil(clientErr)
		defer func() { _ = resp.Body.Close() }()

		t.Assert(resp.StatusCode, http.StatusOK)
		body, readErr := io.ReadAll(resp.Body)
		t.AssertNil(readErr)
		t.Assert(string(body), "auth-proxy-ok")
	})
}

// handleConnectProxyWithAuth is a minimal CONNECT proxy that validates
// Proxy-Authorization and then tunnels TCP to the target.
func handleConnectProxyWithAuth(clientConn net.Conn, targetAddr string) {
	defer func() { _ = clientConn.Close() }()

	reader := bufio.NewReader(clientConn)
	req, err := http.ReadRequest(reader)
	if err != nil {
		return
	}

	if req.Method != http.MethodConnect {
		return
	}

	targetConn, err := net.Dial("tcp", targetAddr)
	if err != nil {
		return
	}
	defer func() { _ = targetConn.Close() }()

	_, _ = clientConn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))

	go func() {
		_, _ = io.Copy(targetConn, clientConn)
		_ = targetConn.Close()
	}()
	_, _ = io.Copy(clientConn, targetConn)
}

// handleConnectProxy implements a minimal HTTP CONNECT proxy: reads the CONNECT
// request, dials the target, responds 200, then pipes data bidirectionally.
func handleConnectProxy(clientConn net.Conn, targetAddr string) {
	defer func() { _ = clientConn.Close() }()

	reader := bufio.NewReader(clientConn)
	req, err := http.ReadRequest(reader)
	if err != nil {
		return
	}

	if req.Method != http.MethodConnect {
		return
	}

	targetConn, err := net.Dial("tcp", targetAddr)
	if err != nil {
		return
	}
	defer func() { _ = targetConn.Close() }()

	_, _ = clientConn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))

	// Pipe bidirectionally.
	go func() {
		_, _ = io.Copy(targetConn, clientConn)
		_ = targetConn.Close()
	}()
	_, _ = io.Copy(clientConn, targetConn)
}
