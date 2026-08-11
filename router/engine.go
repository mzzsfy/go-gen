package router

import (
	"embed"
	"github.com/mzzsfy/go-gen/register"
	"os"
	"path"
	"strings"
)

var (
	//go:embed engine
	files embed.FS
)

type engineGen struct {
	Name string
}

func (g engineGen) GenRouterCore(ctx GlobalCtx) error {
	entries, err := files.ReadDir("engine/" + g.Name)
	if err != nil {
		return err
	}
	outDir := ctx.OutPath
	if err := os.MkdirAll(path.Clean(outDir+"/routers"), 0755); err != nil {
		return err
	}
	for _, entry := range entries {
		bs, err := files.ReadFile("engine/" + g.Name + "/" + entry.Name())
		if err != nil {
			return err
		}
		p := path.Clean(outDir + "/routers/" + strings.TrimSuffix(entry.Name(), "tmp"))
		if err := os.WriteFile(p, bs, 0644); err != nil {
			return err
		}
	}
	return nil
}

func (g engineGen) AfterGenRouter(GlobalCtx) error {
	return nil
}

func init() {
	entries, err := files.ReadDir("engine")
	if err != nil {
		panic(err)
	}
	for _, entry := range entries {
		name := entry.Name()
		register.Register(name+"-router", func() { Gen(engineGen{Name: name}) })
	}
}
