package whatsapp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/zixflow/messaging-simulator/internal/core"
	"github.com/zixflow/messaging-simulator/internal/httputil"
	"github.com/zixflow/messaging-simulator/internal/store"
	"github.com/zixflow/messaging-simulator/pkg/meta"
)

type accountCtxKey struct{}

type Handler struct {
	svc *core.Services
}

func NewHandler(svc *core.Services) *Handler {
	return &Handler{svc: svc}
}

// Routes registers Meta Graph API paths: /v{version}/{phone-number-id}/...
func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	for _, v := range []string{"19.0", "20.0", "21.0"} {
		h.mountVersion(r, v)
	}
	return r
}

func (h *Handler) mountVersion(r chi.Router, version string) {
	prefix := "/v" + version
	r.Route(prefix, func(r chi.Router) {
		r.Get("/{resourceID}", h.HandleResourceGET)

		r.Group(func(r chi.Router) {
			r.Use(h.BearerAuth)
			r.Post("/{phoneNumberID}/messages", h.SendMessage)
			r.Post("/{phoneNumberID}/media", h.UploadMedia)
			r.Delete("/{mediaID}", h.DeleteMedia)
		})
	})
}

// HandleResourceGET handles webhook verify (hub.mode) or authenticated phone/media GET.
func (h *Handler) HandleResourceGET(w http.ResponseWriter, r *http.Request) {
	if r.URL.Query().Get("hub.mode") == "subscribe" {
		h.WebhookVerify(w, r)
		return
	}
	h.BearerAuth(http.HandlerFunc(h.getResource)).ServeHTTP(w, r)
}

func (h *Handler) getResource(w http.ResponseWriter, r *http.Request) {
	resourceID := chi.URLParam(r, "resourceID")
	if r.URL.Query().Get("fields") != "" {
		account, err := h.svc.Accounts.GetByExternalID(r.Context(), "whatsapp", resourceID)
		if err != nil {
			writeGraphError(w, http.StatusNotFound, 100, "Object not found", "GraphMethodException")
			return
		}
		httputil.WriteJSON(w, http.StatusOK, meta.PhoneNumberInfo{
			ID:                 account.ExternalID,
			DisplayPhoneNumber: account.DisplayPhone,
			VerifiedName:       account.Name,
			QualityRating:      "GREEN",
		})
		return
	}
	if mf, err := h.svc.Media.GetByFileID(r.Context(), resourceID); err == nil {
		httputil.WriteJSON(w, http.StatusOK, meta.MediaURLResponse{
			MessagingProduct: "whatsapp",
			URL:              fmt.Sprintf("/media/%s/download", resourceID),
			MimeType:         mf.MimeType,
			FileSize:         mf.SizeBytes,
			ID:               resourceID,
		})
		return
	}
	account, err := h.svc.Accounts.GetByExternalID(r.Context(), "whatsapp", resourceID)
	if err != nil {
		writeGraphError(w, http.StatusNotFound, 100, "Object not found", "GraphMethodException")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, meta.PhoneNumberInfo{
		ID:                 account.ExternalID,
		DisplayPhoneNumber: account.DisplayPhone,
		VerifiedName:       account.Name,
		QualityRating:      "GREEN",
	})
}

func (h *Handler) WebhookVerify(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	mode := q.Get("hub.mode")
	token := q.Get("hub.verify_token")
	challenge := q.Get("hub.challenge")

	if mode == "subscribe" && challenge != "" {
		accounts, _ := h.svc.Accounts.List(r.Context())
		for _, a := range accounts {
			if a.Channel == "whatsapp" && a.WebhookVerifyToken == token {
				w.Header().Set("Content-Type", "text/plain")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(challenge))
				return
			}
		}
	}
	writeGraphError(w, http.StatusForbidden, 190, "Webhook verification failed", "OAuthException")
}

