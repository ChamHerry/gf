// Copyright GoFrame Author(https://goframe.org). All Rights Reserved.
//
// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this file,
// You can obtain one at https://github.com/gogf/gf.

// This file implements the go_native engine, which uses the standard Go
// net/http transport with configurable HTTP/SOCKS5 proxy support.

package httpclient

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"golang.org/x/net/proxy"
)

// nativeEngine implements the Engine interface using standard Go net/http.
// It clones http.DefaultTransport and configures TLS, proxy, and connection pool.
type nativeEngine struct {
	// proxyURL specifies the proxy server (HTTP/HTTPS/SOCKS5). Empty for direct/env proxy.
	proxyURL string

	// timeout specifies the request timeout. Zero means no timeout.
	timeout time.Duration
}

// CreateTransport returns a *http.Transport configured with standard TLS,
// proxy support, and connection pool settings.
func (e *nativeEngine) CreateTransport() (http.RoundTripper, error) {
	transport := http.DefaultTransport.(*http.Transport).Clone()

	// Standard TLS configuration.
	transport.TLSClientConfig = &tls.Config{
		InsecureSkipVerify: false,
		MinVersion:         tls.VersionTLS12,
	}

	// Connection pool tuning.
	transport.MaxIdleConnsPerHost = 10

	// Configure proxy if specified.
	if e.proxyURL != "" {
		if err := configureProxyOnTransport(transport, e.proxyURL); err != nil {
			return nil, err
		}
	}

	// Wrap transport to enforce timeout at the client level.
	// The transport itself does not have a timeout field; the http.Client does.
	// Since gclient.Client embeds http.Client, we rely on the client-level Timeout
	// or the transport-level ResponseHeaderTimeout for timeout enforcement.
	if e.timeout > 0 {
		transport.ResponseHeaderTimeout = e.timeout
	}

	return transport, nil
}

// configureProxyOnTransport sets up HTTP or SOCKS5 proxy on the given transport.
func configureProxyOnTransport(transport *http.Transport, proxyURLStr string) error {
	parsedURL, err := url.Parse(proxyURLStr)
	if err != nil {
		return fmt.Errorf(`failed to parse proxy URL %q: %w`, proxyURLStr, err)
	}

	switch parsedURL.Scheme {
	case "socks5", "socks5h":
		// SOCKS5 proxy: use DialContext with proxy.SOCKS5 dialer.
		dialer, err := createSOCKS5Dialer(parsedURL)
		if err != nil {
			return err
		}
		transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
			return dialer.Dial(network, addr)
		}
		// Disable HTTP proxy to avoid double-proxying through SOCKS5.
		transport.Proxy = nil

	case "http", "https", "":
		// HTTP/HTTPS proxy.
		transport.Proxy = http.ProxyURL(parsedURL)

	default:
		return fmt.Errorf(`unsupported proxy scheme %q (supported: http, https, socks5)`, parsedURL.Scheme)
	}

	return nil
}

// createSOCKS5Dialer creates a SOCKS5 dialer from the parsed proxy URL.
func createSOCKS5Dialer(proxyURL *url.URL) (proxy.Dialer, error) {
	var auth *proxy.Auth
	if proxyURL.User != nil {
		password, _ := proxyURL.User.Password()
		auth = &proxy.Auth{
			User:     proxyURL.User.Username(),
			Password: password,
		}
	}

	proxyAddr := proxyURL.Host
	if proxyURL.Port() == "" {
		// SOCKS5 default port.
		proxyAddr = net.JoinHostPort(proxyURL.Hostname(), "1080")
	}

	dialer, err := proxy.SOCKS5("tcp", proxyAddr, auth, proxy.Direct)
	if err != nil {
		return nil, fmt.Errorf(`failed to create SOCKS5 dialer: %w`, err)
	}

	return dialer, nil
}
