package router

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestGen_OutDirDefault 验证 OutDir 未显式设置时默认使用 WorkDir
func TestGen_OutDirDefault(t *testing.T) {
	srcDir := createTempSrc(t, map[string]string{
		"go.mod": "module testmod\n\ngo 1.22\n",
		"api/handler.go": `package api

// Handler
// @Router /test[GET]
func Handler() {}
`,
	})

	// 不设 OutDir, 只设 WorkDir 为多字符路径
	origOutDir := *OutDir
	*WorkDir = srcDir
	*OutDir = "./"
	*ModuleName = "testmod"
	*RouterGroup = ""
	t.Cleanup(func() { *OutDir = origOutDir })

	// Gen 应自动将 OutDir 设为 WorkDir
	Gen(engineGen{Name: "gin"})

	expectedOutDir := strings.ReplaceAll(srcDir, "\\", "/")
	routerFile := filepath.Join(expectedOutDir, "api", "0_router___.go")
	if _, err := os.Stat(routerFile); os.IsNotExist(err) {
		t.Errorf("OutDir 默认应等于 WorkDir, 缺少文件: %s", routerFile)
	}
}

// TestGen_NonRouterFunctionIgnored 验证无 @Router 注释的函数被跳过
func TestGen_NonRouterFunctionIgnored(t *testing.T) {
	srcDir := createTempSrc(t, map[string]string{
		"go.mod": "module testmod\n\ngo 1.22\n",
		"api/handler.go": `package api

// Handler 有路由
// @Router /routed[GET]
func Handler() {}

// Helper 无路由注释但有文档
func Helper() {}

func bareFunc() {}
`,
	})
	outDir := t.TempDir()
	genWithFlags(t, srcDir, outDir, "testmod", "")

	bs, _ := os.ReadFile(filepath.Join(srcDir, "api", "0_router___.go"))
	content := string(bs)
	if !strings.Contains(content, "Handler") {
		t.Error("有 @Router 的 Handler 应被注册")
	}
	if strings.Contains(content, "Helper") {
		t.Error("无 @Router 的 Helper 不应被注册")
	}
	if strings.Contains(content, "bareFunc") {
		t.Error("无文档的 bareFunc 不应被注册")
	}
}

// TestGen_OutDirIsFilePanics 验证 OutDir 指向文件而非目录时 panic
func TestGen_OutDirIsFilePanics(t *testing.T) {
	srcDir := createTempSrc(t, map[string]string{
		"go.mod": "module testmod\n\ngo 1.22\n",
		"api/handler.go": `package api

// Handler
// @Router /test[GET]
func Handler() {}
`,
	})
	// 创建一个文件作为 outDir (不是目录)
	outDir := filepath.Join(t.TempDir(), "notadir")
	os.WriteFile(outDir, []byte("x"), 0644)

	defer func() {
		if r := recover(); r == nil {
			t.Error("outDir 是文件时应 panic")
		}
	}()
	*WorkDir = srcDir
	*OutDir = outDir
	*ModuleName = "testmod"
	*RouterGroup = ""
	Gen(engineGen{Name: "gin"})
}
