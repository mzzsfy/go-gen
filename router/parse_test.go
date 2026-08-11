package router

import (
	"go/token"
	"path"
	"path/filepath"
	"strings"
	"testing"
)

func testdataDir() string {
	abs, _ := filepath.Abs("testdata/src")
	return path.Clean(strings.ReplaceAll(abs, "\\", "/"))
}

func TestParseDir_BasicPackageParsing(t *testing.T) {
	pkgs, err := ParseDir(token.NewFileSet(), testdataDir(), nil, 0)
	if err != nil {
		t.Fatalf("ParseDir error: %v", err)
	}
	// user, admin, api/quant, api/backtest
	if len(pkgs) != 4 {
		t.Fatalf("expected 4 packages, got %d", len(pkgs))
	}
	// ParseDir 返回的 key 是相对路径, 验证 user 包存在
	found := false
	for k, v := range pkgs {
		if strings.HasSuffix(k, "/user") && v.Name == "user" {
			found = true
		}
	}
	if !found {
		t.Fatalf("未找到 user 包")
	}
}

func TestFindModuleName(t *testing.T) {
	name := findModuleName(testdataDir())
	if name != "testmod" {
		t.Errorf("expected 'testmod', got '%s'", name)
	}
}

// TestParseDir_NestedDirectories 验证嵌套子目录被递归解析
func TestParseDir_NestedDirectories(t *testing.T) {
	pkgs, err := ParseDir(token.NewFileSet(), testdataDir(), nil, 0)
	if err != nil {
		t.Fatalf("ParseDir error: %v", err)
	}
	// user, admin, api/quant, api/backtest
	if len(pkgs) != 4 {
		t.Fatalf("expected 4 packages, got %d", len(pkgs))
	}
}

// TestFindModuleName_NoGoMod 验证无 go.mod 的目录返回空字符串
func TestFindModuleName_NoGoMod(t *testing.T) {
	dir := t.TempDir()
	name := findModuleName(dir)
	if name != "" {
		t.Errorf("expected empty string, got '%s'", name)
	}
}
