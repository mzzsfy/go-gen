package enhance

import (
	"fmt"
	"os"
	"strings"
)

// writeGeneratedFile 写入生成的 Go 源文件, 包含统一的文件头和错误处理
func writeGeneratedFile(workDir, fileName, pkgName string, buildContent func(*strings.Builder)) {
	file, err := os.OpenFile(workDir+"/"+fileName, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		panic(err)
	}
	var sb strings.Builder
	sb.WriteString("//This is an auto-generated file, please do not edit it manually\n")
	sb.WriteString("//这是自动生成的文件,请不要手动编辑\n\n")
	sb.WriteString("package " + pkgName + "\n")
	buildContent(&sb)
	if _, err := file.WriteString(sb.String()); err != nil {
		file.Close()
		panic(err)
	}
	if err := file.Close(); err != nil {
		panic(err)
	}
	fmt.Printf("生成文件: %s/%s\n", workDir, fileName)
}
