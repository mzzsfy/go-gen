package router

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"text/template"
)

// Gen 解析源码并生成路由代码
func Gen(gen GenRouter) {
	workDir := path.Clean(*WorkDir)
	// OutDir 未显式设置时默认使用 WorkDir
	if workDir != "." && path.Clean(*OutDir) == "." {
		*OutDir = strings.ReplaceAll(workDir, "\\", "/")
	}
	outDir := path.Clean(*OutDir)
	baseModuleName := resolveModuleName(workDir)
	checkUsage(workDir, outDir, baseModuleName)
	// workImportBase: 包 import 路径前缀, 基于 workDir 相对 module root 的偏移
	// outImportBase: routers 包 import 路径前缀, 基于 outDir 相对 module root 的偏移
	workImportBase, outImportBase := computeImportBase(workDir, outDir, baseModuleName)

	fmt.Printf("开始生成路由,工作路径: %s\n", workDir)
	contexts := parsePackages(workDir, outImportBase)
	fmt.Printf("开始生成\n")

	mustWriteCoreFiles(outDir)
	fmt.Printf("已写入核心文件\n")

	pkgName := detectPackageName(outDir)
	if pkgName == "" {
		pkgName = outImportBase
		if i := strings.LastIndex(outImportBase, "/"); i > -1 {
			pkgName = outImportBase[i+1:]
		}
	}
	globalCtx := GlobalCtx{
		PackageName:     pkgName,
		PackageBaseName: workImportBase,
		OutPath:         outDir,
		Packages:        contexts,
	}

	must(gen.GenRouterCore(globalCtx))
	fmt.Printf("已写入引擎文件\n")

	fmt.Printf("开始写入路由逻辑\n")
	mustWriteRouterFiles(workDir, contexts)
	mustWriteImportFile(outDir, globalCtx)

	must(gen.AfterGenRouter(globalCtx))
}

// resolveModuleName 解析模块名, 优先用 flag, 否则读 go.mod
func resolveModuleName(workDir string) string {
	if *ModuleName != "" {
		return *ModuleName
	}
	return findModuleName(workDir)
}

// checkUsage 检测常见参数误用, 打印警告
func checkUsage(workDir, outDir, moduleName string) {
	goModName := findModuleName(workDir)
	// 场景1: -moduleName 与 go.mod 不一致
	if *ModuleName != "" && goModName != "" && goModName != *ModuleName {
		fmt.Printf("警告: -moduleName=%q 与 go.mod 声明的 %q 不一致\n", *ModuleName, goModName)
		if strings.HasPrefix(*ModuleName, goModName+"/") {
			fmt.Printf("      -moduleName 应为 go.mod 的 module 名, 子目录偏移由 -workDir 控制\n")
			fmt.Printf("      建议使用: -moduleName=%s -workDir=%s\n", goModName, strings.TrimPrefix(*ModuleName, goModName+"/"))
		} else {
			fmt.Printf("      如果是故意覆盖请忽略, 否则建议: -moduleName=%s\n", goModName)
		}
	}
	// 场景2: 未指定 -moduleName 且找不到 go.mod
	if *ModuleName == "" && goModName == "" {
		fmt.Printf("警告: 未找到 go.mod, 且未指定 -moduleName, import 路径可能不正确\n")
		fmt.Printf("      建议: 指定 -moduleName=<你的模块名>\n")
	}
}

// computeImportBase 计算 import 路径前缀, 包含目录相对 module root 的偏移
func computeImportBase(workDir, outDir, moduleName string) (workBase, outBase string) {
	root := findModuleRoot(workDir)
	if root == "" {
		return moduleName, moduleName
	}
	workBase = joinModulePath(root, workDir, moduleName)
	outBase = joinModulePath(root, outDir, moduleName)
	return
}

// joinModulePath 拼接 module name 和目录相对 module root 的偏移
// 目录不在 module root 内时仅返回 moduleName
func joinModulePath(moduleRoot, dir, moduleName string) string {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return moduleName
	}
	rel, err := filepath.Rel(moduleRoot, abs)
	if err != nil || rel == "." {
		return moduleName
	}
	// 目录不在 module root 内, 无法计算有效 import 偏移
	rel = strings.ReplaceAll(rel, "\\", "/")
	if strings.HasPrefix(rel, "../") {
		return moduleName
	}
	return path.Join(moduleName, rel)
}

