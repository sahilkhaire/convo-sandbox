package admin

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/zixflow/messaging-simulator/internal/auth"
	"github.com/zixflow/messaging-simulator/internal/config"
	"github.com/zixflow/messaging-simulator/internal/core"
	"github.com/zixflow/messaging-simulator/internal/store"
)

type Handler struct {
	svc    *core.Services
	config config.Config
	tokens *auth.TokenService
	auth   *Auth
}

func NewHandler(svc *core.Services, cfg config.Config) (*Handler, error) {
	tokens, err := auth.NewTokenService(cfg.JWTSecret, cfg.JWTExpiryHours)
	if err != nil {
		return nil, err
	}
	h := &Handler{svc: svc, config: cfg, tokens: tokens}
	h.auth = NewAuth(tokens)
	return h, nil
}

func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	h.registerAuthRoutes(r)

	r.Group(func(r chi.Router) {
		r.Use(h.auth.Middleware)

		h.registerProtectedAuthRoutes(r)

		r.With(h.auth.RequirePerm(auth.PermViewAccounts)).Get("/accounts", h.ListAccounts)
		r.With(h.auth.RequirePerm(auth.PermActionAccountsWrite)).Post("/accounts", h.CreateAccount)
		r.With(h.auth.RequirePerm(auth.PermViewAccounts)).Get("/accounts/{id}", h.GetAccount)
		r.With(h.auth.RequirePerm(auth.PermActionAccountsWrite)).Put("/accounts/{id}", h.UpdateAccount)
		r.With(h.auth.RequirePerm(auth.PermActionAccountsWrite)).Delete("/accounts/{id}", h.DeleteAccount)

		r.With(h.auth.RequirePerm(auth.PermViewInbox)).Get("/conversations", h.ListConversations)
		r.With(h.auth.RequirePerm(auth.PermViewInbox)).Get("/conversations/{id}", h.GetConversation)
		r.With(h.auth.RequirePerm(auth.PermViewInbox)).Get("/conversations/{id}/messages", h.ListMessages)
		r.With(h.auth.RequirePerm(auth.PermActionReply)).Post("/conversations/{id}/messages", h.SendInbound)
		r.With(h.auth.RequirePerm(auth.PermViewInbox)).Post("/conversations/{id}/read", h.MarkRead)

		r.With(h.auth.RequirePerm(auth.PermActionDelivery)).Post("/messages/{id}/status", h.TriggerStatus)
		r.With(h.auth.RequirePerm(auth.PermViewWebhooks)).Get("/webhooks", h.ListWebhooks)
		r.Get("/events", h.SSE)

		r.With(h.auth.RequirePerm(auth.PermActionDataPurge)).Delete("/data", h.PurgeData)
		r.Get("/permissions", h.ListPermissions)
	})

	return r
}

func (h *Handler) ListAccounts(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.Accounts.List(r.Context())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	if list == nil {
		list = []store.Account{}
	}
	writeJSON(w, http.StatusOK, list)
}

func (h *Handler) CreateAccount(w http.ResponseWriter, r *http.Request) {
	var input store.Account
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	if input.ExternalID == "" {
		input.ExternalID = uuid.New().String()
	}
	if input.AccessToken == "" {
		input.AccessToken = "sim_" + uuid.New().String()
	}
	if input.ClientSecret == "" {
		input.ClientSecret = "secret_" + uuid.New().String()[:8]
	}
	if input.WebhookURL == "" {
		input.WebhookURL = h.config.DefaultWebhookURL
	}
	if input.SentDelayMs == 0 {
		input.SentDelayMs = 100
	}
	if input.DeliveredDelayMs == 0 {
		input.DeliveredDelayMs = 500
	}
	if input.ReadDelayMs == 0 {
		input.ReadDelayMs = 2000
	}
	if err := h.svc.Accounts.Create(r.Context(), &input); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusCreated, input)
}

