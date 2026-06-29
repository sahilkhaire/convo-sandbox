package core

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/zixflow/messaging-simulator/internal/store"
)

type Services struct {
	DB           *store.DB
	Accounts     *store.AccountRepo
	Conversations *store.ConversationRepo
	Messages     *store.MessageRepo
	Media        *store.MediaRepo
	Webhooks     *store.WebhookRepo
	DeliveryJobs *store.DeliveryJobRepo
	Templates    *store.TemplateRepo
	Data         *store.DataRepo
	Users        *store.UserRepo
	Dispatcher   *Dispatcher
	SSE          *SSEHub
	MediaPath    string
}

func NewServices(db *store.DB, mediaPath string) *Services {
	s := &Services{
		DB:            db,
		Accounts:      store.NewAccountRepo(db),
		Conversations: store.NewConversationRepo(db),
		Messages:      store.NewMessageRepo(db),
		Media:         store.NewMediaRepo(db),
		Webhooks:      store.NewWebhookRepo(db),
		DeliveryJobs:  store.NewDeliveryJobRepo(db),
		Templates:     store.NewTemplateRepo(db),
		Data:          store.NewDataRepo(db),
		Users:         store.NewUserRepo(db),
		SSE:           NewSSEHub(),
		MediaPath:     mediaPath,
	}
	s.Dispatcher = NewDispatcher(s)
	return s
}

type Dispatcher struct {
	svc    *Services
	client *http.Client
	stop   chan struct{}
}

func NewDispatcher(svc *Services) *Dispatcher {
	d := &Dispatcher{
		svc: svc,
		client: &http.Client{Timeout: 15 * time.Second},
		stop: make(chan struct{}),
	}
	go d.worker()
	return d
}

func (d *Dispatcher) Stop() {
	close(d.stop)
}

func (d *Dispatcher) worker() {
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-d.stop:
			return
		case <-ticker.C:
			d.processBatch(context.Background())
		}
	}
}

func (d *Dispatcher) processBatch(ctx context.Context) {
	jobs, err := d.svc.DeliveryJobs.ClaimPending(ctx, 20)
	if err != nil || len(jobs) == 0 {
		return
	}
	for _, job := range jobs {
		d.deliver(ctx, job)
	}
}

func (d *Dispatcher) deliver(ctx context.Context, job store.DeliveryJob) {
	account, err := d.svc.Accounts.GetByID(ctx, job.AccountID)
	if err != nil || account.WebhookURL == "" {
		_ = d.svc.DeliveryJobs.MarkFailed(ctx, job.ID)
		return
	}

	body, _ := json.Marshal(job.Payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, account.WebhookURL, bytes.NewReader(body))
	if err != nil {
		_ = d.svc.DeliveryJobs.MarkFailed(ctx, job.ID)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := d.client.Do(req)
	var status int
	var respBody string
	var errMsg string
	if err != nil {
		errMsg = err.Error()
	} else {
		status = resp.StatusCode
		b, _ := io.ReadAll(resp.Body)
		respBody = string(b)
		resp.Body.Close()
	}

	aid := job.AccountID
	wh := &store.WebhookDelivery{
		AccountID:    &aid,
		Channel:      job.Channel,
		EventType:    job.EventType,
		Payload:      body,
		HTTPStatus:   &status,
		ResponseBody: respBody,
		ErrorMessage: errMsg,
		Attempts:     job.Attempts + 1,
	}
	_ = d.svc.Webhooks.Create(ctx, wh)

	if err != nil || status < 200 || status >= 300 {
		_ = d.svc.DeliveryJobs.MarkFailed(ctx, job.ID)
		return
	}
	_ = d.svc.DeliveryJobs.MarkDone(ctx, job.ID)
	d.svc.SSE.Broadcast(EventDelivery, wh)
}

func (d *Dispatcher) Enqueue(ctx context.Context, account store.Account, messageID *uuid.UUID, channel, eventType string, payload any, delay time.Duration) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	job := &store.DeliveryJob{
		AccountID:   account.ID,
		MessageID:   messageID,
		Channel:     channel,
		EventType:   eventType,
		Payload:     data,
		ScheduledAt: time.Now().Add(delay),
		Status:      "pending",
	}
	return d.svc.DeliveryJobs.Create(ctx, job)
}

func (d *Dispatcher) ScheduleOutboundLifecycle(ctx context.Context, account store.Account, msg store.Message, userPhone string) {
	mid := msg.ID
	ch := account.Channel

	if account.Channel == "rcs" {
		d.enqueueRCSStatus(ctx, account, &mid, userPhone, msg.VendorMessageID, "SEND_MESSAGE_SUCCESS", account.SentDelayMs)
		if account.AutoRead {
			d.enqueueRCSRead(ctx, account, &mid, userPhone, msg.VendorMessageID, account.ReadDelayMs)
		}
		return
	}

	d.enqueueWAStatus(ctx, account, &mid, userPhone, msg.VendorMessageID, "sent", account.SentDelayMs)
	d.enqueueWAStatus(ctx, account, &mid, userPhone, msg.VendorMessageID, "delivered", account.DeliveredDelayMs)
	if account.AutoRead {
		d.enqueueWAStatus(ctx, account, &mid, userPhone, msg.VendorMessageID, "read", account.ReadDelayMs)
	}
	_ = ch
}