func (h *Handler) BearerAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			writeGraphError(w, http.StatusUnauthorized, 190, "Invalid OAuth access token", "OAuthException")
			return
		}
		token := strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
		if token == "" {
			writeGraphError(w, http.StatusUnauthorized, 190, "Invalid OAuth access token", "OAuthException")
			return
		}
		accounts, err := h.svc.Accounts.List(r.Context())
		if err != nil {
			writeGraphError(w, http.StatusInternalServerError, 1, "Internal error", "OAuthException")
			return
		}
		// Accept token matching any WA account, or any non-empty token in dev
		var found *store.Account
		for i := range accounts {
			if accounts[i].Channel == "whatsapp" && accounts[i].AccessToken == token {
				found = &accounts[i]
				break
			}
		}
		if found == nil {
			for i := range accounts {
				if accounts[i].Channel == "whatsapp" {
					found = &accounts[i]
					break
				}
			}
		}
		if found == nil {
			writeGraphError(w, http.StatusUnauthorized, 190, "Invalid OAuth access token", "OAuthException")
			return
		}
		ctx := context.WithValue(r.Context(), accountCtxKey{}, *found)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (h *Handler) SendMessage(w http.ResponseWriter, r *http.Request) {
	phoneNumberID := chi.URLParam(r, "phoneNumberID")
	account, err := h.svc.Accounts.GetByExternalID(r.Context(), "whatsapp", phoneNumberID)
	if err != nil {
		writeGraphError(w, http.StatusNotFound, 100, "Object with ID '"+phoneNumberID+"' does not exist", "GraphMethodException")
		return
	}

	var body map[string]any
	if err := httputil.DecodeJSONLenient(r, &body); err != nil {
		writeGraphError(w, http.StatusBadRequest, 100, "Invalid JSON", "OAuthException")
		return
	}

	if mp, _ := body["messaging_product"].(string); mp != "whatsapp" {
		writeGraphError(w, http.StatusBadRequest, 100, "messaging_product must be whatsapp", "OAuthException")
		return
	}

	// Mark message as read: { messaging_product, status: read, message_id }
	if status, _ := body["status"].(string); status == "read" {
		msgID, _ := body["message_id"].(string)
		if msgID == "" {
			writeGraphError(w, http.StatusBadRequest, 100, "message_id is required when status is read", "OAuthException")
			return
		}
		httputil.WriteJSON(w, http.StatusOK, meta.SuccessResponse{Success: true})
		return
	}

	to, _ := body["to"].(string)
	if to == "" {
		writeGraphError(w, http.StatusBadRequest, 100, "to is required", "OAuthException")
		return
	}

	msgType, _ := body["type"].(string)
	if msgType == "" {
		writeGraphError(w, http.StatusBadRequest, 100, "type is required", "OAuthException")
		return
	}

	if err := validateMessageType(body, msgType); err != nil {
		writeGraphError(w, http.StatusBadRequest, 100, err.Error(), "OAuthException")
		return
	}

	wamid := core.GenerateWAMID()
	raw, _ := json.Marshal(body)

	_, _, err = h.svc.RecordOutbound(r.Context(), core.OutboundInput{
		Account:     account,
		UserPhone:   to,
		VendorID:    wamid,
		MessageType: msgType,
		Body:        raw,
	})
	if err != nil {
		writeGraphError(w, http.StatusInternalServerError, 1, err.Error(), "OAuthException")
		return
	}

	httputil.WriteJSON(w, http.StatusOK, meta.SendMessageResponse{
		MessagingProduct: "whatsapp",
		Contacts: []struct {
			Input string `json:"input"`
			WaID  string `json:"wa_id"`
		}{{Input: to, WaID: core.NormalizePhoneWA(to)}},
		Messages: []struct {
			ID string `json:"id"`
		}{{ID: wamid}},
	})
}

func validateMessageType(body map[string]any, msgType string) error {
	switch msgType {
	case "text":
		if text, ok := body["text"].(map[string]any); !ok || text["body"] == nil {
			return fmt.Errorf("text.body is required for type text")
		}
	case "template":
		if _, ok := body["template"]; !ok {
			return fmt.Errorf("template object is required for type template")
		}
	case "image", "document", "audio", "video", "sticker":
		if _, ok := body[msgType]; !ok {
			return fmt.Errorf("%s object is required for type %s", msgType, msgType)
		}
	case "location", "contacts", "reaction", "interactive":
		if _, ok := body[msgType]; !ok {
			return fmt.Errorf("%s object is required for type %s", msgType, msgType)
		}
	default:
		return fmt.Errorf("unsupported message type: %s", msgType)
	}
	return nil
}

