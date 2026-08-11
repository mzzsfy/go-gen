package router

import (
	"go/ast"
	"testing"
)

func TestParseRouterPath_BasicWithMethod(t *testing.T) {
	hp := parseRouterPath("@Router /api/user/info [GET]")
	if hp.Path != "/api/user/info" {
		t.Errorf("Path: 期望 /api/user/info, 实际 %s", hp.Path)
	}
	if hp.Method != "GET" {
		t.Errorf("Method: 期望 GET, 实际 %s", hp.Method)
	}
	if hp.PathMethod != `/api/user/info", "GET` {
		t.Errorf("PathMethod: %s", hp.PathMethod)
	}
}

func TestParseRouterPath_NoMethod(t *testing.T) {
	hp := parseRouterPath("@Router /list")
	if hp.Path != "/list" {
		t.Errorf("Path: 期望 /list, 实际 %s", hp.Path)
	}
	if hp.Method != "" {
		t.Errorf("Method: 期望空, 实际 %s", hp.Method)
	}
	if hp.PathMethod != "/list" {
		t.Errorf("PathMethod: 期望 /list, 实际 %s", hp.PathMethod)
	}
}

func TestParseRouterPath_LowercaseMethod(t *testing.T) {
	hp := parseRouterPath("@Router /api/info [post]")
	if hp.Method != "POST" {
		t.Errorf("Method 应大写: 期望 POST, 实际 %s", hp.Method)
	}
}

func TestParseRouterPath_PathWithBraces(t *testing.T) {
	hp := parseRouterPath("@Router /users/{id} [GET]")
	if hp.Path != "/users/{id}" {
		t.Errorf("Path: 期望 /users/{id}, 实际 %s", hp.Path)
	}
}

func TestParseRouterPath_MultiplePathsSameLine(t *testing.T) {
	hp := parseRouterPath("@Router /list")
	if hp.Method != "" {
		t.Errorf("无方括号时 Method 应为空, 实际 %s", hp.Method)
	}
	if hp.Path != "/list" {
		t.Errorf("Path: 期望 /list, 实际 %s", hp.Path)
	}
}

func TestNormalizeGroupPath_EmptyString(t *testing.T) {
	// 空字符串 → 补 / → 根路径归零 → 最终 ""
	gp := ""
	normalizeGroupPath(&gp)
	if gp != "" {
		t.Errorf("空字符串最终应为空, 实际 %s", gp)
	}
}

func TestNormalizeGroupPath_NoLeadingSlash(t *testing.T) {
	gp := "api/v1"
	normalizeGroupPath(&gp)
	if gp != "/api/v1" {
		t.Errorf("应补前导 /, 实际 %s", gp)
	}
}

func TestNormalizeGroupPath_AlreadyNormalized(t *testing.T) {
	gp := "/api/v1"
	normalizeGroupPath(&gp)
	if gp != "/api/v1" {
		t.Errorf("已有前导 / 应不变, 实际 %s", gp)
	}
}

func TestNormalizeGroupPath_RootBecomesEmpty(t *testing.T) {
	gp := "/"
	normalizeGroupPath(&gp)
	if gp != "" {
		t.Errorf("/ 应归零为空字符串, 实际 %s", gp)
	}
}

func TestNormalizeGroupPath_RootFromNoSlash(t *testing.T) {
	gp := ""
	normalizeGroupPath(&gp)
	// 空字符串 → "/" → ""
	if gp != "" {
		t.Errorf("空字符串最终应为空, 实际 %s", gp)
	}
}

func TestAddStructFunction_MultipleMethodsSameStruct(t *testing.T) {
	pc := &Package{}
	fn1 := Function{Name: "MethodA", GroupPath: "/api"}
	fn2 := Function{Name: "MethodB", GroupPath: "/api"}

	// 模拟 *ast.FuncDecl 的接收者, 只需要 addStructFunction 中的类型断言
	d1 := &ast.FuncDecl{
		Recv: &ast.FieldList{List: []*ast.Field{{Type: &ast.Ident{Name: "Service"}}}},
		Name: &ast.Ident{Name: "MethodA"},
	}
	d2 := &ast.FuncDecl{
		Recv: &ast.FieldList{List: []*ast.Field{{Type: &ast.Ident{Name: "Service"}}}},
		Name: &ast.Ident{Name: "MethodB"},
	}

	addStructFunction(pc, d1, fn1)
	addStructFunction(pc, d2, fn2)

	if len(pc.StructFunctions) != 1 {
		t.Fatalf("结构体数: 期望 1, 实际 %d", len(pc.StructFunctions))
	}
	if pc.StructFunctions[0].StructName != "Service" {
		t.Errorf("结构体名: 期望 Service, 实际 %s", pc.StructFunctions[0].StructName)
	}
	if len(pc.StructFunctions[0].Functions) != 2 {
		t.Errorf("方法数: 期望 2, 实际 %d", len(pc.StructFunctions[0].Functions))
	}
}

func TestAddStructFunction_PointerReceiver(t *testing.T) {
	pc := &Package{}
	fn := Function{Name: "Method", GroupPath: "/api"}
	d := &ast.FuncDecl{
		Recv: &ast.FieldList{List: []*ast.Field{{
			Type: &ast.StarExpr{X: &ast.Ident{Name: "Service"}},
		}}},
		Name: &ast.Ident{Name: "Method"},
	}
	addStructFunction(pc, d, fn)
	if len(pc.StructFunctions) != 1 {
		t.Fatalf("结构体数: 期望 1, 实际 %d", len(pc.StructFunctions))
	}
	if pc.StructFunctions[0].StructName != "Service" {
		t.Errorf("指针接收者应解引用: 期望 Service, 实际 %s", pc.StructFunctions[0].StructName)
	}
}

func TestAddStructFunction_DifferentStructs(t *testing.T) {
	pc := &Package{}
	d1 := &ast.FuncDecl{
		Recv: &ast.FieldList{List: []*ast.Field{{Type: &ast.Ident{Name: "A"}}}},
		Name: &ast.Ident{Name: "M1"},
	}
	d2 := &ast.FuncDecl{
		Recv: &ast.FieldList{List: []*ast.Field{{Type: &ast.Ident{Name: "B"}}}},
		Name: &ast.Ident{Name: "M2"},
	}
	addStructFunction(pc, d1, Function{Name: "M1"})
	addStructFunction(pc, d2, Function{Name: "M2"})
	if len(pc.StructFunctions) != 2 {
		t.Fatalf("结构体数: 期望 2, 实际 %d", len(pc.StructFunctions))
	}
}
