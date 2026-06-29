package jio

// Jio Business Messaging (JBM) v2.2 API types.

type OAuthResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	Scope       string `json:"scope"`
	TokenType   string `json:"token_type"`
}

type OAuthError struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description,omitempty"`
}

type SendMessageResponse struct {
	Success   bool   `json:"success"`
	MessageID string `json:"messageId"`
}

type RevokeMessageResponse struct {
	Success bool `json:"success"`
}

type AssistantEventRequest struct {
	EventType string `json:"eventType"`
	MessageID string `json:"messageId,omitempty"`
}

type AssistantEventResponse struct {
	Success bool `json:"success"`
}

type CapabilitiesResponse struct {
	Features            []string `json:"features"`
	MessageTrafficTypes []string `json:"messageTrafficTypes,omitempty"`
}

type BatchCapabilitiesRequest struct {
	PhoneNumbers []string `json:"phoneNumbers"`
}

type BatchCapabilityUser struct {
	PhoneNumber         string   `json:"phoneNumber"`
	Features            []string `json:"features"`
	RCSEnabled          bool     `json:"rcsEnabled"`
	MessageTrafficTypes []string `json:"messageTrafficTypes,omitempty"`
}

type BatchCapabilitiesResponse struct {
	Users []BatchCapabilityUser `json:"users"`
}

type UploadFileRequest struct {
	FileName    string `json:"fileName"`
	ContentType string `json:"contentType"`
	FileContent string `json:"fileContent"`
}

type UploadFileResponse struct {
	FileID  string `json:"fileId"`
	Success bool   `json:"success"`
}

type APIError struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

type WebhookEnvelope struct {
	UserPhoneNumber string      `json:"userPhoneNumber"`
	BotID           string      `json:"botId"`
	EntityType      string      `json:"entityType"`
	Entity          interface{} `json:"entity"`
}

var DefaultFeatures = []string{
	"RICHCARD_STANDALONE_SUPPORTED",
	"RICHCARD_CAROUSEL_SUPPORTED",
	"ACTION_OPEN_URL_IN_WEBVIEW",
	"ACTION_DIAL_SUPPORTED",
	"ACTION_VIEW_LOCATION_SUPPORTED",
	"ACTION_CREATE_CALENDAR_EVENT_SUPPORTED",
	"ACTION_SHARE_LOCATION_SUPPORTED",
	"PLAIN_TEXT_SUPPORTED",
}

var DefaultTrafficTypes = []string{
	"AUTHENTICATION",
	"TRANSACTION",
	"PROMOTION",
	"SERVICEREQUEST",
	"ACKNOWLEDGEMENT",
}
