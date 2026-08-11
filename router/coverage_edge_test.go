package router

import (
	"go/token"
	"os"
	"strings"
	"testing"
)

// TestParseDir_SubdirParseErrorPropagation 验证子目录解析错误被传播
func TestParseDir_SubdirParseErrorPropagation(t *testing.T) {
	srcDir := createTempSrc(t, map[string]string{
		"go.mod":       "module testmod\n\ngo 1.22\n",
		"pkg/bad.go":   "package pkg\nthis is bad",
		"pkg/good.go":  "package pkg\ntype Good struct{}\n",
	})

	pkgs, first := ParseDir(token.NewFileSet(), srcDir, nil, 0)
	// 子目录的解析错误应被传播到 first
	if first == nil {
		t.Error("子目录解析错误应被传播到 first error")
	}
	// 好文件仍应被解析
	pkg := findPkgBySuffix(t, pkgs, "/pkg")
	if len(pkg.Files) != 1 {
		t.Errorf("好文件应被解析: 期望 1 文件, 实际 %d", len(pkg.Files))
	}
}

// TestGen_ModuleNameWithSlash 验证 moduleName 含 / 时 pkgName 正确切分
func TestGen_ModuleNameWithSlash(t *testing.T) {
	srcDir := createTempSrc(t, map[string]string{
		"go.mod": "module github.com/org/project\n\ngo 1.22\n",
		"api/handler.go": `package api

// Handler
// @Router /test[GET]
func Handler() {}
`,
	})
	outDir := t.TempDir()
	genWithFlags(t, srcDir, outDir, "github.com/org/project", "")

	// import 文件中 package name 应为最后一段 "project"
	importFile := outDir + "/0_import___.go"
	bs, err := os.ReadFile(importFile)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(bs), "package project") {
		t.Errorf("含 / 的 moduleName 应取最后一段作为 pkgName, 内容:\n%s", string(bs))
	}
}
