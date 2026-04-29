//go:build wasip1
// +build wasip1

package plugin

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// StaticHandler 静态文件处理器
//
// 新版职责（2026.04 改造）：将插件 embed.FS 中的 static/ 目录
// 同步到宿主机数据目录 /static/（对应主机 plugins_data/<entryPath>/static/）。
// 宿主侧 middleware 会自动探测该目录并直通服务静态请求，不再经过 WASM。
//
// 本类型保留仅为向后兼容 API 签名，实例本身不再承担任何运行时职责。
type StaticHandler struct{}

// fingerprintFile 存放上次 embed 内容的 sha256 摘要，用于增量判断是否需要重写
const fingerprintFile = "/static/.embed_fingerprint"

// NewStaticHandler 创建静态文件处理器并将 embed 的 static/ 目录同步到磁盘
//
// 同步策略（按指纹增量）：
//  1. 若 fsys 中无 static/ 目录或目录为空，什么都不做
//  2. 计算 embed 中 static/ 的 SHA256 指纹，与磁盘上旧指纹比对
//  3. 指纹相同：跳过同步（重启时常走此分支）
//  4. 指纹不同：os.RemoveAll("/static") 后全量重写，避免残留旧文件
//
// 参数 rm、ctx 保留仅为兼容旧签名，本函数不使用它们，不再注册任何 WASM 路由。
func NewStaticHandler(fsys fs.FS, rm *RouterManager, ctx context.Context) *StaticHandler {
	_ = rm
	_ = ctx

	// 1. 检测 embed 中是否有 static/ 目录
	entries, err := fs.ReadDir(fsys, "static")
	if err != nil || len(entries) == 0 {
		slog.Info("未检测到 embed 静态资源，跳过同步")
		return &StaticHandler{}
	}

	// 2. 计算 embed 内容指纹
	newFingerprint := computeEmbedFingerprint(fsys, "static")

	// 3. 对比磁盘上的旧指纹
	oldFingerprint, _ := os.ReadFile(fingerprintFile)
	if string(oldFingerprint) == newFingerprint {
		slog.Info("静态资源指纹未变化，跳过同步", "fingerprint", newFingerprint)
		return &StaticHandler{}
	}

	// 4. 指纹变化：清空旧目录并全量重写
	slog.Info("静态资源指纹变化，开始同步", "old", string(oldFingerprint), "new", newFingerprint)
	if err := os.RemoveAll("/static"); err != nil {
		slog.Warn("清理旧 static 目录失败", "error", err)
	}
	if err := os.MkdirAll("/static", 0755); err != nil {
		slog.Error("创建 static 目录失败", "error", err)
		return &StaticHandler{}
	}
	if err := syncEmbedToDisk(fsys, "static", "/static"); err != nil {
		slog.Error("同步 embed 到磁盘失败", "error", err)
		return &StaticHandler{}
	}

	// 5. 写入新指纹
	if err := os.WriteFile(fingerprintFile, []byte(newFingerprint), 0644); err != nil {
		slog.Warn("写入指纹文件失败", "error", err)
	}

	slog.Info("静态资源同步完成")
	return &StaticHandler{}
}

// computeEmbedFingerprint 按 walk 顺序对每个文件的路径和内容做 SHA256 摘要，
// 保证同一 embed 内容在任意运行实例上都算出相同指纹。
func computeEmbedFingerprint(fsys fs.FS, root string) string {
	h := sha256.New()
	_ = fs.WalkDir(fsys, root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		content, readErr := fs.ReadFile(fsys, p)
		if readErr != nil {
			return nil
		}
		h.Write([]byte(p))
		h.Write([]byte{0})
		h.Write(content)
		return nil
	})
	return hex.EncodeToString(h.Sum(nil))
}

// syncEmbedToDisk 递归将 embed.FS 中 srcRoot 下所有文件/目录复制到 destRoot。
// destRoot 在 WASM 视角下是绝对路径（如 "/static"），实际对应宿主机
// plugins_data/<entryPath>/static/ 目录（由 WASI preopen 挂载）。
func syncEmbedToDisk(fsys fs.FS, srcRoot, destRoot string) error {
	return fs.WalkDir(fsys, srcRoot, func(srcPath string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		// 计算相对路径：static/index.html -> /index.html
		rel := strings.TrimPrefix(srcPath, srcRoot)
		destPath := path.Join(destRoot, rel)
		if d.IsDir() {
			return os.MkdirAll(destPath, 0755)
		}
		content, err := fs.ReadFile(fsys, srcPath)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return err
		}
		return os.WriteFile(destPath, content, 0644)
	})
}
