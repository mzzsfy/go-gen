package protobuf

import (
    "github.com/mzzsfy/go-gen/register"
    "go/ast"
    "go/parser"
    "go/token"
    "os"
    "path"
    "strings"
    "sync"
    "text/template"
)

func init() {
    register.Register("protobuf-register", genRegister)
}

var (
    genTemplate = func() *template.Template {
        return template.Must(template.New("").Parse(`    {
        t := &{{.}}{}
        RegisterProtobufGen(t)
    }
`))
    }()
)

func genRegister() {
    workDir := *register.WorkDir
    workDir = path.Clean(workDir)
    dir, err := os.ReadDir(workDir)
    if err != nil {
        panic(err)
    }
    file, err := os.OpenFile(workDir+"gen_register.go", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.ModePerm)
    if err != nil {
        panic(err)
    }
    defer func() {
        file.Write([]byte("}\n"))
        file.Close()
    }()
    once := sync.Once{}
    fileSet := token.NewFileSet()
    for _, f := range dir {
        if strings.HasSuffix(f.Name(), ".pb.go") {
            parseFile, err := parser.ParseFile(fileSet, workDir+"/"+f.Name(), nil, 0)
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
                                genTemplate.Execute(file, d2.Name.Name)
                            }
                        }
                    }
                }
            }
        }
    }
}
