package enhance

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mzzsfy/go-gen/register"
)

func enhanceTestdataDir() string {
	abs, _ := filepath.Abs("testdata/src")
	return filepath.Clean(abs)
}

// copyFixtures 把 fixture 文件复制到临时目录, 因为 enhance 的读写都在 WorkDir
func copyFixtures(t *testing.T, dst string) {
	t.Helper()
	entries, err := os.ReadDir(enhanceTestdataDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") {
			continue
		}
		bs, err := os.ReadFile(filepath.Join(enhanceTestdataDir(), e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		// 排除已有的生成文件
		if strings.Contains(e.Name(), ".gen.go") {
			continue
		}
		os.WriteFile(filepath.Join(dst, e.Name()), bs, 0644)
	}
}

// TestGenRegister 验证结构体注册生成
func TestGenRegister(t *testing.T) {
	workDir := t.TempDir()
	copyFixtures(t, workDir)
	setupEnhanceFlags(t, workDir)

	genRegister()

	bs, err := os.ReadFile(filepath.Join(workDir, *register.FileName))
	if err != nil {
		t.Fatalf("读取生成文件失败: %v", err)
	}
	content := string(bs)

	// 验证包声明和 init 函数
	if !strings.Contains(content, "package src") {
		t.Error("缺少 package 声明")
	}
	if !strings.Contains(content, "func init() {") {
		t.Error("缺少 init 函数")
	}

	// User 结构体: @relation user
	if !strings.Contains(content, "User") || !strings.Contains(content, `"user"`) {
		t.Errorf("User 注册缺失, 内容:\n%s", content)
	}
	// Order 结构体: @relation order,product
	if !strings.Contains(content, "Order") || !strings.Contains(content, `"order"`) || !strings.Contains(content, `"product"`) {
		t.Errorf("Order 注册缺失, 内容:\n%s", content)
	}
}

// TestAddFunction 验证方法注册生成
func TestAddFunction(t *testing.T) {
	workDir := t.TempDir()
	copyFixtures(t, workDir)
	setupEnhanceFlags(t, workDir)
	*register.FunctionName = "GetRelations"
	*register.FileName = "0_addfunction.gen.go"

	addFunction()

	bs, err := os.ReadFile(filepath.Join(workDir, *register.FileName))
	if err != nil {
		t.Fatalf("读取生成文件失败: %v", err)
	}
	content := string(bs)

	if !strings.Contains(content, "package src") {
		t.Error("缺少 package 声明")
	}
	// addFunction 生成方法签名: func (*Type) GetRelations() []string
	if !strings.Contains(content, "func (*User) GetRelations() []string {") {
		t.Errorf("User 方法注册缺失, 内容:\n%s", content)
	}
	if !strings.Contains(content, "func (*Order) GetRelations() []string {") {
		t.Errorf("Order 方法注册缺失, 内容:\n%s", content)
	}
	// 验证返回值
	if !strings.Contains(content, `[]string{"user"}`) {
		t.Error("User relations 返回值错误")
	}
	if !strings.Contains(content, `[]string{"order","product"}`) {
		t.Error("Order relations 返回值错误")
	}
}

func setupEnhanceFlags(t *testing.T, workDir string) {
	t.Helper()
	*register.WorkDir = workDir
	*register.OutDir = workDir
	*register.Annotation = "@relation"
	*register.FunctionName = "RegisterGenRelation"
	*register.FileName = "0_register.gen.go"
	*register.FindFileRegex = ".+.go"
	*register.UsingPointers = true
}

// TestGenRegister_SkipsUnannotated 验证非结构体类型被跳过, 未匹配注解的结构体无注册值
func TestGenRegister_SkipsUnannotated(t *testing.T) {
	edgeDir, _ := filepath.Abs(filepath.Join("testdata", "edge"))
	edgeDir = filepath.Clean(edgeDir)
	setupEnhanceFlags(t, edgeDir)

	genPath := filepath.Join(edgeDir, *register.FileName)
	// 清理可能残留的生成文件
	os.Remove(genPath)
	t.Cleanup(func() {
		os.Remove(genPath)
	})

	genRegister()

	bs, err := os.ReadFile(genPath)
	if err != nil {
		t.Fatalf("读取生成文件失败: %v", err)
	}
	content := string(bs)

	// Tagged 有 @relation tagged1, 应被注册
	if !strings.Contains(content, "Tagged") || !strings.Contains(content, `"tagged1"`) {
		t.Errorf("Tagged 注册缺失, 内容:\n%s", content)
	}
	// 非结构体类型不应出现在生成代码中
	if strings.Contains(content, "MyInt") {
		t.Errorf("MyInt 非结构体, 不应被注册, 内容:\n%s", content)
	}
	if strings.Contains(content, "MyIface") {
		t.Errorf("MyIface 非结构体, 不应被注册, 内容:\n%s", content)
	}
	// Skipped 有文档注释但无匹配注解, 不应有注解值
	if strings.Contains(content, `Skipped]([]string{"`) {
		t.Errorf("Skipped 无匹配注解, 不应有注解值, 内容:\n%s", content)
	}
}

// TestGenRegister_UsingPointersFalse 验证 UsingPointers=false 时不生成指针符号
func TestGenRegister_UsingPointersFalse(t *testing.T) {
	workDir := t.TempDir()
	copyFixtures(t, workDir)
	setupEnhanceFlags(t, workDir)
	*register.UsingPointers = false

	genRegister()

	bs, err := os.ReadFile(filepath.Join(workDir, *register.FileName))
	if err != nil {
		t.Fatalf("读取生成文件失败: %v", err)
	}
	content := string(bs)

	// pointers=false 时 leftString 是 [ 而非 [*
	if strings.Contains(content, "[*") {
		t.Errorf("不应包含指针符号 [*], 内容:\n%s", content)
	}
	if !strings.Contains(content, "[User]") {
		t.Errorf("应包含 [User], 内容:\n%s", content)
	}
	if !strings.Contains(content, "[Order]") {
		t.Errorf("应包含 [Order], 内容:\n%s", content)
	}
}

// TestAddFunction_UsingPointersFalse 验证 addFunction 在 UsingPointers=false 时方法接收者为值类型
func TestAddFunction_UsingPointersFalse(t *testing.T) {
	workDir := t.TempDir()
	copyFixtures(t, workDir)
	setupEnhanceFlags(t, workDir)
	*register.UsingPointers = false
	*register.FunctionName = "GetRelations"
	*register.FileName = "0_addfunction.gen.go"

	addFunction()

	bs, err := os.ReadFile(filepath.Join(workDir, *register.FileName))
	if err != nil {
		t.Fatalf("读取生成文件失败: %v", err)
	}
	content := string(bs)

	// pointers=false 时方法签名为 func (User) 而非 func (*User)
	if !strings.Contains(content, "func (User) GetRelations() []string {") {
		t.Errorf("User 方法签名错误, 应为值接收者, 内容:\n%s", content)
	}
	if !strings.Contains(content, "func (Order) GetRelations() []string {") {
		t.Errorf("Order 方法签名错误, 应为值接收者, 内容:\n%s", content)
	}
}

// TestGenRegister_CustomAnnotation 验证自定义注解标识符
func TestGenRegister_CustomAnnotation(t *testing.T) {
	workDir := t.TempDir()
	setupEnhanceFlags(t, workDir)
	*register.Annotation = "@mytag"

	// 创建带 @mytag 注解的 fixture
	src := `package src

// @mytag user
type User struct{}

// @mytag order,product
type Order struct{}
`
	if err := os.WriteFile(filepath.Join(workDir, "models.go"), []byte(src), 0644); err != nil {
		t.Fatal(err)
	}

	genRegister()

	bs, err := os.ReadFile(filepath.Join(workDir, *register.FileName))
	if err != nil {
		t.Fatalf("读取生成文件失败: %v", err)
	}
	content := string(bs)

	if !strings.Contains(content, "User") || !strings.Contains(content, `"user"`) {
		t.Errorf("User 注册缺失, 内容:\n%s", content)
	}
	if !strings.Contains(content, "Order") || !strings.Contains(content, `"order"`) || !strings.Contains(content, `"product"`) {
		t.Errorf("Order 注册缺失, 内容:\n%s", content)
	}
}

// TestGenRegister_MultipleValues 验证逗号分隔的多值注解解析
func TestGenRegister_MultipleValues(t *testing.T) {
	workDir := t.TempDir()
	copyFixtures(t, workDir)
	setupEnhanceFlags(t, workDir)

	genRegister()

	bs, err := os.ReadFile(filepath.Join(workDir, *register.FileName))
	if err != nil {
		t.Fatalf("读取生成文件失败: %v", err)
	}
	content := string(bs)

	// Order 有 @relation order,product, 应生成多值注册
	if !strings.Contains(content, `[]string{"order","product"}`) {
		t.Errorf("多值注解解析错误, 内容:\n%s", content)
	}
}
