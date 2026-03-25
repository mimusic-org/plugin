//go:build wasip1
// +build wasip1

package plugin

import (
	"bytes"
	"context"
	"io/fs"
	"log/slog"
	"net/http"
	"path"
	"strings"
)

// StaticHandler 静态文件处理器
// 在初始化时预加载所有静态文件内容到内存，避免每次请求都读取 embed.FS
type StaticHandler struct {
	// fileCache 预加载的静态文件内容缓存，key 为请求路径（如 /xiaomi/static/css/style.css）
	fileCache map[string]*RouterResponse
	// basePath 插件的基础路径，如 /xiaomi
	basePath string
}

// getHeadersForFile 根据文件路径返回对应的 Content-Type headers
func getHeadersForFile(filePath string) map[string]string {
	if strings.HasSuffix(filePath, ".css") {
		return map[string]string{"Content-Type": "text/css; charset=utf-8"}
	}
	if strings.HasSuffix(filePath, ".js") {
		return map[string]string{"Content-Type": "application/javascript; charset=utf-8"}
	}
	if strings.HasSuffix(filePath, ".json") {
		return map[string]string{"Content-Type": "application/json; charset=utf-8"}
	}
	if strings.HasSuffix(filePath, ".woff") {
		return map[string]string{"Content-Type": "font/woff"}
	}
	if strings.HasSuffix(filePath, ".woff2") {
		return map[string]string{"Content-Type": "font/woff2"}
	}
	if strings.HasSuffix(filePath, ".svg") {
		return map[string]string{"Content-Type": "image/svg+xml"}
	}
	if strings.HasSuffix(filePath, ".png") {
		return map[string]string{"Content-Type": "image/png"}
	}
	if strings.HasSuffix(filePath, ".jpg") || strings.HasSuffix(filePath, ".jpeg") {
		return map[string]string{"Content-Type": "image/jpeg"}
	}
	if strings.HasSuffix(filePath, ".gif") {
		return map[string]string{"Content-Type": "image/gif"}
	}
	return map[string]string{"Content-Type": "text/html; charset=utf-8"}
}

// NewStaticHandler 创建新的静态文件处理器，自动遍历 static 目录并注册所有路由
// fsys: 静态文件系统，用于遍历目录和读取文件
// basePath: 插件的基础路径，如 /xiaomi
// rm: 路由管理器
// ctx: 上下文
func NewStaticHandler(fsys fs.FS, basePath string, rm *RouterManager, ctx context.Context) *StaticHandler {
	cache := make(map[string]*RouterResponse)

	// 递归遍历 static 目录并注册所有文件
	var walkDir func(currentDir string) error
	walkDir = func(currentDir string) error {
		entries, err := fs.ReadDir(fsys, currentDir)
		if err != nil {
			return err
		}

		for _, entry := range entries {
			fullPath := path.Join(currentDir, entry.Name())

			if entry.IsDir() {
				// 递归处理子目录
				if err := walkDir(fullPath); err != nil {
					slog.Warn("遍历子目录失败", "path", fullPath, "error", err)
				}
				continue
			}

			// 读取文件内容
			content, err := fs.ReadFile(fsys, fullPath)
			if err != nil {
				slog.Warn("读取静态文件失败", "path", fullPath, "error", err)
				continue
			}

			// 对 HTML 文件自动注入 auth-bridge 脚本，
			// 使插件页面能从 URL query parameter 读取 Flutter 传递的 access_token
			if strings.HasSuffix(fullPath, ".html") {
				content = injectAuthBridge(content)
			}

			// 生成路由路径
			var routePath string
			if fullPath == "static/index.html" {
				// 根目录的 index.html 映射到 basePath
				routePath = basePath
			} else {
				// 其他文件：basePath + 完整文件路径
				routePath = basePath + "/" + fullPath
			}

			// 缓存文件内容
			cache[routePath] = &RouterResponse{
				StatusCode: 200,
				Headers:    getHeadersForFile(fullPath),
				Body:       content,
			}

			// 注册路由
			rm.RegisterRouter(ctx, "GET", routePath, func(req *http.Request) (*RouterResponse, error) {
				if resp, exists := cache[req.URL.Path]; exists {
					return resp, nil
				}
				// 处理路径前缀
				idx := strings.Index(req.URL.Path, basePath)
				if idx >= 0 {
					shortPath := req.URL.Path[idx:]
					if resp, exists := cache[shortPath]; exists {
						return resp, nil
					}
				}
				return &RouterResponse{
					StatusCode: 404,
					Headers:    map[string]string{"Content-Type": "text/plain; charset=utf-8"},
					Body:       []byte("file not found"),
				}, nil
			}, false)

			slog.Info("静态文件路由已注册", "route", routePath, "file", fullPath)
		}
		return nil
	}

	// 开始遍历 static 目录
	if err := walkDir("static"); err != nil {
		slog.Warn("遍历静态文件目录失败", "error", err)
	}

	slog.Info("静态文件处理器初始化完成", "cached_files", len(cache), "base_path", basePath)

	return &StaticHandler{
		fileCache: cache,
		basePath:  basePath,
	}
}

// HandleRequest 处理所有静态资源请求（HTML 页面和 CSS, JS 等）
func (h *StaticHandler) HandleRequest(req *http.Request) (*RouterResponse, error) {
	path := req.URL.Path

	// 直接从缓存中查找
	if resp, exists := h.fileCache[path]; exists {
		return resp, nil
	}

	// 路径中可能包含 /api/v1/plugin 前缀，尝试提取 basePath 部分
	idx := strings.Index(path, h.basePath)
	slog.Info("静态文件处理器", "path", path, "idx", idx, "base_path", h.basePath)
	if idx >= 0 {
		shortPath := path[idx:]
		if resp, exists := h.fileCache[shortPath]; exists {
			return resp, nil
		}
	}

	return &RouterResponse{
		StatusCode: 404,
		Headers:    map[string]string{"Content-Type": "text/plain; charset=utf-8"},
		Body:       []byte("file not found"),
	}, nil
}

// authBridgeScript 从 URL query parameter 读取 access_token 并存入 localStorage，
// 使 Flutter 前端通过 ?access_token=xxx 传递的 token 可被插件 JS 的 getAuthToken() 读取。
// 脚本执行后会通过 history.replaceState 清理 URL 中的 token 参数。
var authBridgeScript = []byte(`<script>(function(){var p=new URLSearchParams(window.location.search);var t=p.get("access_token");if(t){localStorage.setItem("mimusic-auth",JSON.stringify({accessToken:t}));p.delete("access_token");var u=window.location.pathname;var r=p.toString();if(r)u+="?"+r;history.replaceState(null,"",u)}})();</script>`)

// injectAuthBridge 在 HTML 内容的 </head> 标签前注入 auth-bridge 脚本，
// 确保脚本在页面其他 JS 之前执行，完成 token 的 localStorage 写入。
func injectAuthBridge(html []byte) []byte {
	idx := bytes.Index(html, []byte("</head>"))
	if idx == -1 {
		// 无 </head> 标签时在文件开头注入
		return append(authBridgeScript, html...)
	}
	result := make([]byte, 0, len(html)+len(authBridgeScript))
	result = append(result, html[:idx]...)
	result = append(result, authBridgeScript...)
	result = append(result, html[idx:]...)
	return result
}
