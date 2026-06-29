package core_test

import (
	"testing"

	"github.com/zixflow/messaging-simulator/internal/core"
)

func TestGenerateMessageID(t *testing.T) {
	id1 := core.GenerateMessageID()
	id2 := core.GenerateMessageID()
	if id1 == "" || id2 == "" {
		t.Fatal("expected non-empty ids")
	}
	if id1 == id2 {
		t.Fatal("expected unique ids")
	}
}

func TestNormalizePhone(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"919876543210", "+919876543210"},
		{"+919876543210", "+919876543210"},
		{" 919876543210 ", "+919876543210"},
		{"%2B919876543210", "+919876543210"},
	}
	for _, tt := range tests {
		if got := core.NormalizePhone(tt.in); got != tt.want {
			t.Errorf("NormalizePhone(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestNormalizePhoneWA(t *testing.T) {
	if got := core.NormalizePhoneWA("+919876543210"); got != "919876543210" {
		t.Errorf("got %q", got)
	}
}
