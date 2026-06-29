package rcs

import (
	"encoding/base64"
	"os"
	"strings"
)

func writeBase64File(path, content string) error {
	// strip data URL prefix if present
	if idx := strings.Index(content, ","); strings.HasPrefix(content, "data:") && idx > 0 {
		content = content[idx+1:]
	}
	data, err := base64.StdEncoding.DecodeString(content)
	if err != nil {
		// try raw bytes
		data = []byte(content)
	}
	return os.WriteFile(path, data, 0o644)
}
