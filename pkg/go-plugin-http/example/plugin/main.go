//go:build wasip1

// Package main 演示如何在 WASM 插件中使用 go-plugin-http。
// 本示例基于 go-plugin 框架，展示如何在插件中发起 HTTP 请求。
package main

import (
	"context"
	"fmt"
	"io"

	"github.com/mimusic-org/plugin/pkg/go-plugin-http/example/proto"
	"github.com/mimusic-org/plugin/pkg/go-plugin-http/http"
)

// main 是 Go 编译为 WASM 所必需的
func main() {}

func init() {
	proto.RegisterGreeter(GreeterPlugin{})
}

// GreeterPlugin 实现 Greeter 接口
type GreeterPlugin struct{}

var _ proto.Greeter = (*GreeterPlugin)(nil)

// Greet 实现 Greeter 接口的 Greet 方法
// 该方法会发起 HTTP 请求来获取数据
func (p GreeterPlugin) Greet(ctx context.Context, request *proto.GreetRequest) (*proto.GreetReply, error) {
	// 获取 host 函数用于日志输出
	hostFunctions := proto.NewHostFunctions()

	// 记录日志
	hostFunctions.Log(ctx, &proto.LogRequest{
		Message: fmt.Sprintf("收到请求: name=%s", request.GetName()),
	})

	// 发起 HTTP GET 请求
	hostFunctions.Log(ctx, &proto.LogRequest{
		Message: "正在发起 HTTP 请求到 httpbin.org...",
	})

	resp, err := http.Get("https://httpbin.org/get?greeting=hello")
	if err != nil {
		return nil, fmt.Errorf("HTTP 请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	hostFunctions.Log(ctx, &proto.LogRequest{
		Message: fmt.Sprintf("HTTP 请求成功, 状态码: %d, 响应长度: %d bytes", resp.StatusCode, len(body)),
	})

	// 返回结果
	return &proto.GreetReply{
		Message: fmt.Sprintf("Hello, %s! HTTP 请求成功，状态码: %d，响应长度: %d bytes",
			request.GetName(), resp.StatusCode, len(body)),
	}, nil
}
