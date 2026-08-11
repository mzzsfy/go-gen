package router

import (
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// engineSpec 描述一个引擎的集成测试配置
type engineSpec struct {
	engine      string
	initContent string
	mainContent string
}

var integrationSpecs = []engineSpec{
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
