package dto

// RegisterInput 是注册请求体。
type RegisterInput struct {
	Username string `json:"username" binding:"required,username"`
	Email    string `json:"email"    binding:"required,email,max=255"`
	Password string `json:"password" binding:"required,min=8,max=72"`
}

// LoginInput 是登录请求体。
//
// Login 字段同时接受「用户名」和「邮箱」，让用户少记一个东西。
type LoginInput struct {
	Login    string `json:"login"    binding:"required"`
	Password string `json:"password" binding:"required"`
}

// AuthResponse 是注册 / 登录 / 刷新成功后返回的数据。
//
// 注意：refresh_token 【不在】这里——它通过 httpOnly Cookie 下发，
// 前端 JS 读不到，从而把 XSS 窃取的风险降到最低。
type AuthResponse struct {
	AccessToken string       `json:"access_token"`
	TokenType   string       `json:"token_type"`
	ExpiresIn   int          `json:"expires_in"` // access_token 有效秒数
	User        UserResponse `json:"user"`
}