func (h *Handler) GetAccount(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	a, err := h.svc.Accounts.GetByID(r.Context(), id)
	if err != nil {
		writeErr(w, http.StatusNotFound, err)
		return
	}
	writeJSON(w, http.StatusOK, a)
}

func (h *Handler) UpdateAccount(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	existing, err := h.svc.Accounts.GetByID(r.Context(), id)
	if err != nil {
		writeErr(w, http.StatusNotFound, err)
		return
	}
	if err := json.NewDecoder(r.Body).Decode(&existing); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	existing.ID = id
	if err := h.svc.Accounts.Update(r.Context(), existing); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, existing)
}

func (h *Handler) DeleteAccount(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	if err := h.svc.Accounts.Delete(r.Context(), id); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) ListConversations(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.Conversations.List(r.Context())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	if list == nil {
		list = []store.Conversation{}
	}
	writeJSON(w, http.StatusOK, list)
}

func (h *Handler) GetConversation(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	c, err := h.svc.Conversations.GetByID(r.Context(), id)
	if err != nil {
		writeErr(w, http.StatusNotFound, err)
		return
	}
	writeJSON(w, http.StatusOK, c)
}

func (h *Handler) ListMessages(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	msgs, err := h.svc.Messages.ListByConversation(r.Context(), id)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	if msgs == nil {
		msgs = []store.Message{}
	}
	writeJSON(w, http.StatusOK, msgs)
}

func (h *Handler) SendInbound(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	var input struct {
		Text        string          `json:"text"`
		MessageType string          `json:"message_type"`
		Body        json.RawMessage `json:"body"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	msg, err := h.svc.RecordInbound(r.Context(), core.InboundInput{
		ConversationID: id,
		Text:           input.Text,
		MessageType:    input.MessageType,
		Body:           input.Body,
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusCreated, msg)
}

func (h *Handler) MarkRead(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	_ = h.svc.Conversations.MarkRead(r.Context(), id)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) TriggerStatus(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	var input struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	if err := h.svc.TriggerMessageStatus(r.Context(), id, input.Status); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) ListWebhooks(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			limit = n
		}
	}
	list, err := h.svc.Webhooks.List(r.Context(), limit)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	if list == nil {
		list = []store.WebhookDelivery{}
	}
	writeJSON(w, http.StatusOK, list)
}

func (h *Handler) PurgeData(w http.ResponseWriter, r *http.Request) {
	if !h.config.EnableDataReset {
		writeErr(w, http.StatusForbidden, errDisabled("data reset disabled"))
		return
	}
	scope := r.URL.Query().Get("scope")
	if scope == "" {
		scope = "messages"
	}
	if scope != "messages" && scope != "all" {
		writeErr(w, http.StatusBadRequest, errDisabled("scope must be messages or all"))
		return
	}
	if err := h.svc.PurgeData(r.Context(), scope); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	h.svc.SSE.Broadcast(core.EventDataCleared, map[string]string{"scope": scope})
	writeJSON(w, http.StatusOK, map[string]string{"status": "cleared", "scope": scope})
}

func (h *Handler) SSE(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeErr(w, http.StatusInternalServerError, errDisabled("streaming not supported"))
		return
	}

	ch := h.svc.SSE.Subscribe()
	defer h.svc.SSE.Unsubscribe(ch)

	_, _ = w.Write([]byte(": connected\n\n"))
	flusher.Flush()

	for {
		select {
		case <-r.Context().Done():
			return
		case ev, ok := <-ch:
			if !ok {
				return
			}
			data, _ := json.Marshal(ev)
			_, _ = w.Write([]byte("data: "))
			_, _ = w.Write(data)
			_, _ = w.Write([]byte("\n\n"))
			flusher.Flush()
		}
	}
}

type simpleErr struct{ msg string }

func (e simpleErr) Error() string { return e.msg }
func errDisabled(msg string) error { return simpleErr{msg: msg} }

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{"error": err.Error()})
}
