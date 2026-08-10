package router

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mzzsfy/go-gen/register"
)

// TestGen_ProducesAllFiles 测试完整生成流水线产生所有预期文件
func TestGen_ProducesAllFiles(t *testing.T) {
	outDir := t.TempDir()
	setupFlags(t, outDir)

	Gen(genGin{Name: "gin"})

	routersDir := filepath.Join(outDir, "routers")
	expectedCoreFiles := []string{
		"0_router___.go",
		"0_context___.go",
		"0_bind___.go",
		"0_default___.go",
		"0_error___.go",
		"0_logger___.go",
		"0_validator___.go",
	}
	for _, name := range expectedCoreFiles {
		p := filepath.Join(routersDir, name)
		if _, err := os.Stat(p); os.IsNotExist(err) {
			t.Errorf("缺少核心生成文件: %s", name)
		}
	}

	// 引擎文件
	expectedEngineFiles := []string{
		"0_router_gin___.go",
		"0_context_gin___.go",
		"0_bind_gin___.go",
	}
	for _, name := range expectedEngineFiles {
		p := filepath.Join(routersDir, name)
		if _, err := os.Stat(p); os.IsNotExist(err) {
			t.Errorf("缺少引擎生成文件: %s", name)
		}
	}

	// 路由注册文件 (按包名生成)
	routerFile := filepath.Join(outDir, "user", "0_router___.go")
	if _, err := os.Stat(routerFile); os.IsNotExist(err) {
		t.Errorf("缺少路由注册文件: user/0_router___.go")
	}

	// import 文件
	importFile := filepath.Join(outDir, "0_import___.go")
	if _, err := os.Stat(importFile); os.IsNotExist(err) {
		t.Errorf("缺少 import 文件: 0_import___.go")
	}
}

// TestGen_RouterRegistrationContent 验证路由注册文件内容正确
func TestGen_RouterRegistrationContent(t *testing.T) {
	outDir := t.TempDir()
	setupFlags(t, outDir)

	Gen(genGin{Name: "gin"})

	routerFile := filepath.Join(outDir, "user", "0_router___.go")
	bs, err := os.ReadFile(routerFile)
	if err != nil {
		t.Fatalf("读取路由注册文件失败: %v", err)
	}
	content := string(bs)

	// 验证函数路由注册
	assertContains(t, content, `Router("/api/user", GetUser, "/info", "GET")`)
	assertContains(t, content, `Router("/api/user", GetUser2, "/detail", "GET")`)
	assertContains(t, content, `Router("/api/user", GetUser2, "/detail2", "POST")`)
	// GetUser4 覆盖了文件级 group
	assertContains(t, content, `Router("/api/v2", GetUser4, "/info", "GET")`)

	// 验证结构体方法路由注册
	assertContains(t, content, `p := &Service{}`)
	assertContains(t, content, `ToBindF(p, p.GetUser3, "GetUser3")`)
	assertContains(t, content, `Router("/api/user", h0, "/list", "GET")`)
}

// TestGen_ImportFileContent 验证 import 文件内容
func TestGen_ImportFileContent(t *testing.T) {
	outDir := t.TempDir()
	setupFlags(t, outDir)

	Gen(genGin{Name: "gin"})

	importFile := filepath.Join(outDir, "0_import___.go")
	bs, err := os.ReadFile(importFile)
	if err != nil {
		t.Fatalf("读取 import 文件失败: %v", err)
	}
	content := string(bs)

	assertContains(t, content, `package testmod`)
	assertContains(t, content, `_ "testmod/user"`)
}

