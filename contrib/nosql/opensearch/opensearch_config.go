// Copyright GoFrame Author(https://goframe.org). All Rights Reserved.
//
// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT was not distributed with this file,
// You can obtain one at https://github.com/gogf/gf.

// OpenSearch client configuration mapping.

package opensearch

import (
	"crypto/tls"
	"net/http"

	opensearchv3 "github.com/opensearch-project/opensearch-go/v3"
	opensearchsigner "github.com/opensearch-project/opensearch-go/v3/signer"

	"github.com/gogf/gf/v2/database/gsearch"
)

const (
	// ExtraKeySigner is the adapter-local Extra key for an official OpenSearch signer.
	ExtraKeySigner = "signer"
)

// newClientConfig maps root gsearch configuration to the official client configuration.
func newClientConfig(config *gsearch.Config) opensearchv3.Config {
	clientConfig := opensearchv3.Config{
		Addresses:            config.Addresses,
		Username:             config.Username,
		Password:             config.Password,
		Header:               newHTTPHeader(config.Headers),
		CACert:               config.CACert,
		RetryOnStatus:        config.RetryOnStatus,
		MaxRetries:           config.MaxRetries,
		CompressRequestBody:  config.CompressRequestBody,
		DiscoverNodesOnStart: config.DiscoverNodesOnStart,
	}
	if signerValue := signerFromExtra(config.Extra); signerValue != nil {
		clientConfig.Signer = signerValue
	}
	if config.TLS || config.TLSSkipVerify {
		clientConfig.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: config.TLSSkipVerify,
			},
		}
	}
	return clientConfig
}

// signerFromExtra extracts an adapter-local OpenSearch signer from Config.Extra.
func signerFromExtra(extra map[string]any) opensearchsigner.Signer {
	if len(extra) == 0 {
		return nil
	}
	signerValue, ok := extra[ExtraKeySigner].(opensearchsigner.Signer)
	if !ok {
		return nil
	}
	return signerValue
}

// newHTTPHeader converts string headers to http.Header.
func newHTTPHeader(headers map[string]string) http.Header {
	if len(headers) == 0 {
		return nil
	}
	httpHeader := make(http.Header, len(headers))
	for key, value := range headers {
		httpHeader.Set(key, value)
	}
	return httpHeader
}
