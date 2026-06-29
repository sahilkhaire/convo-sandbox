package meta

// Meta WhatsApp Cloud API types (Graph API).

type SendMessageResponse struct {
	MessagingProduct string `json:"messaging_product"`
	Contacts         []struct {
		Input string `json:"input"`
		WaID  string `json:"wa_id"`
	} `json:"contacts"`
	Messages []struct {
		ID string `json:"id"`
	} `json:"messages"`
}

type MediaUploadResponse struct {
	ID string `json:"id"`
}

type MediaURLResponse struct {
	MessagingProduct string `json:"messaging_product"`
	URL              string `json:"url"`
	MimeType         string `json:"mime_type"`
	Sha256           string `json:"sha256"`
	FileSize         int64  `json:"file_size"`
	ID               string `json:"id"`
}

type SuccessResponse struct {
	Success bool `json:"success"`
}

type PhoneNumberInfo struct {
	VerifiedName           string `json:"verified_name,omitempty"`
	DisplayPhoneNumber     string `json:"display_phone_number,omitempty"`
	QualityRating          string `json:"quality_rating,omitempty"`
	ID                     string `json:"id"`
	CodeVerificationStatus string `json:"code_verification_status,omitempty"`
}

type GraphError struct {
	Error GraphErrorBody `json:"error"`
}

type GraphErrorBody struct {
	Message   string `json:"message"`
	Type      string `json:"type"`
	Code      int    `json:"code"`
	FbtraceID string `json:"fbtrace_id,omitempty"`
	ErrorData struct {
		MessagingProduct string `json:"messaging_product,omitempty"`
		Details          string `json:"details,omitempty"`
	} `json:"error_data,omitempty"`
}

type WebhookPayload struct {
	Object string `json:"object"`
	Entry  []struct {
		ID      string `json:"id"`
		Changes []struct {
			Value struct {
				MessagingProduct string                   `json:"messaging_product"`
				Metadata         map[string]string        `json:"metadata"`
				Statuses         []map[string]interface{} `json:"statuses,omitempty"`
				Messages         []map[string]interface{} `json:"messages,omitempty"`
			} `json:"value"`
			Field string `json:"field"`
		} `json:"changes"`
	} `json:"entry"`
}