// parsePackages 扫描目录, 提取路由信息, 返回按包路径排序的 Package 列表
func parsePackages(workDir, importBase string) []*Package {
	fset := token.NewFileSet()
	pkgs, err := ParseDir(fset, workDir, nil, parser.ParseComments)
	if err != nil {
		panic(err)
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
		pc := buildPackage(pname, p, workDir, importBase)
		if len(pc.Functions) > 0 || len(pc.StructFunctions) > 0 {
			contexts = append(contexts, pc)
		}
	}
	slices.SortFunc(contexts, func(i, j *Package) int {
		return strings.Compare(i.PackagePathName, j.PackagePathName)
	})
	return contexts
}

// buildPackage 从单个 ast.Package 提取路由和结构体方法
func buildPackage(pname string, p *ast.Package, workDir, importBase string) *Package {
	pc := &Package{
		BasePath:          workDir,
		PackagePathName:   pname,
		PackageName:       path.Base(pname),
		PackageBaseName:   importBase,
	}
	packageGroupPath := findPackageGroupPath(p)
	for _, f := range p.Files {
		fileGroupPath := findFileGroupPath(f, packageGroupPath)
		extractFuncDecls(f, fileGroupPath, pc)
	}
	sortPackageDecls(pc)
	return pc
}

// findPackageGroupPath 从包文档注释中提取 @RouterGroup
func findPackageGroupPath(p *ast.Package) string {
	for _, f := range p.Files {
		if f.Doc == nil {
			continue
		}
		for _, comment := range f.Doc.List {
			text := commentText(comment.Text)
			if strings.HasPrefix(text, routerGroupAnnotation) {
				return strings.TrimSpace(text[len(routerGroupAnnotation)+1:])
			}
		}
	}
	return ""
}

// findFileGroupPath 解析文件级 @RouterGroup, 优先级: 文件级 > 包级 > 全局
func findFileGroupPath(f *ast.File, packageGroupPath string) string {
	fileGroupPath := ""
	for _, commentGroup := range f.Comments {
		if commentGroup == f.Doc {
			continue
		}
		for _, comment := range commentGroup.List {
			text := commentText(comment.Text)
			if strings.HasPrefix(text, routerGroupAnnotation) && fileGroupPath == "" {
				fileGroupPath = strings.TrimSpace(text[len(routerGroupAnnotation)+1:])
			}
		}
	}
	if fileGroupPath == "" {
		fileGroupPath = packageGroupPath
	}
	if fileGroupPath == "" {
		fileGroupPath = *RouterGroup
	}
	return fileGroupPath
}

// extractFuncDecls 从文件的函数声明中提取路由信息
func extractFuncDecls(f *ast.File, fileGroupPath string, pc *Package) {
	for _, dx := range f.Decls {
		d, ok := dx.(*ast.FuncDecl)
		if !ok {
			continue
		}
		if d.Doc == nil || len(d.Doc.List) == 0 {
			continue
		}
		groupPath := fileGroupPath
		httpPaths := parseRouterAnnotations(d.Doc, &groupPath)
		if len(httpPaths) == 0 {
			continue
		}
		normalizeGroupPath(&groupPath)
		for i := range httpPaths {
			httpPaths[i].Path = strings.TrimPrefix(httpPaths[i].Path, groupPath)
			httpPaths[i].PathMethod = strings.TrimPrefix(httpPaths[i].PathMethod, groupPath)
		}
		fn := Function{
			GroupPath: groupPath,
			Paths:     httpPaths,
			Name:      d.Name.Name,
		}
		if d.Recv != nil && len(d.Recv.List) > 0 {
			addStructFunction(pc, d, fn)
		} else {
			pc.Functions = append(pc.Functions, fn)
		}
	}
}

// parseRouterAnnotations 从函数文档注释中提取 @Router 和 @RouterGroup 覆盖
func parseRouterAnnotations(doc *ast.CommentGroup, groupPath *string) []HttpPath {
	var httpPaths []HttpPath
	for _, comment := range doc.List {
		text := commentText(comment.Text)
		switch {
		case strings.HasPrefix(text, routerGroupAnnotation):
			*groupPath = strings.TrimSpace(text[len(routerGroupAnnotation)+1:])
		case strings.HasPrefix(text, routerAnnotation):
			httpPaths = append(httpPaths, parseRouterPath(text))
		}
	}
	return httpPaths
}

// parseRouterPath 解析单条 @Router 注释为 HttpPath
func parseRouterPath(text string) HttpPath {
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
	return e
}

// normalizeGroupPath 确保 groupPath 以 / 开头, 根路径归零
func normalizeGroupPath(groupPath *string) {
	if !strings.HasPrefix(*groupPath, "/") {
		*groupPath = "/" + *groupPath
	}
	if *groupPath == "/" {
		*groupPath = ""
	}
}

