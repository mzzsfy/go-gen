package main

import (
    _ "embed"
    "flag"
    "fmt"
    _ "github.com/mzzsfy/go-gen/enhance"
    _ "github.com/mzzsfy/go-gen/gin"
    "github.com/mzzsfy/go-gen/register"
    "os"
)

var genType = flag.String("genType", "gin-router", "生成类型,默认为gin-router")

func main() {
    flag.Parse()
    cwd, err := os.Getwd()
    if err != nil {
        panic(err)
    }
    fmt.Printf("cwd = %s\n", cwd)
    fmt.Printf("os.Args = %#v\n", os.Args)
    register.Do(*genType)
}
