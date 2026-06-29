package admin

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/zixflow/messaging-simulator/internal/auth"
	"github.com/zixflow/messaging-simulator/internal/store"
)

func (h *Handler) registerAuthRoutes(r interface {
	Get(string, http.HandlerFunc)
	Post(string, http.HandlerFunc)
}) {
	r.Get("/auth/status", h.AuthStatus)
	r.Post("/auth/setup", h.AuthSetup)
	r.Post("/auth/login", h.AuthLogin)
	r.Post("/auth/logout", h.AuthLogout)
}

func (h *Handler) registerProtectedAuthRoutes(r interface {
	Get(string, http.HandlerFunc)
	Post(string, http.HandlerFunc)
	Put(string, http.HandlerFunc)
}) {
	r.Get("/auth/me", h.AuthMe)
	r.Get("/users", h.ListUsers)
	r.Post("/users", h.CreateUser)
	r.Put("/users/{id}", h.UpdateUser)
	r.Put("/users/{id}/password", h.ResetUserPassword)
}

func (h *Handler) AuthStatus(w http.ResponseWriter, r *http.Request) {
	count, err := h.svc.Users.Count(r.Context())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	resp := map[string]any{"needs_setup": count == 0, "authenticated": false}
	if token := extractToken(r); token != "" {
		if claims, err := h.tokens.Verify(token); err == nil {
			resp["authenticated"] = true
			resp["user"] = userFromClaims(claims)
		}
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) AuthSetup(w http.ResponseWriter, r *http.Request) {
	count, err := h.svc.Users.Count(r.Context())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	if count > 0 {
		writeErr(w, http.StatusConflict, errMsg("setup already completed"))
		return
	}
	var input struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	if strings.TrimSpace(input.Name) == "" || strings.TrimSpace(input.Email) == "" {
		writeErr(w, http.StatusBadRequest, errMsg("name and email are required"))
		return
	}
	hash, err := auth.HashPassword(input.Password)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	u := &store.User{
		Name:         input.Name,
		Email:        input.Email,
		PasswordHash: hash,
		IsAdmin:      true,
		IsActive:     true,
		Permissions:  auth.PermissionsJSON(auth.AllPermissions()),
	}
	if err := h.svc.Users.Create(r.Context(), u); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"message": "admin created, please login"})
}

func (h *Handler) AuthLogin(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	u, err := h.svc.Users.GetByEmail(r.Context(), input.Email)
	if err != nil || !auth.CheckPassword(u.PasswordHash, input.Password) {
		writeErr(w, http.StatusUnauthorized, errMsg("invalid email or password"))
		return
	}
	if !u.IsActive {
		writeErr(w, http.StatusForbidden, errMsg("account is deactivated"))
		return
	}
	perms := auth.ParsePermissions(u.Permissions)
	token, err := h.tokens.Sign(u.ID, u.Email, u.Name, u.IsAdmin, perms)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"token": token,
		"user":  publicUser(u, perms),
	})
}

func (h *Handler) AuthLogout(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"message": "logged out"})
}

func (h *Handler) AuthMe(w http.ResponseWriter, r *http.Request) {
	claims, ok := ClaimsFromContext(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, errMsg("authentication required"))
		return
	}
	writeJSON(w, http.StatusOK, userFromClaims(claims))
}

func userFromClaims(claims *auth.Claims) map[string]any {
	perms := claims.Permissions
	if claims.IsAdmin {
		perms = auth.AllPermissions()
	}
	return map[string]any{
		"id":          claims.UserID,
		"email":       claims.Email,
		"name":        claims.Name,
		"is_admin":    claims.IsAdmin,
		"permissions": perms,
	}
}

func publicUser(u store.User, perms []string) map[string]any {
	return map[string]any{
		"id":          u.ID,
		"email":       u.Email,
		"name":        u.Name,
		"is_admin":    u.IsAdmin,
		"is_active":   u.IsActive,
		"permissions": perms,
	}
}

// ensure imports