package enhance

import (
	"path"
	"strings"

	"github.com/mzzsfy/go-gen/register"
)

func init() {
	register.Register("enhance-register", genRegister)
}

func genRegister() {
	workDir := path.Clean(*register.WorkDir)
	pkgName, structs, err := scanAnnotatedStructs(workDir, *FindFileRegex, *Annotation)
	if err != nil {
		panic(err)
	}
	if pkgName == "" {
		return
	}
	writeGeneratedFile(workDir, *FileName, pkgName, func(sb *strings.Builder) {
		sb.WriteString("\nfunc init() {\n")
		leftString := "["
		if *UsingPointers {
			leftString += "*"
		}
		for _, s := range structs {
			sb.WriteString("\t")
			sb.WriteString(*FunctionName)
			sb.WriteString(leftString)
			sb.WriteString(s.Name)
			sb.WriteString("]([]string{")
			sb.WriteString(strings.Join(s.Values, ","))
			sb.WriteString("})\n")
		}
		sb.WriteString("}\n")
	})
}
