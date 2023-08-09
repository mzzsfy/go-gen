# go-genGinRouter

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
    reg.AddGroupHandlers("<<组>>", func(context *gin.Context) {
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
