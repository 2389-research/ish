// ABOUTME: HTTP handlers for Twilio API endpoints
// ABOUTME: Implements SMS and Voice API routes

package twilio

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
)

func (p *TwilioPlugin) sendMessage(w http.ResponseWriter, r *http.Request) {
	accountSid := r.Context().Value("account_sid").(string)

	if err := r.ParseForm(); err != nil {
		writeError(w, http.StatusBadRequest, 21602, "Missing required parameter")
		return
	}

	to := r.FormValue("To")
	from := r.FormValue("From")
	body := r.FormValue("Body")

	if to == "" || from == "" || body == "" {
		writeError(w, http.StatusBadRequest, 21602, "Missing required parameter To, From, or Body")
		return
	}

	message, err := p.store.CreateMessage(accountSid, from, to, body)
	if err != nil {
		writeError(w, http.StatusInternalServerError, 20005, "Internal server error")
		return
	}

	// Queue immediate webhook for "queued" status
	p.QueueMessageWebhook(message.Sid, "queued", 0)

	// Start async lifecycle simulation
	go p.SimulateMessageLifecycle(message.Sid)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(messageToResponse(message))
}

func (p *TwilioPlugin) getMessage(w http.ResponseWriter, r *http.Request) {
	messageSid := chi.URLParam(r, "MessageSid")

	message, err := p.store.GetMessage(messageSid)
	if err != nil {
		writeError(w, http.StatusNotFound, 20404, "Message not found")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(messageToResponse(message))
}

func (p *TwilioPlugin) listMessages(w http.ResponseWriter, r *http.Request) {
	accountSid := r.Context().Value("account_sid").(string)

	pageSize := 50
	if ps := r.URL.Query().Get("PageSize"); ps != "" {
		if parsed, err := strconv.Atoi(ps); err == nil && parsed > 0 && parsed <= 1000 {
			pageSize = parsed
		}
	}

	messages, err := p.store.ListMessages(accountSid, pageSize)
	if err != nil {
		writeError(w, http.StatusInternalServerError, 20005, "Internal server error")
		return
	}

	responseMessages := make([]map[string]interface{}, len(messages))
	for i, msg := range messages {
		responseMessages[i] = messageToResponse(&msg)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"messages":  responseMessages,
		"page":      0,
		"page_size": pageSize,
	})
}

func messageToResponse(msg *Message) map[string]interface{} {
	response := map[string]interface{}{
		"sid":           msg.Sid,
		"account_sid":   msg.AccountSid,
		"from":          msg.FromNumber,
		"to":            msg.ToNumber,
		"body":          msg.Body,
		"status":        msg.Status,
		"direction":     msg.Direction,
		"date_created":  msg.DateCreated.Format(time.RFC1123Z),
		"date_updated":  msg.DateUpdated.Format(time.RFC1123Z),
		"num_segments":  msg.NumSegments,
		"price":         msg.Price,
		"price_unit":    msg.PriceUnit,
		"error_code":    nil,
		"error_message": nil,
	}

	if msg.DateSent != nil {
		response["date_sent"] = msg.DateSent.Format(time.RFC1123Z)
	} else {
		response["date_sent"] = nil
	}

	return response
}

func writeError(w http.ResponseWriter, statusCode, errorCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"code":    errorCode,
		"message": message,
		"status":  statusCode,
	})
}

func (p *TwilioPlugin) initiateCall(w http.ResponseWriter, r *http.Request) {
	accountSid := r.Context().Value("account_sid").(string)

	if err := r.ParseForm(); err != nil {
		writeError(w, http.StatusBadRequest, 21602, "Missing required parameter")
		return
	}

	to := r.FormValue("To")
	from := r.FormValue("From")
	url := r.FormValue("Url")

	if to == "" || from == "" || url == "" {
		writeError(w, http.StatusBadRequest, 21602, "Missing required parameter To, From, or Url")
		return
	}

	call, err := p.store.CreateCall(accountSid, from, to)
	if err != nil {
		writeError(w, http.StatusInternalServerError, 20005, "Internal server error")
		return
	}

	// Queue immediate webhook for "initiated" status
	p.QueueCallWebhook(call.Sid, "initiated", 0)

	// Start async lifecycle simulation
	go p.SimulateCallLifecycle(call.Sid)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(callToResponse(call))
}

func (p *TwilioPlugin) getCall(w http.ResponseWriter, r *http.Request) {
	callSid := chi.URLParam(r, "CallSid")

	call, err := p.store.GetCall(callSid)
	if err != nil {
		writeError(w, http.StatusNotFound, 20404, "Call not found")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(callToResponse(call))
}

func (p *TwilioPlugin) listCalls(w http.ResponseWriter, r *http.Request) {
	accountSid := r.Context().Value("account_sid").(string)

	pageSize := 50
	if ps := r.URL.Query().Get("PageSize"); ps != "" {
		if parsed, err := strconv.Atoi(ps); err == nil && parsed > 0 && parsed <= 1000 {
			pageSize = parsed
		}
	}

	calls, err := p.store.ListCalls(accountSid, pageSize)
	if err != nil {
		writeError(w, http.StatusInternalServerError, 20005, "Internal server error")
		return
	}

	responseCalls := make([]map[string]interface{}, len(calls))
	for i, call := range calls {
		responseCalls[i] = callToResponse(&call)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"calls":     responseCalls,
		"page":      0,
		"page_size": pageSize,
	})
}

func callToResponse(call *Call) map[string]interface{} {
	response := map[string]interface{}{
		"sid":          call.Sid,
		"account_sid":  call.AccountSid,
		"from":         call.FromNumber,
		"to":           call.ToNumber,
		"status":       call.Status,
		"direction":    call.Direction,
		"date_created": call.DateCreated.Format(time.RFC1123Z),
		"date_updated": call.DateUpdated.Format(time.RFC1123Z),
		"answered_by":  call.AnsweredBy,
	}

	if call.Duration != nil {
		response["duration"] = strconv.Itoa(*call.Duration)
	} else {
		response["duration"] = nil
	}

	return response
}

func (p *TwilioPlugin) listPhoneNumbers(w http.ResponseWriter, r *http.Request) {
	accountSid := r.Context().Value("account_sid").(string)

	numbers, err := p.store.ListPhoneNumbers(accountSid)
	if err != nil {
		writeError(w, http.StatusInternalServerError, 20005, "Internal server error")
		return
	}

	responseNumbers := make([]map[string]interface{}, len(numbers))
	for i, num := range numbers {
		responseNumbers[i] = map[string]interface{}{
			"sid":                      num.Sid,
			"account_sid":              num.AccountSid,
			"phone_number":             num.PhoneNumber,
			"friendly_name":            num.FriendlyName,
			"voice_url":                num.VoiceURL,
			"voice_method":             num.VoiceMethod,
			"sms_url":                  num.SmsURL,
			"sms_method":               num.SmsMethod,
			"status_callback":          num.StatusCallback,
			"status_callback_method":   num.StatusCallbackMethod,
			"date_created":             num.CreatedAt.Format(time.RFC1123Z),
			"date_updated":             num.UpdatedAt.Format(time.RFC1123Z),
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"incoming_phone_numbers": responseNumbers,
	})
}