// TestGen_CoreTemplateCopiedCorrectly 验证核心模板被正确复制(包含所有接口定义)
func TestGen_CoreTemplateCopiedCorrectly(t *testing.T) {
	outDir := t.TempDir()
	setupFlags(t, outDir)

	Gen(genGin{Name: "gin"})

	routerCore := filepath.Join(outDir, "routers", "0_router___.go")
	bs, err := os.ReadFile(routerCore)
	if err != nil {
		t.Fatalf("读取核心路由文件失败: %v", err)
	}
	content := string(bs)

	// 验证所有接口定义存在于生成的文件中
	interfaceChecks := map[string]bool{
		"Get(key string) any":         false, // setValue (echo)
		"Get(key string) (any, bool)": false, // setValue1 (旧 gin)
		"Get(key any) (any, bool)":    false, // setValue2 (gin v1.12.0)
		"Request() *http.Request":     false, // requestHolder
		"h.(setValue)":                false,
		"h.(setValue1)":               false,
		"h.(setValue2)":               false,
		"h.(requestHolder)":           false,
	}
	for pattern := range interfaceChecks {
		if strings.Contains(content, pattern) {
			interfaceChecks[pattern] = true
		}
	}
	for pattern, found := range interfaceChecks {
		if !found {
			t.Errorf("生成文件缺少: %s", pattern)
		}
	}
}

// --- helpers ---

func setupFlags(t *testing.T, outDir string) {
	t.Helper()
	*register.WorkDir = testdataDir()
	*register.OutDir = outDir
	*register.ModuleName = "testmod"
	*register.RouterGroup = ""
}

func assertContains(t *testing.T, content, expected string) {
	t.Helper()
	if !strings.Contains(content, expected) {
		t.Errorf("生成内容缺少期望片段:\n  期望包含: %s\n  实际内容:\n%s", expected, content)
	}
}

// TestGen_RouterWithoutMethod 验证无方法的 @Router 只生成3个参数(无方法)
func TestGen_RouterWithoutMethod(t *testing.T) {
	outDir := t.TempDir()
	setupFlags(t, outDir)

	Gen(genGin{Name: "gin"})

	routerFile := filepath.Join(outDir, "user", "0_router___.go")
	bs, err := os.ReadFile(routerFile)
	if err != nil {
		t.Fatalf("读取路由注册文件失败: %v", err)
	}
	content := string(bs)

	// 无方法时只有3个参数, 无 HTTP 方法
	assertContains(t, content, `Router("/api/user", AllMethods, "/all")`)
}

// TestGen_MultipleStructMethods 验证多方法结构体正确生成 h0/h1 变量名
func TestGen_MultipleStructMethods(t *testing.T) {
	outDir := t.TempDir()
	setupFlags(t, outDir)

	Gen(genGin{Name: "gin"})

	routerFile := filepath.Join(outDir, "user", "0_router___.go")
	bs, err := os.ReadFile(routerFile)
	if err != nil {
		t.Fatalf("读取路由注册文件失败: %v", err)
	}
	content := string(bs)

	assertContains(t, content, `ToBindF(p, p.Multi1, "Multi1")`)
	assertContains(t, content, `Router("/api/user", h0, "/multi1", "GET")`)
	assertContains(t, content, `ToBindF(p, p.Multi2, "Multi2")`)
	assertContains(t, content, `Router("/api/user", h1, "/multi2", "POST")`)
}

// TestGen_EchoEngine 验证 echo 引擎生成正确的文件
func TestGen_EchoEngine(t *testing.T) {
	outDir := t.TempDir()
	setupFlags(t, outDir)

	Gen(genGin{Name: "echo"})

	routersDir := filepath.Join(outDir, "routers")
	echoFiles := []string{
		"0_router_echo___.go",
		"0_context_echo___.go",
		"0_bind_echo___.go",
	}
	for _, name := range echoFiles {
		p := filepath.Join(routersDir, name)
		if _, err := os.Stat(p); os.IsNotExist(err) {
			t.Errorf("缺少 echo 引擎文件: %s", name)
		}
	}
}

