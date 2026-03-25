//go:build wasip1
// +build wasip1

package plugin

import (
	"encoding/json"
)

// JSONResponse 创建 JSON 格式的 HTTP 响应
func JSONResponse(statusCode int, data interface{}) *RouterResponse {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return &RouterResponse{
			StatusCode: 500,
			Headers:    map[string]string{"Content-Type": "application/json"},
			Body:       []byte(`{"success":false,"error":"json marshal error"}`),
		}
	}
	return &RouterResponse{
		StatusCode: statusCode,
		Headers:    map[string]string{"Content-Type": "application/json"},
		Body:       jsonData,
	}
}

// ErrorResponse 创建错误响应
func ErrorResponse(statusCode int, message string) *RouterResponse {
	return JSONResponse(statusCode, map[string]interface{}{
		"success": false,
		"error":   message,
	})
}

// SuccessResponse 创建成功响应
func SuccessResponse(data interface{}) *RouterResponse {
	return JSONResponse(200, map[string]interface{}{
		"success": true,
		"data":    data,
	})
}
