package user

// @RouterGroup /api/user

// AllMethods @Router无方法表示所有HTTP方法
// @Router /all
func AllMethods() {}

type MultiService struct{}

// Multi1
// @Router /multi1[GET]
func (s *MultiService) Multi1() {}

// Multi2
// @Router /multi2[POST]
func (s *MultiService) Multi2() {}
