package main

import (
	_ "embed"
	_ "github.com/mzzsfy/go-gen/enhance"
	_ "github.com/mzzsfy/go-gen/router"
)

import (
	"flag"
	"fmt"
	"github.com/mzzsfy/go-gen/register"
	"os"
)

var genType = flag.String("genType", "echo-router", "生成类型: echo-router, gin-router, enhance-register, enhance-addFunction")

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
