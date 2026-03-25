//go:build !wasip1

// Package main 演示如何在 host 端使用 go-plugin-http。
// 本示例展示如何基于 go-plugin 框架加载 WASM 插件并注入 HTTP Library。
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/tetratelabs/wazero"

	"github.com/mimusic-org/plugin/pkg/go-plugin-http/example/proto"
	httpexport "github.com/mimusic-org/plugin/pkg/go-plugin-http/export"
	httpimpl "github.com/mimusic-org/plugin/pkg/go-plugin-http/impl"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	ctx := context.Background()

	// 创建插件加载器，注入 HTTP Library
	// 使用默认配置，go-plugin 会自动处理启动函数
	p, err := proto.NewGreeterPlugin(ctx,
		proto.WazeroRuntime(func(ctx context.Context) (wazero.Runtime, error) {
			r, err := proto.DefaultWazeroRuntime()(ctx)
			if err != nil {
				return nil, err
			}
			// 注入 HTTP Library - 这是关键步骤
			return r, httpexport.Instantiate(ctx, r, httpimpl.HttpLibraryImpl{})
		}))
	if err != nil {
		return fmt.Errorf("创建插件加载器失败: %w", err)
	}

	// 加载 WASM 插件
	plugin, err := p.Load(ctx, "../build/plugin.wasm", myHostFunctions{})
	if err != nil {
		// 如果加载失败，打印详细错误信息
		fmt.Fprintf(os.Stderr, "加载插件失败: %v\n", err)
		fmt.Fprintf(os.Stderr, "\n提示: 请确保已正确编译 WASM 插件\n")
		fmt.Fprintf(os.Stderr, "运行: make plugin\n")
		return err
	}
	defer plugin.Close(ctx)

	fmt.Println("=== go-plugin-http 示例 ===")
	fmt.Println("插件加载成功，正在调用 Greet 方法...")
	fmt.Println()

	// 调用插件的 Greet 方法
	reply, err := plugin.Greet(ctx, &proto.GreetRequest{
		Name: "World",
	})
	if err != nil {
		return fmt.Errorf("调用插件失败: %w", err)
	}

	fmt.Println()
	fmt.Printf("插件返回: %s\n", reply.GetMessage())
	fmt.Println()
	fmt.Println("=== 示例完成 ===")

	return nil
}

// myHostFunctions 实现插件需要的 host 函数
type myHostFunctions struct{}

var _ proto.HostFunctions = (*myHostFunctions)(nil)

func (m myHostFunctions) Log(_ context.Context, request *proto.LogRequest) (*proto.LogResponse, error) {
	fmt.Printf("[Plugin Log] %s\n", request.GetMessage())
	return &proto.LogResponse{Success: true}, nil
}
