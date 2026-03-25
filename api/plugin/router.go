//go:build wasip1
// +build wasip1

package plugin

import (
	"bufio"
	"bytes"
	"context"
	"log/slog"
	"net/http"

	"github.com/mimusic-org/plugin/api/pbplugin"
)

// RouterCallback 路由回调函数类型
type RouterCallback func(*http.Request) (*RouterResponse, error)

// RouterResponse 路由响应结构
type RouterResponse struct {
	StatusCode int
	Headers    map[string]string
	Body       []byte
}

// RouterManager 路由管理器
type RouterManager struct {
	routes   map[uint64]*Route
	nextID   uint64
	pluginId int64
}

// Route 路由结构
type Route struct {
	ID       uint64
	Method   string
	Pattern  string
	Callback RouterCallback
}

// NewRouterManager 创建新的路由管理器
func NewRouterManager() *RouterManager {
	return &RouterManager{
		routes: make(map[uint64]*Route),
	}
}

// SetPluginId 设置插件ID
func (rm *RouterManager) SetPluginId(pluginId int64) {
	rm.pluginId = pluginId
}

// RegisterRouter 注册路由
func (rm *RouterManager) RegisterRouter(ctx context.Context, method, pattern string, callback RouterCallback, requiresAuth ...bool) uint64 {
	rm.nextID++
	route := &Route{
		ID:       rm.nextID,
		Method:   method,
		Pattern:  pattern,
		Callback: callback,
	}
	rm.routes[route.ID] = route

	// 默认需要认证，除非明确指定为 false
	auth := true
	if len(requiresAuth) > 0 {
		auth = requiresAuth[0]
	}

	hostFunctions := pbplugin.NewHostFunctions()
	hostFunctions.RegisterRouter(ctx, &pbplugin.RegisterRouterRequest{
		Method:        method,
		Pattern:       pattern,
		HandlerFuncId: route.ID,
		PluginId:      rm.pluginId,
		RequiresAuth:  auth,
	})

	return route.ID
}

// GetFunc 获取路由回调函数
func (rm *RouterManager) GetFunc(routeID uint64) RouterCallback {
	route, exists := rm.routes[routeID]
	if !exists {
		return nil
	}
	return route.Callback
}

// OnRouterCallback 处理路由回调
func (rm *RouterManager) OnRouterCallback(ctx context.Context, request *pbplugin.OnRouterCallbackRequest) (*pbplugin.OnRouterCallbackResponse, error) {
	// slog.Info("OnRouterCallback executed", "request", request)

	// 从请求数据中恢复HTTP请求
	httpReq, err := http.ReadRequest(bufio.NewReader(bytes.NewReader(request.GetRequestData())))
	if err != nil {
		slog.Error("Failed to parse HTTP request", "error", err)
		return nil, err
	}

	callback := rm.GetFunc(request.GetHandlerFuncId())
	if callback != nil {
		resp, err := callback(httpReq)
		if err != nil {
			slog.Error("Router callback failed", "error", err)
			return nil, err
		}

		// 构造响应
		response := &pbplugin.OnRouterCallbackResponse{
			Success:    true,
			StatusCode: int32(resp.StatusCode),
			Headers:    resp.Headers,
			Body:       resp.Body,
		}

		// slog.Info("OnRouterCallback callback executed", "request", request)
		// 注意：不像定时器，路由不应该在这里删除，因为路由可能是被多次调用的
		// delete(rm.routes, request.GetHandlerFuncId())
		return response, nil
	}

	slog.Warn("No callback found for route", "routeID", request.GetHandlerFuncId())
	return &pbplugin.OnRouterCallbackResponse{
		Success: false,
		Message: "No callback found for route",
	}, nil
}
