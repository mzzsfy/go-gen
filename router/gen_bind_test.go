package router

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// bind 测试: 验证 gin/echo 的 BindMultipleMap 和 BindBody 正确工作
// gin: 覆盖 noValidate 不 panic + MapFormWithTag
// echo: 覆盖 bindMapWithTag 全类型分支

// bindHandlerSrc 覆盖 query 绑定(各类型) 和 json body 绑定
const bindHandlerSrc = "package user\n" +
	"\n" +
	"import \"testmod/routers\"\n" +
	"\n" +
	"type QueryAPI struct {\n" +
	"\tName   string   `query:\"name\"`\n" +
	"\tAge    int      `query:\"age\"`\n" +
	"\tScore  float64  `query:\"score\"`\n" +
	"\tActive bool     `query:\"active\"`\n" +
	"\tTags   []string `query:\"tags\"`\n" +
	"\tCount  uint     `query:\"count\"`\n" +
	"}\n" +
	"\n" +
	"// QueryBind\n" +
	"// @RouterGroup /api\n" +
	"// @Router /query[GET]\n" +
	"func (q *QueryAPI) QueryBind(ctx routers.Ctx) any {\n" +
	"\treturn routers.Ok(map[string]any{\n" +
	"\t\t\"name\":   q.Name,\n" +
	"\t\t\"age\":    q.Age,\n" +
	"\t\t\"score\":  q.Score,\n" +
	"\t\t\"active\": q.Active,\n" +
	"\t\t\"tags\":   q.Tags,\n" +
	"\t\t\"count\":  q.Count,\n" +
	"\t})\n" +
	"}\n" +
	"\n" +
	"type BodyAPI struct {\n" +
	"\tName string `json:\"name\"`\n" +
	"\tAge  int    `json:\"age\"`\n" +
	"}\n" +
	"\n" +
	"// BodyBind\n" +
	"// @RouterGroup /api\n" +
	"// @Router /body[POST]\n" +
	"func (b *BodyAPI) BodyBind(ctx routers.Ctx) any {\n" +
	"\treturn routers.Ok(map[string]any{\n" +
	"\t\t\"name\": b.Name,\n" +
	"\t\t\"age\":  b.Age,\n" +
	"\t})\n" +
	"}\n"

func TestIntegration_Bind(t *testing.T) {
	for _, spec := range integrationSpecs {
		t.Run(spec.engine, func(t *testing.T) {
			testBindEngine(t, spec)
		})
	}
}

func testBindEngine(t *testing.T, spec engineSpec) {
	t.Helper()
	modDir := t.TempDir()
	writeFile(t, modDir, "go.mod", "module testmod\n\ngo 1.22\n")
	writeFile(t, modDir, "user/handler.go", bindHandlerSrc)

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

	// 1. query 绑定: 覆盖 string/int/float/bool/[]string/uint
	t.Run("query", func(t *testing.T) {
		url := fmt.Sprintf("http://%s/api/query?name=test&age=42&score=3.14&active=true&tags=a&tags=b&count=7", addr)
		resp, err := http.Get(url)
		if err != nil {
			t.Fatalf("请求失败: %v", err)
		}
		defer resp.Body.Close()
		body := readBindBody(t, resp)
		if resp.StatusCode != 200 {
			t.Fatalf("状态码 %d, body: %s", resp.StatusCode, body)
		}
		var res struct {
			Code int `json:"code"`
			Data struct {
				Name   string   `json:"name"`
				Age    int      `json:"age"`
				Score  float64  `json:"score"`
				Active bool     `json:"active"`
				Tags   []string `json:"tags"`
				Count  uint     `json:"count"`
			} `json:"data"`
		}
		if err := json.Unmarshal([]byte(body), &res); err != nil {
			t.Fatalf("解析响应失败: %v, body: %s", err, body)
		}
		if res.Code != 0 {
			t.Fatalf("code=%d, body: %s", res.Code, body)
		}
		if res.Data.Name != "test" {
			t.Errorf("Name: 期望 test, 实际 %s", res.Data.Name)
		}
		if res.Data.Age != 42 {
			t.Errorf("Age: 期望 42, 实际 %d", res.Data.Age)
		}
		if res.Data.Score != 3.14 {
			t.Errorf("Score: 期望 3.14, 实际 %f", res.Data.Score)
		}
		if !res.Data.Active {
			t.Errorf("Active: 期望 true, 实际 false")
		}
		if len(res.Data.Tags) != 2 || res.Data.Tags[0] != "a" || res.Data.Tags[1] != "b" {
			t.Errorf("Tags: 期望 [a b], 实际 %v", res.Data.Tags)
		}
		if res.Data.Count != 7 {
			t.Errorf("Count: 期望 7, 实际 %d", res.Data.Count)
		}
	})

	// 2. json body 绑定: gin 下覆盖 noValidate 不 panic
	t.Run("json_body", func(t *testing.T) {
		jsonBody := `{"name":"alice","age":30}`
		resp, err := http.Post(
			fmt.Sprintf("http://%s/api/body", addr),
			"application/json",
			strings.NewReader(jsonBody),
		)
		if err != nil {
			t.Fatalf("请求失败: %v", err)
		}
		defer resp.Body.Close()
		body := readBindBody(t, resp)
		if resp.StatusCode != 200 {
			t.Fatalf("状态码 %d, body: %s", resp.StatusCode, body)
		}
		var res struct {
			Code int `json:"code"`
			Data struct {
				Name string `json:"name"`
				Age  int    `json:"age"`
			} `json:"data"`
		}
		if err := json.Unmarshal([]byte(body), &res); err != nil {
			t.Fatalf("解析响应失败: %v, body: %s", err, body)
		}
		if res.Code != 0 {
			t.Fatalf("code=%d, body: %s", res.Code, body)
		}
		if res.Data.Name != "alice" {
			t.Errorf("Name: 期望 alice, 实际 %s", res.Data.Name)
		}
		if res.Data.Age != 30 {
			t.Errorf("Age: 期望 30, 实际 %d", res.Data.Age)
		}
	})

	// 3. 错误类型: 非法整数应返回错误响应 (DefaultResultHandler 对 error 固定 500)
	t.Run("invalid_int", func(t *testing.T) {
		url := fmt.Sprintf("http://%s/api/query?age=notanint", addr)
		resp, err := http.Get(url)
		if err != nil {
			t.Fatalf("请求失败: %v", err)
		}
		defer resp.Body.Close()
		body := readBindBody(t, resp)
		if resp.StatusCode != 500 {
			t.Errorf("非法 int 期望 500, 实际 %d", resp.StatusCode)
		}
		if !strings.Contains(body, "notanint") {
			t.Errorf("错误信息应包含 notanint, 实际: %s", body)
		}
	})
}

func readBindBody(t *testing.T, resp *http.Response) string {
	t.Helper()
	bs, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	return strings.TrimSpace(string(bs))
}
