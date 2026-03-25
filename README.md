# MiMusic 插件协议库

这个仓库包含了 [MiMusic](https://github.com/mimusic-org/mimusic) 的插件协议和示例插件，[MiMusic](https://github.com/mimusic-org/mimusic) 是一个用 Go 构建的轻量级音乐服务器。
## 概述

MiMusic 支持通过基于 WASM 的插件扩展其功能。这个仓库提供了：

1. 插件协议定义（Protocol Buffers）
2. 插件开发基础API
3. 示例实现，展示如何创建自定义插件
4. 插件开发文档

## 插件协议

插件协议使用 Protocol Buffers 在 [`plugin.proto`](api/pbplugin/plugin.proto) 中定义。它定义了所有插件必须实现的接口。

插件系统的主要特性：
- 设备认证（支持多种认证方式）
- 远程设备音乐播放控制
- 基于 RPC 的可扩展通信
- 定时器管理
- 路由管理

## 插件API

插件开发基础API位于 [`api/plugin`](api/plugin) 目录中，提供了以下核心功能：

### 基础插件服务
- `BasePlugin`: 所有插件的基类，处理插件生命周期管理
- `RegisterPlugin`: 注册具体插件实现
- `GetPluginId`: 获取插件唯一标识符

### 定时器管理
- `TimerManager`: 管理插件的定时器功能
- `RegisterDelayTimer`: 注册延迟定时器
- `TimerCallback`: 定时器回调函数类型

### 路由管理
- `RouterManager`: 管理插件的路由功能
- `RegisterRouter`: 注册HTTP路由
- `RouterCallback`: 路由回调函数类型

## 代码生成

使用 [`gen.sh`](gen.sh) 脚本从 `.proto` 文件生成 Go 代码：

```bash
./gen.sh
```

## 示例

示例插件可以在单独的仓库中找到：

- [mimusic-plugin-example](https://github.com/mimusic-org/mimusic-plugin-example)：一个完整的示例，展示如何实现各种插件功能

## 开始使用

有关开发自己的插件的详细说明，请参阅 [mimusic-plugin-example](https://github.com/mimusic-org/mimusic-plugin-example) 仓库。

## License

本项目基于 [Apache License 2.0](LICENSE) 开源协议发布。