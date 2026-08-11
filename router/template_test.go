package router

import (
	"strings"
	"testing"
)

// TestCoreTemplate_NoGetRouterContext 验证已移除 getRouterContext 和相关接口
func TestCoreTemplate_NoGetRouterContext(t *testing.T) {
	content := readCoreTemplate(t, "0_router___.gotmp")
	removed := []string{
		"getRouterContext",
		"type setValue interface",
		"type setValue1 interface",
		"type setValue2 interface",
		"type requestHolder interface",
	}
	for _, pattern := range removed {
		if strings.Contains(content, pattern) {
			t.Errorf("核心模板应已移除: %s", pattern)
		}
	}
	// routerContextKey 常量保留, 供引擎 getOrMakeCtx 使用
	if !strings.Contains(content, "routerContextKey") {
		t.Error("routerContextKey 常量应保留")
	}
}

// TestEngineTemplate_HasGetOrMakeCtx 验证引擎模板包含引擎专属 context 缓存
func TestEngineTemplate_HasGetOrMakeCtx(t *testing.T) {
	ginRouter := readEngineTemplate(t, "gin", "0_router_gin___.gotmp")
	if !strings.Contains(ginRouter, "getOrMakeCtx") {
		t.Error("gin 引擎模板缺少 getOrMakeCtx")
	}
	if strings.Contains(ginRouter, "getRouterContext") {
		t.Error("gin 引擎模板应移除 getRouterContext 调用")
	}

	echoRouter := readEngineTemplate(t, "echo", "0_router_echo___.gotmp")
	if !strings.Contains(echoRouter, "getOrMakeCtx") {
		t.Error("echo 引擎模板缺少 getOrMakeCtx")
	}
	if strings.Contains(echoRouter, "getRouterContext") {
		t.Error("echo 引擎模板应移除 getRouterContext 调用")
	}
}

// TestEngineTemplates_Exist 验证引擎模板文件存在
func TestEngineTemplates_Exist(t *testing.T) {
	engines := []string{"gin", "echo"}
	for _, eng := range engines {
		entries, err := files.ReadDir("engine/" + eng)
		if err != nil {
			t.Errorf("读取引擎目录 engine/%s 失败: %v", eng, err)
			continue
		}
		if len(entries) == 0 {
			t.Errorf("引擎目录 engine/%s 为空", eng)
		}
		hasContext := false
		hasRouter := false
		for _, e := range entries {
			if strings.Contains(e.Name(), "context") {
				hasContext = true
			}
			if strings.Contains(e.Name(), "router") {
				hasRouter = true
			}
		}
		if !hasContext {
			t.Errorf("引擎 %s 缺少 context 模板", eng)
		}
		if !hasRouter {
			t.Errorf("引擎 %s 缺少 router 模板", eng)
		}
	}
}

// TestCoreTemplates_Exist 验证核心模板文件存在
func TestCoreTemplates_Exist(t *testing.T) {
	expected := []string{
		"0_router___.gotmp",
		"0_context___.gotmp",
		"0_bind___.gotmp",
		"0_default___.gotmp",
		"0_error___.gotmp",
		"0_logger___.gotmp",
		"0_validator___.gotmp",
		"0__init___.gotmp",
	}
	for _, name := range expected {
		_, err := core.ReadFile("core/" + name)
		if err != nil {
			t.Errorf("缺少核心模板: %s", name)
		}
	}
}

func readCoreTemplate(t *testing.T, name string) string {
	t.Helper()
	bs, err := core.ReadFile("core/" + name)
	if err != nil {
		t.Fatalf("读取模板 %s 失败: %v", name, err)
	}
	return string(bs)
}

func readEngineTemplate(t *testing.T, engine, name string) string {
	t.Helper()
	bs, err := files.ReadFile("engine/" + engine + "/" + name)
	if err != nil {
		t.Fatalf("读取引擎模板 %s/%s 失败: %v", engine, name, err)
	}
	return string(bs)
}
