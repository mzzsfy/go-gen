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

// TestTemplate_NoMultiWriteBug 验证模板中不存在丢弃多次写入的bug模式
// 根因: writePrepare 曾有 if w.Written() && w.size > 0 { return false },
// 导致首次写入后所有后续 Write/WriteString 返回 0,nil 丢弃数据
func TestTemplate_NoMultiWriteBug(t *testing.T) {
	content := readCoreTemplate(t, "0_context___.gotmp")

	// 不应存在阻止多次写入的早退逻辑
	bugPatterns := []string{
		"w.size > 0",
		"return 0, nil",
		"return false",
	}
	for _, p := range bugPatterns {
		if strings.Contains(content, p) {
			t.Errorf("模板不应包含阻止多次写入的代码: %q", p)
		}
	}

	// writePrepare 应无返回值 (不再有 bool 返回)
	if strings.Contains(content, "writePrepare() bool") {
		t.Error("writePrepare 不应有 bool 返回值")
	}

	// Write 和 WriteString 应直接调用 writePrepare(), 不检查返回值
	if strings.Contains(content, "if !w.writePrepare()") {
		t.Error("Write/WriteString 不应检查 writePrepare 返回值")
	}
}

// TestTemplate_WriteAccumulatesSize 验证模板中 Write 累加 size 的逻辑存在
func TestTemplate_WriteAccumulatesSize(t *testing.T) {
	content := readCoreTemplate(t, "0_context___.gotmp")

	// 每次 Write 后 size 应累加, 不是赋值
	if !strings.Contains(content, "w.size += n") {
		t.Error("Write 应累加 size (w.size += n), 支持多次写入")
	}
}

// TestIntegration_MultipleWrites 集成测试: 多次 Write 的数据完整到达客户端
// 回归: responseWriter.Write 曾在首次写入后丢弃后续写入
func TestIntegration_MultipleWrites(t *testing.T) {
	handlerSrc := `package user

import (
	"testmod/routers"
)

// MultiWrite
// @RouterGroup /api
// @Router /multi[GET]
func MultiWrite(ctx routers.Ctx) any {
	ctx.Status(200)
	ctx.Response().Write([]byte("part1-"))
	ctx.Response().Write([]byte("part2-"))
	ctx.Response().Write([]byte("part3"))
	return routers.ResultDoNothing
}
`
	expected := "part1-part2-part3"

	for _, spec := range integrationSpecs {
		t.Run(spec.engine, func(t *testing.T) {
			runIntegrationTest(t, spec, handlerSrc, "/api/multi", expected)
		})
	}
}

// TestIntegration_StreamLargeBody 集成测试: Stream (io.Copy 内部多次 Write) 数据完整
// 回归: Stream 底层 io.Copy 分块读取写入, bug 导致只有第一块到达客户端
func TestIntegration_StreamLargeBody(t *testing.T) {
	// 生成足够长的内容, 迫使 io.Copy 分多块写入
	longBody := strings.Repeat("A", 32*1024)
	handlerSrc := fmt.Sprintf(`package user

import (
	"strings"
	"testmod/routers"
)

// StreamTest
// @RouterGroup /api
// @Router /stream[GET]
func StreamTest(ctx routers.Ctx) any {
	ctx.Stream(200, "text/plain", strings.NewReader(%q))
	return routers.ResultDoNothing
}
`, longBody)

	for _, spec := range integrationSpecs {
		t.Run(spec.engine, func(t *testing.T) {
			runIntegrationTest(t, spec, handlerSrc, "/api/stream", longBody)
		})
	}
}

