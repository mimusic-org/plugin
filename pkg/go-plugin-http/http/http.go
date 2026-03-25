//go:build !wasip1

package http

import "net/http"

type Client = http.Client
type Request = http.Request
type Response = http.Response
type Cookie = http.Cookie

var NewRequest = http.NewRequest
var ErrUseLastResponse = http.ErrUseLastResponse

// CheckRedirectFunc 重定向检查函数类型
type CheckRedirectFunc = func(req *Request, via []*Request) error
