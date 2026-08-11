package router

import (
	"flag"
	"github.com/mzzsfy/go-gen/register"
)

var (
	ModuleName  = flag.String("moduleName", "", "手动指定主module名称,否则读取go.mod文件夹")
	RouterGroup = flag.String("routerGroup", "", "全局路由组前缀")
)

// 共享 flag 的便捷引用
var (
	WorkDir = register.WorkDir
	OutDir  = register.OutDir
)
