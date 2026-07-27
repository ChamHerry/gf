// Copyright GoFrame Author(https://goframe.org). All Rights Reserved.
//
// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT was not distributed with this file,
// You can obtain one at https://github.com/gogf/gf.

// Package s3 为 gstorage 提供 S3-compatible（含 MinIO）适配器。
package s3

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"

	"github.com/aws/aws-sdk-go-v2/aws"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/gogf/gf/v2/errors/gcode"
	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/storage/gstorage"
)

const adapterType = "s3"

// Adapter 是基于 AWS SDK for Go v2 的 S3-compatible 适配器。
// SDK client 保持私有，公共 API 不提供逃逸入口。
type Adapter struct {
	client    *awss3.Client
	transport *http.Transport
	closed    atomic.Bool
}

func init() {
	if err := gstorage.RegisterAdapterFunc(adapterType, func(config *gstorage.Config) (gstorage.Adapter, error) {
		return New(config)
	}); err != nil {
		panic(err)
	}
}

// New 使用 gcfg 已加载的强类型配置创建 S3-compatible 适配器。
// 这里直接构造 SDK 配置，不读取 SDK 环境变量、共享凭据文件或命令行参数。
func New(config *gstorage.Config) (*Adapter, error) {
	endpoint, err := validateConfig(config)
	if err != nil {
		return nil, err
	}
	transport := &http.Transport{}
	if defaultTransport, ok := http.DefaultTransport.(*http.Transport); ok {
		transport = defaultTransport.Clone()
	}
	// endpoint 必须完全来自 gcfg，禁止代理环境变量改变实际目标。
	transport.Proxy = nil
	httpClient := &http.Client{
		Transport: transport,
		Timeout:   config.Timeout,
	}
	credentials := aws.Credentials{
		AccessKeyID:     config.Credentials.AccessKey,
		SecretAccessKey: config.Credentials.SecretKey,
		SessionToken:    config.Credentials.SessionToken,
		Source:          "gcfg-yaml",
	}
	awsConfig := aws.Config{
		Region: config.Region,
		Credentials: aws.CredentialsProviderFunc(func(context.Context) (aws.Credentials, error) {
			return credentials, nil
		}),
		HTTPClient:                 httpClient,
		Retryer:                    func() aws.Retryer { return aws.NopRetryer{} },
		RequestChecksumCalculation: aws.RequestChecksumCalculationWhenRequired,
		ResponseChecksumValidation: aws.ResponseChecksumValidationWhenRequired,
	}
	client := awss3.NewFromConfig(awsConfig, func(options *awss3.Options) {
		options.BaseEndpoint = aws.String(endpoint)
		options.UsePathStyle = config.PathStyle
	})
	return &Adapter{
		client:    client,
		transport: transport,
	}, nil
}

func validateConfig(config *gstorage.Config) (string, error) {
	if config == nil {
		return "", gerror.NewCode(gcode.CodeInvalidConfiguration, `storage configuration cannot be nil`)
	}
	if strings.TrimSpace(config.Type) != adapterType {
		return "", gerror.NewCodef(gcode.CodeInvalidConfiguration, `storage type must be %q`, adapterType)
	}
	if strings.TrimSpace(config.Endpoint) == "" {
		return "", gerror.NewCode(gcode.CodeInvalidConfiguration, `storage endpoint is required`)
	}
	if strings.TrimSpace(config.Region) == "" {
		return "", gerror.NewCode(gcode.CodeInvalidConfiguration, `storage region is required`)
	}
	if strings.TrimSpace(config.Credentials.AccessKey) == "" || strings.TrimSpace(config.Credentials.SecretKey) == "" {
		return "", gerror.NewCode(gcode.CodeInvalidConfiguration, `storage accessKey and secretKey are required`)
	}
	if config.Timeout <= 0 {
		return "", gerror.NewCode(gcode.CodeInvalidConfiguration, `storage timeout must be greater than zero`)
	}
	endpoint := strings.TrimSpace(config.Endpoint)
	if !strings.Contains(endpoint, "://") {
		if config.TLS {
			endpoint = "https://" + endpoint
		} else {
			endpoint = "http://" + endpoint
		}
	}
	parsed, err := url.Parse(endpoint)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return "", gerror.NewCodef(gcode.CodeInvalidConfiguration, `invalid storage endpoint %q`, config.Endpoint)
	}
	if parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || (parsed.Path != "" && parsed.Path != "/") {
		return "", gerror.NewCode(gcode.CodeInvalidConfiguration, `storage endpoint must not contain credentials, path, query or fragment`)
	}
	if (parsed.Scheme == "https") != config.TLS {
		return "", gerror.NewCode(gcode.CodeInvalidConfiguration, `storage endpoint scheme and tls setting must match`)
	}
	return strings.TrimRight(parsed.String(), "/"), nil
}
