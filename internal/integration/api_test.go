//go:build integration

package integration_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"
)

const baseURL = "http://localhost:8080"

func TestHealth(t *testing.T) {
	resp, err := http.Get(baseURL + "/api/health")
	if err != nil {
		t.Skip("server not running:", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
}

func TestJioOAuthProductionPath(t *testing.T) {
	url := baseURL + "/v1/oauth/token?grant_type=client_credentials&client_id=6544c5b408febf98e5fc5ec4&client_secret=demo_secret&scope=read"
	resp, err := http.Get(url)
	if err != nil {
		t.Skip("server not running:", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
}

func TestRCSSendCreatesConversation(t *testing.T) {
	body := bytes.NewBufferString(`{"content":{"plainText":"integration test"}}`)
	req, _ := http.NewRequest(http.MethodPost,
		baseURL+"/v1/messaging/users/%2B919111111111/assistantMessages/async?assistantId=6544c5b408febf98e5fc5ec4&messageId=inttest1",
		body)
	req.Header.Set("Authorization", "Bearer rcs_token_demo")
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Skip("server not running:", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("status %d: %s", resp.StatusCode, b)
	}

	time.Sleep(300 * time.Millisecond)
	convResp, err := http.Get(baseURL + "/api/conversations")
	if err != nil {
		t.Fatal(err)
	}
	defer convResp.Body.Close()
	var convs []map[string]any
	_ = json.NewDecoder(convResp.Body).Decode(&convs)
	found := false
	for _, c := range convs {
		if c["user_phone"] == "+919111111111" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected conversation for +919111111111")
	}
}

func TestWhatsAppSendProductionPath(t *testing.T) {
	body := bytes.NewBufferString(`{"messaging_product":"whatsapp","recipient_type":"individual","to":"919876543210","type":"text","text":{"body":"Hello"}}`)
	req, _ := http.NewRequest(http.MethodPost, baseURL+"/v19.0/123456789012345/messages", body)
	req.Header.Set("Authorization", "Bearer wa_token_demo")
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Skip("server not running:", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("status %d: %s", resp.StatusCode, b)
	}
}

func TestJioBatchCapabilities(t *testing.T) {
	body := bytes.NewBufferString(`{"phoneNumbers":["+919111111111","+919222222222"]}`)
	req, _ := http.NewRequest(http.MethodPost, baseURL+"/v1/messaging/usersBatchGet", body)
	req.Header.Set("Authorization", "Bearer rcs_token_demo")
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Skip("server not running:", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("status %d: %s", resp.StatusCode, b)
	}
	var result map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&result)
	if _, ok := result["users"]; !ok {
		t.Fatal("expected users in batch response")
	}
}

func TestPurgeMessagesOnly(t *testing.T) {
	req, _ := http.NewRequest(http.MethodDelete, baseURL+"/api/data?scope=messages", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Skip("server not running:", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("status %d: %s", resp.StatusCode, b)
	}
}
