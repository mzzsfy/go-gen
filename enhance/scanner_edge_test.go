package enhance

import (
	"testing"
)

// TestScanAnnotatedStructs_DocAndInlineComment 验证行内注释优先于文档注释
func TestScanAnnotatedStructs_DocAndInlineComment(t *testing.T) {
	dir := t.TempDir()
	writeGoFile(t, dir, "models.go", `package src

// @relation docValue
type Both struct{} // @relation inlineValue
`)
	_, structs, err := scanAnnotatedStructs(dir, ".+.go", "@relation")
	if err != nil {
		t.Fatal(err)
	}
	if len(structs) != 1 {
		t.Fatalf("结构体数: 期望 1, 实际 %d", len(structs))
	}
	if structs[0].Name != "Both" {
		t.Errorf("结构体名: 期望 Both, 实际 %s", structs[0].Name)
	}
	// 行内注释优先
	found := false
	for _, v := range structs[0].Values {
		if v == `"inlineValue"` {
			found = true
		}
	}
	if !found {
		t.Errorf("应使用行内注释值 inlineValue, 实际 %v", structs[0].Values)
	}
}

// TestScanAnnotatedStructs_DocCommentOnly 验证仅文档注释时正常提取
func TestScanAnnotatedStructs_DocCommentOnly(t *testing.T) {
	dir := t.TempDir()
	writeGoFile(t, dir, "models.go", `package src

// @relation docOnly
type Doc struct{}
`)
	_, structs, _ := scanAnnotatedStructs(dir, ".+.go", "@relation")
	if len(structs) != 1 {
		t.Fatalf("结构体数: 期望 1, 实际 %d", len(structs))
	}
	if len(structs[0].Values) != 1 || structs[0].Values[0] != `"docOnly"` {
		t.Errorf("Values: %v", structs[0].Values)
	}
}

// TestScanAnnotatedStructs_EmptyAnnotationValue 验证空注解值
func TestScanAnnotatedStructs_EmptyAnnotationValue(t *testing.T) {
	dir := t.TempDir()
	writeGoFile(t, dir, "models.go", `package src

// @relation
type Empty struct{}
`)
	_, structs, _ := scanAnnotatedStructs(dir, ".+.go", "@relation")
	// @relation 后无值, 应生成一个空字符串元素
	if len(structs) != 1 {
		t.Fatalf("结构体数: 期望 1, 实际 %d", len(structs))
	}
	if len(structs[0].Values) != 1 || structs[0].Values[0] != `""` {
		t.Errorf("空注解值: 期望 [\"\"], 实际 %v", structs[0].Values)
	}
}

// TestScanAnnotatedStructs_CustomAnnotation 验证自定义注解标识符不误匹配
func TestScanAnnotatedStructs_CustomAnnotation(t *testing.T) {
	dir := t.TempDir()
	writeGoFile(t, dir, "models.go", `package src

// @relation shouldNotMatch
type Wrong struct{}

// @mytag correct
type Right struct{}
`)
	_, structs, _ := scanAnnotatedStructs(dir, ".+.go", "@mytag")
	if len(structs) != 1 {
		t.Fatalf("结构体数: 期望 1, 实际 %d", len(structs))
	}
	if structs[0].Name != "Right" {
		t.Errorf("应只匹配 @mytag 的 Right, 实际 %s", structs[0].Name)
	}
}
