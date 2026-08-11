package register

import "flag"

var (
	WorkDir = flag.String("workDir", "./", "扫描源码的根目录, 支持非 module root (自动向上递归查找 go.mod)")
	OutDir  = flag.String("outDir", "./", "核心文件和 import 文件的输出目录, 不影响各包的 0_router___.go")
)
