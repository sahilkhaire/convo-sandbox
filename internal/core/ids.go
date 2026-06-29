package core

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/url"
	"strings"
	"time"
)

func GenerateMessageID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

func GenerateWAMID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return "wamid." + hex.EncodeToString(b)
}

func GenerateFileID() string {
	b := make([]byte, 12)
	_, _ = rand.Read(b)
	return "file_" + hex.EncodeToString(b)
}

func NormalizePhone(phone string) string {
	p := strings.TrimSpace(phone)
	if decoded, err := url.PathUnescape(p); err == nil {
		p = decoded
	}
	p = strings.ReplaceAll(p, " ", "")
	if !strings.HasPrefix(p, "+") && len(p) > 0 {
		p = "+" + p
	}
	return p
}

func NormalizePhoneWA(phone string) string {
	p := strings.TrimSpace(phone)
	p = strings.ReplaceAll(p, " ", "")
	p = strings.TrimPrefix(p, "+")
	return p
}

func UnixTimestamp() string {
	return fmt.Sprintf("%d", time.Now().Unix())
}