// TestGen_MultiplePackages 验证多包生成正确
func TestGen_MultiplePackages(t *testing.T) {
	outDir := t.TempDir()
	setupFlags(t, outDir)

	Gen(genGin{Name: "gin"})

	// 两个包的路由文件都应存在
	userFile := filepath.Join(outDir, "user", "0_router___.go")
	if _, err := os.Stat(userFile); os.IsNotExist(err) {
		t.Errorf("缺少 user/0_router___.go")
	}
	adminFile := filepath.Join(outDir, "admin", "0_router___.go")
	if _, err := os.Stat(adminFile); os.IsNotExist(err) {
		t.Errorf("缺少 admin/0_router___.go")
	}

	// admin 包有包级 @RouterGroup /api/admin (来自 doc.go)
	bs, err := os.ReadFile(adminFile)
	if err != nil {
		t.Fatalf("读取 admin 路由文件失败: %v", err)
	}
	assertContains(t, string(bs), `Router("/api/admin", AdminHandler, "/list", "GET")`)
	assertContains(t, string(bs), `Router("/api/admin", AdminSearch, "/search", "GET")`)

	// import 文件包含两个包
	importFile := filepath.Join(outDir, "0_import___.go")
	bs, err = os.ReadFile(importFile)
	if err != nil {
		t.Fatalf("读取 import 文件失败: %v", err)
	}
	content := string(bs)
	assertContains(t, content, `_ "testmod/user"`)
	assertContains(t, content, `_ "testmod/admin"`)
}

// TestGen_FunctionSorting 验证函数按名称排序
func TestGen_FunctionSorting(t *testing.T) {
	outDir := t.TempDir()
	setupFlags(t, outDir)

	Gen(genGin{Name: "gin"})

	routerFile := filepath.Join(outDir, "user", "0_router___.go")
	bs, err := os.ReadFile(routerFile)
	if err != nil {
		t.Fatalf("读取路由注册文件失败: %v", err)
	}
	content := string(bs)

	// AllMethods 应在 GetUser 之前 (A < G)
	idxAll := strings.Index(content, "AllMethods")
	idxGetUser := strings.Index(content, "GetUser")
	if idxAll < 0 || idxGetUser < 0 {
		t.Fatalf("未找到 AllMethods 或 GetUser")
	}
	if idxAll >= idxGetUser {
		t.Errorf("AllMethods 应出现在 GetUser 之前, idxAll=%d, idxGetUser=%d", idxAll, idxGetUser)
	}

	// GetUser 应在 GetUser2 之前
	idxGetUser2 := strings.Index(content, "GetUser2")
	if idxGetUser2 < 0 {
		t.Fatalf("未找到 GetUser2")
	}
	if idxGetUser >= idxGetUser2 {
		t.Errorf("GetUser 应出现在 GetUser2 之前, idxGetUser=%d, idxGetUser2=%d", idxGetUser, idxGetUser2)
	}
}

// TestGen_InitFileSkipIfExists 验证 0__ 前缀文件已存在时不被覆盖
func TestGen_InitFileSkipIfExists(t *testing.T) {
	outDir := t.TempDir()
	setupFlags(t, outDir)

	// 预先创建 init 文件, 写入特定内容
	initPath := filepath.Join(outDir, "routers", "0__init___.go")
	os.MkdirAll(filepath.Dir(initPath), os.ModeDir)
	customContent := "// 自定义内容, 不应被覆盖"
	err := os.WriteFile(initPath, []byte(customContent), os.ModePerm)
	if err != nil {
		t.Fatalf("写入预设文件失败: %v", err)
	}

	Gen(genGin{Name: "gin"})

	// 验证内容未被覆盖
	bs, err := os.ReadFile(initPath)
	if err != nil {
		t.Fatalf("读取 init 文件失败: %v", err)
	}
	if string(bs) != customContent {
		t.Errorf("init 文件被覆盖, 期望: %s, 实际: %s", customContent, string(bs))
	}
}

// --- 全局/包级 @RouterGroup 和 swag 兼容测试 ---

