package router

import (
    "bytes"
    "embed"
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
    routerNameAnnotation  = "@RouterName"
    //go:embed core *.gotmp
    core embed.FS
)

type GlobalCtx struct {
    PackageName     string
    PackageBaseName string
    OutPath         string
    Packages        []*Package
}

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
    BasePath          string
    WritePath         string
    PackageBaseName   string
    PackagePathName   string
    PackageModuleName string
    Functions         []Function
    StructFunctions   []StructFunction
}

type GenRouter interface {
    GenRouterCore(GlobalCtx) error
    AfterGenRouter(GlobalCtx) error
}

func Gen(gen GenRouter) {
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
        pc := &Package{
            BasePath: workDir,
        }
        pc.PackageBaseName = findModuleName(path.Base(pname))
        if pc.PackageBaseName == "" {
            pc.PackageBaseName = baseModuleName
        }
        pc.PackagePathName = pname
        pc.PackageModuleName = strings.ReplaceAll(pname, "/", ".")
        // 包级路由组: 扫描所有文件的包文档注释(package xxx 上方)
        packageGroupPath := ""
        for _, f := range p.Files {
            if f.Doc == nil {
                continue
            }
            for _, comment := range f.Doc.List {
                text := strings.TrimSpace(comment.Text[2:])
                if strings.HasPrefix(text, routerGroupAnnotation) {
                    packageGroupPath = strings.TrimSpace(text[len(routerGroupAnnotation)+1:])
                    break
                }
            }
            if packageGroupPath != "" {
                break
            }
        }
        for fname, f := range p.Files {
            fileGroupPath := ""
            // 包文档注释单独处理为包级 group, 文件级扫描跳过它
            for _, commentGroup := range f.Comments {
                if commentGroup == f.Doc {
                    continue
                }
                for _, comment := range commentGroup.List {
                    text := strings.TrimSpace(comment.Text[2:])
                    if strings.HasPrefix(text, routerGroupAnnotation) {
                        if fileGroupPath == "" {
                            fileGroupPath = strings.TrimSpace(text[len(routerGroupAnnotation)+1:])
                        }
                    }
                }
            }
            // 优先级: 文件级 > 包级 > 全局
            if fileGroupPath == "" {
                fileGroupPath = packageGroupPath
            }
            if fileGroupPath == "" {
                fileGroupPath = *register.RouterGroup
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
                        if strings.HasPrefix(text, routerGroupAnnotation) {
                            groupPath = strings.TrimSpace(text[len(routerGroupAnnotation)+1:])
                        } else if strings.HasPrefix(text, routerAnnotation) {
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
    slices.SortFunc(contexts, func(i, j *Package) int { return strings.Compare(i.PackagePathName, j.PackagePathName) })
    fmt.Printf("开始生成\n")
    outDir := path.Clean(*register.OutDir)
    {
        f1, _ := core.ReadDir("core")
        err = os.MkdirAll(path.Clean(outDir+"/routers"), os.ModeDir)
        if err != nil {
            panic(err)
        }
        for _, f := range f1 {
            name := f.Name()
            bs, _ := core.ReadFile("core/" + name)
            p := path.Clean(outDir + "/routers/" + strings.TrimSuffix(name, "tmp"))
            if strings.HasPrefix(name, "0__") {
                stat, _ := os.Stat(p)
                if stat != nil {
                    continue
                }
            }
            err = os.WriteFile(p, bs, os.ModePerm)
            if err != nil {
                panic(err)
            }
        }
    }
    fmt.Printf("已写入核心文件\n")
    pkgName := baseModuleName
    {
        i := strings.LastIndex(baseModuleName, "/")
        if i > -1 {
            pkgName = baseModuleName[i+1:]
        }
    }
    globalCtx := GlobalCtx{
        PackageName:     pkgName,
        PackageBaseName: baseModuleName,
        OutPath:         outDir,
        Packages:        contexts,
    }
    err = gen.GenRouterCore(globalCtx)
    if err != nil {
        panic(err)
    }
    fmt.Printf("已写入引擎文件\n")
    {
        fmt.Printf("开始写入路由逻辑\n")
        t := template.New("router.go")
        bs, _ := core.ReadFile("_router.gotmp")
        t.Parse(string(bs))
        for _, context := range contexts {
            wPath := path.Clean(outDir + "/" + context.PackagePathName + "/0_router___.go")
            context.WritePath = wPath
            os.MkdirAll(filepath.Dir(wPath), os.ModeDir)
            b := &bytes.Buffer{}
            err = t.Execute(b, context)
            if err != nil {
                panic(err)
            }
            i := b.Bytes()
            err = os.WriteFile(wPath, i, os.ModePerm)
            if err != nil {
                panic(err)
            }
            fmt.Printf("已写入:%s\n", wPath)
        }
    }

    {
        t := template.New("import.go")
        bs, _ := core.ReadFile("_import.gotmp")
        t.Parse(string(bs))
        wPath := path.Clean(outDir + "/0_import___.go")
        b := &bytes.Buffer{}
        err = t.Execute(b, globalCtx)
        if err != nil {
            panic(err)
        }
        i := b.Bytes()
        err = os.WriteFile(wPath, i, os.ModePerm)
        if err != nil {
            panic(err)
        }
        fmt.Printf("已写入:%s\n", wPath)
    }

    err = gen.AfterGenRouter(globalCtx)
    if err != nil {
        panic(err)
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