// TestIntegration_ContentLengthConsistency 验证多次 Write 后 Content-Length 与实际 body 一致
// 回归: responseWriter 曾丢弃后续写入, 导致 size 计数小于实际 body
func TestIntegration_ContentLengthConsistency(t *testing.T) {
	// 构造可预测的分段写入, 总长度已知
	chunk := "0123456789"
	repeat := 5 // 总长 50
	handlerSrc := fmt.Sprintf(`package user

import (
	"testmod/routers"
)

// ContentLengthTest
// @RouterGroup /api
// @Router /clen[GET]
func ContentLengthTest(ctx routers.Ctx) any {
	ctx.Status(200)
	for i := 0; i < %d; i++ {
		ctx.Response().Write([]byte(%q))
	}
	return routers.ResultDoNothing
}
`, repeat, chunk)
	expectedBody := strings.Repeat(chunk, repeat)

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

			resp, err := http.Get(fmt.Sprintf("http://%s/api/clen", addr))
			if err != nil {
				t.Fatalf("请求失败: %v", err)
			}
			defer resp.Body.Close()
			body := readFullBody(t, resp)

			// 实际 body 必须等于预期 (不被截断)
			if body != expectedBody {
				t.Errorf("body 不匹配:\n期望(%d): %s\n实际(%d): %s", len(expectedBody), expectedBody, len(body), body)
			}

			// Content-Length 必须与实际 body 长度一致
			cl := resp.Header.Get("Content-Length")
			if cl != "" && cl != fmt.Sprintf("%d", len(expectedBody)) {
				t.Errorf("Content-Length 不一致: header=%s, body实际=%d", cl, len(body))
			}
		})
	}
}

// runIntegrationTest 是 gin/echo 集成测试的公共流程
func runIntegrationTest(t *testing.T, spec engineSpec, handlerSrc, path, expectedBody string) {
	t.Helper()
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

	resp, err := http.Get(fmt.Sprintf("http://%s%s", addr, path))
	if err != nil {
		t.Fatalf("请求失败: %v", err)
	}
	defer resp.Body.Close()
	body := readFullBody(t, resp)
	if body != expectedBody {
		t.Fatalf("响应体不匹配:\n期望(%d): %s\n实际(%d): %s", len(expectedBody), expectedBody, len(body), body)
	}
}

// TestIntegration_ManualStreamWithFlush 验证手动模拟 stream (循环 Write + Flush) 每次刷新数据都正确到达客户端
// 回归: responseWriter 曾丢弃首次写入后的所有 Write, 且 Flush 依赖 WriteHeaderNow 链路
func TestIntegration_ManualStreamWithFlush(t *testing.T) {
	// 模拟 SSE / 流式推送: 写入 chunk 后立即 Flush, 重复多次
	chunkCount := 5
	handlerSrc := fmt.Sprintf(`package user

import (
	"fmt"
	"testmod/routers"
)

// ManualStream
// @RouterGroup /api
// @Router /manual[GET]
func ManualStream(ctx routers.Ctx) any {
	flusher, ok := ctx.Response().(interface{ Flush() })
	if !ok {
		ctx.String(500, "no flusher")
		return routers.ResultDoNothing
	}
	ctx.Status(200)
	for i := 0; i < %d; i++ {
		ctx.Response().Write([]byte(fmt.Sprintf("chunk-%%d\n", i)))
		flusher.Flush()
	}
	return routers.ResultDoNothing
}
`, chunkCount)

	// 所有 chunk 拼接
	var expected strings.Builder
	for i := 0; i < chunkCount; i++ {
		expected.WriteString(fmt.Sprintf("chunk-%d\n", i))
	}

	for _, spec := range integrationSpecs {
		t.Run(spec.engine, func(t *testing.T) {
			runIntegrationTest(t, spec, handlerSrc, "/api/manual", expected.String())
		})
	}
}

func readFullBody(t *testing.T, resp *http.Response) string {
	t.Helper()
	buf := make([]byte, 0, 8192)
	tmp := make([]byte, 4096)
	for {
		n, err := resp.Body.Read(tmp)
		if n > 0 {
			buf = append(buf, tmp[:n]...)
		}
		if err != nil {
			break
		}
	}
	return string(buf)
}