func (h *Handler) UploadMedia(w http.ResponseWriter, r *http.Request) {
	phoneNumberID := chi.URLParam(r, "phoneNumberID")
	account, err := h.svc.Accounts.GetByExternalID(r.Context(), "whatsapp", phoneNumberID)
	if err != nil {
		writeGraphError(w, http.StatusNotFound, 100, "Object not found", "GraphMethodException")
		return
	}

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		writeGraphError(w, http.StatusBadRequest, 100, "Invalid multipart form", "OAuthException")
		return
	}

	if mp := r.FormValue("messaging_product"); mp != "" && mp != "whatsapp" {
		writeGraphError(w, http.StatusBadRequest, 100, "messaging_product must be whatsapp", "OAuthException")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeGraphError(w, http.StatusBadRequest, 100, "file is required", "OAuthException")
		return
	}
	defer file.Close()

	mimeType := r.FormValue("type")
	if mimeType == "" {
		mimeType = header.Header.Get("Content-Type")
	}

	mediaID := core.GenerateFileID()
	path := core.MediaFilePath(h.svc.MediaPath, mediaID)
	out, err := os.Create(path)
	if err != nil {
		writeGraphError(w, http.StatusInternalServerError, 1, err.Error(), "OAuthException")
		return
	}
	n, _ := io.Copy(out, file)
	out.Close()

	aid := account.ID
	mf := &store.MediaFile{
		AccountID:   &aid,
		FileID:      mediaID,
		MimeType:    mimeType,
		FileName:    header.Filename,
		SizeBytes:   n,
		StoragePath: path,
	}
	_ = h.svc.Media.Create(r.Context(), mf)

	httputil.WriteJSON(w, http.StatusOK, meta.MediaUploadResponse{ID: mediaID})
}

func (h *Handler) GetMedia(w http.ResponseWriter, r *http.Request) {
	mediaID := chi.URLParam(r, "mediaID")
	mf, err := h.svc.Media.GetByFileID(r.Context(), mediaID)
	if err != nil {
		writeGraphError(w, http.StatusNotFound, 100, "Media not found", "GraphMethodException")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, meta.MediaURLResponse{
		MessagingProduct: "whatsapp",
		URL:              fmt.Sprintf("/media/%s/download", mediaID),
		MimeType:         mf.MimeType,
		FileSize:         mf.SizeBytes,
		ID:               mediaID,
	})
}

func (h *Handler) DeleteMedia(w http.ResponseWriter, r *http.Request) {
	mediaID := chi.URLParam(r, "mediaID")
	mf, err := h.svc.Media.GetByFileID(r.Context(), mediaID)
	if err != nil {
		writeGraphError(w, http.StatusNotFound, 100, "Media not found", "GraphMethodException")
		return
	}
	_ = os.Remove(mf.StoragePath)
	httputil.WriteJSON(w, http.StatusOK, meta.SuccessResponse{Success: true})
}

func (h *Handler) GetPhoneNumber(w http.ResponseWriter, r *http.Request) {
	phoneNumberID := chi.URLParam(r, "phoneNumberID")
	if phoneNumberID == "" {
		phoneNumberID = chi.URLParam(r, "resourceID")
	}
	account, err := h.svc.Accounts.GetByExternalID(r.Context(), "whatsapp", phoneNumberID)
	if err != nil {
		writeGraphError(w, http.StatusNotFound, 100, "Object not found", "GraphMethodException")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, meta.PhoneNumberInfo{
		ID:                 account.ExternalID,
		DisplayPhoneNumber: account.DisplayPhone,
		VerifiedName:       account.Name,
		QualityRating:      "GREEN",
	})
}

func writeGraphError(w http.ResponseWriter, status, code int, message, errType string) {
	httputil.WriteJSON(w, status, meta.GraphError{
		Error: meta.GraphErrorBody{
			Message: message,
			Type:    errType,
			Code:    code,
		},
	})
}
