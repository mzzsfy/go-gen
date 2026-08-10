package user

// @RouterGroup /api/user

// GetUser
// @Router /info[GET]
func GetUser() {}

// GetUser2
// @Router /detail[GET]
// @Router /detail2[POST]
func GetUser2() {}

type Service struct{}

// GetUser3
// @Router /list[GET]
func (s *Service) GetUser3() {}

// @RouterGroup /api/v2
// GetUser4
// @Router /info[GET]
func GetUser4() {}
