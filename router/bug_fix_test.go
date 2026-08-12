package router

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestBug1_OutDirNotAffectRouterFiles 验证 -outDir 不影响包路由文件输出路径
// 复现: go-gen -outDir ./pkg/ 时, 包文件不应输出到 pkg/pkg/api/ (双重嵌套)
func TestBug1_OutDirNotAffectRouterFiles(t *testing.T) {
	srcDir := createTempSrc(t, map[string]string{
		"go.mod": "module gate\n\ngo 1.22\n",
		"pkg/api/health.go": `package api

import "gate/routers"

// HealthCheck
// @RouterGroup /api
// @Router /health[GET]
func HealthCheck(ctx routers.Ctx) any {
	return routers.Ok("ok")
}
`,
	})
	outDir := t.TempDir()

	*WorkDir = srcDir
	*OutDir = filepath.Join(outDir, "pkg")
	*ModuleName = "gate"
	*RouterGroup = ""
	Gen(engineGen{Name: "gin"})

	// 包文件应在 workDir/pkg/api/, 不在 outDir/pkg/api/
	routerFile := filepath.Join(srcDir, "pkg", "api", "0_router___.go")
	if _, err := os.Stat(routerFile); os.IsNotExist(err) {
		t.Errorf("包文件应在 workDir 自身目录: %s", routerFile)
	}

	// 验证不存在双重嵌套
	doubleNested := filepath.Join(outDir, "pkg", "pkg", "api", "0_router___.go")
	if _, err := os.Stat(doubleNested); err == nil {
		t.Errorf("不应出现双重嵌套路径: %s", doubleNested)
	}

	// 核心文件和 import 文件应在 outDir
	coreFile := filepath.Join(outDir, "pkg", "routers", "0_router___.go")
	if _, err := os.Stat(coreFile); os.IsNotExist(err) {
		t.Errorf("核心文件应在 outDir: %s", coreFile)
	}
	importFile := filepath.Join(outDir, "pkg", "0_import___.go")
	if _, err := os.Stat(importFile); os.IsNotExist(err) {
		t.Errorf("import 文件应在 outDir: %s", importFile)
	}
}

// TestBug2_WorkDirNonModuleRoot_ModuleResolution 验证 workDir 非 module root 时递归查找 go.mod
func TestBug2_WorkDirNonModuleRoot_ModuleResolution(t *testing.T) {
	moduleRoot := t.TempDir()
	writeFile(t, moduleRoot, "go.mod", "module gate\n\ngo 1.22\n")
	writeFile(t, moduleRoot, "pkg/api/health.go", `package api

// HealthCheck
// @RouterGroup /api
// @Router /health[GET]
func HealthCheck() {}
`)

	// workDir 设为子目录, go.mod 在上一级
	subDir := filepath.Join(moduleRoot, "pkg")
	*WorkDir = subDir
	*OutDir = subDir
	*ModuleName = ""
	*RouterGroup = ""
	Gen(engineGen{Name: "gin"})

	// 模块名应正确解析为 gate, 而非空
	routerFile := filepath.Join(subDir, "api", "0_router___.go")
	bs, err := os.ReadFile(routerFile)
	if err != nil {
		t.Fatalf("读取路由文件失败: %v", err)
	}
	content := string(bs)
	// 模板中 import routers 路径应包含 gate/pkg (module name + workDir 偏移)
	if !strings.Contains(content, `"gate/pkg/routers"`) {
		t.Errorf("import 路径应包含 module name 和偏移, 内容:\n%s", content)
	}
}

