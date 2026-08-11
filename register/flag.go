package register

import "flag"

var (
	WorkDir = flag.String("workDir", "./", "需要操作的目录")
	OutDir  = flag.String("outDir", "./", "输出目录")
)
