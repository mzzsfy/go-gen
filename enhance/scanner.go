package enhance

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path"
	"regexp"
	"strings"
)

// annotatedStruct 带匹配注解的结构体信息
type annotatedStruct struct {
	Name   string
	Values []string
}

// scanAnnotatedStructs 扫描目录, 解析所有匹配文件, 提取带指定注解的结构体
func scanAnnotatedStructs(workDir, fileRegex, annotation string) (packageName string, result []annotatedStruct, err error) {
	reg, err := regexp.Compile(fileRegex)
	if err != nil {
		return "", nil, err
	}
	workDir = path.Clean(workDir)
	entries, err := os.ReadDir(workDir)
	if err != nil {
		return "", nil, err
	}
	fileSet := token.NewFileSet()
	for _, entry := range entries {
		if !reg.MatchString(entry.Name()) {
			continue
		}
		filePath := workDir + "/" + entry.Name()
		f, parseErr := parser.ParseFile(fileSet, filePath, nil, parser.ParseComments)
		if parseErr != nil {
			fmt.Printf("跳过文件: %s, 错误: %v\n", entry.Name(), parseErr)
			continue
		}
		if packageName == "" {
			packageName = f.Name.Name
		}
		result = append(result, extractAnnotatedStructs(f, annotation)...)
	}
	return packageName, result, nil
}

// extractAnnotatedStructs 从 AST 提取带匹配注解的结构体
func extractAnnotatedStructs(f *ast.File, annotation string) []annotatedStruct {
	var result []annotatedStruct
	for _, decl := range f.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			if _, ok := typeSpec.Type.(*ast.StructType); !ok {
				continue
			}
			commentGroup := typeSpec.Comment
			if commentGroup == nil {
				commentGroup = genDecl.Doc
			}
			if commentGroup == nil {
				continue
			}
			values := parseAnnotationValues(commentGroup, annotation)
			if values == nil {
				continue
			}
			result = append(result, annotatedStruct{
				Name:   typeSpec.Name.Name,
				Values: values,
			})
		}
	}
	return result
}

// parseAnnotationValues 从注释组提取注解后的逗号分隔值
func parseAnnotationValues(group *ast.CommentGroup, annotation string) []string {
	var values []string
	for _, comment := range group.List {
		text := strings.TrimSpace(strings.TrimPrefix(comment.Text, "//"))
		if !strings.HasPrefix(text, annotation) {
			continue
		}
		rest := ""
		if len(text) > len(annotation) {
			rest = strings.TrimSpace(text[len(annotation)+1:])
		}
		for _, s := range strings.Split(rest, ",") {
			values = append(values, `"`+s+`"`)
		}
	}
	return values
}
