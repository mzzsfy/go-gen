package router

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// 集成测试: 生成真实 Go 服务, 编译并运行, 验证 HTTP 路由可访问.
// gin 和 echo 各一个子测试, 临时目录由 t.TempDir() 自动清理.

func TestIntegration_BuildAndRun(t *testing.T) {
	for _, spec := range integrationSpecs {
		t.Run(spec.engine, func(t *testing.T) {
			testBuildAndRunEngine(t, spec)
		})
	}
}

func testBuildAndRunEngine(t *testing.T, spec engineSpec) {
	t.Helper()

	// 创建临时模块目录
	modDir := t.TempDir()
	writeFile(t, modDir, "go.mod", "module testmod\n\ngo 1.22\n")

	// 写入 handler 源码 (签名 func(routers.Ctx) any)
	writeFile(t, modDir, "user/handler.go", `package user

import "testmod/routers"

// GetUser
// @RouterGroup /api
// @Router /info[GET]
func GetUser(ctx routers.Ctx) any {
	return routers.Ok("hello")
}
`)

	// 运行 Gen 生成路由代码
	*WorkDir = filepath.Join(modDir, "user")
	// WorkDir 需包含 go.mod 所在目录, 否则 findModuleName 找不到模块名
	// Gen 的 ParseDir 递归扫描 WorkDir, 设置为模块根目录
	*WorkDir = modDir
	*OutDir = modDir
	*ModuleName = "testmod"
	*RouterGroup = ""
	Gen(engineGen{Name: spec.engine})

	// 覆盖 0__init___.go, 去掉 panic("fixme")
	writeFile(t, modDir, "routers/0__init___.go", spec.initContent)

	// 写入 main.go
	writeFile(t, modDir, "cmd/server/main.go", spec.mainContent)

	// go mod tidy
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = modDir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Skipf("go mod tidy 失败 (可能无网络): %v\n%s", err, out)
	}

	// go build
	binaryName := "testserver"
	if runtime.GOOS == "windows" {
		binaryName = "testserver.exe"
	}
	binaryPath := filepath.Join(modDir, binaryName)
	cmd = exec.Command("go", "build", "-o", binaryPath, "./cmd/server")
	cmd.Dir = modDir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go build 失败: %v\n%s", err, out)
	}

	// 启动服务
	port := freePort(t)
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	cmd = exec.Command(binaryPath)
	cmd.Env = append(os.Environ(), "LISTEN_ADDR="+addr)
	if err := cmd.Start(); err != nil {
		t.Fatalf("启动服务失败: %v", err)
	}
	defer func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}()

	// 等待服务就绪
	if !waitReady(t, addr, 5*time.Second) {
		t.Fatalf("服务未在超时内就绪: %s", addr)
	}

	// 循环发送请求验证 sync.Pool 复用后响应仍正确
	expected := `{"code":0,"data":"hello"}`
	for i := 0; i < 10; i++ {
		resp, err := http.Get(fmt.Sprintf("http://%s/api/info", addr))
		if err != nil {
			t.Fatalf("第%d次请求失败: %v", i+1, err)
		}
		body := make([]byte, 1024)
		n, _ := resp.Body.Read(body)
		resp.Body.Close()
		if resp.StatusCode != 200 {
			t.Errorf("第%d次请求期望状态码 200, 实际 %d", i+1, resp.StatusCode)
		}
		bodyStr := strings.TrimSpace(string(body[:n]))
		if bodyStr != expected {
			t.Errorf("第%d次响应体不匹配:\n期望: %s\n实际: %s", i+1, expected, bodyStr)
		}
	}
}

// 回归测试: SetValue 存入 values map 的值必须能通过 Context().Value() 读取
// 根因: baseCtx.Context() 原先返回 b.ctx, 而 SetValue 存入 b.values, 两者不互通
func TestIntegration_SetValueVisibleViaContext(t *testing.T) {
	handlerSrc := `package user

import "testmod/routers"

type ctxKey string

const userKey ctxKey = "user"

// SetValueTest
// @RouterGroup /api
// @Router /setvalue[GET]
func SetValueTest(ctx routers.Ctx) any {
	ctx.SetValue(userKey, "alice")
	val := ctx.Context().Value(userKey)
	if val == nil {
		return routers.Err("value not found via Context().Value()")
	}
	s, ok := val.(string)
	if !ok || s != "alice" {
		return routers.Err("unexpected value")
	}
	return routers.Ok(s)
}
`
	expected := `{"code":0,"data":"alice"}`

	for _, spec := range integrationSpecs {
		t.Run(spec.engine, func(t *testing.T) {
			modDir := t.TempDir()
			writeFile(t, modDir, "go.mod", "module testmod\n\ngo 1.22\n")
			writeFile(t, modDir, "user/handler.go", handlerSrc)

			*WorkDir = modDir
			*OutDir = modDir
			*ModuleName = "testmod"
			*RouterGroup = ""
			Gen(engineGen{Name: spec.engine})

			writeFile(t, modDir, "routers/0__init___.go", spec.initContent)
			writeFile(t, modDir, "cmd/server/main.go", spec.mainContent)

			cmd := exec.Command("go", "mod", "tidy")
			cmd.Dir = modDir
			if out, err := cmd.CombinedOutput(); err != nil {
				t.Skipf("go mod tidy 失败 (可能无网络): %v\n%s", err, out)
			}

			binaryName := "testserver"
			if runtime.GOOS == "windows" {
				binaryName = "testserver.exe"
			}
			binaryPath := filepath.Join(modDir, binaryName)
			cmd = exec.Command("go", "build", "-o", binaryPath, "./cmd/server")
			cmd.Dir = modDir
			if out, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("go build 失败: %v\n%s", err, out)
			}

			port := freePort(t)
			addr := fmt.Sprintf("127.0.0.1:%d", port)
			cmd = exec.Command(binaryPath)
			cmd.Env = append(os.Environ(), "LISTEN_ADDR="+addr)
			if err := cmd.Start(); err != nil {
				t.Fatalf("启动服务失败: %v", err)
			}
			defer func() {
				_ = cmd.Process.Kill()
				_ = cmd.Wait()
			}()

			if !waitReady(t, addr, 5*time.Second) {
				t.Fatalf("服务未就绪: %s", addr)
			}

			resp, err := http.Get(fmt.Sprintf("http://%s/api/setvalue", addr))
			if err != nil {
				t.Fatalf("请求失败: %v", err)
			}
			body := make([]byte, 1024)
			n, _ := resp.Body.Read(body)
			resp.Body.Close()
			bodyStr := strings.TrimSpace(string(body[:n]))
			if bodyStr != expected {
				t.Fatalf("响应不匹配:\n期望: %s\n实际: %s", expected, bodyStr)
			}
		})
	}
}
