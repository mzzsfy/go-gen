package router

import (
	"strings"
	"testing"
)

// TestCoreTemplate_ContainsAllInterfaces 验证核心模板包含所有框架版本所需的接口定义
// 回归测试: gin v1.12.0 将 Get/Set key 从 string 改为 any, 需要同时兼容
func TestCoreTemplate_ContainsAllInterfaces(t *testing.T) {
	content := readCoreTemplate(t, "0_router___.gotmp")

	// setValue: echo v4 (Get 返回 any, key 为 string)
	if !strings.Contains(content, "Get(key string) any") {
		t.Error("缺少 setValue 接口: echo 需要 Get(key string) any")
	}
	// setValue1: gin < v1.12.0 (Get 返回 (any, bool), key 为 string)
	if !strings.Contains(content, "Get(key string) (any, bool)") {
		t.Error("缺少 setValue1 接口: 旧版 gin 需要 Get(key string) (any, bool)")
	}
	// setValue2: gin >= v1.12.0 (Get 返回 (any, bool), key 为 any)
	if !strings.Contains(content, "Get(key any) (any, bool)") {
		t.Error("缺少 setValue2 接口: gin v1.12.0 需要 Get(key any) (any, bool)")
	}
	// requestHolder: echo (Request()/SetRequest() 方法)
	if !strings.Contains(content, "Request() *http.Request") {
		t.Error("缺少 requestHolder 接口")
	}
	if !strings.Contains(content, "SetRequest(*http.Request)") {
		t.Error("缺少 requestHolder 接口: SetRequest")
	}
}

// TestCoreTemplate_GetRouterContextChecksAllTypes 验证 getRouterContext 检查所有接口类型
func TestCoreTemplate_GetRouterContextChecksAllTypes(t *testing.T) {
	content := readCoreTemplate(t, "0_router___.gotmp")

	typeAsserts := []string{
		"h.(setValue)",
		"h.(setValue1)",
		"h.(setValue2)",
		"h.(requestHolder)",
	}
	for _, assert := range typeAsserts {
		if !strings.Contains(content, assert) {
			t.Errorf("getRouterContext 缺少类型断言: %s", assert)
		}
	}
}

// TestCoreTemplate_NoDuplicateAnyKeyInterfaces 验证不会同时存在
// 只有 any key 而没有 string key 的接口(即旧 bug: 把 string 全改成 any)
func TestCoreTemplate_NoDuplicateAnyKeyInterfaces(t *testing.T) {
	content := readCoreTemplate(t, "0_router___.gotmp")
	// setValue 的 Get 必须返回 any (非 bool), 且 key 为 string
	// 如果被错误改成 any key, 则 setValue 和 setValue2 签名会重复
	setValueCount := strings.Count(content, "Get(key string) any")
	if setValueCount != 1 {
		t.Errorf("期望 Get(key string) any 出现 1 次, 实际 %d 次", setValueCount)
	}
	setValue1Count := strings.Count(content, "Get(key string) (any, bool)")
	if setValue1Count != 1 {
		t.Errorf("期望 Get(key string) (any, bool) 出现 1 次, 实际 %d 次", setValue1Count)
	}
}

// TestEngineTemplates_Exist 验证引擎模板文件存在
func TestEngineTemplates_Exist(t *testing.T) {
	engines := []string{"gin", "echo"}
	for _, eng := range engines {
		entries, err := files.ReadDir("engin/" + eng)
		if err != nil {
			t.Errorf("读取引擎目录 engin/%s 失败: %v", eng, err)
			continue
		}
		if len(entries) == 0 {
			t.Errorf("引擎目录 engin/%s 为空", eng)
		}
		// 每个引擎至少有 context 和 router 文件
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
