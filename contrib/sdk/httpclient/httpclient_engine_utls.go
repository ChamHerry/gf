// Copyright GoFrame Author(https://goframe.org). All Rights Reserved.
//
// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this file,
// You can obtain one at https://github.com/gogf/gf.

// This file implements the utls engine, which uses refraction-networking/utls
// to simulate custom TLS fingerprints (JA3/JA4) during the TLS handshake.

package httpclient

import (
	"bufio"
	"context"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	utls "github.com/refraction-networking/utls"
	"golang.org/x/net/http2"
	"golang.org/x/net/proxy"
)

// utlsEngine implements the Engine interface using uTLS for TLS fingerprint
// simulation. It creates an *http.Transport with a custom DialTLSContext
// that replaces the standard TLS handshake with utls.UClient.
type utlsEngine struct {
	// proxyURL specifies the proxy server (HTTP/HTTPS/SOCKS5). Empty for direct.
	proxyURL string

	// timeout specifies the request timeout. Zero means no timeout.
	timeout time.Duration

	// fingerprint specifies the TLS fingerprint profile to use.
	fingerprint FingerprintProfile

	// customSpec provides a custom ClientHelloSpec when fingerprint is "custom".
	customSpec *utls.ClientHelloSpec

	// insecureSkipVerify controls whether TLS certificate verification is skipped.
	insecureSkipVerify bool
}

// CreateTransport returns an *http.Transport with a custom DialTLSContext
// that uses uTLS to simulate the configured TLS fingerprint.
func (e *utlsEngine) CreateTransport() (http.RoundTripper, error) {
	// Resolve the fingerprint to a ClientHelloID and optional ClientHelloSpec.
	helloID, helloSpec, err := resolveFingerprint(e.fingerprint, e.customSpec)
	if err != nil {
		return nil, fmt.Errorf(`failed to resolve fingerprint profile %q: %w`, e.fingerprint, err)
	}

	dialer := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}

	transport := &http.Transport{
		DialTLSContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return e.dialTLS(ctx, network, addr, dialer, helloID, helloSpec)
		},
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   10,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		ForceAttemptHTTP2:     true,
		DisableCompression:    false,
	}

	// Enable HTTP/2 support (auto-negotiated via ALPN during TLS handshake).
	if err := http2.ConfigureTransport(transport); err != nil {
		return nil, fmt.Errorf(`failed to configure HTTP/2 for utls transport: %w`, err)
	}

	if e.timeout > 0 {
		transport.ResponseHeaderTimeout = e.timeout
	}

	return transport, nil
}

// dialTLS establishes a TLS connection using uTLS with the configured fingerprint.
// It performs: TCP dial (direct or via proxy) → utls.UClient → TLS handshake.
func (e *utlsEngine) dialTLS(
	ctx context.Context,
	network, addr string,
	dialer *net.Dialer,
	helloID utls.ClientHelloID,
	helloSpec *utls.ClientHelloSpec,
) (net.Conn, error) {
	// Extract server name (hostname) from the address for SNI.
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		host = addr
	}

	// Step 1: Establish TCP connection (directly or through proxy).
	var tcpConn net.Conn
	if e.proxyURL != "" {
		tcpConn, err = e.dialThroughProxy(ctx, network, addr, dialer)
	} else {
		tcpConn, err = dialer.DialContext(ctx, network, addr)
	}
	if err != nil {
		return nil, fmt.Errorf(`TCP dial failed for %q: %w`, addr, err)
	}

	// Step 2: Create uTLS config with SNI.
	tlsConfig := &utls.Config{
		ServerName:         host,
		InsecureSkipVerify: e.insecureSkipVerify,
		MinVersion:         utls.VersionTLS12,
	}

	// Step 3: Create uTLS client connection.
	uConn := utls.UClient(tcpConn, tlsConfig, helloID)

	// Step 4: Apply custom ClientHelloSpec if provided (for preset profiles like nodejs_v24).
	if helloSpec != nil {
		if applyErr := uConn.ApplyPreset(helloSpec); applyErr != nil {
			closeErr := tcpConn.Close()
			if closeErr != nil {
				return nil, fmt.Errorf(`apply TLS preset failed: %w (close error: %v)`, applyErr, closeErr)
			}
			return nil, fmt.Errorf(`apply TLS preset failed: %w`, applyErr)
		}
	}

	// Step 5: Perform TLS handshake.
	if err := uConn.HandshakeContext(ctx); err != nil {
		closeErr := uConn.Close()
		if closeErr != nil {
			return nil, fmt.Errorf(`TLS handshake failed: %w (close error: %v)`, err, closeErr)
		}
		return nil, fmt.Errorf(`TLS handshake failed: %w`, err)
	}

	return uConn, nil
}

