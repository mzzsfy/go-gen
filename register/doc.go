// Package register 提供 genType 注册与分发机制.
// 各生成器通过 Register 注册, main 调用 Do 按名称执行.
// 同时定义共享 flag: workDir (扫描目录) 和 outDir (输出目录).
package register