// TestBug2_WorkDirNonModuleRoot_ImportPath 验证 import 文件路径含 module 偏移
func TestBug2_WorkDirNonModuleRoot_ImportPath(t *testing.T) {
	moduleRoot := t.TempDir()
	writeFile(t, moduleRoot, "go.mod", "module gate\n\ngo 1.22\n")
	writeFile(t, moduleRoot, "pkg/api/handler.go", `package api

// Handler
// @Router /test[GET]
func Handler() {}
`)

	subDir := filepath.Join(moduleRoot, "pkg")
	*WorkDir = subDir
	*OutDir = subDir
	*ModuleName = ""
	*RouterGroup = ""
	Gen(engineGen{Name: "gin"})

	importFile := filepath.Join(subDir, "0_import___.go")
	bs, err := os.ReadFile(importFile)
	if err != nil {
		t.Fatalf("读取 import 文件失败: %v", err)
	}
	content := string(bs)
	// import 路径应为 gate/pkg/api (module name + workDir 偏移 + 包路径)
	if !strings.Contains(content, `_ "gate/pkg/api"`) {
		t.Errorf("import 路径应含 module 偏移, 内容:\n%s", content)
	}
}

// TestBug3_HyphenatedModuleName_PackageName 验证 module name 含连字符时
// import 文件包名使用 outDir 实际包名而非 module name 派生
func TestBug3_HyphenatedModuleName_PackageName(t *testing.T) {
	srcDir := createTempSrc(t, map[string]string{
		"go.mod": "module im-go\n\ngo 1.22\n",
		"main.go": `package main

func main() {}
`,
		"api/handler.go": `package api

// Handler
// @Router /test[GET]
func Handler() {}
`,
	})
	*WorkDir = srcDir
	*OutDir = srcDir
	*ModuleName = "im-go"
	*RouterGroup = ""
	Gen(engineGen{Name: "gin"})

	importFile := filepath.Join(srcDir, "0_import___.go")
	bs, err := os.ReadFile(importFile)
	if err != nil {
		t.Fatalf("读取 import 文件失败: %v", err)
	}
	content := string(bs)
	if !strings.Contains(content, "package main") {
		t.Errorf("包名应使用 outDir 实际包名 main, 内容:\n%s", content)
	}
	if strings.Contains(content, "package im-go") {
		t.Errorf("不应使用含连字符的 module name 作为包名, 内容:\n%s", content)
	}
}

// TestBug3_IdempotentRun 两次 Gen 不应 panic (生成的文件不应干扰二次解析)
func TestBug3_IdempotentRun(t *testing.T) {
	srcDir := createTempSrc(t, map[string]string{
		"go.mod": "module im-go\n\ngo 1.22\n",
		"main.go": `package main

func main() {}
`,
		"api/handler.go": `package api

// Handler
// @Router /test[GET]
func Handler() {}
`,
	})
	*WorkDir = srcDir
	*OutDir = srcDir
	*ModuleName = "im-go"
	*RouterGroup = ""

	Gen(engineGen{Name: "gin"})
	Gen(engineGen{Name: "gin"}) // 第二次不应 panic
}

// TestFindModuleRoot_WalksUp 验证 findModuleRoot 向上递归查找
func TestFindModuleRoot_WalksUp(t *testing.T) {
	moduleRoot := t.TempDir()
	writeFile(t, moduleRoot, "go.mod", "module testmod\n\ngo 1.22\n")

	subDir := filepath.Join(moduleRoot, "a", "b", "c")
	os.MkdirAll(subDir, 0755)

	root := findModuleRoot(subDir)
	absModuleRoot, _ := filepath.Abs(moduleRoot)
	if root != absModuleRoot {
		t.Errorf("findModuleRoot 应向上递归: 期望 %s, 实际 %s", absModuleRoot, root)
	}
}

// TestFindModuleName_FromSubdirectory 验证 findModuleName 从子目录向上查找 go.mod
func TestFindModuleName_FromSubdirectory(t *testing.T) {
	moduleRoot := t.TempDir()
	writeFile(t, moduleRoot, "go.mod", "module mymodule\n\ngo 1.22\n")

	subDir := filepath.Join(moduleRoot, "deep", "nested", "dir")
	os.MkdirAll(subDir, 0755)

	name := findModuleName(subDir)
	if name != "mymodule" {
		t.Errorf("子目录应向上查找 module name: 期望 mymodule, 实际 %s", name)
	}
}
