package v1

import (
	"errors"
	"net/http"
	"strings"

	authsupport "github.com/ICE-awa/acmrank/server/internal/auth"
	"github.com/ICE-awa/acmrank/server/internal/model"
	"github.com/ICE-awa/acmrank/server/internal/service"
	"github.com/gin-gonic/gin"
)

const authenticatedUserContextKey = "authenticatedUser"

type AuthMiddleware struct {
	service        AuthService
	adminUsernames map[string]struct{}
}

func NewAuthMiddleware(service AuthService, adminUsernames ...string) *AuthMiddleware {
	normalizedAdmins := make(map[string]struct{}, len(adminUsernames))
	for _, username := range adminUsernames {
		trimmed := strings.TrimSpace(strings.ToLower(username))
		if trimmed == "" {
			continue
		}

		normalizedAdmins[trimmed] = struct{}{}
	}

	return &AuthMiddleware{
		service:        service,
		adminUsernames: normalizedAdmins,
	}
}

func (m *AuthMiddleware) RequireAuthenticated() gin.HandlerFunc {
	return func(c *gin.Context) {
		user, err := m.service.Authenticate(c.Request.Context(), readCookie(c, authsupport.AccessTokenCookieName))
		if err != nil {
			statusCode := http.StatusInternalServerError
			switch {
			case errors.Is(err, service.ErrUnauthorized):
				statusCode = http.StatusUnauthorized
			case errors.Is(err, service.ErrEmailVerificationRequired), errors.Is(err, service.ErrUserDisabled):
				statusCode = http.StatusForbidden
			}

			c.AbortWithStatusJSON(statusCode, gin.H{"error": err.Error()})
			return
		}

		c.Set(authenticatedUserContextKey, user)
		c.Next()
	}
}

func (m *AuthMiddleware) RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := currentUserFromContext(c)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
			return
		}

		if _, ok := m.adminUsernames[strings.ToLower(user.Username)]; !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "admin privileges required"})
			return
		}

		c.Next()
	}
}

func currentUserFromContext(c *gin.Context) (model.User, bool) {
	value, ok := c.Get(authenticatedUserContextKey)
	if !ok {
		return model.User{}, false
	}

	user, ok := value.(model.User)
	return user, ok
}
