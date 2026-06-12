# httpclient

`httpclient`包提供了带有可插拔 TLS 传输引擎的 HTTP 客户端 SDK，支持 TLS 指纹模拟、自定义代理和 Chrome 模拟。

# 安装

```
go get -u github.com/gogf/gf/contrib/sdk/httpclient/v2
```

# 功能特性

- **三种传输引擎**：`go_native`、`utls`、`chrome_impersonate`
- **TLS 指纹模拟**：Node.js v24、Chrome、Firefox、随机指纹，或自定义`ClientHelloSpec`
- **代理支持**：HTTP CONNECT、HTTPS、SOCKS5 代理
- **向后兼容**：未配置引擎时，使用默认的`gclient`传输

# 用法

## 基本用法（无引擎）

未指定`Engine`时，客户端行为与之前完全一致。

```go
package main

import (
    "context"

    "github.com/gogf/gf/v2/frame/g"

    httpclient "github.com/gogf/gf/contrib/sdk/httpclient/v2"
)

func main() {
    client := httpclient.New(httpclient.Config{
        URL: "http://example.com",
    })

    type Req struct {
        g.Meta `path:"/api/get" method:"get"`
    }
    type Res struct {
        Message string `json:"message"`
    }

    var (
        ctx = context.Background()
        req = &Req{}
        res = &Res{}
    )
    if err := client.Request(ctx, req, res); err != nil {
        panic(err)
    }
    g.Dump(res)
}
```

## 使用 uTLS 引擎（Node.js v24 指纹）

模拟 Node.js v24.6.0 的 TLS 指纹，绕过 JA3/JA4 指纹检测。

```go
client := httpclient.New(httpclient.Config{
    URL: "https://api.example.com",
    Engine: &httpclient.EngineConfig{
        Type:        httpclient.EngineTypeUTLS,
        Fingerprint: httpclient.FingerprintNodeJSV24,
    },
})
```

## 使用 uTLS 引擎和 SOCKS5 代理

```go
client := httpclient.New(httpclient.Config{
    URL: "https://api.example.com",
    Engine: &httpclient.EngineConfig{
        Type:        httpclient.EngineTypeUTLS,
        Fingerprint: httpclient.FingerprintChromeAuto,
        ProxyURL:    "socks5://user:pass@127.0.0.1:1080",
        Timeout:     30 * time.Second,
    },
})
```

## 使用 Chrome 模拟引擎

通过`imroc/req/v3`实现完整的 Chrome TLS 指纹（包括 HTTP/2 设置）。

```go
client := httpclient.New(httpclient.Config{
    URL: "https://api.example.com",
    Engine: &httpclient.EngineConfig{
        Type: httpclient.EngineTypeChromeImpersonate,
    },
})
```

## 使用自定义 ClientHelloSpec

```go
client := httpclient.New(httpclient.Config{
    URL: "https://api.example.com",
    Engine: &httpclient.EngineConfig{
        Type:        httpclient.EngineTypeUTLS,
        Fingerprint: httpclient.FingerprintCustom,
        CustomSpec:  myCustomSpec, // *utls.ClientHelloSpec
    },
})
```

# 引擎类型

| 引擎 | 描述 | 适用场景 |
|---|---|---|
| `go_native` | 标准 Go `net/http` 传输，支持代理 | 默认行为，不模拟指纹 |
| `utls` | 基于`refraction-networking/utls`的自定义 TLS 指纹 | 通过预设或自定义指纹绕过 JA3/JA4 检测 |
| `chrome_impersonate` | 基于`imroc/req/v3`的`ImpersonateChrome` | 完整 Chrome 模拟（TLS + HTTP/2 设置） |

# 指纹配置

| 指纹 | 描述 | JA3 哈希 |
|---|---|---|
| `nodejs_v24` | Node.js v24.6.0 TLS 指纹 | `944d1e1858cd278718f8a46b65d3212f` |
| `chrome_auto` | 最新 Chrome 指纹（由 uTLS 管理） | N/A |
| `firefox_auto` | 最新 Firefox 指纹（由 uTLS 管理） | N/A |
| `randomized` | 随机 TLS 指纹（由 uTLS 管理） | N/A |
| `custom` | 用户提供的`ClientHelloSpec` | 用户自定义 |

## 许可证

`GoFrame httpclient`基于[MIT 许可证](../../../LICENSE)发布，完全免费开源。
