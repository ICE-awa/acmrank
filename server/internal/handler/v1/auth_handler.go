package v1

import (
	"context"
	"errors"
	"net/http"
	"time"

	authsupport "github.com/ICE-awa/acmrank/server/internal/auth"
	dtov1 "github.com/ICE-awa/acmrank/server/internal/dto/v1"
	"github.com/ICE-awa/acmrank/server/internal/model"
	"github.com/ICE-awa/acmrank/server/internal/service"
	"github.com/gin-gonic/gin"
)

type AuthService interface {
	Register(ctx context.Context, input service.RegisterInput) (service.RegisterResult, error)
	VerifyEmail(ctx context.Context, token string) (model.User, error)
	Login(ctx context.Context, input service.LoginInput) (service.SessionResult, error)
	Refresh(ctx context.Context, refreshToken string) (service.SessionResult, error)
	Logout(ctx context.Context, refreshToken string) error
	Authenticate(ctx context.Context, accessToken string) (model.User, error)
}

type AuthHandler struct {
	service      AuthService
	cookieSecure bool
}

func NewAuthHandler(service AuthService, cookieSecure bool) *AuthHandler {
	return &AuthHandler{
		service:      service,
		cookieSecure: cookieSecure,
	}
}

func (h *AuthHandler) Register(c *gin.Context) {
	var request dtov1.RegisterRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.service.Register(c.Request.Context(), service.RegisterInput{
		Username: request.Username,
		Email:    request.Email,
		RealName: request.RealName,
		Password: request.Password,
	})
	if err != nil {
		writeAuthError(c, err)
		return
	}

	c.JSON(http.StatusCreated, dtov1.RegisterResponse{
		User:                       toUserResponse(result.User),
		EmailVerificationToken:     result.EmailVerification.Token,
		EmailVerificationExpiresAt: result.EmailVerification.ExpiresAt.UTC().Format(time.RFC3339),
	})
}

func (h *AuthHandler) VerifyEmail(c *gin.Context) {
	var request dtov1.VerifyEmailRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := h.service.VerifyEmail(c.Request.Context(), request.Token)
	if err != nil {
		writeAuthError(c, err)
		return
	}

	c.JSON(http.StatusOK, dtov1.VerifyEmailResponse{
		User: toUserResponse(user),
	})
}

func (h *AuthHandler) Login(c *gin.Context) {
	var request dtov1.LoginRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.service.Login(c.Request.Context(), service.LoginInput{
		Username: request.Username,
		Password: request.Password,
	})
	if err != nil {
		writeAuthError(c, err)
		return
	}

	h.writeSessionCookies(c, result.Session)
	c.JSON(http.StatusOK, toSessionResponse(result))
}

func (h *AuthHandler) Refresh(c *gin.Context) {
	result, err := h.service.Refresh(c.Request.Context(), readCookie(c, authsupport.RefreshTokenCookieName))
	if err != nil {
		writeAuthError(c, err)
		return
	}

	h.writeSessionCookies(c, result.Session)
	c.JSON(http.StatusOK, toSessionResponse(result))
}

func (h *AuthHandler) Logout(c *gin.Context) {
	err := h.service.Logout(c.Request.Context(), readCookie(c, authsupport.RefreshTokenCookieName))
	h.clearSessionCookies(c)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *AuthHandler) writeSessionCookies(c *gin.Context, session model.IssuedSession) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     authsupport.AccessTokenCookieName,
		Value:    session.AccessToken,
		Path:     "/",
		MaxAge:   cookieMaxAge(session.AccessTokenExpiresAt),
		HttpOnly: true,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     authsupport.RefreshTokenCookieName,
		Value:    session.RefreshToken,
		Path:     "/",
		MaxAge:   cookieMaxAge(session.RefreshTokenExpiresAt),
		HttpOnly: true,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
}

func (h *AuthHandler) clearSessionCookies(c *gin.Context) {
	expiredAt := time.Unix(0, 0).UTC()
	for _, name := range []string{
		authsupport.AccessTokenCookieName,
		authsupport.RefreshTokenCookieName,
	} {
		http.SetCookie(c.Writer, &http.Cookie{
			Name:     name,
			Value:    "",
			Path:     "/",
			Expires:  expiredAt,
			MaxAge:   -1,
			HttpOnly: true,
			Secure:   h.cookieSecure,
			SameSite: http.SameSiteLaxMode,
		})
	}
}

func toSessionResponse(result service.SessionResult) dtov1.SessionResponse {
	return dtov1.SessionResponse{
		User:                  toUserResponse(result.User),
		AccessTokenExpiresAt:  result.Session.AccessTokenExpiresAt.UTC().Format(time.RFC3339),
		RefreshTokenExpiresAt: result.Session.RefreshTokenExpiresAt.UTC().Format(time.RFC3339),
	}
}

func toUserResponse(user model.User) dtov1.UserResponse {
	var emailVerifiedAt *string
	if user.EmailVerifiedAt != nil {
		formatted := user.EmailVerifiedAt.UTC().Format(time.RFC3339)
		emailVerifiedAt = &formatted
	}

	return dtov1.UserResponse{
		ID:              user.ID,
		Username:        user.Username,
		Email:           user.Email,
		RealName:        user.RealName,
		Status:          string(user.Status),
		EmailVerifiedAt: emailVerifiedAt,
		CreatedAt:       user.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:       user.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

func writeAuthError(c *gin.Context, err error) {
	statusCode := http.StatusInternalServerError
	switch {
	case errors.Is(err, service.ErrValidation):
		statusCode = http.StatusBadRequest
	case errors.Is(err, service.ErrUsernameUnavailable), errors.Is(err, service.ErrEmailUnavailable):
		statusCode = http.StatusConflict
	case errors.Is(err, service.ErrInvalidCredentials), errors.Is(err, service.ErrUnauthorized):
		statusCode = http.StatusUnauthorized
	case errors.Is(err, service.ErrEmailVerificationRequired), errors.Is(err, service.ErrUserDisabled):
		statusCode = http.StatusForbidden
	case errors.Is(err, service.ErrInvalidEmailVerificationTok):
		statusCode = http.StatusBadRequest
	}

	c.JSON(statusCode, gin.H{"error": err.Error()})
}

func readCookie(c *gin.Context, name string) string {
	value, err := c.Cookie(name)
	if err != nil {
		return ""
	}

	return value
}

func cookieMaxAge(expiresAt time.Time) int {
	seconds := int(time.Until(expiresAt.UTC()).Seconds())
	if seconds < 0 {
		return 0
	}

	return seconds
}
