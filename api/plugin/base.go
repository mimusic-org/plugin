//go:build wasip1
// +build wasip1

// Package plugin 提供了创建基于 WASM 的插件的基础。
// 它定义了核心插件服务接口和所有插件必须继承的基础实现。
// 它还管理诸如定时器和插件生命周期事件等通用功能。
package plugin

import (
	"context"
	"log/slog"

	"github.com/mimusic-org/plugin/api/pbplugin"
	"github.com/knqyf263/go-plugin/types/known/emptypb"
)

// PluginService 定义了所有插件必须实现的接口。
// 它包含宿主应用程序调用的三个核心生命周期方法：
//   - GetPluginInfo: 返回插件的元数据
//   - Init: 在插件加载时调用
//   - Deinit: 在插件卸载时调用
type PluginService interface {
	// GetPluginInfo 检索插件的元数据，包括名称、版本、描述、作者和主页。
	GetPluginInfo(ctx context.Context, request *emptypb.Empty) (*pbplugin.GetPluginInfoResponse, error)

	// Init 使用包含插件ID的请求初始化插件。
	// 插件在此处执行其设置操作。
	Init(ctx context.Context, request *pbplugin.InitRequest) (*emptypb.Empty, error)

	// Deinit 在插件被卸载时清理资源。
	// 插件应在此处释放初始化期间获取的任何资源。
	Deinit(ctx context.Context, request *emptypb.Empty) (*emptypb.Empty, error)
}

// BasePlugin 封装了所有插件共享的通用功能。
// 它实现了 PluginService 接口，并将具体的业务逻辑委托给通过 RegisterPlugin 注册的具体插件实现。
type BasePlugin struct {
	// timerManager 处理插件的定时器相关功能
	timerManager *TimerManager

	// routerManager 处理插件的路由相关功能
	routerManager *RouterManager

	// pluginId 在宿主应用程序中唯一标识此插件实例
	pluginId int64

	// impl 是提供业务逻辑的具体插件实现
	impl PluginService
}

// basePlugin 是在整个应用程序中使用的 BasePlugin 单例实例
var basePlugin *BasePlugin

// init 创建具有新 TimerManager 的单例 BasePlugin 实例。
// 此函数在导入包时自动运行。
func init() {
	basePlugin = &BasePlugin{
		timerManager:  NewTimerManager(),
		routerManager: NewRouterManager(),
	}
}

// RegisterPlugin 将具体插件实现注册到插件框架中。
// 每个插件在初始化期间必须调用此函数以使自身对宿主应用程序可用。
func RegisterPlugin(impl PluginService) {
	basePlugin.impl = impl
	pbplugin.RegisterPluginService(basePlugin)
}

// GetPluginInfo 将获取插件元数据的调用委托给具体实现。
// 这允许每个插件提供其自己的特定信息。
func (bp *BasePlugin) GetPluginInfo(ctx context.Context, request *emptypb.Empty) (*pbplugin.GetPluginInfoResponse, error) {
	return bp.impl.GetPluginInfo(ctx, request)
}

// Init 在委托给具体实现之前执行基本插件初始化。
// 它设置插件ID并配置定时器管理器，然后调用插件的Init方法。
func (bp *BasePlugin) Init(ctx context.Context, request *pbplugin.InitRequest) (*emptypb.Empty, error) {
	// BasePlugin 的通用初始化逻辑
	bp.pluginId = request.GetPluginId()
	bp.timerManager.SetPluginId(bp.pluginId)
	bp.routerManager.SetPluginId(bp.pluginId)

	slog.Info("正在初始化插件", "pluginId", bp.pluginId)

	// 委托给具体插件实现
	return bp.impl.Init(ctx, request)
}

// Deinit 在委托给具体实现之前清理基本插件资源。
// 它调用插件的Deinit方法以允许自定义清理逻辑。
func (bp *BasePlugin) Deinit(ctx context.Context, request *emptypb.Empty) (*emptypb.Empty, error) {
	// 委托给具体插件实现
	return bp.impl.Deinit(ctx, request)
}

// OnTimerCallback 通过委托给定时器管理器来处理定时器事件。
// 当预定的定时器到期时，宿主应用程序会调用此方法。
func (bp *BasePlugin) OnTimerCallback(ctx context.Context, request *pbplugin.OnTimerCallbackRequest) (*emptypb.Empty, error) {
	// slog.Info("OnTimerCallback", "request", request)
	return bp.timerManager.OnTimerCallback(ctx, request)
}

// OnRouterCallback 通过委托给路由管理器来处理路由事件。
// 当注册的路由被访问时，宿主应用程序会调用此方法。
func (bp *BasePlugin) OnRouterCallback(ctx context.Context, request *pbplugin.OnRouterCallbackRequest) (*pbplugin.OnRouterCallbackResponse, error) {
	// slog.Info("OnRouterCallback", "request", request)
	return bp.routerManager.OnRouterCallback(ctx, request)
}

// GetTimerManager 返回供插件实现使用的定时器管理器实例。
// 插件可以使用此方法安排延迟回调或周期性定时器。
func GetTimerManager() *TimerManager {
	return basePlugin.timerManager
}

// GetRouterManager 返回供插件实现使用的路由管理器实例。
// 插件可以使用此方法注册路由。
func GetRouterManager() *RouterManager {
	return basePlugin.routerManager
}

// GetPluginId 返回分配给此插件实例的唯一标识符。
// 插件可以使用此ID将自己与其他实例区分开来。
func GetPluginId() int64 {
	return basePlugin.pluginId
}
