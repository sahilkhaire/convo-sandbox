package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/google/uuid"
	"github.com/zixflow/messaging-simulator/internal/config"
	"github.com/zixflow/messaging-simulator/internal/store"
)

func main() {
	cfg := config.Load()
	ctx := context.Background()
	db, err := store.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	accounts := store.NewAccountRepo(db)

	rcsID := "6544c5b408febf98e5fc5ec4"
	rcs := &store.Account{
		Channel:            "rcs",
		Name:               "Demo RCS Assistant",
		ExternalID:         rcsID,
		ClientSecret:       "demo_secret",
		AccessToken:        "rcs_token_demo",
		WebhookURL:         envOr("SEED_WEBHOOK_URL", "http://host.docker.internal:9000/webhooks/rcs"),
		WebhookVerifyToken: "verify_token",
		SentDelayMs:        200,
		DeliveredDelayMs:   800,
		ReadDelayMs:        3000,
		AutoRead:           false,
		Capabilities:       json.RawMessage(`["RICHCARD_STANDALONE_SUPPORTED","RICHCARD_CAROUSEL_SUPPORTED","PLAIN_TEXT_SUPPORTED"]`),
	}

	waID := "123456789012345"
	wa := &store.Account{
		Channel:            "whatsapp",
		Name:               "Demo WhatsApp Number",
		ExternalID:         waID,
		AccessToken:        "wa_token_demo",
		WebhookURL:         envOr("SEED_WEBHOOK_URL", "http://host.docker.internal:9000/webhooks/whatsapp"),
		WebhookVerifyToken: "verify_token",
		WabaID:             "waba_" + uuid.New().String()[:8],
		DisplayPhone:       "+919876543210",
		SentDelayMs:        150,
		DeliveredDelayMs:   600,
		ReadDelayMs:        2500,
		AutoRead:           true,
	}

	for _, a := range []*store.Account{rcs, wa} {
		existing, err := accounts.GetByExternalID(ctx, a.Channel, a.ExternalID)
		if err == nil {
			fmt.Printf("account exists: %s %s (%s)\n", a.Channel, a.Name, existing.ID)
			continue
		}
		if err := accounts.Create(ctx, a); err != nil {
			log.Fatalf("create %s: %v", a.Channel, err)
		}
		fmt.Printf("created %s account: %s (external_id=%s)\n", a.Channel, a.ID, a.ExternalID)
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
