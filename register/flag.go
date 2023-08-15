package register

import "flag"

var (
    WorkDir       = flag.String("workDir", "./", "需要操作的目录")
    OutDir        = flag.String("outDir", "./", "输出目录")
    ModuleName    = flag.String("moduleName", "", "手动指定主module名称,否则读取go.mod文件夹")
    Annotation    = flag.String("annotation", "@relation", "需要识别的注释")
    FunctionName  = flag.String("functionName", "RegisterGenRelation", "方法名称")
    FileName      = flag.String("fileName", "0_register.gen.go", "生成文件名")
    FindFileRegex = flag.String("findFileRegex", ".+.go", "匹配的文件")
)
