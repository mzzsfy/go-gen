package router

import (
	"os"
	"testing"
)

// TestGen_NonexistentWorkDirPanics 验证不存在的 workDir 导致 panic
func TestGen_NonexistentWorkDirPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("不存在的 workDir 应 panic")
		}
	}()
	*WorkDir = "/nonexistent/path/xyz"
	*OutDir = t.TempDir()
	*ModuleName = "testmod"
	*RouterGroup = ""
	Gen(engineGen{Name: "gin"})
}

// TestGen_NonexistentEnginePanics 验证不存在的引擎名导致 panic
func TestGen_NonexistentEnginePanics(t *testing.T) {
	srcDir := createTempSrc(t, map[string]string{
		"go.mod": "module testmod\n\ngo 1.22\n",
		"api/h.go": `package api

// H
// @Router /x[GET]
func H() {}
`,
	})
	defer func() {
		if r := recover(); r == nil {
			t.Error("不存在的引擎名应 panic")
		}
	}()
	*WorkDir = srcDir
	*OutDir = t.TempDir()
	*ModuleName = "testmod"
	*RouterGroup = ""
	Gen(engineGen{Name: "nonexistent-engine"})
}

// TestGenRouterCore_BadEngineName 验证 GenRouterCore 直接调用时返回错误
func TestGenRouterCore_BadEngineName(t *testing.T) {
	e := engineGen{Name: "no-such-engine"}
	err := e.GenRouterCore(GlobalCtx{OutPath: t.TempDir()})
	if err == nil {
		t.Error("不存在的引擎名应返回错误")
	}
}

// TestGenRouterCore_OutDirIsFile 验证 GenRouterCore 输出路径为文件时返回错误
func TestGenRouterCore_OutDirIsFile(t *testing.T) {
	outDir := t.TempDir() + "/afile"
	if err := os.WriteFile(outDir, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	e := engineGen{Name: "gin"}
	err := e.GenRouterCore(GlobalCtx{OutPath: outDir})
	if err == nil {
		t.Error("输出路径为文件时应返回错误")
	}
}
