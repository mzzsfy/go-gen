package router

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

// findPkgBySuffix 在 ParseDir 结果中按路径后缀查找包
func findPkgBySuffix(t *testing.T, pkgs map[string]*ast.Package, suffix string) *ast.Package {
	t.Helper()
	for k, v := range pkgs {
		if strings.HasSuffix(k, suffix) {
			return v
		}
	}
	t.Fatalf("未找到路径后缀为 %s 的包", suffix)
	return nil
}

// TestParseDir_ModePropagation 回归测试: 递归子目录应透传 mode 参数
// bug: 递归调用曾硬编码 parser.ParseComments 而非透传 mode
func TestParseDir_ModePropagation(t *testing.T) {
	srcDir := createTempSrc(t, map[string]string{
		"go.mod": "module testmod\n\ngo 1.22\n",
		"pkg/a.go": `package pkg

// FuncA 有注释
func FuncA() {}
`,
	})

	// 用 mode=0 调用, 子目录文件不应包含注释
	pkgs, err := ParseDir(token.NewFileSet(), srcDir, nil, 0)
	if err != nil {
		t.Fatal(err)
	}

	pkg := findPkgBySuffix(t, pkgs, "/pkg")
	for _, f := range pkg.Files {
		if len(f.Comments) > 0 {
			t.Errorf("mode=0 时文件不应包含注释, 文件 %s 有 %d 个注释组",
				f.Name.Name, len(f.Comments))
		}
	}
}

// TestParseDir_ModeWithComments 验证 mode=ParseComments 时子目录确实有注释
func TestParseDir_ModeWithComments(t *testing.T) {
	srcDir := createTempSrc(t, map[string]string{
		"go.mod": "module testmod\n\ngo 1.22\n",
		"pkg/a.go": `package pkg

// FuncA 注释
func FuncA() {}
`,
	})

	pkgs, err := ParseDir(token.NewFileSet(), srcDir, nil, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}

	pkg := findPkgBySuffix(t, pkgs, "/pkg")
	found := false
	for _, f := range pkg.Files {
		if len(f.Comments) > 0 {
			found = true
		}
	}
	if !found {
		t.Error("mode=ParseComments 时子目录应包含注释")
	}
}

func TestParseDir_SkipsNonGoFiles(t *testing.T) {
	srcDir := createTempSrc(t, map[string]string{
		"go.mod":     "module testmod\n\ngo 1.22\n",
		"pkg/a.go":   "package pkg",
		"pkg/b.txt":  "not a go file",
		"pkg/c.json": `{"key": "value"}`,
	})

	pkgs, err := ParseDir(token.NewFileSet(), srcDir, nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	pkg := findPkgBySuffix(t, pkgs, "/pkg")
	for fname := range pkg.Files {
		if !strings.HasSuffix(fname, ".go") {
			t.Errorf("非 .go 文件不应被解析: %s", fname)
		}
	}
}

func TestParseDir_SkipsHiddenDirs(t *testing.T) {
	srcDir := createTempSrc(t, map[string]string{
		"go.mod":       "module testmod\n\ngo 1.22\n",
		".hidden/a.go": "package hidden",
		"pkg/a.go":     "package pkg",
	})

	pkgs, err := ParseDir(token.NewFileSet(), srcDir, nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	for k := range pkgs {
		if strings.Contains(k, ".hidden") {
			t.Error("隐藏目录 .hidden 不应被扫描")
		}
	}
}
