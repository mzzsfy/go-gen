package register

var (
    reg map[string]func()
)

func Register(name string, fn func()) {
    reg[name] = fn
}

func Do(name string) {
    f := reg[name]
    if f == nil {
        panic("no such generate type: " + name)
    }
    f()
}
