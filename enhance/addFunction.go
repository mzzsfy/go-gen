package enhance

import (
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
	writeGeneratedFile(workDir, *FileName, pkgName, func(sb *strings.Builder) {
		recvPrefix := ""
		if *UsingPointers {
			recvPrefix = "*"
		}
		for _, s := range structs {
			sb.WriteString("\nfunc (")
			sb.WriteString(recvPrefix)
			sb.WriteString(s.Name)
			sb.WriteString(") ")
			sb.WriteString(*FunctionName)
			sb.WriteString("() []string {\n\treturn []string{")
			sb.WriteString(strings.Join(s.Values, ","))
			sb.WriteString("}\n}\n")
		}
	})
}
