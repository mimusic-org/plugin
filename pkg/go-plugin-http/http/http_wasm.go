//go:build wasip1
// +build wasip1

// Package http 提供了在 WASM 插件中发起 HTTP 请求的能力。
// 由于 WASM 环境不支持 net/http 包，本包通过 host 函数代理实现 HTTP 请求。
//
// 使用方式与标准库 net/http 类似，只需将 import "net/http" 改为
// import "github.com/mimusic-org/plugin/pkg/go-plugin-http/plugin" 即可。
//
// 示例：
//
//	import http "github.com/mimusic-org/plugin/pkg/go-plugin-http/plugin"
//
//	resp, err := http.Get("https://example.com/api")
//	if err != nil {
//	    // 处理错误
//	}
//	defer resp.Body.Close()
//
//	body, _ := io.ReadAll(resp.Body)
package http

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"time"

	"github.com/mimusic-org/plugin/pkg/go-plugin-http/export"
)

// ErrUseLastResponse 用于在 CheckRedirect 中阻止自动重定向
var ErrUseLastResponse = errors.New("net/http: use last response")

// Header 表示 HTTP 头部，与 net/http.Header 类似
type Header map[string][]string

// Add 添加一个键值对到头部
func (h Header) Add(key, value string) {
	h[key] = append(h[key], value)
}

// Set 设置头部的键值对，会覆盖已有的值
func (h Header) Set(key, value string) {
	h[key] = []string{value}
}

// Get 获取头部的第一个值
func (h Header) Get(key string) string {
	if values := h[key]; len(values) > 0 {
		return values[0]
	}
	return ""
}

// Del 删除头部的键值对
func (h Header) Del(key string) {
	delete(h, key)
}

// Cookie 表示 HTTP Cookie，与 net/http.Cookie 类似
type Cookie struct {
	Name  string
	Value string
}

// URL 表示解析后的 URL，与 net/url.URL 类似
type URL struct {
	Scheme   string
	Host     string
	Path     string
	RawQuery string
	raw      string
}

// Query 返回 URL 的查询参数
func (u *URL) Query() Values {
	v, _ := ParseQuery(u.RawQuery)
	return v
}

// String 返回 URL 的字符串表示
func (u *URL) String() string {
	result := u.raw
	if u.RawQuery != "" {
		if u.raw != "" && !containsQuery(u.raw) {
			result = u.raw + "?" + u.RawQuery
		}
	}
	return result
}

func containsQuery(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] == '?' {
			return true
		}
	}
	return false
}

// Values 表示 URL 查询参数
type Values map[string][]string

// Get 获取第一个值
func (v Values) Get(key string) string {
	if vs := v[key]; len(vs) > 0 {
		return vs[0]
	}
	return ""
}

// Set 设置值
func (v Values) Set(key, value string) {
	v[key] = []string{value}
}

// Add 添加值
func (v Values) Add(key, value string) {
	v[key] = append(v[key], value)
}

// Encode 编码为查询字符串
func (v Values) Encode() string {
	if len(v) == 0 {
		return ""
	}
	var buf strings.Builder
	first := true
	for key, values := range v {
		for _, value := range values {
			if !first {
				buf.WriteByte('&')
			}
			first = false
			buf.WriteString(urlEncode(key))
			buf.WriteByte('=')
			buf.WriteString(urlEncode(value))
		}
	}
	return buf.String()
}

// ParseQuery 解析查询字符串
func ParseQuery(query string) (Values, error) {
	m := make(Values)
	for query != "" {
		key := query
		if i := strings.IndexByte(key, '&'); i >= 0 {
			key, query = key[:i], key[i+1:]
		} else {
			query = ""
		}
		if key == "" {
			continue
		}
		value := ""
		if i := strings.IndexByte(key, '='); i >= 0 {
			key, value = key[:i], key[i+1:]
		}
		m[urlDecode(key)] = append(m[urlDecode(key)], urlDecode(value))
	}
	return m, nil
}

func urlEncode(s string) string {
	var buf strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if shouldEscape(c) {
			buf.WriteByte('%')
			buf.WriteByte("0123456789ABCDEF"[c>>4])
			buf.WriteByte("0123456789ABCDEF"[c&15])
		} else {
			buf.WriteByte(c)
		}
	}
	return buf.String()
}

func urlDecode(s string) string {
	var buf strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '%' && i+2 < len(s) {
			if h, ok := hexToByte(s[i+1], s[i+2]); ok {
				buf.WriteByte(h)
				i += 2
				continue
			}
		} else if s[i] == '+' {
			buf.WriteByte(' ')
			continue
		}
		buf.WriteByte(s[i])
	}
	return buf.String()
}

func hexToByte(h1, h2 byte) (byte, bool) {
	n1, ok1 := hexDigit(h1)
	n2, ok2 := hexDigit(h2)
	if !ok1 || !ok2 {
		return 0, false
	}
	return n1<<4 | n2, true
}

func hexDigit(c byte) (byte, bool) {
	switch {
	case '0' <= c && c <= '9':
		return c - '0', true
	case 'a' <= c && c <= 'f':
		return c - 'a' + 10, true
	case 'A' <= c && c <= 'F':
		return c - 'A' + 10, true
	}
	return 0, false
}

