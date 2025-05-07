package router

import (
    "embed"
    "github.com/mzzsfy/go-gen/register"
    "os"
    "path"
    "strings"
)

var (
    //go:embed engin
    files embed.FS
)

type genGin struct {
    Name string
}

func (g genGin) GenRouterCore(ctx GlobalCtx) error {
    fs, err := files.ReadDir("engin/" + g.Name)
    if err != nil {
        return err
    }
    outDir := ctx.OutPath
    err = os.MkdirAll(path.Clean(outDir+"/routers"), os.ModeDir)
    if err != nil {
        return err
    }
    for _, f := range fs {
        bs, _ := files.ReadFile("engin/" + g.Name + "/" + f.Name())
        err = os.WriteFile(path.Clean(outDir+"/routers/"+strings.TrimSuffix(f.Name(), "tmp")), bs, os.ModePerm)
        if err != nil {
            return err
        }
    }
    return nil
}

func (g genGin) AfterGenRouter(GlobalCtx) error {
    return nil
}

func init() {
    fs, err := files.ReadDir("engin")
    if err != nil {
        panic(err)
    }
    for _, f := range fs {
        name := f.Name()
        register.Register(name+"-router", func() { Gen(genGin{Name: name}) })
    }
}
