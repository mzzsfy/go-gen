package router

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mzzsfy/go-gen/register"
)

// TestParams_AllDefaults 验证所有 Router 相关 flag 的默认值
func TestParams_AllDefaults(t *testing.T) {
	tests := []struct {
		flag string
		want string
	}{
		{"workDir", "./"},
		{"outDir", "./"},
		{"moduleName", ""},
		{"routerGroup", ""},
	}
	for _, tt := range tests {
		t.Run(tt.flag, func(t *testing.T) {
			f := flag.Lookup(tt.flag)
			if f == nil {
				t.Fatalf("flag %q 未注册", tt.flag)
			}
			if f.DefValue != tt.want {
				t.Errorf("flag %q 默认值: 得到 %q, 期望 %q", tt.flag, f.DefValue, tt.want)
			}
		})
	}
}

// TestGenType_DefaultRegistered 验证 genType 默认值 "echo-router" 已注册
// genType 在 main 包定义, 但默认值 echo-router 对应 router 包 init 注册的生成器
func TestGenType_DefaultRegistered(t *testing.T) {
	// 使用空临时目录, 避免污染项目
	emptyDir := t.TempDir()
	*WorkDir = emptyDir
	*OutDir = emptyDir
	*ModuleName = ""
	*RouterGroup = ""

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("默认 genType echo-router 应已注册, panic: %v", r)
		}
	}()
	register.Do("echo-router")
}

// TestGenType_CustomRegistered 验证 genType 自定义值 "gin-router" 已注册
func TestGenType_CustomRegistered(t *testing.T) {
	emptyDir := t.TempDir()
	*WorkDir = emptyDir
	*OutDir = emptyDir
	*ModuleName = ""
	*RouterGroup = ""

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("genType gin-router 应已注册, panic: %v", r)
		}
	}()
	register.Do("gin-router")
}
func TestModuleName_DefaultBehavior(t *testing.T) {
	srcDir := createTempSrc(t, map[string]string{
		"go.mod": "module auto-detected\n\ngo 1.22\n",
		"api/handler.go": `package api

// Handler
// @Router /test[GET]
func Handler() {}
`,
	})
	outDir := t.TempDir()

	*WorkDir = srcDir
	*OutDir = outDir
	*ModuleName = "" // 默认值
	*RouterGroup = ""
	Gen(engineGen{Name: "gin"})

	importFile := filepath.Join(outDir, "0_import___.go")
	bs, err := os.ReadFile(importFile)
	if err != nil {
		t.Fatalf("读取 import 文件失败: %v", err)
	}
	if !strings.Contains(string(bs), `"auto-detected/api"`) {
		t.Errorf("moduleName 为空时应自动从 go.mod 读取, 内容:\n%s", string(bs))
	}
}

// TestModuleName_CustomValue 验证 moduleName 自定义值覆盖 go.mod
func TestModuleName_CustomValue(t *testing.T) {
	srcDir := createTempSrc(t, map[string]string{
		"go.mod": "module real-name\n\ngo 1.22\n",
		"api/handler.go": `package api

// Handler
// @Router /test[GET]
func Handler() {}
`,
	})
	outDir := t.TempDir()

	*WorkDir = srcDir
	*OutDir = outDir
	*ModuleName = "custom-name" // 自定义覆盖
	*RouterGroup = ""
	Gen(engineGen{Name: "gin"})

	importFile := filepath.Join(outDir, "0_import___.go")
	bs, _ := os.ReadFile(importFile)
	if !strings.Contains(string(bs), `"custom-name/api"`) {
		t.Errorf("自定义 moduleName 应覆盖 go.mod, 内容:\n%s", string(bs))
	}
	if strings.Contains(string(bs), `"real-name/api"`) {
		t.Errorf("不应使用 go.mod 中的名称, 内容:\n%s", string(bs))
	}
}

// TestRouterGroup_DefaultBehavior 验证 routerGroup 默认(空)时无前缀
func TestRouterGroup_DefaultBehavior(t *testing.T) {
	srcDir := createTempSrc(t, map[string]string{
		"go.mod": "module testmod\n\ngo 1.22\n",
		"api/handler.go": `package api

// Handler
// @Router /test[GET]
func Handler() {}
`,
	})
	outDir := t.TempDir()
	genWithFlags(t, srcDir, outDir, "testmod", "") // 空 routerGroup

	bs, _ := os.ReadFile(filepath.Join(srcDir, "api", "0_router___.go"))
	// 无 group 时 Router 第一个参数为空
	if !strings.Contains(string(bs), `Router("", Handler, "/test", "GET")`) {
		t.Errorf("默认 routerGroup 为空时不应有前缀, 内容:\n%s", string(bs))
	}
}

