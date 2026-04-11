package v1

type CreatePlatformAccountRequest struct {
	Platform string `json:"platform" binding:"required"`
	Handle   string `json:"handle" binding:"required"`
}

type ReviewPlatformAccountRequest struct {
	Reason string `json:"reason"`
}

type PlatformAccountOwnerResponse struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
	RealName string `json:"real_name"`
}

type PlatformAccountResponse struct {
	ID            int64                         `json:"id"`
	Platform      string                        `json:"platform"`
	Handle        string                        `json:"handle"`
	DisplayHandle string                        `json:"display_handle"`
	Status        string                        `json:"status"`
	VerifiedAt    *string                       `json:"verified_at,omitempty"`
	LastSyncedAt  *string                       `json:"last_synced_at,omitempty"`
	CreatedAt     string                        `json:"created_at"`
	UpdatedAt     string                        `json:"updated_at"`
	Owner         *PlatformAccountOwnerResponse `json:"owner,omitempty"`
}

type PlatformAccountsResponse struct {
	Accounts []PlatformAccountResponse `json:"accounts"`
}
