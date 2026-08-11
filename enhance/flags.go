package enhance

import "flag"

var (
	Annotation    = flag.String("annotation", "@relation", "需要识别的注释")
	FunctionName  = flag.String("functionName", "RegisterGenRelation", "方法名称")
	FileName      = flag.String("fileName", "0_register.gen.go", "生成文件名")
	FindFileRegex = flag.String("findFileRegex", ".+.go", "匹配的文件")
	UsingPointers = flag.Bool("usingPointers", false, "生成是否使用指针")
)
