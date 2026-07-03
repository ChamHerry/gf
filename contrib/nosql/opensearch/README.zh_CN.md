# opensearch

`opensearch`包基于 OpenSearch 官方 Go client 注册`database/gsearch`适配器。

# 安装

```bash
go get -u github.com/gogf/gf/contrib/nosql/opensearch/v2
```

# 用法

## 启用适配器

调用`g.Search()`或`gsearch.New()`前，通过空白导入完成适配器注册。

```go
package main

import (
	_ "github.com/gogf/gf/contrib/nosql/opensearch/v2"

	"github.com/gogf/gf/v2/database/gsearch"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gctx"
)

func main() {
	ctx := gctx.New()
	response, err := g.Search().Search(ctx, &gsearch.SearchRequest{
		Index: []string{"movies"},
		Body:  []byte(`{"query":{"match_all":{}}}`),
	})
	if err != nil {
		panic(err)
	}
	g.Dump(response.Hits)
}
```

## 配置

```yaml
search:
  default:
    type: opensearch
    addresses:
      - "http://127.0.0.1:9200"
  secured:
    type: opensearch
    addresses:
      - "https://127.0.0.1:9200"
    username: "search-user"
    password: "search-password"
```

示例中只使用占位凭证。生产凭证应放在应用自己的密钥管理流程中。

# 操作

## Search

```go
response, err := g.Search().Search(ctx, &gsearch.SearchRequest{
	Index: []string{"movies"},
	Query: map[string]string{"allow_partial_search_results": "true"},
	Body:  []byte(`{"query":{"match":{"title":"goframe"}}}`),
})
if err != nil {
	panic(err)
}
if response.TimedOut || response.Shards.Failed > 0 {
	g.Dump(response.Shards.Failures)
}
```

## Bulk

```go
response, err := g.Search().Bulk(ctx, &gsearch.BulkRequest{
	Index: "movies",
	Body: []byte(
		"{\"index\":{\"_id\":\"1\"}}\n" +
			"{\"title\":\"GoFrame\"}\n",
	),
})
if err != nil {
	panic(err)
}
if response.Errors {
	g.Dump(response.Items)
}
```

## Raw request

```go
response, err := g.Search().Perform(ctx, &gsearch.Request{
	Method: gsearch.RequestMethodGet,
	Path:   "/_cluster/health",
})
if err != nil {
	panic(err)
}
g.Dump(response.StatusCode, response.Body)
```

## 适配器局部 signer

OpenSearch 请求签名不进入 root`database/gsearch`配置。手动创建适配器时，可以通过`Config.Extra`传入官方 OpenSearch`signer.Signer`。需要使用`ExtraKeySigner`时，应对本模块进行命名导入。

```go
search, err := gsearch.New(&gsearch.Config{
	Type:      gsearch.EngineTypeOpenSearch,
	Addresses: []string{"https://search-domain.example.com"},
	Extra: map[string]any{
		opensearch.ExtraKeySigner: mySigner,
	},
})
if err != nil {
	panic(err)
}
if err = search.Ping(ctx); err != nil {
	panic(err)
}
```

## 原生客户端逃生口

`Client()`以`any`返回官方`*opensearch.Client`对象。需要使用`database/gsearch`未建模的 OpenSearch API 时，可以通过它访问原生能力。

# 配置项

| 字段 | 说明 |
|---|---|
| `type` | 必须为`opensearch`。 |
| `addresses` | OpenSearch 节点地址。 |
| `username`/`password` | Basic Auth 认证。 |
| `headers` | 附加 HTTP header。 |
| `caCert` | TLS 信任材料。 |
| `tls`/`tlsSkipVerify` | TLS 传输行为。 |
| `retryOnStatus`/`maxRetries` | 重试策略。 |
| `compressRequestBody` | 请求体压缩。 |
| `discoverNodesOnStart` | 启动时节点发现。 |
| `extra.signer` | 适配器局部官方 OpenSearch signer，在 Go 代码中通过`ExtraKeySigner`设置。 |

# 兼容性

本模块使用`github.com/opensearch-project/opensearch-go/v3 v3.1.0`。实施时，`opensearch-go/v4 v4.6.0`要求 Go 1.24，`opensearch-go/v5@latest`仍是未发布伪版本且要求 Go 1.25.9，因此首版适配器选择稳定 v3，以匹配当前仓库 Go 1.23 module 策略。

本适配器用于 OpenSearch 集群。适配器会校验`Info`响应并拒绝非 OpenSearch distribution。不要把它作为 Elasticsearch 兼容层；Elasticsearch 请导入`contrib/nosql/elasticsearch/v2`。

# 错误语义

Search 和 Bulk 请求即使 HTTP 状态成功，也可能存在局部失败。调用方应检查：

| 响应 | 字段 |
|---|---|
| `SearchResponse` | `TimedOut`、`Shards.Failed`、`Shards.Failures` |
| `BulkResponse` | `Errors`、`Items[].Error` |

# 许可与专利边界

`GoFrame opensearch`基于[MIT 许可证](../../../LICENSE)发布。OpenSearch 官方 Go client 由其上游项目基于 Apache-2.0 许可发布。

本适配器只调用公开 REST API，不实现服务端索引、排序、查询规划、向量搜索或存储算法。添加服务端搜索类高级能力前，仍需进行专利审查。