func (d *Dispatcher) enqueueRCSStatus(ctx context.Context, account store.Account, msgID *uuid.UUID, userPhone, vendorMsgID, eventType string, delayMs int) {
	payload := map[string]any{
		"userPhoneNumber": userPhone,
		"botId":           account.ExternalID,
		"entityType":      "STATUS_EVENT",
		"entity": map[string]any{
			"eventType": eventType,
			"messageId": vendorMsgID,
			"sendTime":  time.Now().UTC().Format(time.RFC3339Nano),
			"eventId":   GenerateMessageID(),
		},
	}
	_ = d.Enqueue(ctx, account, msgID, "rcs", eventType, payload, time.Duration(delayMs)*time.Millisecond)
}

func (d *Dispatcher) enqueueRCSRead(ctx context.Context, account store.Account, msgID *uuid.UUID, userPhone, vendorMsgID string, delayMs int) {
	payload := map[string]any{
		"userPhoneNumber": userPhone,
		"botId":           account.ExternalID,
		"entityType":      "USER_EVENT",
		"entity": map[string]any{
			"eventType":         "MESSAGE_READ",
			"messageId":           vendorMsgID,
			"sendTime":            time.Now().UTC().Format(time.RFC3339Nano),
			"senderPhoneNumber": userPhone,
			"eventId":             GenerateMessageID(),
		},
	}
	_ = d.Enqueue(ctx, account, msgID, "rcs", "MESSAGE_READ", payload, time.Duration(delayMs)*time.Millisecond)
}

func (d *Dispatcher) enqueueWAStatus(ctx context.Context, account store.Account, msgID *uuid.UUID, userPhone, vendorMsgID, status string, delayMs int) {
	payload := map[string]any{
		"object": "whatsapp_business_account",
		"entry": []map[string]any{{
			"id": account.WabaID,
			"changes": []map[string]any{{
				"value": map[string]any{
					"messaging_product": "whatsapp",
					"metadata": map[string]string{
						"display_phone_number": account.DisplayPhone,
						"phone_number_id":      account.ExternalID,
					},
					"statuses": []map[string]any{{
						"id":           vendorMsgID,
						"status":       status,
						"timestamp":    UnixTimestamp(),
						"recipient_id": NormalizePhoneWA(userPhone),
					}},
				},
				"field": "messages",
			}},
		}},
	}
	_ = d.Enqueue(ctx, account, msgID, "whatsapp", status, payload, time.Duration(delayMs)*time.Millisecond)
}

func (d *Dispatcher) SendInboundRCS(ctx context.Context, account store.Account, userPhone string, entity map[string]any) error {
	payload := map[string]any{
		"userPhoneNumber": userPhone,
		"botId":           account.ExternalID,
		"entityType":      "USER_MESSAGE",
		"entity":          entity,
	}
	return d.Enqueue(ctx, account, nil, "rcs", "USER_MESSAGE", payload, 50*time.Millisecond)
}

func (d *Dispatcher) SendInboundWA(ctx context.Context, account store.Account, userPhone string, msgBody map[string]any) error {
	payload := map[string]any{
		"object": "whatsapp_business_account",
		"entry": []map[string]any{{
			"id": account.WabaID,
			"changes": []map[string]any{{
				"value": map[string]any{
					"messaging_product": "whatsapp",
					"metadata": map[string]string{
						"display_phone_number": account.DisplayPhone,
						"phone_number_id":      account.ExternalID,
					},
					"contacts": []map[string]any{{
						"profile": map[string]string{"name": "Test User"},
						"wa_id":   NormalizePhoneWA(userPhone),
					}},
					"messages": []map[string]any{msgBody},
				},
				"field": "messages",
			}},
		}},
	}
	return d.Enqueue(ctx, account, nil, "whatsapp", "inbound_message", payload, 50*time.Millisecond)
}

func (s *Services) EnsureMediaDir() error {
	return os.MkdirAll(s.MediaPath, 0o755)
}

func (s *Services) DeleteMediaFiles(ctx context.Context) error {
	paths, err := s.Media.ListPaths(ctx)
	if err != nil {
		return err
	}
	for _, p := range paths {
		_ = os.Remove(p)
	}
	return nil
}

func (s *Services) PurgeData(ctx context.Context, scope string) error {
	_ = s.DeliveryJobs.CancelAll(ctx)
	if err := s.DeleteMediaFiles(ctx); err != nil {
		log.Printf("media cleanup warning: %v", err)
	}
	if scope == "all" {
		return s.Data.PurgeAll(ctx)
	}
	return s.Data.PurgeMessagesOnly(ctx)
}

func MediaFilePath(base, fileID string) string {
	return filepath.Join(base, fileID)
}
