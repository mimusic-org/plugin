# go-plugin-http 使用示例

本目录包含 go-plugin-http 的完整使用示例，基于 [go-plugin](https://github.com/knqyf263/go-plugin) 框架，展示如何在 host 端和 plugin 端使用 HTTP 请求功能。

[English Documentation](./README_EN.md)

## 目录结构

```
example/
├── build/              # 编译输出目录（自动生成）
│   ├── host            # Host 端可执行程序
│   └── plugin.wasm     # WASM 插件
├── host/               # Host 端示例（主程序）
│   └── main.go
├── plugin/             # Plugin 端示例（WASM 插件）
│   └── main.go
├── proto/              # 协议定义
│   ├── greeter.proto   # 插件接口定义
│   └── *.pb.go         # 生成的代码（自动生成）
├── gen.sh              # proto 代码生成脚本
├── go.mod              # Go 模块配置
├── Makefile            # 构建脚本
└── README.md
```

## 快速开始

```bash
# 一键编译并运行
make run

# 或者分步执行
make gen      # 生成 proto 代码
make plugin   # 编译 WASM 插件（使用 -buildmode=c-shared）
make host     # 编译 Host 程序
make run      # 运行示例
```

## 运行结果

```
=== go-plugin-http 示例 ===
插件加载成功，正在调用 Greet 方法...

[Plugin Log] 收到请求: name=World
[Plugin Log] 正在发起 HTTP 请求到 httpbin.org...
INFO HttpLibrary.DoRequest method=GET url="https://httpbin.org/get?greeting=hello"
INFO HTTP 请求成功 statusCode=200 bodyLen=313
[Plugin Log] HTTP 请求成功, 状态码: 200, 响应长度: 313 bytes

插件返回: Hello, World! HTTP 请求成功，状态码: 200，响应长度: 313 bytes

=== 示例完成 ===
```

## 重要说明

### WASM 编译模式

Go 1.24+ 需要使用 `-buildmode=c-shared` 来构建 WASI reactor：

```bash
GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -o plugin.wasm ./plugin/main.go
```

这会生成 `_initialize` 函数而不是 `_start`，使得模块可以在初始化后保持活跃，支持多次调用导出的函数。

## 示例说明

### Host 端 (host/main.go)

演示如何：
1. 创建 go-plugin 插件加载器
2. 注入 HTTP Library 到 wazero runtime
3. 加载 WASM 插件并调用其方法

关键代码：
```go
p, err := proto.NewGreeterPlugin(ctx, proto.WazeroRuntime(func(ctx context.Context) (wazero.Runtime, error) {
    r, err := proto.DefaultWazeroRuntime()(ctx)
    if err != nil {
        return nil, err
    }
    // 注入 HTTP Library
    return r, httpexport.Instantiate(ctx, r, httpimpl.HttpLibraryImpl{})
}))
```

### Plugin 端 (plugin/main.go)

演示如何：
1. 实现 go-plugin 插件接口
2. 在插件中使用 HTTP 请求功能
3. 调用 host 函数进行日志输出

关键代码：
```go
import http "github.com/mimusic-org/plugin/pkg/go-plugin-http/plugin"

resp, err := http.Get("https://httpbin.org/get")
if err != nil {
    return nil, err
}
defer resp.Body.Close()
```

## 清理

```bash
make clean
```
