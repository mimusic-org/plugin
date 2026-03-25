//go:build !wasip1

// Package impl 提供 HTTP Library 的 host 端实现。
// 该实现会被注入到 wazero runtime 中，供 WASM 插件调用。
//
// 使用方式：
//
//	import (
//	    httpexport "github.com/mimusic-org/plugin/pkg/go-plugin-http/export"
//	    httpimpl "github.com/mimusic-org/plugin/pkg/go-plugin-http/impl"
//	)
//
//	// 在创建 wazero runtime 时注入
//	httpexport.Instantiate(ctx, runtime, httpimpl.HttpLibraryImpl{})
package impl

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/mimusic-org/plugin/pkg/go-plugin-http/export"
)

// 确保 HttpLibraryImpl 实现了 export.HttpLibrary 接口
var _ export.HttpLibrary = (*HttpLibraryImpl)(nil)

// HttpLibraryImpl 实现 export.HttpLibrary 接口
type HttpLibraryImpl struct{}

// DoRequest 执行 HTTP 请求
func (h HttpLibraryImpl) DoRequest(ctx context.Context, req *export.HttpRequest) (*export.HttpResponse, error) {
	slog.Info("HttpLibrary.DoRequest", "method", req.GetMethod(), "url", req.GetUrl())

	// 忽略传入的 ctx，使用 context.Background() 避免被外部取消
	// 在 WASI 环境下，定时器回调中的 context 可能会被取消
	_ = ctx

	// 创建 HTTP 客户端，设置超时
	client := &http.Client{}
	timeoutMs := req.GetTimeoutMs()
	if timeoutMs > 0 {
		client.Timeout = time.Duration(timeoutMs) * time.Millisecond
	} else {
		client.Timeout = 30 * time.Second // 默认 30 秒超时
	}

	// 创建请求体
	var bodyReader io.Reader
	if len(req.GetBody()) > 0 {
		bodyReader = strings.NewReader(string(req.GetBody()))
	}

	// 创建 HTTP 请求，使用 context.Background() 避免被取消
	httpReq, err := http.NewRequestWithContext(context.Background(), req.GetMethod(), req.GetUrl(), bodyReader)
	if err != nil {
		slog.Error("创建 HTTP 请求失败", "error", err)
		return &export.HttpResponse{
			Success: false,
			Error:   "创建请求失败: " + err.Error(),
		}, nil
	}

	// 设置请求头
	for key, value := range req.GetHeaders() {
		httpReq.Header.Set(key, value)
	}

	// 发起 HTTP 请求
	resp, err := client.Do(httpReq)
	if err != nil {
		slog.Error("HTTP 请求失败", "error", err)
		return &export.HttpResponse{
			Success: false,
			Error:   "请求失败: " + err.Error(),
		}, nil
	}
	defer resp.Body.Close()

	// 读取响应体
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		slog.Error("读取响应体失败", "error", err)
		return &export.HttpResponse{
			Success: false,
			Error:   "读取响应失败: " + err.Error(),
		}, nil
	}

	// 构建响应头
	respHeaders := make(map[string]string)
	for key, values := range resp.Header {
		if len(values) > 0 {
			respHeaders[key] = strings.Join(values, ", ")
		}
	}

	slog.Info("HTTP 请求成功", "statusCode", resp.StatusCode, "bodyLen", len(respBody))

	return &export.HttpResponse{
		Success:    true,
		StatusCode: int32(resp.StatusCode),
		Headers:    respHeaders,
		Body:       respBody,
	}, nil
}
