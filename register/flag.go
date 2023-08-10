package register

import "flag"

var (
    WorkDir    = flag.String("WorkDir", "./", "需要操作的目录")
    ModuleName = flag.String("ModuleName", "", "手动指定主module名称,否则读取go.mod文件夹")
)