// addStructFunction 将方法函数添加到 Package 的结构体方法列表
func addStructFunction(pc *Package, d *ast.FuncDecl, fn Function) {
	structType := d.Recv.List[0].Type
	if expr, ok := structType.(*ast.StarExpr); ok {
		structType = expr.X
	}
	ident, ok := structType.(*ast.Ident)
	if !ok {
		return
	}
	for i := range pc.StructFunctions {
		if pc.StructFunctions[i].StructName == ident.Name {
			pc.StructFunctions[i].Functions = append(pc.StructFunctions[i].Functions, fn)
			return
		}
	}
	pc.StructFunctions = append(pc.StructFunctions, StructFunction{
		StructName: ident.Name,
		Functions:  []Function{fn},
	})
}

// sortPackageDecls 对包内的函数和结构体方法按名称排序
func sortPackageDecls(pc *Package) {
	slices.SortFunc(pc.StructFunctions, func(i, j StructFunction) int {
		return strings.Compare(i.StructName, j.StructName)
	})
	for _, sf := range pc.StructFunctions {
		slices.SortFunc(sf.Functions, func(i, j Function) int {
			return strings.Compare(i.Name, j.Name)
		})
	}
	slices.SortFunc(pc.Functions, func(i, j Function) int {
		return strings.Compare(i.Name, j.Name)
	})
}

// mustWriteCoreFiles 写入核心模板文件, 0__ 前缀文件为用户可编辑文件, 已存在则跳过
func mustWriteCoreFiles(outDir string) {
	entries, err := core.ReadDir("core")
	if err != nil {
		panic(fmt.Errorf("读取 core 目录失败: %w", err))
	}
	routersDir := path.Clean(outDir + "/routers")
	if err := os.MkdirAll(routersDir, 0755); err != nil {
		panic(err)
	}
	for _, entry := range entries {
		name := entry.Name()
		bs, err := core.ReadFile("core/" + name)
		if err != nil {
			panic(fmt.Errorf("读取核心模板 %s 失败: %w", name, err))
		}
		p := path.Clean(outDir + "/routers/" + strings.TrimSuffix(name, "tmp"))
		if strings.HasPrefix(name, "0__") {
			if stat, _ := os.Stat(p); stat != nil {
				continue
			}
		}
		if err := os.WriteFile(p, bs, 0644); err != nil {
			panic(err)
		}
	}
}

// mustWriteRouterFiles 按包渲染路由注册文件, 始终写入包自身目录 (不受 outDir 影响)
func mustWriteRouterFiles(workDir string, contexts []*Package) {
	t := template.Must(template.New("router.go").Parse(mustReadCoreTemplate("_router.gotmp")))
	for _, ctx := range contexts {
		wPath := path.Clean(workDir + "/" + ctx.PackagePathName + "/0_router___.go")
		ctx.WritePath = wPath
		if err := os.MkdirAll(filepath.Dir(wPath), 0755); err != nil {
			panic(err)
		}
		b := &bytes.Buffer{}
		if err := t.Execute(b, ctx); err != nil {
			panic(err)
		}
		if err := os.WriteFile(wPath, b.Bytes(), 0644); err != nil {
			panic(err)
		}
		fmt.Printf("已写入:%s\n", wPath)
	}
}

// mustWriteImportFile 渲染全局 import 文件
func mustWriteImportFile(outDir string, globalCtx GlobalCtx) {
	t := template.Must(template.New("import.go").Parse(mustReadCoreTemplate("_import.gotmp")))
	wPath := path.Clean(outDir + "/0_import___.go")
	b := &bytes.Buffer{}
	if err := t.Execute(b, globalCtx); err != nil {
		panic(err)
	}
	if err := os.WriteFile(wPath, b.Bytes(), 0644); err != nil {
		panic(err)
	}
	fmt.Printf("已写入:%s\n", wPath)
}

func mustReadCoreTemplate(name string) string {
	bs, err := core.ReadFile(name)
	if err != nil {
		panic(fmt.Errorf("读取模板 %s 失败: %w", name, err))
	}
	return string(bs)
}

// commentText 去除行注释前缀并 trim
func commentText(text string) string {
	return strings.TrimSpace(strings.TrimPrefix(text, "//"))
}

// detectPackageName 从目录中已有 .go 文件读取 package 声明名
// 无可解析文件时返回空, 由调用方回退到路径推导
func detectPackageName(dir string) string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, filepath.Join(dir, entry.Name()), nil, parser.PackageClauseOnly)
		if err != nil {
			continue
		}
		return f.Name.Name
	}
	return ""
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
