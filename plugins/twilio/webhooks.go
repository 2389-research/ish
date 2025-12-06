// ABOUTME: Async webhook delivery system for Twilio plugin
// ABOUTME: Simulates realistic status callback timing

package twilio

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"time"
)

// StartWebhookWorker polls the webhook queue and delivers pending webhooks
func (p *TwilioPlugin) StartWebhookWorker(ctx context.Context) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.processWebhookQueue()
		}
	}
}

func (p *TwilioPlugin) processWebhookQueue() {
	// Get pending webhooks that are ready to deliver
	webhooks, err := p.store.GetPendingWebhooks(time.Now())
	if err != nil {
		log.Printf("Error fetching pending webhooks: %v", err)
		return
	}

	for _, webhook := range webhooks {
		p.deliverWebhook(webhook)
	}
}

func (p *TwilioPlugin) deliverWebhook(webhook WebhookQueueItem) {
	// Parse payload as form values
	values, err := url.ParseQuery(webhook.Payload)
	if err != nil {
		log.Printf("Error parsing webhook payload: %v", err)
		p.store.MarkWebhookFailed(webhook.ID)
		return
	}

	// Send POST request
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.PostForm(webhook.WebhookURL, values)
	if err != nil {
		log.Printf("Error delivering webhook to %s: %v", webhook.WebhookURL, err)
		p.store.MarkWebhookFailed(webhook.ID)
		return
	}
	defer resp.Body.Close()

	// Mark as delivered
	if err := p.store.MarkWebhookDelivered(webhook.ID); err != nil {
		log.Printf("Error marking webhook delivered: %v", err)
	}
}

// QueueMessageWebhook schedules a webhook for a message status change
func (p *TwilioPlugin) QueueMessageWebhook(messageSid, status string, delay time.Duration) error {
	msg, err := p.store.GetMessage(messageSid)
	if err != nil {
		return err
	}

	// Get phone number config to find webhook URLs
	phoneNumbers, err := p.store.ListPhoneNumbers(msg.AccountSid)
	if err != nil {
		return err
	}

	var webhookURL string
	for _, pn := range phoneNumbers {
		if pn.PhoneNumber == msg.FromNumber && pn.StatusCallback != "" {
			webhookURL = pn.StatusCallback
			break
		}
	}

	// No webhook configured, skip
	if webhookURL == "" {
		return nil
	}

	// Build form-encoded payload
	payload := url.Values{}
	payload.Set("MessageSid", msg.Sid)
	payload.Set("MessageStatus", status)
	payload.Set("From", msg.FromNumber)
	payload.Set("To", msg.ToNumber)
	payload.Set("Body", msg.Body)
	payload.Set("AccountSid", msg.AccountSid)

	return p.store.QueueWebhook(messageSid, webhookURL, payload.Encode(), time.Now().Add(delay))
}

// QueueCallWebhook schedules a webhook for a call status change
func (p *TwilioPlugin) QueueCallWebhook(callSid, status string, delay time.Duration) error {
	call, err := p.store.GetCall(callSid)
	if err != nil {
		return err
	}

	phoneNumbers, err := p.store.ListPhoneNumbers(call.AccountSid)
	if err != nil {
		return err
	}

	var webhookURL string
	for _, pn := range phoneNumbers {
		if pn.PhoneNumber == call.FromNumber && pn.StatusCallback != "" {
			webhookURL = pn.StatusCallback
			break
		}
	}

	if webhookURL == "" {
		return nil
	}

	payload := url.Values{}
	payload.Set("CallSid", call.Sid)
	payload.Set("CallStatus", status)
	payload.Set("From", call.FromNumber)
	payload.Set("To", call.ToNumber)
	payload.Set("AccountSid", call.AccountSid)

	if call.Duration != nil {
		payload.Set("CallDuration", fmt.Sprintf("%d", *call.Duration))
	}

	return p.store.QueueWebhook(callSid, webhookURL, payload.Encode(), time.Now().Add(delay))
}

// SimulateMessageLifecycle progresses a message through realistic status transitions
func (p *TwilioPlugin) SimulateMessageLifecycle(messageSid string) {
	// queued → sent (100ms)
	time.AfterFunc(100*time.Millisecond, func() {
		p.store.UpdateMessageStatus(messageSid, "sent")
		p.QueueMessageWebhook(messageSid, "sent", 0)
	})

	// sent → delivered (500ms)
	time.AfterFunc(600*time.Millisecond, func() {
		p.store.UpdateMessageStatus(messageSid, "delivered")
		p.QueueMessageWebhook(messageSid, "delivered", 0)
	})
}

// SimulateCallLifecycle progresses a call through realistic status transitions
func (p *TwilioPlugin) SimulateCallLifecycle(callSid string) {
	// initiated → ringing (200ms)
	time.AfterFunc(200*time.Millisecond, func() {
		p.store.UpdateCallStatus(callSid, "ringing", nil)
		p.QueueCallWebhook(callSid, "ringing", 0)
	})

	// ringing → in-progress (800ms)
	time.AfterFunc(1000*time.Millisecond, func() {
		p.store.UpdateCallStatus(callSid, "in-progress", nil)
		p.QueueCallWebhook(callSid, "in-progress", 0)
	})

	// in-progress → completed (5-30s)
	completionDelay := time.Duration(5000+rand.Intn(25000)) * time.Millisecond
	time.AfterFunc(1000*time.Millisecond+completionDelay, func() {
		duration := 5 + rand.Intn(26) // 5-30 seconds
		p.store.UpdateCallStatus(callSid, "completed", &duration)
		p.QueueCallWebhook(callSid, "completed", 0)
	})
}
