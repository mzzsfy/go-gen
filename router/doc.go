// Package router 从源码注解生成 HTTP 路由注册代码.
// 支持 echo 和 gin 引擎, 通过 @Router 和 @RouterGroup 注解声明路由.
//
// 文件输出规则:
//   - routers/ 核心文件和 0_import___.go: 受 -outDir 控制
//   - 各包的 0_router___.go: 始终写入包自身目录, 不受 -outDir 影响
//
// -workDir 支持 非 module root, 自动向上递归查找 go.mod 计算 import 路径.
package router