// TestRouterGroup_CustomValue 验证 routerGroup 自定义值作为全局前缀
func TestRouterGroup_CustomValue(t *testing.T) {
	srcDir := createTempSrc(t, map[string]string{
		"go.mod": "module testmod\n\ngo 1.22\n",
		"api/handler.go": `package api

// Handler
// @Router /test[GET]
func Handler() {}
`,
	})
	outDir := t.TempDir()
	genWithFlags(t, srcDir, outDir, "testmod", "/api/v1") // 自定义 routerGroup

	bs, _ := os.ReadFile(filepath.Join(srcDir, "api", "0_router___.go"))
	if !strings.Contains(string(bs), `Router("/api/v1", Handler, "/test", "GET")`) {
		t.Errorf("自定义 routerGroup 应作为前缀, 内容:\n%s", string(bs))
	}
}

// TestWorkDir_DefaultBehavior 验证 workDir 默认(./)时扫描当前目录
func TestWorkDir_DefaultBehavior(t *testing.T) {
	srcDir := createTempSrc(t, map[string]string{
		"go.mod": "module testmod\n\ngo 1.22\n",
		"api/handler.go": `package api

// Handler
// @Router /test[GET]
func Handler() {}
`,
	})
	outDir := t.TempDir()

	// workDir 设为 module root (等价于 ./ 相对 module root)
	*WorkDir = srcDir
	*OutDir = outDir
	*ModuleName = "testmod"
	*RouterGroup = ""
	Gen(engineGen{Name: "gin"})

	// 包路由文件在 workDir/api/ 下
	routerFile := filepath.Join(srcDir, "api", "0_router___.go")
	if _, err := os.Stat(routerFile); os.IsNotExist(err) {
		t.Errorf("workDir 为 module root 时路由文件应在 workDir 子目录: %s", routerFile)
	}
}

// TestOutDir_DefaultBehavior 验证 outDir 默认(./)时与 workDir 一致
func TestOutDir_DefaultBehavior(t *testing.T) {
	srcDir := createTempSrc(t, map[string]string{
		"go.mod": "module testmod\n\ngo 1.22\n",
		"api/handler.go": `package api

// Handler
// @Router /test[GET]
func Handler() {}
`,
	})

	// 不设 OutDir (保持默认 ./), 设 WorkDir 为子目录
	origOutDir := *OutDir
	*WorkDir = srcDir
	*OutDir = "./"
	*ModuleName = "testmod"
	*RouterGroup = ""
	t.Cleanup(func() { *OutDir = origOutDir })

	Gen(engineGen{Name: "gin"})

	// outDir 默认应等于 workDir
	importFile := filepath.Join(srcDir, "0_import___.go")
	if _, err := os.Stat(importFile); os.IsNotExist(err) {
		t.Errorf("outDir 默认应等于 workDir, import 文件应在: %s", importFile)
	}
}

// TestOutDir_CustomValue 验证 outDir 自定义值控制核心文件位置
func TestOutDir_CustomValue(t *testing.T) {
	srcDir := createTempSrc(t, map[string]string{
		"go.mod": "module testmod\n\ngo 1.22\n",
		"api/handler.go": `package api

// Handler
// @Router /test[GET]
func Handler() {}
`,
	})
	outDir := t.TempDir()
	genWithFlags(t, srcDir, outDir, "testmod", "")

	// 核心文件在 outDir
	coreFile := filepath.Join(outDir, "routers", "0_router___.go")
	if _, err := os.Stat(coreFile); os.IsNotExist(err) {
		t.Errorf("核心文件应在 outDir: %s", coreFile)
	}
	// 包路由文件在 workDir (srcDir), 不在 outDir
	routerFile := filepath.Join(srcDir, "api", "0_router___.go")
	if _, err := os.Stat(routerFile); os.IsNotExist(err) {
		t.Errorf("包路由文件应在 workDir: %s", routerFile)
	}
	routerFileInOut := filepath.Join(outDir, "api", "0_router___.go")
	if _, err := os.Stat(routerFileInOut); err == nil {
		t.Errorf("包路由文件不应在 outDir (避免双重嵌套): %s", routerFileInOut)
	}
}
