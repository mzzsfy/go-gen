package enhance

import "flag"

var (
	Annotation    = flag.String("annotation", "@relation", "需要识别的注释标识")
	FunctionName  = flag.String("functionName", "RegisterGenRelation", "生成的方法/函数名称")
	FileName      = flag.String("fileName", "0_register.gen.go", "生成文件名")
	FindFileRegex = flag.String("findFileRegex", ".+.go", "匹配文件的 regex (非锚定, 支持部分匹配)")
	UsingPointers = flag.Bool("usingPointers", false, "生成是否使用指针类型")
)
