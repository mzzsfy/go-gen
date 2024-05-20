# go-gen

## addFunction

添加特定注释的所有struct添加方法

```go
//@addFn a,b,c,123
type test struct {
}
```

```shell
go-gen -genType=enhance-addFunction -usingPointers=true -annotation=addFn -functionName=ADDFN -findFileRegex=.*\.pb\.go -fileName=addFunction.gen.go
```

```go
//addFunction.gen.go
func (t *test) ADDFN() []string {
    return []string{"a","b","c","123"}
}
```

## register

让添加特定注释的struct在init函数中调用方法

```go
//@addFn a,b,c,1234
type test struct {
}

func CALLFN[T any](ss []string) {
    println(ss)
}
```

```shell
go-gen -genType=enhance-register -usingPointers=false -annotation=call -functionName=CALLFN -findFileRegex=.*\.pb\.go -fileName=register.gen.go
```

```go
//register.gen.go
func init() {
    CALLFN([]string{"a","b","c","1234"})
}
```

## gin-router

[![](https://hits.seeyoufarm.com/api/count/incr/badge.svg?url=https%3A%2F%2Fgithub.com%2Fmzzsfy%2Fgo-genGin&count_bg=%2379C83D&title_bg=%23555555&icon=&icon_color=%23E7E7E7&title=hits&edge_flat=false)](https://github.com/mzzsfy)  
按 https://github.com/swaggo/gin-swagger 编写注释,然后自动生成路由

优势: 
- 生成代码简单,低侵入性
- 生成代码暴露部分核心部分,方便二次开发
- 携带参数绑定功能,支持tag: `query`,`form`,`json`,`header`,`path`,可动态添加,见生成文件中:routers.BindByTag
- 携带参数检验功能,使用 https://github.com/go-playground/validator/v10
- 支持每个文件一个@RouterGroup,也可在方法上单独覆盖,不支持全局@RouterGroup,@RouterGroup逻辑为gin.Group(),方便统一添加中间件

编写 go 文件并添加注释
```go
package xxx
// @RouterGroup /api/v1

// HelloWorld PingExample godoc
// @Summary ping example
// @Schemes
// @Description do ping
// @Tags example
// @Accept json
// @Produce json
// @Success 200 {string} HelloWorld
// @Router /api/v1/example/helloworld [delete]
func HelloWorld(g *gin.Context) {
    g.String(http.StatusOK, "helloworld")
}
type Test struct {
    Name string `json:"name" query:"name" validate:"min=5"`
}
// HelloWorld1 PingExample godoc
// @Summary ping example
// @Schemes
// @Description do ping
// @Tags example
// @Accept json
// @Produce json
// @Success 200 {string} HelloWorld
// @Router /api/v1/example/helloworld1 [get]
func (t Test) HelloWorld1(g *gin.Context) {
    g.String(http.StatusOK, t.Name)
}
```

```go
import (
    _ "<<你的包名>>/routers"
    "<<你的包名>>/routers/reg"
)

//#生成swagger文档
//go:generate go install github.com/swaggo/swag/cmd/swag@latest
//go:generate swag init

//#生成路由
//go:generate go install github.com/mzzsfy/go-genGinRouter@latest
//go:generate go-genGinRouter

func main() {
    g := gin.Default()
    // 在这里添加根中间件
    
    // 简单的统一异常处理,可不注册自己编写
    reg.RegisterErrorHandle(g)
    // 在这里添加组中间件
    reg.AddGroupHandlers("<组>", func(context *gin.Context) {
    })
    // 注册生成的路由
    reg.RegisterRouter(g)
    
    //在这里注册自定义路由
    
    g.Run(":8080")
}
```

执行
```
go generate
```
