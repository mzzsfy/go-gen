package router

import (
	"embed"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

var (
	routerAnnotation      = "@Router"
	routerGroupAnnotation = "@RouterGroup"
	//go:embed core *.gotmp
	core embed.FS
)

type GlobalCtx struct {
	PackageName     string
	PackageBaseName string
	OutPath         string
	Packages        []*Package
}

type HttpPath struct {
	Path       string
	Method     string
	PathMethod string
}
type Function struct {
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
	PackagePathName   string // 相对workDir的完整路径, 如 "api/quant"
	PackageName       string // package声明名, 为路径的base name, 如 "quant"
	PackageModuleName string
	Functions         []Function
	StructFunctions   []StructFunction
}

type GenRouter interface {
	GenRouterCore(GlobalCtx) error
	AfterGenRouter(GlobalCtx) error
}

func findModuleName(dir string) string {
	file, e := os.ReadFile(dir + "/go.mod")
	if e == nil {
		for _, s := range strings.Split(string(file), "\n") {
			fields := strings.Fields(s)
			if len(fields) >= 2 && fields[0] == "module" {
				return fields[1]
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
			p, f := ParseDir(fset, filepath.Join(pathStr, d.Name()), filter, mode)
			if f != nil && first == nil {
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
