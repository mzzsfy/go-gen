package enhance

import (
    "github.com/mzzsfy/go-gen/register"
    "go/ast"
    "go/parser"
    "go/token"
    "os"
    "path"
    "regexp"
    "strings"
    "sync"
)

func init() {
    register.Register("enhance-register", genRegister)
}

func genRegister() {
    reg, _ := regexp.Compile(".*")
    {
        expr := *register.FindFileRegex
        if expr != "" {
            var err error
            reg, err = regexp.Compile(expr)
            if err != nil {
                panic(err)
            }
        }
    }
    workDir := *register.WorkDir
    workDir = path.Clean(workDir)
    dir, err := os.ReadDir(workDir)
    if err != nil {
        panic(err)
    }
    file, err := os.OpenFile(workDir+"/"+*register.FileName, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.ModePerm)
    if err != nil {
        panic(err)
    } else {
        println("生成文件: " + workDir + "/" + *register.FileName)
    }
    defer func() {
        file.Write([]byte("}\n"))
        file.Close()
    }()
    once := sync.Once{}
    fileSet := token.NewFileSet()
    for _, f := range dir {
        if reg.MatchString(f.Name()) {
            parseFile, err := parser.ParseFile(fileSet, workDir+"/"+f.Name(), nil, parser.ParseComments)
            if err != nil {
                println("跳过文件: " + f.Name() + ", 错误: " + err.Error())
                continue
            }
            once.Do(func() {
                file.Write([]byte(`//This is an auto-generated file, please do not edit it manually
//这是自动生成的文件,请不要手动编辑

package ` + parseFile.Name.Name + "\n\nfunc init() {\n"))
            })
            for _, dx := range parseFile.Decls {
                switch d := dx.(type) {
                case *ast.GenDecl:
                    for _, d1 := range d.Specs {
                        switch d2 := d1.(type) {
                        case *ast.TypeSpec:
                            switch d2.Type.(type) {
                            case *ast.StructType:
                                if d2.Comment == nil && d.Doc == nil {
                                    continue
                                }
                                var commentGroup *ast.CommentGroup
                                if d2.Comment == nil {
                                    commentGroup = d.Doc
                                } else {
                                    commentGroup = d2.Comment
                                }
                                var values []string
                                for _, comment := range commentGroup.List {
                                    text := strings.TrimSpace(comment.Text[2:])
                                    if strings.HasPrefix(text, *register.Annotation) {
                                        ss := strings.TrimSpace(text[len(*register.Annotation)+1:])
                                        for _, s := range strings.Split(ss, ",") {
                                            values = append(values, `"`+s+`"`)
                                        }
                                    }
                                }
                                file.Write([]byte("    " + *register.FunctionName + "[" + d2.Name.Name + "]([]string{" + strings.Join(values, ",") + "})\n"))
                            }
                        }
                    }
                }
            }
        }
    }
}
