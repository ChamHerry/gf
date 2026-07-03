# elasticsearch

`elasticsearch`包基于 Elasticsearch 官方 Go client 注册`database/gsearch`适配器。

# 安装

```bash
go get -u github.com/gogf/gf/contrib/nosql/elasticsearch/v2
```

# 用法

## 启用适配器

调用`g.Search()`或`gsearch.New()`前，通过空白导入完成适配器注册。

```go
package main

import (
	_ "github.com/gogf/gf/contrib/nosql/elasticsearch/v2"

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
    type: elasticsearch
    addresses:
      - "http://127.0.0.1:9200"
  secured:
    type: elasticsearch
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

## 原生客户端逃生口

`Client()`以`any`返回官方`*elasticsearch.Client`对象。需要使用`database/gsearch`未建模的 Elasticsearch API 时，可以通过它访问原生能力。

# 配置项

| 字段 | 说明 |
|---|---|
| `type` | 必须为`elasticsearch`。 |
| `addresses` | Elasticsearch 节点地址。 |
| `username`/`password` | Basic Auth 认证。 |
| `apiKey` | Elasticsearch API key 认证。 |
| `serviceToken` | Elasticsearch service token 认证。 |
| `cloudId` | Elastic Cloud deployment ID。 |
| `headers` | 附加 HTTP header。 |
| `caCert`/`certificateFingerprint` | TLS 信任材料。 |
| `tls`/`tlsSkipVerify` | TLS 传输行为。 |
| `retryOnStatus`/`maxRetries` | 重试策略。 |
| `compressRequestBody` | 请求体压缩。 |
| `discoverNodesOnStart` | 启动时节点发现。 |

# 兼容性

本模块使用`github.com/elastic/go-elasticsearch/v8 v8.19.6`。该依赖只存在于当前 contrib module，不会进入 root`go.mod`。

本适配器用于 Elasticsearch 集群。不要把它作为 OpenSearch 兼容层；OpenSearch 请导入`contrib/nosql/opensearch/v2`。

# 错误语义

Search 和 Bulk 请求即使 HTTP 状态成功，也可能存在局部失败。调用方应检查：

| 响应 | 字段 |
|---|---|
| `SearchResponse` | `TimedOut`、`Shards.Failed`、`Shards.Failures` |
| `BulkResponse` | `Errors`、`Items[].Error` |

# 许可与专利边界

`GoFrame elasticsearch`基于[MIT 许可证](../../../LICENSE)发布。Elasticsearch 官方 Go client 使用其上游项目许可。

本适配器只调用公开 REST API，不实现服务端索引、排序、查询规划、向量搜索或存储算法。添加服务端搜索类高级能力前，仍需进行专利审查。
