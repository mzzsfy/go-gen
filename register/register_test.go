package register

import "testing"

func TestRegisterAndDo(t *testing.T) {
	called := false
	Register("test-gen", func() {
		called = true
	})
	Do("test-gen")
	if !called {
		t.Error("Do 未调用注册的函数")
	}
}

func TestDo_UnknownName(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("未知 genType 应 panic")
		}
	}()
	Do("nonexistent-gen-type")
}

func TestRegister_Overwrite(t *testing.T) {
	v1 := false
	v2 := false
	Register("overwrite-test", func() { v1 = true })
	Register("overwrite-test", func() { v2 = true })
	Do("overwrite-test")
	if v1 {
		t.Error("旧函数不应被调用")
	}
	if !v2 {
		t.Error("新函数应被调用")
	}
}
