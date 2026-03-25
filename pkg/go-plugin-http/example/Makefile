# go-plugin-http 示例 Makefile
# 用于编译 host 端程序和 WASM 插件

.PHONY: all clean host plugin gen run help

# 输出目录
BUILD_DIR := build

# 默认目标：编译所有
all: gen plugin host
	@echo "编译完成！"
	@echo "  Host 程序: $(BUILD_DIR)/host"
	@echo "  WASM 插件: $(BUILD_DIR)/plugin.wasm"

# 创建输出目录
$(BUILD_DIR):
	mkdir -p $(BUILD_DIR)

# 生成 proto 代码
gen:
	@echo "生成 proto 代码..."
	./gen.sh

# 编译 host 端程序
host: $(BUILD_DIR)
	@echo "编译 Host 程序..."
	go build -o $(BUILD_DIR)/host ./host/main.go

# 编译 WASM 插件
# 使用 -buildmode=c-shared 构建 WASI reactor，生成 _initialize 函数
plugin: $(BUILD_DIR)
	@echo "编译 WASM 插件..."
	GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -o $(BUILD_DIR)/plugin.wasm ./plugin/main.go

# 运行示例
run: all
	@echo "运行 Host 程序..."
	cd host && ../$(BUILD_DIR)/host

# 清理编译产物
clean:
	@echo "清理编译产物..."
	rm -rf $(BUILD_DIR)

# 帮助信息
help:
	@echo "可用目标："
	@echo "  make all     - 生成代码并编译 host 程序和 WASM 插件（默认）"
	@echo "  make gen     - 生成 proto 代码"
	@echo "  make host    - 仅编译 host 程序"
	@echo "  make plugin  - 仅编译 WASM 插件"
	@echo "  make run     - 编译并运行示例"
	@echo "  make clean   - 清理编译产物"
	@echo "  make help    - 显示帮助信息"
