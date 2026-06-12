// Copyright GoFrame Author(https://goframe.org). All Rights Reserved.
//
// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this file,
// You can obtain one at https://github.com/gogf/gf.

// This file implements the chrome_impersonate engine, which uses imroc/req/v3
// with ImpersonateChrome to produce a full Chrome TLS fingerprint.

package httpclient

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"time"

	reqClient "github.com/imroc/req/v3"
)

// chromeEngine implements the Engine interface using imroc/req/v3
// with Chrome impersonation. Each request creates a fresh req.Client
// to ensure clean sessions (no cookie leakage between requests).
type chromeEngine struct {
	// proxyURL specifies the proxy server. Empty for direct connection.
	proxyURL string

	// timeout specifies the request timeout. Zero means no timeout.
	timeout time.Duration
}

// CreateTransport returns an http.RoundTripper that delegates each request
// to a fresh req.Client configured with ImpersonateChrome.
func (e *chromeEngine) CreateTransport() (http.RoundTripper, error) {
	return &chromeTransport{
		proxyURL: e.proxyURL,
		timeout:  e.timeout,
	}, nil
}

// chromeTransport implements http.RoundTripper by wrapping req.Client.
// Each RoundTrip call creates a new req.Client to maintain a clean session
// with accurate Chrome TLS fingerprint on every request.
type chromeTransport struct {
	// proxyURL specifies the proxy server for outgoing requests.
	proxyURL string

	// timeout specifies the per-request timeout.
	timeout time.Duration
}

// RoundTrip executes an HTTP request using a fresh Chrome-impersonated req.Client.
// It converts the standard *http.Request to a req request and converts the
// response back to a standard *http.Response.
func (t *chromeTransport) RoundTrip(httpReq *http.Request) (*http.Response, error) {
	// Create a fresh req.Client for each request (clean session, accurate fingerprint).
	client := reqClient.C().
		ImpersonateChrome().
		SetCookieJar(nil) // Disable CookieJar for clean sessions.

	if t.timeout > 0 {
		client.SetTimeout(t.timeout)
	}

	if t.proxyURL != "" {
		client.SetProxyURL(t.proxyURL)
	}

	// Read request body for forwarding (req requires a reader, not http.Request.Body directly).
	var bodyReader io.Reader
	if httpReq.Body != nil {
		bodyBytes, err := io.ReadAll(httpReq.Body)
		if err != nil {
			return nil, err
		}
		if closeErr := httpReq.Body.Close(); closeErr != nil {
			return nil, fmt.Errorf(`failed to close request body: %w`, closeErr)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	// Execute the request via req.Client.
	resp, err := client.R().
		SetBody(bodyReader).
		Send(httpReq.Method, httpReq.URL.String())
	if err != nil {
		return nil, err
	}

	// Convert req.Response to standard *http.Response.
	httpResp := &http.Response{
		StatusCode: resp.StatusCode,
		Header:     make(http.Header),
		Proto:      resp.Proto,
		ProtoMajor: resp.ProtoMajor,
		ProtoMinor: resp.ProtoMinor,
	}

	// Copy headers.
	for key, values := range resp.Header {
		for _, value := range values {
			httpResp.Header.Add(key, value)
		}
	}

	// Copy body.
	if resp.Body != nil {
		httpResp.Body = io.NopCloser(resp.Body)
	}

	return httpResp, nil
}