// dialThroughProxy establishes a TCP connection through an HTTP CONNECT or SOCKS5 proxy.
func (e *utlsEngine) dialThroughProxy(
	ctx context.Context,
	network, addr string,
	dialer *net.Dialer,
) (net.Conn, error) {
	parsedURL, err := url.Parse(e.proxyURL)
	if err != nil {
		return nil, fmt.Errorf(`failed to parse proxy URL %q: %w`, e.proxyURL, err)
	}

	switch parsedURL.Scheme {
	case "socks5", "socks5h":
		return e.dialThroughSOCKS5(ctx, network, addr, parsedURL, dialer)

	case "http", "https", "":
		return e.dialThroughHTTPConnect(ctx, network, addr, parsedURL, dialer)

	default:
		return nil, fmt.Errorf(`unsupported proxy scheme %q for utls engine`, parsedURL.Scheme)
	}
}

// dialThroughSOCKS5 establishes a connection through a SOCKS5 proxy.
func (e *utlsEngine) dialThroughSOCKS5(
	ctx context.Context,
	network, addr string,
	parsedURL *url.URL,
	dialer *net.Dialer,
) (net.Conn, error) {
	var auth *proxy.Auth
	if parsedURL.User != nil {
		password, _ := parsedURL.User.Password()
		auth = &proxy.Auth{
			User:     parsedURL.User.Username(),
			Password: password,
		}
	}

	proxyAddr := parsedURL.Host
	if parsedURL.Port() == "" {
		proxyAddr = net.JoinHostPort(parsedURL.Hostname(), "1080")
	}

	socks5Dialer, err := proxy.SOCKS5(network, proxyAddr, auth, dialer)
	if err != nil {
		return nil, fmt.Errorf(`failed to create SOCKS5 dialer: %w`, err)
	}

	// proxy.Dialer.Dial does not accept context; wrap with timeout handling.
	type dialResult struct {
		conn net.Conn
		err  error
	}
	resultCh := make(chan dialResult, 1)

	go func() {
		conn, dialErr := socks5Dialer.Dial(network, addr)
		resultCh <- dialResult{conn: conn, err: dialErr}
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case result := <-resultCh:
		if result.err != nil {
			return nil, fmt.Errorf(`SOCKS5 dial failed: %w`, result.err)
		}
		return result.conn, nil
	}
}

// dialThroughHTTPConnect establishes a connection through an HTTP CONNECT proxy tunnel.
func (e *utlsEngine) dialThroughHTTPConnect(
	ctx context.Context,
	network, addr string,
	parsedURL *url.URL,
	dialer *net.Dialer,
) (net.Conn, error) {
	// Connect to the proxy server.
	proxyConn, err := dialer.DialContext(ctx, network, parsedURL.Host)
	if err != nil {
		return nil, fmt.Errorf(`connect to proxy %q failed: %w`, parsedURL.Host, err)
	}

	// Build the CONNECT request.
	connectReq := fmt.Sprintf("CONNECT %s HTTP/1.1\r\nHost: %s\r\n", addr, addr)

	// Add proxy authentication if credentials are present.
	if parsedURL.User != nil {
		username := parsedURL.User.Username()
		password, hasPassword := parsedURL.User.Password()
		if hasPassword {
			credentials := base64.StdEncoding.EncodeToString([]byte(username + ":" + password))
			connectReq += "Proxy-Authorization: Basic " + credentials + "\r\n"
		}
	}
	connectReq += "\r\n"

	// Send CONNECT request.
	if _, err := proxyConn.Write([]byte(connectReq)); err != nil {
		closeErr := proxyConn.Close()
		if closeErr != nil {
			return nil, fmt.Errorf(`send CONNECT failed: %w (close error: %v)`, err, closeErr)
		}
		return nil, fmt.Errorf(`send CONNECT failed: %w`, err)
	}

	// Read CONNECT response.
	resp, err := http.ReadResponse(bufio.NewReader(proxyConn), &http.Request{Method: http.MethodConnect})
	if err != nil {
		closeErr := proxyConn.Close()
		if closeErr != nil {
			return nil, fmt.Errorf(`read CONNECT response failed: %w (close error: %v)`, err, closeErr)
		}
		return nil, fmt.Errorf(`read CONNECT response failed: %w`, err)
	}

	if resp.StatusCode != http.StatusOK {
		closeErr := proxyConn.Close()
		if closeErr != nil {
			return nil, fmt.Errorf(`proxy CONNECT failed with status %d (close error: %v)`, resp.StatusCode, closeErr)
		}
		return nil, fmt.Errorf(`proxy CONNECT failed with status %d`, resp.StatusCode)
	}

	return proxyConn, nil
}
