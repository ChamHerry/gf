// Copyright GoFrame Author(https://goframe.org). All Rights Reserved.
//
// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT was not distributed with this file,
// You can obtain one at https://github.com/gogf/gf.

// Elasticsearch client configuration mapping.

package elasticsearch

import (
	"crypto/tls"
	"net/http"

	elasticv8 "github.com/elastic/go-elasticsearch/v8"

	"github.com/gogf/gf/v2/database/gsearch"
)

// newClientConfig maps root gsearch configuration to the official client configuration.
func newClientConfig(config *gsearch.Config) elasticv8.Config {
	clientConfig := elasticv8.Config{
		Addresses:              config.Addresses,
		Username:               config.Username,
		Password:               config.Password,
		APIKey:                 config.APIKey,
		ServiceToken:           config.ServiceToken,
		CloudID:                config.CloudID,
		Header:                 newHTTPHeader(config.Headers),
		CACert:                 config.CACert,
		CertificateFingerprint: config.CertificateFingerprint,
		RetryOnStatus:          config.RetryOnStatus,
		MaxRetries:             config.MaxRetries,
		CompressRequestBody:    config.CompressRequestBody,
		DiscoverNodesOnStart:   config.DiscoverNodesOnStart,
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
