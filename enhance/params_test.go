package enhance

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mzzsfy/go-gen/register"
)

// TestParams_AllDefaults 验证所有 flag 的默认值 (DefValue) 与预期一致
// 通过 flag.Lookup 读取注册时的 DefValue, 不受其他测试修改影响
func TestParams_AllDefaults(t *testing.T) {
	tests := []struct {
		flag string
		want string
	}{
		{"annotation", "@relation"},
		{"functionName", "RegisterGenRelation"},
		{"fileName", "0_register.gen.go"},
		{"findFileRegex", ".+.go"},
		{"usingPointers", "false"},
		{"workDir", "./"},
		{"outDir", "./"},
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

// TestFindFileRegex_CustomValue 验证自定义 findFileRegex 只扫描匹配文件
func TestFindFileRegex_CustomValue(t *testing.T) {
	workDir := t.TempDir()
	// model_user.go 匹配 `model.+\.go`, handler.go 不匹配
	files := map[string]string{
		"model_user.go": `package src

// @relation user
type User struct{}
`,
		"handler.go": `package src

// @relation handler
type Handler struct{}
`,
	}
	for name, content := range files {
		full := filepath.Join(workDir, name)
		if err := os.WriteFile(full, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	*register.WorkDir = workDir
	*register.OutDir = workDir
	*Annotation = "@relation"
	*FunctionName = "RegisterGenRelation"
	*FileName = "0_register.gen.go"
	*FindFileRegex = `model.+\.go`
	*UsingPointers = true

	genRegister()

	bs, err := os.ReadFile(filepath.Join(workDir, *FileName))
	if err != nil {
		t.Fatalf("读取生成文件失败: %v", err)
	}
	content := string(bs)

	// User 在 model_user.go 中, 应被扫描
	if !strings.Contains(content, "User") {
		t.Errorf("model_user.go 应被匹配, User 应出现, 内容:\n%s", content)
	}
	// Handler 在 handler.go 中, 不应被扫描
	if strings.Contains(content, "Handler") {
		t.Errorf("handler.go 不应被匹配, Handler 不应出现, 内容:\n%s", content)
	}
}

// TestFindFileRegex_DefaultValue 验证默认 findFileRegex (.+.go) 匹配所有 .go 文件
func TestFindFileRegex_DefaultValue(t *testing.T) {
	workDir := t.TempDir()
	files := map[string]string{
		"a.go": `package src

// @relation alpha
type Alpha struct{}
`,
		"b.go": `package src

// @relation beta
type Beta struct{}
`,
	}
	for name, content := range files {
		full := filepath.Join(workDir, name)
		if err := os.WriteFile(full, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	*register.WorkDir = workDir
	*register.OutDir = workDir
	*Annotation = "@relation"
	*FunctionName = "RegisterGenRelation"
	*FileName = "0_register.gen.go"
	*FindFileRegex = ".+.go"
	*UsingPointers = true

	genRegister()

	bs, err := os.ReadFile(filepath.Join(workDir, *FileName))
	if err != nil {
		t.Fatalf("读取生成文件失败: %v", err)
	}
	content := string(bs)

	if !strings.Contains(content, "Alpha") {
		t.Errorf("默认 regex 应匹配 a.go, 内容:\n%s", content)
	}
	if !strings.Contains(content, "Beta") {
		t.Errorf("默认 regex 应匹配 b.go, 内容:\n%s", content)
	}
}

// TestFindFileRegex_PartialMatch 验证 findFileRegex 支持部分匹配 (非锚定)
// 用户可只指定文件名的一部分, 如 "model" 匹配所有含 model 的 .go 文件
func TestFindFileRegex_PartialMatch(t *testing.T) {
	workDir := t.TempDir()
	files := map[string]string{
		"model_user.go":   `package src` + "\n\n// @relation user\ntype User struct{}\n",
		"model_order.go":  `package src` + "\n\n// @relation order\ntype Order struct{}\n",
		"unrelated.go":    `package src` + "\n\n// @relation skip\ntype Skip struct{}\n",
	}
	for name, content := range files {
		full := filepath.Join(workDir, name)
		if err := os.WriteFile(full, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	*register.WorkDir = workDir
	*register.OutDir = workDir
	*Annotation = "@relation"
	*FunctionName = "RegisterGenRelation"
	*FileName = "0_register.gen.go"
	*FindFileRegex = `model`
	*UsingPointers = true

	genRegister()

	bs, err := os.ReadFile(filepath.Join(workDir, *FileName))
	if err != nil {
		t.Fatalf("读取生成文件失败: %v", err)
	}
	content := string(bs)

	if !strings.Contains(content, "User") {
		t.Errorf("部分匹配 model 应包含 model_user.go 的 User, 内容:\n%s", content)
	}
	if !strings.Contains(content, "Order") {
		t.Errorf("部分匹配 model 应包含 model_order.go 的 Order, 内容:\n%s", content)
	}
	if strings.Contains(content, "Skip") {
		t.Errorf("unrelated.go 不含 model, 不应被匹配, 内容:\n%s", content)
	}
}
func TestAnnotation_DefaultAndCustom(t *testing.T) {
	workDir := t.TempDir()
	src := `package src

// @relation default
type DefaultStruct struct{}

// @custom customVal
type CustomStruct struct{}
`
	if err := os.WriteFile(filepath.Join(workDir, "models.go"), []byte(src), 0644); err != nil {
		t.Fatal(err)
	}

	// 默认 annotation (@relation): 只找到 DefaultStruct
	*register.WorkDir = workDir
	*register.OutDir = workDir
	*Annotation = "@relation"
	*FunctionName = "RegisterGenRelation"
	*FileName = "0_register.gen.go"
	*FindFileRegex = ".+.go"
	*UsingPointers = true

	genRegister()
	bs, _ := os.ReadFile(filepath.Join(workDir, *FileName))
	content := string(bs)
	if !strings.Contains(content, "DefaultStruct") {
		t.Errorf("默认 annotation 应找到 DefaultStruct, 内容:\n%s", content)
	}
	if strings.Contains(content, "CustomStruct") {
		t.Errorf("默认 annotation 不应找到 CustomStruct, 内容:\n%s", content)
	}

	// 清理后用自定义 annotation (@custom): 只找到 CustomStruct
	os.Remove(filepath.Join(workDir, *FileName))
	*Annotation = "@custom"
	genRegister()
	bs, _ = os.ReadFile(filepath.Join(workDir, *FileName))
	content = string(bs)
	if !strings.Contains(content, "CustomStruct") {
		t.Errorf("自定义 annotation 应找到 CustomStruct, 内容:\n%s", content)
	}
	if strings.Contains(content, "DefaultStruct") {
		t.Errorf("自定义 annotation 不应找到 DefaultStruct, 内容:\n%s", content)
	}
}

// TestUsingPointers_DefaultAndCustom 验证 usingPointers 默认值(false)与自定义值(true)
func TestUsingPointers_DefaultAndCustom(t *testing.T) {
	workDir := t.TempDir()
	src := `package src

// @relation user
type User struct{}
`
	if err := os.WriteFile(filepath.Join(workDir, "models.go"), []byte(src), 0644); err != nil {
		t.Fatal(err)
	}

	baseFlags := func() {
		*register.WorkDir = workDir
		*register.OutDir = workDir
		*Annotation = "@relation"
		*FunctionName = "RegisterGenRelation"
		*FileName = "0_register.gen.go"
		*FindFileRegex = ".+.go"
	}

	// 默认 false: 生成 [User] 而非 [*User]
	baseFlags()
	*UsingPointers = false
	genRegister()
	bs, _ := os.ReadFile(filepath.Join(workDir, *FileName))
	content := string(bs)
	if strings.Contains(content, "[*User]") {
		t.Errorf("默认 usingPointers=false 不应有指针, 内容:\n%s", content)
	}
	if !strings.Contains(content, "[User]") {
		t.Errorf("应有 [User], 内容:\n%s", content)
	}

	// 自定义 true: 生成 [*User]
	os.Remove(filepath.Join(workDir, *FileName))
	baseFlags()
	*UsingPointers = true
	genRegister()
	bs, _ = os.ReadFile(filepath.Join(workDir, *FileName))
	content = string(bs)
	if !strings.Contains(content, "[*User]") {
		t.Errorf("usingPointers=true 应有 [*User], 内容:\n%s", content)
	}
}
