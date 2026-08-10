package router

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/mzzsfy/go-gen/register"
)

// 集成测试: 生成真实 Go 服务, 编译并运行, 验证 HTTP 路由可访问.
// gin 和 echo 各一个子测试, 临时目录由 t.TempDir() 自动清理.

type engineSpec struct {
	engine      string
	initContent string // 覆盖 routers/0__init___.go, 替换 panic("fixme")
	mainContent string // cmd/server/main.go
}

func TestIntegration_BuildAndRun(t *testing.T) {
	specs := []engineSpec{
		{
			engine: "gin",
			initContent: `package routers

import "github.com/gin-gonic/gin"

var DefaultEngine = gin.New()

func init() {
	DefaultRouters["default"] = NewGinRouter(DefaultEngine.Group(""))
}
`,
			mainContent: `package main

import (
	"net/http"
	"os"
	"testmod/routers"
	_ "testmod/user"
)

func main() {
	addr := os.Getenv("LISTEN_ADDR")
	if addr == "" {
		addr = ":18080"
	}
	http.ListenAndServe(addr, routers.DefaultEngine)
}
`,
		},
		{
			engine: "echo",
			initContent: `package routers

import "github.com/labstack/echo/v4"

var DefaultEngine = echo.New()

func init() {
	DefaultRouters["default"] = NewEchoRouter(DefaultEngine.Group(""))
}
`,
			mainContent: `package main

import (
	"os"
	"testmod/routers"
	_ "testmod/user"
)

func main() {
	addr := os.Getenv("LISTEN_ADDR")
	if addr == "" {
		addr = ":18080"
	}
	routers.DefaultEngine.Start(addr)
}
`,
		},
	}

	for _, spec := range specs {
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
	*register.WorkDir = filepath.Join(modDir, "user")
	// WorkDir 需包含 go.mod 所在目录, 否则 findModuleName 找不到模块名
	// Gen 的 ParseDir 递归扫描 WorkDir, 设置为模块根目录
	*register.WorkDir = modDir
	*register.OutDir = modDir
	*register.ModuleName = "testmod"
	*register.RouterGroup = ""
	Gen(genGin{Name: spec.engine})

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

	// 发请求验证路由
	resp, err := http.Get(fmt.Sprintf("http://%s/api/info", addr))
	if err != nil {
		t.Fatalf("请求失败: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Errorf("期望状态码 200, 实际 %d", resp.StatusCode)
	}
	body := make([]byte, 1024)
	n, _ := resp.Body.Read(body)
	bodyStr := strings.TrimSpace(string(body[:n]))
	expected := `{"code":0,"data":"hello"}`
	if bodyStr != expected {
		t.Errorf("响应体不匹配:\n期望: %s\n实际: %s", expected, bodyStr)
	}
}

func writeFile(t *testing.T, modDir, relPath, content string) {
	t.Helper()
	full := filepath.Join(modDir, filepath.FromSlash(relPath))
	if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

func waitReady(t *testing.T, addr string, timeout time.Duration) bool {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		conn, err := net.Dial("tcp", addr)
		if err == nil {
			conn.Close()
			return true
		}
		time.Sleep(50 * time.Millisecond)
	}
	return false
}
