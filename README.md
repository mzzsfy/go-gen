# go-gen

## 参数说明

所有参数通过 flag 传入, 可在 `go:generate` 中使用.

### 通用参数

| 参数 | 默认值 | 说明 |
|---|---|---|
| `-genType` | `echo-router` | 生成类型: `echo-router`, `gin-router`, `enhance-register`, `enhance-addFunction` |
| `-workDir` | `./` | 扫描源码的根目录, 支持非 module root (自动向上递归查找 go.mod) |
| `-outDir` | `./` | 核心文件和 import 文件的输出目录. **不影响各包的 `0_router___.go`** (始终写入包自身目录) |

### router 专属参数

| 参数 | 默认值 | 说明 |
|---|---|---|
| `-moduleName` | `""` | 手动指定 module 名称, 为空时自动从 go.mod 读取 |
| `-routerGroup` | `""` | 全局路由组前缀, 如 `/api/v1` |

### enhance 专属参数

| 参数 | 默认值 | 说明 |
|---|---|---|
| `-annotation` | `@relation` | 需要识别的注释标识 |
| `-functionName` | `RegisterGenRelation` | 生成的方法/函数名称 |
| `-fileName` | `0_register.gen.go` | 生成文件名 |
| `-findFileRegex` | `.+.go` | 匹配文件的 regex (非锚定, 支持部分匹配) |
| `-usingPointers` | `false` | 生成是否使用指针类型 |

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
    CALLFN[test]([]string{"a","b","c","1234"})
}
```

## router

### swagger

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

```
//#生成swagger文档
//go:generate go install github.com/swaggo/swag/cmd/swag@latest
//go:generate swag init
```

### echo

```go
//go:generate go install github.com/mzzsfy/go-gen@latest
//go:generate go-gen -genType=echo-router
```

### gin

```go
//go:generate go install github.com/mzzsfy/go-gen@latest
//go:generate go-gen -genType=gin-router
```

### 文件输出规则

- `routers/` 核心文件和 `0_import___.go`: 受 `-outDir` 控制
- 各包的 `0_router___.go`: **始终写入包自身目录** (不受 `-outDir` 影响)
- `-workDir` 非 module root 时自动向上递归查找 go.mod, import 路径包含目录偏移
