// Package enhance 从结构体注释生成注册代码或方法.
// enhance-register: 在 init 函数中调用指定方法注册带注解的结构体.
// enhance-addFunction: 为带注解的结构体生成返回注解值的方法.
//
// 通过 -annotation 指定注解标识, -findFileRegex 过滤扫描文件 (非锚定, 支持部分匹配).
package enhance
