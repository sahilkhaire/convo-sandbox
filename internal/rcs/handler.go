package rcs

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/zixflow/messaging-simulator/internal/core"
	"github.com/zixflow/messaging-simulator/internal/httputil"
	"github.com/zixflow/messaging-simulator/internal/store"
	"github.com/zixflow/messaging-simulator/pkg/jio"
)

type accountCtxKey struct{}

type Handler struct {
	svc *core.Services
}

func NewHandler(svc *core.Services) *Handler {
	return &Handler{svc: svc}
}

// Routes registers Jio JBM v2.2 paths exactly as documented.
func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/v1/oauth/token", h.OAuthToken)
	r.Group(func(r chi.Router) {
		r.Use(h.BearerAuth)
		r.Post("/v1/messaging/users/{userPhoneNumber}/assistantMessages/async", h.SendMessage)
		r.Delete("/v1/messaging/users/{userPhoneNumber}/assistantMessages/{messageID}", h.RevokeMessage)
		r.Post("/v1/messaging/users/{userPhoneNumber}/assistantEvents", h.AssistantEvents)
		r.Get("/v1/messaging/users/{userPhoneNumber}/capabilities", h.GetCapabilities)
		r.Post("/v1/messaging/usersBatchGet", h.BatchCapabilities)
		r.Post("/v1/messaging/upload/files", h.UploadFiles)
	})
	return r
}

func (h *Handler) OAuthToken(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	grantType := q.Get("grant_type")
	clientID := q.Get("client_id")
	clientSecret := q.Get("client_secret")
	scope := q.Get("scope")

	if grantType != "client_credentials" {
		httputil.WriteJSON(w, http.StatusBadRequest, jio.OAuthError{Error: "unsupported_grant_type"})
		return
	}
	if clientID == "" || clientSecret == "" {
		httputil.WriteJSON(w, http.StatusBadRequest, jio.OAuthError{Error: "invalid_request", ErrorDescription: "client_id and client_secret required"})
		return
	}
	if scope != "" && scope != "read" {
		httputil.WriteJSON(w, http.StatusBadRequest, jio.OAuthError{Error: "invalid_scope"})
		return
	}

	account, err := h.svc.Accounts.GetByExternalID(r.Context(), "rcs", clientID)
	if err != nil || account.ClientSecret != clientSecret {
		httputil.WriteJSON(w, http.StatusUnauthorized, jio.OAuthError{Error: "invalid_client"})
		return
	}

	httputil.WriteJSON(w, http.StatusOK, jio.OAuthResponse{
		AccessToken: account.AccessToken,
		ExpiresIn:   3600,
		Scope:       "read",
		TokenType:   "Bearer",
	})
}

func (h *Handler) BearerAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			writeJioError(w, http.StatusUnauthorized, "401", "Unauthorized")
			return
		}
		token := strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
		accounts, err := h.svc.Accounts.List(r.Context())
		if err != nil {
			writeJioError(w, http.StatusInternalServerError, "500", "Internal error")
			return
		}
		var found *store.Account
		for i := range accounts {
			if accounts[i].Channel == "rcs" && accounts[i].AccessToken == token {
				found = &accounts[i]
				break
			}
		}
		if found == nil {
			writeJioError(w, http.StatusUnauthorized, "401", "Invalid access token")
			return
		}
		ctx := context.WithValue(r.Context(), accountCtxKey{}, *found)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func accountFromCtx(r *http.Request) (store.Account, bool) {
	a, ok := r.Context().Value(accountCtxKey{}).(store.Account)
	return a, ok
}

func (h *Handler) SendMessage(w http.ResponseWriter, r *http.Request) {
	userPhone := core.NormalizePhone(httputil.DecodePathPhone(chi.URLParam(r, "userPhoneNumber")))
	assistantID := r.URL.Query().Get("assistantId")
	messageID := r.URL.Query().Get("messageId")

	if assistantID == "" {
		writeJioError(w, http.StatusBadRequest, "400", "assistantId query parameter is required")
		return
	}
	if messageID == "" {
		writeJioError(w, http.StatusBadRequest, "400", "messageId query parameter is required")
		return
	}

	account, err := h.svc.Accounts.GetByExternalID(r.Context(), "rcs", assistantID)
	if err != nil {
		writeJioError(w, http.StatusNotFound, "404", "Assistant not found")
		return
	}

	var body map[string]any
	if err := httputil.DecodeJSONLenient(r, &body); err != nil {
		writeJioError(w, http.StatusBadRequest, "400", "Invalid JSON body")
		return
	}

	if err := validateSendBody(body); err != nil {
		writeJioError(w, http.StatusBadRequest, "400", err.Error())
		return
	}

	msgType := detectRCSMessageType(body)
	raw, _ := json.Marshal(body)

	_, _, err = h.svc.RecordOutbound(r.Context(), core.OutboundInput{
		Account:     account,
		UserPhone:   userPhone,
		VendorID:    messageID,
		MessageType: msgType,
		Body:        raw,
	})
	if err != nil {
		writeJioError(w, http.StatusInternalServerError, "500", err.Error())
		return
	}

	httputil.WriteJSON(w, http.StatusCreated, jio.SendMessageResponse{Success: true, MessageID: messageID})
}

func validateSendBody(body map[string]any) error {
	if _, ok := body["richCardDetails"]; ok {
		return nil
	}
	if _, ok := body["carousel"]; ok {
		return nil
	}
	if content, ok := body["content"].(map[string]any); ok {
		if _, hasText := content["plainText"]; hasText {
			return nil
		}
	}
	// conversation / suggestion payloads
	if _, ok := body["suggestions"]; ok {
		return nil
	}
	return errValidation("request body must include content.plainText, richCardDetails, carousel, or suggestions")
}

