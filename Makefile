# 颜色输出
BLUE=\033[0;34m
GREEN=\033[0;32m
NC=\033[0m # No Color

.PHONY: help
help: ## 显示帮助信息
	@echo "$(BLUE)MiMusic Plugin - Makefile 命令$(NC)"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[0;32m%-20s\033[0m %s\n", $$1, $$2}'
	@echo ""

.PHONY: gen
gen: ## 生成 protobuf 文件
	./gen.sh

.PHONY: clean
clean: ## 清理所有构建产物
	rm -f api/pbplugin/*.pb.go
