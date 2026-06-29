package httputil

import (
	"encoding/json"
	"net/http"
	"net/url"
)

func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func DecodePathPhone(raw string) string {
	p := raw
	if decoded, err := url.PathUnescape(p); err == nil {
		p = decoded
	}
	return p
}

func DecodeJSON(r *http.Request, dst any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(dst)
}

func DecodeJSONLenient(r *http.Request, dst any) error {
	return json.NewDecoder(r.Body).Decode(dst)
}
