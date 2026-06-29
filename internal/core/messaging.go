package core

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/zixflow/messaging-simulator/internal/store"
)

type OutboundInput struct {
	Account    store.Account
	UserPhone  string
	VendorID   string
	MessageType string
	Body       json.RawMessage
}

func (s *Services) RecordOutbound(ctx context.Context, in OutboundInput) (store.Message, store.Conversation, error) {
	phone := NormalizePhone(in.UserPhone)
	conv, err := s.Conversations.GetOrCreate(ctx, in.Account.ID, in.Account.Channel, phone)
	if err != nil {
		return store.Message{}, store.Conversation{}, err
	}

	msg := &store.Message{
		ConversationID:  conv.ID,
		AccountID:       in.Account.ID,
		Direction:       "outbound",
		VendorMessageID: in.VendorID,
		MessageType:     in.MessageType,
		Status:          "accepted",
		Body:            in.Body,
	}
	if err := s.Messages.Create(ctx, msg); err != nil {
		return store.Message{}, store.Conversation{}, err
	}
	_ = s.Conversations.Touch(ctx, conv.ID, false)
	s.SSE.Broadcast(EventNewMessage, msg)
	s.Dispatcher.ScheduleOutboundLifecycle(ctx, in.Account, *msg, phone)
	return *msg, conv, nil
}

type InboundInput struct {
	ConversationID uuid.UUID
	Text           string
	MessageType    string
	Body           json.RawMessage
}

func (s *Services) RecordInbound(ctx context.Context, in InboundInput) (store.Message, error) {
	conv, err := s.Conversations.GetByID(ctx, in.ConversationID)
	if err != nil {
		return store.Message{}, err
	}
	account, err := s.Accounts.GetByID(ctx, conv.AccountID)
	if err != nil {
		return store.Message{}, err
	}

	vendorID := GenerateMessageID()
	if account.Channel == "whatsapp" {
		vendorID = GenerateWAMID()
	}

	body := in.Body
	if body == nil {
		if account.Channel == "rcs" {
			body, _ = json.Marshal(map[string]any{"text": in.Text})
		} else {
			body, _ = json.Marshal(map[string]any{"text": map[string]string{"body": in.Text}})
		}
	}
	msgType := in.MessageType
	if msgType == "" {
		msgType = "text"
	}

	msg := &store.Message{
		ConversationID:  conv.ID,
		AccountID:     account.ID,
		Direction:     "inbound",
		VendorMessageID: vendorID,
		MessageType:   msgType,
		Status:        "received",
		Body:          body,
	}
	if err := s.Messages.Create(ctx, msg); err != nil {
		return store.Message{}, err
	}
	_ = s.Conversations.Touch(ctx, conv.ID, true)
	until := time.Now().Add(24 * time.Hour)
	_ = s.Conversations.ExtendWindow(ctx, conv.ID, until)

	phone := conv.UserPhone
	if account.Channel == "rcs" {
		entity := map[string]any{
			"messageId": vendorID,
			"sendTime":  time.Now().UTC().Format(time.RFC3339Nano),
			"text":      in.Text,
		}
		_ = s.Dispatcher.SendInboundRCS(ctx, account, phone, entity)
	} else {
		waMsg := map[string]any{
			"from":      NormalizePhoneWA(phone),
			"id":        vendorID,
			"timestamp": UnixTimestamp(),
			"type":      msgType,
		}
		if msgType == "text" {
			waMsg["text"] = map[string]string{"body": in.Text}
		} else {
			var extra map[string]any
			_ = json.Unmarshal(body, &extra)
			for k, v := range extra {
				waMsg[k] = v
			}
		}
		_ = s.Dispatcher.SendInboundWA(ctx, account, phone, waMsg)
	}

	s.SSE.Broadcast(EventNewMessage, msg)
	return *msg, nil
}

func (s *Services) TriggerMessageStatus(ctx context.Context, messageID uuid.UUID, status string) error {
	msg, err := s.Messages.GetByID(ctx, messageID)
	if err != nil {
		return err
	}
	conv, err := s.Conversations.GetByID(ctx, msg.ConversationID)
	if err != nil {
		return err
	}
	account, err := s.Accounts.GetByID(ctx, msg.AccountID)
	if err != nil {
		return err
	}
	_ = s.Messages.UpdateStatus(ctx, messageID, status)
	mid := msg.ID

	if account.Channel == "rcs" {
		switch status {
		case "delivered":
			s.Dispatcher.enqueueRCSStatus(ctx, account, &mid, conv.UserPhone, msg.VendorMessageID, "SEND_MESSAGE_SUCCESS", 0)
		case "read":
			s.Dispatcher.enqueueRCSRead(ctx, account, &mid, conv.UserPhone, msg.VendorMessageID, 0)
		case "failed":
			payload := map[string]any{
				"userPhoneNumber": conv.UserPhone,
				"botId":           account.ExternalID,
				"entityType":      "STATUS_EVENT",
				"entity": map[string]any{
					"eventType": "SEND_MESSAGE_FAILURE",
					"messageId": msg.VendorMessageID,
					"sendTime":  time.Now().UTC().Format(time.RFC3339Nano),
					"error": map[string]any{
						"code":    "500",
						"errCode": "SIMULATED_FAILURE",
						"message": "Simulated delivery failure",
					},
					"eventId": GenerateMessageID(),
				},
			}
			_ = s.Dispatcher.Enqueue(ctx, account, &mid, "rcs", "SEND_MESSAGE_FAILURE", payload, 0)
		case "revoked":
			payload := map[string]any{
				"userPhoneNumber": conv.UserPhone,
				"botId":           account.ExternalID,
				"entityType":      "SERVER_EVENT",
				"entity": map[string]any{
					"eventType": "TTL_EXPIRATION_REVOKED",
					"messageId": msg.VendorMessageID,
					"sendTime":  time.Now().UTC().Format(time.RFC3339Nano),
					"eventId":   GenerateMessageID(),
				},
			}
			_ = s.Dispatcher.Enqueue(ctx, account, &mid, "rcs", "TTL_EXPIRATION_REVOKED", payload, 0)
		}
	} else {
		s.Dispatcher.enqueueWAStatus(ctx, account, &mid, conv.UserPhone, msg.VendorMessageID, status, 0)
	}
	return nil
}
