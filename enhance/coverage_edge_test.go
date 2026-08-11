package enhance

import (
	"os"
	"path/filepath"
	"testing"
)

// TestGenRegister_EmptyDirSkips 验证空目录(无 .go 文件)时不生成文件
func TestGenRegister_EmptyDirSkips(t *testing.T) {
	dir := t.TempDir()
	setupEnhanceFlags(t, dir)

	genRegister()

	genPath := filepath.Join(dir, *FileName)
	if _, err := os.Stat(genPath); !os.IsNotExist(err) {
		t.Errorf("空目录不应生成文件")
	}
}

// TestAddFunction_EmptyDirSkips 验证空目录时不生成文件
func TestAddFunction_EmptyDirSkips(t *testing.T) {
	dir := t.TempDir()
	setupEnhanceFlags(t, dir)
	*FunctionName = "GetRelations"
	*FileName = "0_addfunction.gen.go"

	addFunction()

	genPath := filepath.Join(dir, *FileName)
	if _, err := os.Stat(genPath); !os.IsNotExist(err) {
		t.Errorf("空目录不应生成文件")
	}
}

// TestScanAnnotatedStructs_NonexistentDir 验证不存在的目录返回错误
func TestScanAnnotatedStructs_NonexistentDir(t *testing.T) {
	_, _, err := scanAnnotatedStructs("/nonexistent/path/xyz", ".+.go", "@relation")
	if err == nil {
		t.Error("不存在的目录应返回错误")
	}
}

// TestScanAnnotatedStructs_FileWithImports 验证含 import 的文件不误触非 TypeSpec 分支
func TestScanAnnotatedStructs_FileWithImports(t *testing.T) {
	dir := t.TempDir()
	writeGoFile(t, dir, "models.go", `package src

import "fmt"

// @relation tagged
type Tagged struct{}

var _ = fmt.Println
`)
	_, structs, err := scanAnnotatedStructs(dir, ".+.go", "@relation")
	if err != nil {
		t.Fatal(err)
	}
	if len(structs) != 1 {
		t.Fatalf("含 import 的文件: 结构体数: 期望 1, 实际 %d", len(structs))
	}
	if structs[0].Name != "Tagged" {
		t.Errorf("应只识别 Tagged, 实际 %s", structs[0].Name)
	}
}

// TestGenRegister_NonexistentDirPanic 验证不存在的目录导致 panic
func TestGenRegister_NonexistentDirPanic(t *testing.T) {
	setupEnhanceFlags(t, "/nonexistent/path/xyz")
	defer func() {
		if r := recover(); r == nil {
			t.Error("不存在的目录应 panic")
		}
	}()
	genRegister()
}
