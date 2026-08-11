package router

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"testing"
)

// TestResolveModuleName_Fallback 验证 ModuleName 为空时 fallback 到 findModuleName
func TestResolveModuleName_Fallback(t *testing.T) {
	dir := t.TempDir()
	goMod := "module myproject\n\ngo 1.22\n"
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatal(err)
	}
	// 保存和恢复 flag 状态
	orig := *ModuleName
	*ModuleName = ""
	t.Cleanup(func() { *ModuleName = orig })

	got := resolveModuleName(dir)
	if got != "myproject" {
		t.Errorf("fallback 模块名: 期望 myproject, 实际 %s", got)
	}
}

// TestResolveModuleName_FlagOverride 验证 ModuleName 非空时直接使用
func TestResolveModuleName_FlagOverride(t *testing.T) {
	orig := *ModuleName
	*ModuleName = "override-mod"
	t.Cleanup(func() { *ModuleName = orig })

	got := resolveModuleName("/any/path")
	if got != "override-mod" {
		t.Errorf("flag 覆盖: 期望 override-mod, 实际 %s", got)
	}
}

// TestParseDir_WithFilter 验证 filter 参数生效
func TestParseDir_WithFilter(t *testing.T) {
	srcDir := createTempSrc(t, map[string]string{
		"go.mod": "module testmod\n\ngo 1.22\n",
		"a.go":   "package pkg\n// FuncA\nfunc FuncA() {}\n",
		"b.go":   "package pkg\n// FuncB\nfunc FuncB() {}\n",
	})

	// filter 只允许 a.go
	pkgs, err := ParseDir(token.NewFileSet(), srcDir, func(fi os.FileInfo) bool {
		return fi.Name() == "a.go"
	}, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}

	pkg, ok := pkgs["pkg"]
	if !ok {
		t.Fatal("未找到 pkg 包")
	}
	if len(pkg.Files) != 1 {
		t.Errorf("filter 后文件数: 期望 1, 实际 %d", len(pkg.Files))
	}
	for fname := range pkg.Files {
		if filepath.Base(fname) != "a.go" {
			t.Errorf("filter 后应只有 a.go, 实际包含 %s", filepath.Base(fname))
		}
	}
}

// TestParseDir_NonexistentDir 验证不存在的目录返回错误
func TestParseDir_NonexistentDir(t *testing.T) {
	_, err := ParseDir(token.NewFileSet(), "/nonexistent/path/xyz", nil, 0)
	if err == nil {
		t.Error("不存在的目录应返回错误")
	}
}

// TestParseDir_MultipleFilesSamePackage 验证同包多文件合并
func TestParseDir_MultipleFilesSamePackage(t *testing.T) {
	srcDir := createTempSrc(t, map[string]string{
		"go.mod": "module testmod\n\ngo 1.22\n",
		"a.go":   "package pkg\nfunc A() {}\n",
		"b.go":   "package pkg\nfunc B() {}\n",
	})

	pkgs, err := ParseDir(token.NewFileSet(), srcDir, nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	pkg, ok := pkgs["pkg"]
	if !ok {
		t.Fatal("未找到 pkg 包")
	}
	if len(pkg.Files) != 2 {
		t.Errorf("同包多文件应合并: 期望 2 文件, 实际 %d", len(pkg.Files))
	}
}

// TestParseDir_ParseErrorHandling 验证语法错误的文件被跳过, 不阻止其他文件解析
func TestParseDir_ParseErrorHandling(t *testing.T) {
	srcDir := createTempSrc(t, map[string]string{
		"go.mod":  "module testmod\n\ngo 1.22\n",
		"bad.go":  "package pkg\nthis is bad go",
		"good.go": "package pkg\ntype Good struct{}\n",
	})

	pkgs, _ := ParseDir(token.NewFileSet(), srcDir, nil, 0)
	pkg, ok := pkgs["pkg"]
	if !ok {
		t.Fatal("应有 pkg 包 (含坏文件但不阻止解析)")
	}
	if len(pkg.Files) != 1 {
		t.Errorf("坏文件应跳过, 好文件应解析: 期望 1 文件, 实际 %d", len(pkg.Files))
	}
}
