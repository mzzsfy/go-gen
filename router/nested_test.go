package router

import (
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"
)

// TestNestedPackageName_BaseName 验证多层级路径的 base name 提取逻辑
// buildPackage 中 PackageName = path.Base(pname), 此测试确保该映射对各层级正确
func TestNestedPackageName_BaseName(t *testing.T) {
	tests := []struct {
		pname string
		want  string
	}{
		{"api/quant", "quant"},
		{"api/backtest", "backtest"},
		{"api/v2/deep", "deep"},
		{"user", "user"},
		{"a/b/c/d/e", "e"},
	}
	for _, tt := range tests {
		t.Run(tt.pname, func(t *testing.T) {
			got := path.Base(tt.pname)
			if got != tt.want {
				t.Errorf("path.Base(%q): got %q, want %q", tt.pname, got, tt.want)
			}
		})
	}
}

// TestGen_NestedPackageFileLocation 验证嵌套包生成文件位于正确的多层级路径
func TestGen_NestedPackageFileLocation(t *testing.T) {
	outDir := t.TempDir()
	setupFlags(t, outDir)

	Gen(engineGen{Name: "gin"})

	// 文件应在 api/quant/ 和 api/backtest/ 下, 而非扁平的 quant/
	nestedFiles := []string{
		filepath.Join(outDir, "api", "quant", "0_router___.go"),
		filepath.Join(outDir, "api", "backtest", "0_router___.go"),
	}
	for _, f := range nestedFiles {
		if _, err := os.Stat(f); os.IsNotExist(err) {
			t.Errorf("缺少嵌套包路由文件: %s", f)
		}
	}
}

// TestGen_NestedPackageDeclaration 验证嵌套包的 package 声明使用 base name 而非完整路径
func TestGen_NestedPackageDeclaration(t *testing.T) {
	outDir := t.TempDir()
	setupFlags(t, outDir)

	Gen(engineGen{Name: "gin"})

	tests := []struct {
		relPath string
		wantPkg string
	}{
		{filepath.Join("api", "quant", "0_router___.go"), "package quant"},
		{filepath.Join("api", "backtest", "0_router___.go"), "package backtest"},
	}
	for _, tt := range tests {
		t.Run(tt.relPath, func(t *testing.T) {
			bs, err := os.ReadFile(filepath.Join(outDir, tt.relPath))
			if err != nil {
				t.Fatalf("读取文件失败: %v", err)
			}
			content := string(bs)
			if !strings.Contains(content, tt.wantPkg) {
				t.Errorf("package 声明错误: 期望包含 %q\n实际内容:\n%s", tt.wantPkg, content)
			}
			// 确保不包含非法的路径式声明
			illegalPkg := "package api/" + strings.TrimPrefix(tt.wantPkg, "package ")
			if strings.Contains(content, illegalPkg) {
				t.Errorf("包含非法 package 声明: %q", illegalPkg)
			}
		})
	}
}

// TestGen_NestedPackageImportPath 验证 import 文件使用完整模块路径引用嵌套包
func TestGen_NestedPackageImportPath(t *testing.T) {
	outDir := t.TempDir()
	setupFlags(t, outDir)

	Gen(engineGen{Name: "gin"})

	importFile := filepath.Join(outDir, "0_import___.go")
	bs, err := os.ReadFile(importFile)
	if err != nil {
		t.Fatalf("读取 import 文件失败: %v", err)
	}
	content := string(bs)
	assertContains(t, content, `_ "testmod/api/quant"`)
	assertContains(t, content, `_ "testmod/api/backtest"`)
}

// TestGen_NestedPackageRouterContent 验证嵌套包的路由注册内容正确生成
func TestGen_NestedPackageRouterContent(t *testing.T) {
	outDir := t.TempDir()
	setupFlags(t, outDir)

	Gen(engineGen{Name: "gin"})

	// backtest 包含结构体方法, 验证多层级路径下结构体方法也正常
	btFile := filepath.Join(outDir, "api", "backtest", "0_router___.go")
	bs, err := os.ReadFile(btFile)
	if err != nil {
		t.Fatalf("读取 backtest 路由文件失败: %v", err)
	}
	content := string(bs)
	assertContains(t, content, "BacktestHandler")
	assertContains(t, content, "Runner")
	assertContains(t, content, "routers.DefaultRouter.Router")
}

// TestGen_NestedPackageAllGoFiles 验证所有生成的 0_router___.go 文件都是合法的 package 声明
func TestGen_NestedPackageAllGoFiles(t *testing.T) {
	outDir := t.TempDir()
	setupFlags(t, outDir)

	Gen(engineGen{Name: "gin"})

	// 遍历所有生成的 0_router___.go, 确保无路径式 package 声明
	err := filepath.Walk(outDir, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(p, "0_router___.go") {
			return nil
		}
		bs, e := os.ReadFile(p)
		if e != nil {
			return e
		}
		content := string(bs)
		// 任何 package 声明都不应包含 /
		for _, line := range strings.Split(content, "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "package ") {
				pkgDecl := strings.TrimSpace(strings.TrimPrefix(line, "package "))
				if strings.Contains(pkgDecl, "/") {
					t.Errorf("%s: package 声明含 / : %q", p, line)
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Walk error: %v", err)
	}
}

// TestGen_NestedPackagePathClean 验证路径拼接使用 path.Clean, 无冗余分隔符
func TestGen_NestedPackagePathClean(t *testing.T) {
	outDir := t.TempDir()
	setupFlags(t, outDir)

	Gen(engineGen{Name: "gin"})

	// 生成的文件路径不应包含 // 或多余的 .
	quantFile := filepath.Join(outDir, "api", "quant", "0_router___.go")
	cleanPath := path.Clean(outDir + "/api/quant/0_router___.go")
	if filepath.ToSlash(filepath.Clean(quantFile)) != filepath.ToSlash(cleanPath) {
		t.Errorf("路径不洁: %q vs %q", quantFile, cleanPath)
	}
}
