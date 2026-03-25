# go-plugin-http

[![Go Reference](https://pkg.go.dev/badge/github.com/mimusic-org/plugin/pkg/go-plugin-http.svg)](https://pkg.go.dev/github.com/mimusic-org/plugin/pkg/go-plugin-http)
[![License: Apache 2.0](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

为基于 [go-plugin](https://github.com/knqyf263/go-plugin) 的 WASM 插件提供 HTTP 请求能力。

[English Documentation](./README_EN.md)

由于 WASM 环境不支持 Go 标准库的 `net/http` 包，本库通过 host 函数代理的方式，让 WASM 插件能够发起 HTTP 请求。

## 要求

- Go 1.24+（需要 `-buildmode=c-shared` 支持 WASI reactor）
- [go-plugin](https://github.com/knqyf263/go-plugin) v0.9.0+
- [wazero](https://github.com/tetratelabs/wazero) v1.11.0+

## 安装

```bash
go get github.com/mimusic-org/plugin/pkg/go-plugin-http
```

## 使用方式

### Host 端（主程序）

在创建 wazero runtime 时注入 HTTP Library：

```go
import (
    httpexport "github.com/mimusic-org/plugin/pkg/go-plugin-http/export"
    httpimpl "github.com/mimusic-org/plugin/pkg/go-plugin-http/impl"
)

// 在创建插件加载器时注入 HTTP Library
loader, err := plugins.NewPluginServicePlugin(ctx,
    plugins.WazeroRuntime(func(ctx context.Context) (wazero.Runtime, error) {
        r, err := plugins.DefaultWazeroRuntime()(ctx)
        if err != nil {
            return nil, err
        }
        // 注入 HTTP Library
        if err := httpexport.Instantiate(ctx, r, httpimpl.HttpLibraryImpl{}); err != nil {
            return nil, err
        }
        return r, nil
    }))
```

### Plugin 端（WASM 插件）

在插件代码中使用 HTTP 请求：

```go
import http "github.com/mimusic-org/plugin/pkg/go-plugin-http/plugin"

// GET 请求
resp, err := http.Get("https://example.com/api")
if err != nil {
    // 处理错误
}
defer resp.Body.Close()
body, _ := io.ReadAll(resp.Body)

// POST 请求
resp, err := http.Post("https://example.com/api", "application/json", bytes.NewReader(jsonData))

// 自定义请求
req, _ := http.NewRequest("PUT", "https://example.com/api", body)
req.Header.Set("Authorization", "Bearer token")
resp, err := http.Do(req)

// 使用自定义客户端（带超时）
client := &http.Client{Timeout: 10 * time.Second}
resp, err := client.Get("https://example.com/api")
```

## API 参考

### 类型

- `Header` - HTTP 头部，类似 `net/http.Header`
- `Request` - HTTP 请求，类似 `net/http.Request`
- `Response` - HTTP 响应，类似 `net/http.Response`
- `Client` - HTTP 客户端，类似 `net/http.Client`

### 函数

- `NewRequest(method, url string, body io.Reader) (*Request, error)` - 创建请求
- `Get(url string) (*Response, error)` - 发起 GET 请求
- `Post(url, contentType string, body io.Reader) (*Response, error)` - 发起 POST 请求
- `Head(url string) (*Response, error)` - 发起 HEAD 请求
- `Do(req *Request) (*Response, error)` - 执行请求

### Client 方法

- `Do(req *Request) (*Response, error)` - 执行请求
- `DoContext(ctx context.Context, req *Request) (*Response, error)` - 执行带 context 的请求
- `Get(url string) (*Response, error)` - 发起 GET 请求
- `Post(url, contentType string, body io.Reader) (*Response, error)` - 发起 POST 请求
- `Head(url string) (*Response, error)` - 发起 HEAD 请求

## 完整示例

查看 [example](./example) 目录获取完整的使用示例，包括：

- Host 端如何加载 WASM 插件并注入 HTTP Library
- Plugin 端如何使用 HTTP 请求功能
- 如何编译 WASI reactor 模式的 WASM 插件

运行示例：

```bash
cd example
make run
```

## WASM 编译说明

Go 1.24+ 需要使用 `-buildmode=c-shared` 来构建 WASI reactor：

```bash
GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -o plugin.wasm ./plugin/main.go
```

这会生成 `_initialize` 函数而不是 `_start`，使得模块可以在初始化后保持活跃，支持多次调用导出的函数。

## 开发

### 重新生成 protobuf 代码

```bash
./gen.sh
```

### 运行测试

```bash
cd example
make run
```

## License

Apache License 2.0 - 详见 [LICENSE](./LICENSE) 文件