type validationError string

func (e validationError) Error() string { return string(e) }
func errValidation(msg string) error      { return validationError(msg) }

func detectRCSMessageType(body map[string]any) string {
	if _, ok := body["richCardDetails"]; ok {
		return "rich_card"
	}
	if _, ok := body["carousel"]; ok {
		return "carousel"
	}
	if _, ok := body["content"]; ok {
		return "basic"
	}
	if _, ok := body["suggestions"]; ok {
		return "suggestion"
	}
	return "text"
}

func (h *Handler) RevokeMessage(w http.ResponseWriter, r *http.Request) {
	_ = httputil.DecodePathPhone(chi.URLParam(r, "userPhoneNumber"))
	messageID := chi.URLParam(r, "messageID")

	msg, err := h.svc.Messages.GetByVendorID(r.Context(), messageID)
	if err != nil {
		writeJioError(w, http.StatusNotFound, "404", "Message not found")
		return
	}
	_ = h.svc.TriggerMessageStatus(r.Context(), msg.ID, "revoked")
	httputil.WriteJSON(w, http.StatusCreated, jio.RevokeMessageResponse{Success: true})
}

func (h *Handler) AssistantEvents(w http.ResponseWriter, r *http.Request) {
	_ = core.NormalizePhone(httputil.DecodePathPhone(chi.URLParam(r, "userPhoneNumber")))

	var req jio.AssistantEventRequest
	if err := httputil.DecodeJSONLenient(r, &req); err != nil {
		writeJioError(w, http.StatusBadRequest, "400", "Invalid JSON body")
		return
	}
	if req.EventType != "TYPING" && req.EventType != "MESSAGE_READ" {
		writeJioError(w, http.StatusBadRequest, "400", "eventType must be TYPING or MESSAGE_READ")
		return
	}
	if req.EventType == "MESSAGE_READ" && req.MessageID == "" {
		writeJioError(w, http.StatusBadRequest, "400", "messageId is required for MESSAGE_READ")
		return
	}

	httputil.WriteJSON(w, http.StatusCreated, jio.AssistantEventResponse{Success: true})
}

func (h *Handler) GetCapabilities(w http.ResponseWriter, r *http.Request) {
	_ = core.NormalizePhone(httputil.DecodePathPhone(chi.URLParam(r, "userPhoneNumber")))
	assistantID := r.URL.Query().Get("assistantId")
	if assistantID == "" {
		writeJioError(w, http.StatusBadRequest, "400", "assistantId query parameter is required")
		return
	}

	account, err := h.svc.Accounts.GetByExternalID(r.Context(), "rcs", assistantID)
	if err != nil {
		writeJioError(w, http.StatusNotFound, "404", "Assistant not found")
		return
	}

	resp := jio.CapabilitiesResponse{
		Features:            jio.DefaultFeatures,
		MessageTrafficTypes: jio.DefaultTrafficTypes,
	}
	if len(account.Capabilities) > 2 {
		var caps []string
		if json.Unmarshal(account.Capabilities, &caps) == nil && len(caps) > 0 {
			resp.Features = caps
		}
	}
	httputil.WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) BatchCapabilities(w http.ResponseWriter, r *http.Request) {
	var req jio.BatchCapabilitiesRequest
	if err := httputil.DecodeJSONLenient(r, &req); err != nil {
		writeJioError(w, http.StatusBadRequest, "400", "Invalid JSON body")
		return
	}
	if len(req.PhoneNumbers) < 1 {
		writeJioError(w, http.StatusBadRequest, "400", "phoneNumbers array is required")
		return
	}

	users := make([]jio.BatchCapabilityUser, 0, len(req.PhoneNumbers))
	for _, pn := range req.PhoneNumbers {
		users = append(users, jio.BatchCapabilityUser{
			PhoneNumber:         core.NormalizePhone(pn),
			Features:            jio.DefaultFeatures,
			RCSEnabled:          true,
			MessageTrafficTypes: jio.DefaultTrafficTypes,
		})
	}
	httputil.WriteJSON(w, http.StatusOK, jio.BatchCapabilitiesResponse{Users: users})
}

func (h *Handler) UploadFiles(w http.ResponseWriter, r *http.Request) {
	var req jio.UploadFileRequest
	if err := httputil.DecodeJSONLenient(r, &req); err != nil {
		writeJioError(w, http.StatusBadRequest, "400", "Invalid JSON body")
		return
	}
	if req.FileName == "" || req.ContentType == "" || req.FileContent == "" {
		writeJioError(w, http.StatusBadRequest, "400", "fileName, contentType, and fileContent are required")
		return
	}

	fileID := core.GenerateFileID()
	path := core.MediaFilePath(h.svc.MediaPath, fileID)
	if err := writeBase64File(path, req.FileContent); err != nil {
		writeJioError(w, http.StatusInternalServerError, "500", err.Error())
		return
	}

	mf := &store.MediaFile{
		FileID:      fileID,
		MimeType:    req.ContentType,
		FileName:    req.FileName,
		SizeBytes:   int64(len(req.FileContent)),
		StoragePath: path,
	}
	_ = h.svc.Media.Create(r.Context(), mf)

	httputil.WriteJSON(w, http.StatusOK, jio.UploadFileResponse{FileID: fileID, Success: true})
}

func writeJioError(w http.ResponseWriter, status int, code, message string) {
	httputil.WriteJSON(w, status, jio.APIError{
		Error: struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		}{Code: code, Message: message},
	})
}
