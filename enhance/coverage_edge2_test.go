package enhance

import "testing"

// TestScanAnnotatedStructs_FileWithFuncDecl 验证含函数声明的文件不误触非 GenDecl 分支
func TestScanAnnotatedStructs_FileWithFuncDecl(t *testing.T) {
	dir := t.TempDir()
	writeGoFile(t, dir, "models.go", `package src

// @relation tagged
type Tagged struct{}

func DoSomething() {}
`)
	_, structs, err := scanAnnotatedStructs(dir, ".+.go", "@relation")
	if err != nil {
		t.Fatal(err)
	}
	if len(structs) != 1 {
		t.Fatalf("含函数声明的文件: 结构体数: 期望 1, 实际 %d", len(structs))
	}
	if structs[0].Name != "Tagged" {
		t.Errorf("应只识别 Tagged, 实际 %s", structs[0].Name)
	}
}

// TestAddFunction_NonexistentDirPanic 验证不存在的目录导致 panic
func TestAddFunction_NonexistentDirPanic(t *testing.T) {
	setupEnhanceFlags(t, "/nonexistent/path/xyz")
	*FunctionName = "GetRelations"
	*FileName = "0_addfunction.gen.go"
	defer func() {
		if r := recover(); r == nil {
			t.Error("不存在的目录应 panic")
		}
	}()
	addFunction()
}
