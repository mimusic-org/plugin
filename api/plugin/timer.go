//go:build wasip1
// +build wasip1

package plugin

import (
	"context"
	"log/slog"

	"github.com/mimusic-org/plugin/api/pbplugin"
	"github.com/knqyf263/go-plugin/types/known/emptypb"
)

// TimerCallback 定时器回调函数类型
type TimerCallback func()

// TimerManager 定时器管理器
type TimerManager struct {
	timers   map[uint64]*Timer
	nextID   uint64
	pluginId int64
}

// NewTimerManager 创建新的定时器管理器
func NewTimerManager() *TimerManager {
	return &TimerManager{
		timers: make(map[uint64]*Timer),
	}
}

func (tm *TimerManager) SetPluginId(pluginId int64) {
	tm.pluginId = pluginId
}

// Timer 定时器结构
type Timer struct {
	ID           uint64
	DelaySeconds int64
	Callback     TimerCallback
}

// RegisterDelayTimer 注册延迟定时器
func (tm *TimerManager) RegisterDelayTimer(ctx context.Context, delayMilliseconds int64, callback TimerCallback) uint64 {
	tm.nextID++
	timer := &Timer{
		ID:           tm.nextID,
		DelaySeconds: delayMilliseconds,
		Callback:     callback,
	}
	tm.timers[timer.ID] = timer

	hostFunctions := pbplugin.NewHostFunctions()
	hostFunctions.RegisterDelayTimer(ctx, &pbplugin.RegisterDelayTimerRequest{
		TimerId:           timer.ID,
		DelayMilliseconds: delayMilliseconds,
		PluginId:          tm.pluginId,
	})
	return timer.ID
}

// CancelTimer 取消已注册的定时器
func (tm *TimerManager) CancelTimer(ctx context.Context, timerID uint64) error {
	// 从本地映射中删除定时器
	_, exists := tm.timers[timerID]
	if !exists {
		return nil // 定时器不存在或已被执行
	}

	// 删除本地映射
	delete(tm.timers, timerID)

	// 调用主机函数取消定时器
	hostFunctions := pbplugin.NewHostFunctions()
	_, err := hostFunctions.CancelDelayTimer(ctx, &pbplugin.CancelTimerRequest{
		TimerId:  timerID,
		PluginId: tm.pluginId,
	})

	if err != nil {
		slog.Warn("Failed to cancel timer on host", "timerId", timerID, "error", err)
		return err
	}

	slog.Info("Timer canceled successfully", "timerId", timerID)
	return nil
}

// GetFunc 获取定时器回调函数
func (tm *TimerManager) GetFunc(timerID uint64) TimerCallback {
	return tm.timers[timerID].Callback
}

// OnTimerCallback 处理定时器回调
func (tm *TimerManager) OnTimerCallback(ctx context.Context, request *pbplugin.OnTimerCallbackRequest) (*emptypb.Empty, error) {
	slog.Info("OnTimerCallback executed", "request", request)
	callback := tm.GetFunc(request.TimerId)
	if callback != nil {
		callback()
	}
	slog.Info("OnTimerCallback callback executed", "request", request)
	delete(tm.timers, request.TimerId)
	return &emptypb.Empty{}, nil
}
