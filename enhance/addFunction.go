package enhance

import (
	"os"
	"path"
	"strings"

	"github.com/mzzsfy/go-gen/register"
)

func init() {
	register.Register("enhance-addFunction", addFunction)
}

func addFunction() {
	workDir := path.Clean(*register.WorkDir)
	pkgName, structs, err := scanAnnotatedStructs(workDir, *FindFileRegex, *Annotation)
	if err != nil {
		panic(err)
	}
	if pkgName == "" {
		return
	}
	file, err := os.OpenFile(workDir+"/"+*FileName, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.ModePerm)
	if err != nil {
		panic(err)
	}
	defer file.Close()
	println("生成文件: " + workDir + "/" + *FileName)

	recvPrefix := ""
	if *UsingPointers {
		recvPrefix = "*"
	}
	var sb strings.Builder
	sb.WriteString("//This is an auto-generated file, please do not edit it manually\n")
	sb.WriteString("//这是自动生成的文件,请不要手动编辑\n\n")
	sb.WriteString("package " + pkgName + "\n")
	for _, s := range structs {
		sb.WriteString("\nfunc (")
		sb.WriteString(recvPrefix)
		sb.WriteString(s.Name)
		sb.WriteString(") ")
		sb.WriteString(*FunctionName)
		sb.WriteString("() []string {\n	return []string{")
		sb.WriteString(strings.Join(s.Values, ","))
		sb.WriteString("}\n}\n")
	}
	file.Write([]byte(sb.String()))
}