// createTempSrc 在临时目录创建 Go 源码树, files 为相对路径->内容
func createTempSrc(t *testing.T, files map[string]string) string {
	t.Helper()
	srcDir := t.TempDir()
	for relPath, content := range files {
		full := filepath.Join(srcDir, relPath)
		os.MkdirAll(filepath.Dir(full), 0755)
		if err := os.WriteFile(full, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	return srcDir
}

func genWithFlags(t *testing.T, srcDir, outDir, moduleName, globalGroup string) {
	t.Helper()
	*register.WorkDir = srcDir
	*register.OutDir = outDir
	*register.ModuleName = moduleName
	*register.RouterGroup = globalGroup
	Gen(genGin{Name: "gin"})
}

// TestGen_PackageLevelGroup 验证包文档注释中的 @RouterGroup 对包内所有文件生效
func TestGen_PackageLevelGroup(t *testing.T) {
	srcDir := createTempSrc(t, map[string]string{
		"go.mod": "module testmod\n\ngo 1.18\n",
		"api/doc.go": `// @RouterGroup /api/v2
package api
`,
		"api/handler.go": `package api

// Handler1
// @Router /list[GET]
func Handler1() {}
`,
		"api/handler2.go": `package api

// Handler2
// @Router /detail[GET]
func Handler2() {}
`,
	})
	outDir := t.TempDir()
	genWithFlags(t, srcDir, outDir, "testmod", "")

	bs, err := os.ReadFile(filepath.Join(outDir, "api", "0_router___.go"))
	if err != nil {
		t.Fatalf("读取路由文件失败: %v", err)
	}
	content := string(bs)
	// 两个文件都没有文件级 @RouterGroup, 应继承包级 /api/v2
	assertContains(t, content, `Router("/api/v2", Handler1, "/list", "GET")`)
	assertContains(t, content, `Router("/api/v2", Handler2, "/detail", "GET")`)
}

// TestGen_GlobalGroup 验证全局 group flag 对无 group 注解的路由生效
func TestGen_GlobalGroup(t *testing.T) {
	srcDir := createTempSrc(t, map[string]string{
		"go.mod": "module testmod\n\ngo 1.18\n",
		"api/handler.go": `package api

// Handler
// @Router /test[GET]
func Handler() {}
`,
	})
	outDir := t.TempDir()
	genWithFlags(t, srcDir, outDir, "testmod", "/api/global")

	bs, err := os.ReadFile(filepath.Join(outDir, "api", "0_router___.go"))
	if err != nil {
		t.Fatalf("读取路由文件失败: %v", err)
	}
	assertContains(t, string(bs), `Router("/api/global", Handler, "/test", "GET")`)
}

// TestGen_GroupPriority 验证优先级: 函数级 > 文件级 > 包级 > 全局
func TestGen_GroupPriority(t *testing.T) {
	srcDir := createTempSrc(t, map[string]string{
		"go.mod": "module testmod\n\ngo 1.18\n",
		// 包级 group
		"pkg/doc.go": `// @RouterGroup /pkg
package pkg
`,
		// 文件级 group 覆盖包级
		"pkg/a.go": `package pkg

// @RouterGroup /file

// FileHandler 使用文件级 group
// @Router /a[GET]
func FileHandler() {}

// PkgHandler 使用包级 group (无文件级, 无函数级)
// @Router /b[GET]
func PkgHandler() {}

// FuncHandler 使用函数级 group 覆盖文件级
// @RouterGroup /func
// @Router /c[GET]
func FuncHandler() {}
`,
	})
	outDir := t.TempDir()
	// 全局 group /global, 应被所有更高优先级覆盖
	genWithFlags(t, srcDir, outDir, "testmod", "/global")

	bs, err := os.ReadFile(filepath.Join(outDir, "pkg", "0_router___.go"))
	if err != nil {
		t.Fatalf("读取路由文件失败: %v", err)
	}
	content := string(bs)
	// 文件级 /file 覆盖包级 /pkg 和全局 /global
	assertContains(t, content, `Router("/file", FileHandler, "/a", "GET")`)
	// 包级 /pkg 覆盖全局 /global (a.go 中无文件级注解的部分使用文件级; PkgHandler 无函数级, 但文件级已设为 /file)
	// 实际上 PkgHandler 在 a.go 中, 文件级 group 是 /file, 所以也用 /file
	assertContains(t, content, `Router("/file", PkgHandler, "/b", "GET")`)
	// 函数级 /func 覆盖文件级 /file
	assertContains(t, content, `Router("/func", FuncHandler, "/c", "GET")`)
}

// TestGen_GroupPriority_PackageOverGlobal 验证包级覆盖全局
func TestGen_GroupPriority_PackageOverGlobal(t *testing.T) {
	srcDir := createTempSrc(t, map[string]string{
		"go.mod": "module testmod\n\ngo 1.18\n",
		"pkg/doc.go": `// @RouterGroup /pkg
package pkg
`,
		"pkg/handler.go": `package pkg

// Handler
// @Router /test[GET]
func Handler() {}
`,
	})
	outDir := t.TempDir()
	genWithFlags(t, srcDir, outDir, "testmod", "/global")

	bs, _ := os.ReadFile(filepath.Join(outDir, "pkg", "0_router___.go"))
	// 包级 /pkg 覆盖全局 /global
	assertContains(t, string(bs), `Router("/pkg", Handler, "/test", "GET")`)
}

// TestGen_SwagFormat 验证 swag 格式 @Router /path [GET] (空格+方括号)
func TestGen_SwagFormat(t *testing.T) {
	srcDir := createTempSrc(t, map[string]string{
		"go.mod": "module testmod\n\ngo 1.18\n",
		"api/handler.go": `package api

// SwagHandler swag格式: 路径与方法间有空格
// @Router /users/{id} [GET]
func SwagHandler() {}

// SwagNoMethod swag格式无方法
// @Router /list
func SwagNoMethod() {}
`,
	})
	outDir := t.TempDir()
	genWithFlags(t, srcDir, outDir, "testmod", "")

	bs, _ := os.ReadFile(filepath.Join(outDir, "api", "0_router___.go"))
	content := string(bs)
	// swag 格式 @Router /users/{id} [GET] 正确解析
	assertContains(t, content, `Router("", SwagHandler, "/users/{id}", "GET")`)
	// swag 格式 @Router /list 无方法
	assertContains(t, content, `Router("", SwagNoMethod, "/list")`)
}

// TestGen_PackageDocNotTreatedAsFileLevel 验证包文档注释中的 @RouterGroup
// 不会被当作文件级 group, 同一文件中的文件级 @RouterGroup 仍优先生效
func TestGen_PackageDocNotTreatedAsFileLevel(t *testing.T) {
	srcDir := createTempSrc(t, map[string]string{
		"go.mod": "module testmod\n\ngo 1.18\n",
		"pkg/handler.go": `// @RouterGroup /pkg
package pkg

// @RouterGroup /file

// FileHandler 使用文件级 /file, 不受包文档 /pkg 影响
// @Router /a[GET]
func FileHandler() {}
`,
		"pkg/other.go": `package pkg

// OtherHandler 无文件级 group, 继承包级 /pkg
// @Router /b[GET]
func OtherHandler() {}
`,
	})
	outDir := t.TempDir()
	genWithFlags(t, srcDir, outDir, "testmod", "")

	bs, _ := os.ReadFile(filepath.Join(outDir, "pkg", "0_router___.go"))
	content := string(bs)
	// handler.go 有文件级 /file, 应覆盖包文档 /pkg
	assertContains(t, content, `Router("/file", FileHandler, "/a", "GET")`)
	// other.go 无文件级 group, 继承包文档 /pkg
	assertContains(t, content, `Router("/pkg", OtherHandler, "/b", "GET")`)
}
