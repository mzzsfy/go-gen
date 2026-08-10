package edge

// 未标注的结构体,应被跳过
type Skipped struct{}

// @relation tagged1
type Tagged struct{}

// 非结构体类型,应被跳过
type MyInt int

type MyIface interface {
	Do()
}
