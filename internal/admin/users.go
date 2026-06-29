package admin

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/zixflow/messaging-simulator/internal/auth"
	"github.com/zixflow/messaging-simulator/internal/store"
)

func (h *Handler) requireAdmin(w http.ResponseWriter, r *http.Request) (*auth.Claims, bool) {
	claims, ok := ClaimsFromContext(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, errMsg("authentication required"))
		return nil, false
	}
	if !claims.IsAdmin && !auth.HasPermission(claims.IsAdmin, claims.Permissions, auth.PermActionUsersManage) {
		writeErr(w, http.StatusForbidden, errMsg("admin access required"))
		return nil, false
	}
	return claims, true
}

func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.requireAdmin(w, r); !ok {
		return
	}
	list, err := h.svc.Users.List(r.Context())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	if list == nil {
		list = []store.User{}
	}
	out := make([]map[string]any, 0, len(list))
	for _, u := range list {
		out = append(out, map[string]any{
			"id":          u.ID,
			"name":        u.Name,
			"email":       u.Email,
			"is_admin":    u.IsAdmin,
			"is_active":   u.IsActive,
			"permissions": auth.ParsePermissions(u.Permissions),
			"created_at":  u.CreatedAt,
		})
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.requireAdmin(w, r); !ok {
		return
	}
	var input struct {
		Name        string   `json:"name"`
		Email       string   `json:"email"`
		Password    string   `json:"password"`
		Permissions []string `json:"permissions"`
		IsAdmin     bool     `json:"is_admin"`
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
	perms := input.Permissions
	if input.IsAdmin {
		perms = auth.AllPermissions()
	}
	u := &store.User{
		Name:         input.Name,
		Email:        input.Email,
		PasswordHash: hash,
		IsAdmin:      input.IsAdmin,
		IsActive:     true,
		Permissions:  auth.PermissionsJSON(perms),
	}
	if err := h.svc.Users.Create(r.Context(), u); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusCreated, publicUser(*u, perms))
}

func (h *Handler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.requireAdmin(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	existing, err := h.svc.Users.GetByID(r.Context(), id)
	if err != nil {
		writeErr(w, http.StatusNotFound, err)
		return
	}
	var input struct {
		Name        string   `json:"name"`
		IsActive    bool     `json:"is_active"`
		Permissions []string `json:"permissions"`
		IsAdmin     bool     `json:"is_admin"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	if actor.UserID == id.String() && !input.IsActive {
		writeErr(w, http.StatusBadRequest, errMsg("cannot deactivate your own account"))
		return
	}
	existing.Name = input.Name
	existing.IsActive = input.IsActive
	existing.IsAdmin = input.IsAdmin
	perms := input.Permissions
	if input.IsAdmin {
		perms = auth.AllPermissions()
	}
	existing.Permissions = auth.PermissionsJSON(perms)
	if err := h.svc.Users.Update(r.Context(), existing); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, publicUser(existing, perms))
}

func (h *Handler) ResetUserPassword(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.requireAdmin(w, r); !ok {
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	if _, err := h.svc.Users.GetByID(r.Context(), id); err != nil {
		writeErr(w, http.StatusNotFound, err)
		return
	}
	var input struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	hash, err := auth.HashPassword(input.Password)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	if err := h.svc.Users.SetPassword(r.Context(), id, hash); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "password updated"})
}

func (h *Handler) ListPermissions(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, auth.AllPermissions())
}
