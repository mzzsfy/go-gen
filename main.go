package main

import (
    _ "embed"
    "flag"
    _ "github.com/mzzsfy/go-gen/gin"
    "github.com/mzzsfy/go-gen/register"
)

var genType = flag.String("genType", "gin-router", "生成类型,默认为gin-router")

func main() {
    flag.Parse()
    register.Do(*genType)
}
