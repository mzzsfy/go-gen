package router

import (
    "bytes"
    _ "embed"
    "fmt"
    "github.com/mzzsfy/go-gen/register"
    "go/ast"
    "go/parser"
    "go/token"
    "io/fs"
    "os"
    "path"
    "path/filepath"
    "slices"
    "strings"
    "text/template"
)

var (
    routerAnnotation      = "@Router"
    routerGroupAnnotation = "@RouterGroup"
    //go:embed template/ginByPackage.gotemp
    ginByPackageTemplate []byte
    //go:embed template/main.gotemp
    mainTemplate []byte
)

type FileInfo struct {
    Path string
}
type HttpPath struct {
    Path       string
    Method     string
    PathMethod string
}
type Function struct {
    FileInfo
    GroupPath string
    Name      string
    Paths     []HttpPath
}
type StructFunction struct {
    Functions  []Function
    StructName string
}
type Package struct {
    PackageBaseName string
    PackageName     string
    Functions       []Function
    StructFunctions []StructFunction
}

func gen() {
    workDir := *register.WorkDir
    workDir = path.Clean(workDir)
    if len(workDir) > 2 && len(*register.OutDir) <= 2 {
        *register.OutDir = strings.ReplaceAll(workDir, "\\", "/")
    }
    pkgs, err := ParseDir(token.NewFileSet(), workDir, nil, parser.ParseComments)
    fmt.Printf("开始生成路由,工作路径: %s, \n", workDir)
    if err != nil {
        panic(err)
    }
    baseModuleName := *register.ModuleName
    if baseModuleName == "" {
        baseModuleName = findModuleName(workDir)
    }
    if len(workDir) > 2 {
        basePkgPath := strings.ReplaceAll(workDir, "\\", "/")
        for key, v := range pkgs {
            if strings.HasPrefix(key, basePkgPath) {
                pkgs[strings.ReplaceAll(key, basePkgPath+"/", "")] = v
                delete(pkgs, key)
            }
        }
    }
    var contexts []*Package
    for pname, p := range pkgs {
        fmt.Printf("分析中: %s\n", pname)
        pc := &Package{}
        pc.PackageBaseName = findModuleName(path.Base(pname))
        if pc.PackageBaseName == "" {
            pc.PackageBaseName = baseModuleName
        }
        pc.PackageName = pname
        for fname, f := range p.Files {
            fileGroupPath := ""
            for _, commentGroup := range f.Comments {
                for _, comment := range commentGroup.List {
                    text := strings.TrimSpace(comment.Text[2:])
                    if strings.HasPrefix(text, routerGroupAnnotation) {
                        if fileGroupPath == "" || len(commentGroup.List) <= 3 {
                            fileGroupPath = strings.TrimSpace(text[len(routerGroupAnnotation)+1:])
                        }
                    }
                }
            }
            for _, dx := range f.Decls {
                groupPath := fileGroupPath
                switch d := dx.(type) {
                case *ast.FuncDecl:
                    if d.Doc == nil || len(d.Doc.List) == 0 {
                        continue
                    }
                    var httpPath []HttpPath
                    for _, comment := range d.Doc.List {
                        text := strings.TrimSpace(comment.Text[2:])
                        if strings.HasPrefix(text, routerAnnotation) {
                            m := ""
                            p := strings.TrimSpace(text[len(routerAnnotation)+1:])
                            if strings.Contains(p, "[") {
                                p, m, _ = strings.Cut(p, "[")
                                p = strings.TrimSpace(p)
                                m = strings.ToUpper(strings.TrimSpace(strings.TrimRight(m, "]")))
                            }
                            e := HttpPath{
                                Path:       p,
                                Method:     m,
                                PathMethod: p,
                            }
                            if m != "" {
                                e.PathMethod = p + `", "` + m
                            }
                            httpPath = append(httpPath, e)
                        } else if strings.HasPrefix(text, routerGroupAnnotation) {
                            groupPath = strings.TrimSpace(text[len(routerGroupAnnotation)+1:])
                        }
                    }
                    if len(httpPath) <= 0 {
                        continue
                    }
                    if !strings.HasPrefix(groupPath, "/") {
                        groupPath = "/" + groupPath
                    }
                    if groupPath == "/" {
                        groupPath = ""
                    }
                    for i, p := range httpPath {
                        // 如果用户填写了全路径,去掉groupPath
                        // @RouterGroup /api/user
                        // /api/user/info => /info
                        httpPath[i].Path = strings.TrimPrefix(p.Path, groupPath)
                        httpPath[i].PathMethod = strings.TrimPrefix(p.PathMethod, groupPath)
                    }
                    if d.Recv != nil && len(d.Recv.List) > 0 {
                        structType := d.Recv.List[0].Type
                        if expr, ok := structType.(*ast.StarExpr); ok {
                            structType = expr.X
                        }
                        ident := structType.(*ast.Ident)
                        //找到结构体
                        var structFunction *StructFunction
                        for i, sf := range pc.StructFunctions {
                            if sf.StructName == ident.Name {
                                structFunction = &pc.StructFunctions[i]
                                break
                            }
                        }
                        //没有找到,创建
                        b := structFunction == nil
                        if b {
                            structFunction = &StructFunction{
                                StructName: ident.Name,
                            }
                        }
                        //添加方法
                        structFunction.Functions = append(structFunction.Functions, Function{
                            FileInfo:  FileInfo{Path: fname},
                            GroupPath: groupPath,
                            Paths:     httpPath,
                            Name:      d.Name.Name,
                        })
                        if b {
                            pc.StructFunctions = append(pc.StructFunctions, *structFunction)
                        }
                    } else {
                        pc.Functions = append(pc.Functions, Function{
                            FileInfo:  FileInfo{Path: fname},
                            GroupPath: groupPath,
                            Paths:     httpPath,
                            Name:      d.Name.Name,
                        })
                    }
                default:
                }
            }
            slices.SortFunc(pc.StructFunctions, func(i, j StructFunction) int { return strings.Compare(i.StructName, j.StructName) })
            for _, function := range pc.StructFunctions {
                slices.SortFunc(function.Functions, func(i, j Function) int { return strings.Compare(i.Name, j.Name) })
            }
            slices.SortFunc(pc.Functions, func(i, j Function) int { return strings.Compare(i.Name, j.Name) })
        }
        if len(pc.Functions) > 0 || len(pc.StructFunctions) > 0 {
            contexts = append(contexts, pc)
        }
    }
    t := template.New("main.go")
    parse, _ := t.Parse(string(mainTemplate))
    b := &bytes.Buffer{}
    slices.SortFunc(contexts, func(i, j *Package) int { return strings.Compare(i.PackageName, j.PackageName) })
    parse.Execute(b, contexts)
    outDir := path.Clean(*register.OutDir)
    os.Mkdir(path.Clean(outDir+"/routers"), os.ModeDir)
    os.Mkdir(path.Clean(outDir+"/routers/reg"), os.ModeDir)
    name := path.Clean(outDir + "/routers/reg/core.go")
    err = os.WriteFile(name, b.Bytes(), os.ModePerm)
    if err != nil {
        panic(err)
    }
    fmt.Printf("已写入:%s\n", name)
    for _, context := range contexts {
        wPath := path.Clean(outDir + "/routers/" + context.PackageName + ".go")
        t := template.New(context.PackageName + ".go")
        _, err := t.Parse(string(ginByPackageTemplate))
        if err != nil {
            panic(err)
        }
        b := &bytes.Buffer{}
        err = t.Execute(b, context)
        if err != nil {
            panic(err)
        }
        i := b.Bytes()
        os.Mkdir(path.Dir(wPath), os.ModeDir)
        err = os.WriteFile(wPath, i, os.ModePerm)
        if err != nil {
            panic(err)
        }
        fmt.Printf("已写入:%s\n", wPath)
    }
}

