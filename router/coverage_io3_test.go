package router

import (
	"os"
	"path"
	"testing"
)

// TestMustWriteCoreFiles_WriteFileFails 验证目标路径为目录时 WriteFile panic
func TestMustWriteCoreFiles_WriteFileFails(t *testing.T) {
	outDir := t.TempDir()
	routersDir := path.Clean(outDir + "/routers")
	os.MkdirAll(routersDir, 0755)
	// 预创建一个目录占位目标文件路径
	os.MkdirAll(path.Clean(routersDir+"/0_bind___.go"), 0755)

	defer func() {
		if r := recover(); r == nil {
			t.Error("目标路径为目录时应 panic")
		}
	}()
	mustWriteCoreFiles(outDir)
}

// TestMustWriteRouterFiles_WriteFileFails 验证目标路径为目录时 WriteFile panic
func TestMustWriteRouterFiles_WriteFileFails(t *testing.T) {
	outDir := t.TempDir()
	pkgDir := path.Clean(outDir + "/pkg")
	os.MkdirAll(pkgDir, 0755)
	// 预创建目录占位
	os.MkdirAll(path.Clean(pkgDir+"/0_router___.go"), 0755)

	defer func() {
		if r := recover(); r == nil {
			t.Error("目标路径为目录时应 panic")
		}
	}()
	mustWriteRouterFiles(outDir, []*Package{{
		PackagePathName: "pkg",
	}})
}

// TestMustWriteRouterFiles_MkdirAllFails 验证 outDir 为文件时 MkdirAll panic
func TestMustWriteRouterFiles_MkdirAllFails(t *testing.T) {
	outDir := t.TempDir() + "/afile"
	os.WriteFile(outDir, []byte("x"), 0644)

	defer func() {
		if r := recover(); r == nil {
			t.Error("outDir 为文件时 MkdirAll 应 panic")
		}
	}()
	mustWriteRouterFiles(outDir, []*Package{{
		PackagePathName: "pkg",
	}})
}

// TestMustWriteImportFile_WriteFileFails 验证 outDir 不存在时 WriteFile panic
func TestMustWriteImportFile_WriteFileFails(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("outDir 不存在时应 panic")
		}
	}()
	mustWriteImportFile("/nonexistent/path/xyz", GlobalCtx{
		PackageName:     "testmod",
		PackageBaseName: "testmod",
	})
}

// TestGenRouterCore_WriteFileFails 验证目标路径为目录时 WriteFile 返回错误
func TestGenRouterCore_WriteFileFails(t *testing.T) {
	outDir := t.TempDir()
	routersDir := path.Clean(outDir + "/routers")
	os.MkdirAll(routersDir, 0755)
	// 预创建目录占位引擎输出文件
	os.MkdirAll(path.Clean(routersDir+"/0_bind_gin___.go"), 0755)

	e := engineGen{Name: "gin"}
	err := e.GenRouterCore(GlobalCtx{OutPath: outDir})
	if err == nil {
		t.Error("目标路径为目录时应返回错误")
	}
}
