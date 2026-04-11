package v1

type RegisterRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	RealName string `json:"real_name"`
	Password string `json:"password"`
}

type VerifyEmailRequest struct {
	Token string `json:"token"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type UserResponse struct {
	ID              int64   `json:"id"`
	Username        string  `json:"username"`
	Email           string  `json:"email"`
	RealName        string  `json:"real_name"`
	Status          string  `json:"status"`
	EmailVerifiedAt *string `json:"email_verified_at,omitempty"`
	CreatedAt       string  `json:"created_at"`
	UpdatedAt       string  `json:"updated_at"`
}

type RegisterResponse struct {
	User                       UserResponse `json:"user"`
	EmailVerificationToken     string       `json:"email_verification_token"`
	EmailVerificationExpiresAt string       `json:"email_verification_expires_at"`
}

type SessionResponse struct {
	User                  UserResponse `json:"user"`
	AccessTokenExpiresAt  string       `json:"access_token_expires_at"`
	RefreshTokenExpiresAt string       `json:"refresh_token_expires_at"`
}

type VerifyEmailResponse struct {
	User UserResponse `json:"user"`
}