func findModuleName(dir string) string {
    file, e := os.ReadFile(dir + "/go.mod")
    if e == nil {
        split := strings.Split(string(file), "\n")
        for _, s := range split {
            if s != "" && strings.HasPrefix(strings.TrimSpace(s), "module") {
                return strings.TrimSpace(s[7:])
            }
        }
    }
    return ""
}

func ParseDir(fset *token.FileSet, pathStr string, filter func(fs.FileInfo) bool, mode parser.Mode) (pkgs map[string]*ast.Package, first error) {
    list, err := os.ReadDir(pathStr)
    if err != nil {
        return nil, err
    }
    pkgs = make(map[string]*ast.Package)
    for _, d := range list {
        if strings.HasPrefix(d.Name(), ".") {
            continue
        }
        if d.IsDir() {
            p, f := ParseDir(fset, filepath.Join(pathStr, d.Name()), filter, parser.ParseComments)
            if f != nil {
                first = f
            }
            for s, a := range p {
                pkgs[strings.TrimLeft(strings.ReplaceAll(filepath.Join(pathStr, s), "\\", "/"), "/")] = a
            }
            continue
        }
        if !strings.HasSuffix(d.Name(), ".go") {
            continue
        }
        if filter != nil {
            info, err := d.Info()
            if err != nil {
                return nil, err
            }
            if !filter(info) {
                continue
            }
        }
        filename := filepath.Join(pathStr, d.Name())
        if src, err := parser.ParseFile(fset, filename, nil, mode); err == nil {
            name := src.Name.Name
            pName := name
            if pName == "main" && pathStr == "" {
                name = ""
            }
            pkg, found := pkgs[name]
            if !found {
                pkg = &ast.Package{
                    Name:  pName,
                    Files: make(map[string]*ast.File),
                }
                pkgs[name] = pkg
            }
            pkg.Files[filename] = src
        } else if first == nil {
            first = err
        }
    }

    return
}

func init() {
    register.Register("gin-router", gen)
}
