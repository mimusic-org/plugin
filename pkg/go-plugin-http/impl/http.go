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
	"net/http/cookiejar"
	"strings"
	"time"

	"github.com/mimusic-org/plugin/pkg/go-plugin-http/export"
)

// setCookieCollector 包装 http.RoundTripper，收集重定向链中所有响应的 Set-Cookie 头
type setCookieCollector struct {
	transport  http.RoundTripper
	setCookies []string
}

func (c *setCookieCollector) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := c.transport.RoundTrip(req)
	if err != nil {
		return resp, err
	}
	// 收集每个响应（包括重定向中间响应）的 Set-Cookie
	c.setCookies = append(c.setCookies, resp.Header.Values("Set-Cookie")...)
	return resp, err
}

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

	// 创建 cookie jar，确保重定向链中的 cookie 能被正确传递
	jar, _ := cookiejar.New(nil)

	// 创建 Set-Cookie 收集器，收集重定向链中所有响应的 Set-Cookie 头
	collector := &setCookieCollector{
		transport: &http.Transport{},
	}

	// 创建 HTTP 客户端，设置超时
	client := &http.Client{
		Jar:       jar,
		Transport: collector,
	}

	// 根据插件请求决定是否禁用自动重定向
	// 不设置 CheckRedirect 则使用 Go 默认行为（自动跟随重定向，最多 10 次）
	if req.GetDisableRedirect() {
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}
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

	originalURL := httpReq.URL.String()

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

	// 重定向检测日志
	finalURL := resp.Request.URL.String()
	if finalURL != originalURL {
		slog.Info("[HttpHost] redirect detected", "originalURL", originalURL, "finalURL", finalURL)
	} else {
		slog.Info("[HttpHost] no redirect", "url", finalURL)
	}

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
			if strings.EqualFold(key, "Set-Cookie") {
				// Set-Cookie 不能用 ", " 拼接（cookie 值可能含逗号），使用 "\n"
				continue // 下面统一处理
			}
			respHeaders[key] = strings.Join(values, ", ")
		}
	}

	// 合并重定向链中所有 Set-Cookie 头（包括中间响应和最终响应）
	allSetCookies := make([]string, 0, len(collector.setCookies))
	seen := make(map[string]bool)
	// collector 已收集所有响应（含最终响应）的 Set-Cookie
	for _, sc := range collector.setCookies {
		if !seen[sc] {
			seen[sc] = true
			allSetCookies = append(allSetCookies, sc)
		}
	}
	if len(allSetCookies) > 0 {
		respHeaders["Set-Cookie"] = strings.Join(allSetCookies, "\n")
	}

	slog.Info("HTTP 请求成功", "statusCode", resp.StatusCode, "bodyLen", len(respBody))

	return &export.HttpResponse{
		Success:    true,
		StatusCode: int32(resp.StatusCode),
		Headers:    respHeaders,
		Body:       respBody,
	}, nil
}