func shouldEscape(c byte) bool {
	if 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z' || '0' <= c && c <= '9' {
		return false
	}
	switch c {
	case '-', '_', '.', '~':
		return false
	}
	return true
}

// Request 表示 HTTP 请求，与 net/http.Request 类似
type Request struct {
	Method string
	URL    *URL
	Header Header
	Body   io.Reader
}

// AddCookie 添加 Cookie 到请求
func (r *Request) AddCookie(c *Cookie) {
	if c == nil {
		return
	}
	existing := r.Header.Get("Cookie")
	if existing != "" {
		r.Header.Set("Cookie", existing+"; "+c.Name+"="+c.Value)
	} else {
		r.Header.Set("Cookie", c.Name+"="+c.Value)
	}
}

// Response 表示 HTTP 响应，与 net/http.Response 类似
type Response struct {
	StatusCode int
	Header     Header
	Body       io.ReadCloser
}

// Cookies 从响应头中解析 cookies
func (r *Response) Cookies() []*Cookie {
	var cookies []*Cookie
	for _, line := range r.Header["Set-Cookie"] {
		if cookie := parseCookie(line); cookie != nil {
			cookies = append(cookies, cookie)
		}
	}
	return cookies
}

// parseCookie 解析单个 Set-Cookie 头
func parseCookie(line string) *Cookie {
	parts := strings.Split(line, ";")
	if len(parts) == 0 {
		return nil
	}
	// 第一部分是 name=value
	nv := strings.SplitN(strings.TrimSpace(parts[0]), "=", 2)
	if len(nv) != 2 {
		return nil
	}
	return &Cookie{
		Name:  nv[0],
		Value: nv[1],
	}
}

// bodyReadCloser 实现 io.ReadCloser 接口
type bodyReadCloser struct {
	*bytes.Reader
}

func (b *bodyReadCloser) Close() error {
	return nil
}

// Client 表示 HTTP 客户端，与 net/http.Client 类似
type Client struct {
	Timeout       time.Duration
	CheckRedirect func(req *Request, via []*Request) error
}

// DefaultClient 是默认的 HTTP 客户端
var DefaultClient = &Client{}

// NewRequest 创建一个新的 HTTP 请求
func NewRequest(method, urlStr string, body io.Reader) (*Request, error) {
	return &Request{
		Method: method,
		URL:    &URL{raw: urlStr},
		Header: make(Header),
		Body:   body,
	}, nil
}

// Do 执行 HTTP 请求
func (c *Client) Do(req *Request) (*Response, error) {
	return c.DoContext(context.Background(), req)
}

// DoContext 执行带 context 的 HTTP 请求
func (c *Client) DoContext(ctx context.Context, req *Request) (*Response, error) {
	// 读取请求体
	var bodyBytes []byte
	if req.Body != nil {
		var err error
		bodyBytes, err = io.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}
	}

	// 转换请求头为 map[string]string
	headers := make(map[string]string)
	for key, values := range req.Header {
		if len(values) > 0 {
			headers[key] = values[0]
		}
	}

	// 计算超时时间（毫秒）
	var timeoutMs int64
	if c.Timeout > 0 {
		timeoutMs = c.Timeout.Milliseconds()
	}

	// 调用 HTTP Library 发起 HTTP 请求
	httpLibrary := export.NewHttpLibrary()
	resp, err := httpLibrary.DoRequest(ctx, &export.HttpRequest{
		Method:    req.Method,
		Url:       req.URL.String(),
		Headers:   headers,
		Body:      bodyBytes,
		TimeoutMs: timeoutMs,
	})
	if err != nil {
		return nil, err
	}

	// 如果请求失败，返回错误
	if !resp.Success {
		return nil, &RequestError{Message: resp.Error}
	}

	// 转换响应头为 Header 类型
	respHeader := make(Header)
	for key, value := range resp.Headers {
		respHeader[key] = []string{value}
	}

	return &Response{
		StatusCode: int(resp.StatusCode),
		Header:     respHeader,
		Body:       &bodyReadCloser{bytes.NewReader(resp.Body)},
	}, nil
}

// RequestError 表示 HTTP 请求错误
type RequestError struct {
	Message string
}

func (e *RequestError) Error() string {
	return e.Message
}

// Get 发起 GET 请求（使用默认客户端）
func Get(url string) (*Response, error) {
	return DefaultClient.Get(url)
}

// Get 发起 GET 请求
func (c *Client) Get(url string) (*Response, error) {
	req, err := NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	return c.Do(req)
}

// Post 发起 POST 请求（使用默认客户端）
func Post(url, contentType string, body io.Reader) (*Response, error) {
	return DefaultClient.Post(url, contentType, body)
}

// Post 发起 POST 请求
func (c *Client) Post(url, contentType string, body io.Reader) (*Response, error) {
	req, err := NewRequest("POST", url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", contentType)
	return c.Do(req)
}

// Head 发起 HEAD 请求（使用默认客户端）
func Head(url string) (*Response, error) {
	return DefaultClient.Head(url)
}

// Head 发起 HEAD 请求
func (c *Client) Head(url string) (*Response, error) {
	req, err := NewRequest("HEAD", url, nil)
	if err != nil {
		return nil, err
	}
	return c.Do(req)
}

// Do 使用默认客户端执行 HTTP 请求
func Do(req *Request) (*Response, error) {
	return DefaultClient.Do(req)
}
