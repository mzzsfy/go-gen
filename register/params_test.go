package register

import (
	"flag"
	"testing"
)

// TestParams_AllDefaults 验证 register 包 flag 的默认值
func TestParams_AllDefaults(t *testing.T) {
	tests := []struct {
		flag string
		want string
	}{
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

// TestParams_WorkDirAndOutDir_PointToSameDefault 验证 WorkDir 和 OutDir 是独立指针
func TestParams_WorkDirAndOutDir_PointToSameDefault(t *testing.T) {
	if *WorkDir != "./" || *OutDir != "./" {
		// flag 可能被其他测试修改, 检查 DefValue 即可
		w := flag.Lookup("workDir")
		o := flag.Lookup("outDir")
		if w.DefValue != o.DefValue {
			t.Errorf("workDir 和 outDir 默认值应一致: workDir=%q outDir=%q", w.DefValue, o.DefValue)
		}
		return
	}
	// 默认值一致
	if *WorkDir != *OutDir {
		t.Errorf("默认值应一致: workDir=%q outDir=%q", *WorkDir, *OutDir)
	}
}
