package protobuf

import (
    "github.com/mzzsfy/go-gen/register"
    "go/ast"
    "go/parser"
    "go/token"
    "os"
    "strings"
    "sync"
    "text/template"
)

func init() {
    register.Register("protobuf-register", genRegister)
}

var (
    genTemplate = func() *template.Template {
        return template.Must(template.New("").Parse(`func init() {
	t := &{{.}}{}
	Register(t)
}`))
    }()
)

func genRegister() {
    workDir := *register.WorkDir
    dir, err := os.ReadDir(workDir)
    if err != nil {
        panic(err)
    }
    file, err := os.OpenFile("template/gen_register.go", os.O_CREATE|os.O_WRONLY, os.ModePerm)
    if err != nil {
        panic(err)
    }
    defer file.Close()
    once := sync.Once{}
    fileSet := token.NewFileSet()
    for _, f := range dir {
        if strings.HasSuffix(f.Name(), ".pb.go") {
            parseFile, err := parser.ParseFile(fileSet, f.Name(), nil, 0)
            if err != nil {
                println("跳过文件: " + f.Name() + ", 错误: " + err.Error())
                continue
            }
            once.Do(func() {
                file.Write([]byte(`//This is an auto-generated file, please do not edit it manually
//这是自动生成的文件,请不要手动编辑
package ` + parseFile.Name.Name + "\n\n"))
            })
            for _, dx := range parseFile.Decls {
                switch d := dx.(type) {
                case *ast.GenDecl:
                    for _, d1 := range d.Specs {
                        switch d2 := d1.(type) {
                        case *ast.TypeSpec:
                            switch d2.Type.(type) {
                            case *ast.StructType:
                                genTemplate.Execute(file, d2.Name.Name)
                            }

                        }
                    }
                }
            }
            genTemplate.Execute(file, f.Name())
        }
    }
}
