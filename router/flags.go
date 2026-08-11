package router

import (
	"flag"
	"github.com/mzzsfy/go-gen/register"
)

var (
	ModuleName  = flag.String("moduleName", "", "手动指定 module 名称, 为空时自动从 go.mod 读取 (支持向上递归查找)")
	RouterGroup = flag.String("routerGroup", "", "全局路由组前缀, 如 /api/v1")
)

// 共享 flag 的便捷引用
var (
	WorkDir = register.WorkDir
	OutDir  = register.OutDir
)
