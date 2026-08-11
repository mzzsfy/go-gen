package enhance

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScanAnnotatedStructs_Basic(t *testing.T) {
	dir := t.TempDir()
	writeGoFile(t, dir, "models.go", `package src

// @relation user
type User struct{}

// @relation order,product
type Order struct{}
`)
	pkgName, structs, err := scanAnnotatedStructs(dir, ".+.go", "@relation")
	if err != nil {
		t.Fatal(err)
	}
	if pkgName != "src" {
		t.Errorf("包名: 期望 src, 实际 %s", pkgName)
	}
	if len(structs) != 2 {
		t.Fatalf("结构体数量: 期望 2, 实际 %d", len(structs))
	}
	findStruct := func(name string) annotatedStruct {
		for _, s := range structs {
			if s.Name == name {
				return s
			}
		}
		t.Fatalf("未找到结构体 %s", name)
		return annotatedStruct{}
	}
	user := findStruct("User")
	if len(user.Values) != 1 || user.Values[0] != `"user"` {
		t.Errorf("User values: %v", user.Values)
	}
	order := findStruct("Order")
	if len(order.Values) != 2 || order.Values[0] != `"order"` || order.Values[1] != `"product"` {
		t.Errorf("Order values: %v", order.Values)
	}
}

func TestScanAnnotatedStructs_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	pkgName, structs, err := scanAnnotatedStructs(dir, ".+.go", "@relation")
	if err != nil {
		t.Fatal(err)
	}
	if pkgName != "" {
		t.Errorf("空目录包名应为空, 实际 %s", pkgName)
	}
	if len(structs) != 0 {
		t.Errorf("空目录结构体数: 期望 0, 实际 %d", len(structs))
	}
}

func TestScanAnnotatedStructs_NoMatchingFiles(t *testing.T) {
	dir := t.TempDir()
	writeGoFile(t, dir, "models.go", `package src
type User struct{}
`)
	_, structs, err := scanAnnotatedStructs(dir, `.*\.pb\.go`, "@relation")
	if err != nil {
		t.Fatal(err)
	}
	if len(structs) != 0 {
		t.Errorf("无匹配文件时结构体数: 期望 0, 实际 %d", len(structs))
	}
}

func TestScanAnnotatedStructs_SkipsNonStructTypes(t *testing.T) {
	dir := t.TempDir()
	writeGoFile(t, dir, "types.go", `package src

// @relation tagged
type Tagged struct{}

// @relation skipped
type MyInt int

type MyIface interface {
	Foo()
}
`)
	_, structs, err := scanAnnotatedStructs(dir, ".+.go", "@relation")
	if err != nil {
		t.Fatal(err)
	}
	if len(structs) != 1 {
		t.Fatalf("结构体数: 期望 1, 实际 %d", len(structs))
	}
	if structs[0].Name != "Tagged" {
		t.Errorf("应只保留 Tagged, 实际 %s", structs[0].Name)
	}
}

func TestScanAnnotatedStructs_UnannotatedStructSkipped(t *testing.T) {
	dir := t.TempDir()
	writeGoFile(t, dir, "models.go", `package src

// @relation tagged
type Tagged struct{}

type Plain struct{}
`)
	_, structs, _ := scanAnnotatedStructs(dir, ".+.go", "@relation")
	if len(structs) != 1 {
		t.Fatalf("结构体数: 期望 1, 实际 %d", len(structs))
	}
	if structs[0].Name != "Tagged" {
		t.Errorf("应只保留 Tagged")
	}
}

func TestScanAnnotatedStructs_MultipleFiles(t *testing.T) {
	dir := t.TempDir()
	writeGoFile(t, dir, "a.go", `package src
// @relation a1
type StructA struct{}
`)
	writeGoFile(t, dir, "b.go", `package src
// @relation b1
type StructB struct{}
`)
	pkgName, structs, _ := scanAnnotatedStructs(dir, ".+.go", "@relation")
	if pkgName != "src" {
		t.Errorf("包名: 期望 src, 实际 %s", pkgName)
	}
	if len(structs) != 2 {
		t.Errorf("跨文件结构体数: 期望 2, 实际 %d", len(structs))
	}
}

func TestScanAnnotatedStructs_MalformedGoSkipped(t *testing.T) {
	dir := t.TempDir()
	writeGoFile(t, dir, "bad.go", "package src\nthis is not valid go")
	writeGoFile(t, dir, "good.go", `package src
// @relation ok
type Good struct{}
`)
	_, structs, _ := scanAnnotatedStructs(dir, ".+.go", "@relation")
	if len(structs) != 1 {
		t.Fatalf("坏文件应跳过, 结构体数: 期望 1, 实际 %d", len(structs))
	}
	if structs[0].Name != "Good" {
		t.Errorf("应保留 Good")
	}
}

func TestScanAnnotatedStructs_InlineComment(t *testing.T) {
	dir := t.TempDir()
	writeGoFile(t, dir, "models.go", `package src
type Inline struct{} // @relation inline
`)
	_, structs, _ := scanAnnotatedStructs(dir, ".+.go", "@relation")
	if len(structs) != 1 {
		t.Fatalf("行内注释结构体数: 期望 1, 实际 %d", len(structs))
	}
	if structs[0].Name != "Inline" {
		t.Errorf("应识别行内注释的 Inline")
	}
}

func TestScanAnnotatedStructs_InvalidRegex(t *testing.T) {
	dir := t.TempDir()
	_, _, err := scanAnnotatedStructs(dir, "[invalid", "@relation")
	if err == nil {
		t.Error("无效正则应返回错误")
	}
}

func writeGoFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}
