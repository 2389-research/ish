// ABOUTME: Database layer for Twilio plugin
// ABOUTME: Manages accounts, phone numbers, messages, calls, and webhook queue

package twilio

import (
	"crypto/rand"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
	"time"
)

type TwilioStore struct {
	db *sql.DB
}

type Account struct {
	AccountSid   string
	AuthToken    string
	FriendlyName string
	Status       string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func NewTwilioStore(db *sql.DB) (*TwilioStore, error) {
	store := &TwilioStore{db: db}
	if err := store.initTables(); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *TwilioStore) initTables() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS twilio_accounts (
			account_sid TEXT PRIMARY KEY,
			auth_token TEXT NOT NULL,
			friendly_name TEXT,
			status TEXT DEFAULT 'active',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		`CREATE TABLE IF NOT EXISTS twilio_phone_numbers (
			sid TEXT PRIMARY KEY,
			account_sid TEXT NOT NULL,
			phone_number TEXT NOT NULL,
			friendly_name TEXT,
			voice_url TEXT,
			voice_method TEXT DEFAULT 'POST',
			sms_url TEXT,
			sms_method TEXT DEFAULT 'POST',
			status_callback TEXT,
			status_callback_method TEXT DEFAULT 'POST',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (account_sid) REFERENCES twilio_accounts(account_sid)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_phone_numbers_account ON twilio_phone_numbers(account_sid)`,

		`CREATE TABLE IF NOT EXISTS twilio_messages (
			sid TEXT PRIMARY KEY,
			account_sid TEXT NOT NULL,
			from_number TEXT NOT NULL,
			to_number TEXT NOT NULL,
			body TEXT,
			status TEXT DEFAULT 'queued',
			direction TEXT,
			date_created TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			date_sent TIMESTAMP,
			date_updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			num_segments INTEGER DEFAULT 1,
			price REAL,
			price_unit TEXT DEFAULT 'USD',
			FOREIGN KEY (account_sid) REFERENCES twilio_accounts(account_sid)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_messages_account ON twilio_messages(account_sid)`,
		`CREATE INDEX IF NOT EXISTS idx_messages_status ON twilio_messages(status)`,
		`CREATE INDEX IF NOT EXISTS idx_messages_date ON twilio_messages(date_created)`,

		`CREATE TABLE IF NOT EXISTS twilio_calls (
			sid TEXT PRIMARY KEY,
			account_sid TEXT NOT NULL,
			from_number TEXT NOT NULL,
			to_number TEXT NOT NULL,
			status TEXT DEFAULT 'initiated',
			direction TEXT,
			duration INTEGER,
			date_created TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			date_updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			answered_by TEXT,
			FOREIGN KEY (account_sid) REFERENCES twilio_accounts(account_sid)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_calls_account ON twilio_calls(account_sid)`,
		`CREATE INDEX IF NOT EXISTS idx_calls_status ON twilio_calls(status)`,

		`CREATE TABLE IF NOT EXISTS twilio_webhook_configs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			account_sid TEXT NOT NULL,
			resource_type TEXT NOT NULL,
			event_type TEXT NOT NULL,
			url TEXT NOT NULL,
			method TEXT DEFAULT 'POST',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (account_sid) REFERENCES twilio_accounts(account_sid)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_webhook_configs_account ON twilio_webhook_configs(account_sid)`,

		`CREATE TABLE IF NOT EXISTS twilio_webhook_queue (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			resource_sid TEXT NOT NULL,
			webhook_url TEXT NOT NULL,
			payload TEXT NOT NULL,
			scheduled_at TIMESTAMP NOT NULL,
			delivered_at TIMESTAMP,
			status TEXT DEFAULT 'pending',
			attempts INTEGER DEFAULT 0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_webhook_queue_schedule ON twilio_webhook_queue(scheduled_at, status)`,
	}

	for _, query := range queries {
		if _, err := s.db.Exec(query); err != nil {
			return err
		}
	}
	return nil
}

func generateAuthToken() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func (s *TwilioStore) GetOrCreateAccount(accountSid string) (*Account, error) {
	// Try to get existing account
	var account Account
	var friendlyName sql.NullString
	err := s.db.QueryRow(`
		SELECT account_sid, auth_token, friendly_name, status, created_at, updated_at
		FROM twilio_accounts
		WHERE account_sid = ?
	`, accountSid).Scan(
		&account.AccountSid,
		&account.AuthToken,
		&friendlyName,
		&account.Status,
		&account.CreatedAt,
		&account.UpdatedAt,
	)

	if err == nil {
		if friendlyName.Valid {
			account.FriendlyName = friendlyName.String
		}
		return &account, nil
	}

	// Account doesn't exist, create it
	authToken, err := generateAuthToken()
	if err != nil {
		return nil, err
	}

	_, err = s.db.Exec(`
		INSERT INTO twilio_accounts (account_sid, auth_token)
		VALUES (?, ?)
	`, accountSid, authToken)
	if err != nil {
		return nil, err
	}

	// Fetch the newly created account
	err = s.db.QueryRow(`
		SELECT account_sid, auth_token, friendly_name, status, created_at, updated_at
		FROM twilio_accounts
		WHERE account_sid = ?
	`, accountSid).Scan(
		&account.AccountSid,
		&account.AuthToken,
		&friendlyName,
		&account.Status,
		&account.CreatedAt,
		&account.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	if friendlyName.Valid {
		account.FriendlyName = friendlyName.String
	}

	return &account, nil
}

func (s *TwilioStore) ValidateAccount(accountSid, authToken string) bool {
	var storedToken string
	err := s.db.QueryRow(`
		SELECT auth_token FROM twilio_accounts
		WHERE account_sid = ? AND status = 'active'
	`, accountSid).Scan(&storedToken)

	if err != nil {
		return false
	}

	return subtle.ConstantTimeCompare([]byte(storedToken), []byte(authToken)) == 1
}

type Message struct {
	Sid         string
	AccountSid  string
	FromNumber  string
	ToNumber    string
	Body        string
	Status      string
	Direction   string
	DateCreated time.Time
	DateSent    *time.Time
	DateUpdated time.Time
	NumSegments int
	Price       float64
	PriceUnit   string
}

func generateSID(prefix string) (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return prefix + hex.EncodeToString(bytes), nil
}

func (s *TwilioStore) CreateMessage(accountSid, from, to, body string) (*Message, error) {
	sid, err := generateSID("SM")
	if err != nil {
		return nil, err
	}

	// Calculate segments (160 chars per segment)
	numSegments := (len(body) + 159) / 160
	if numSegments == 0 {
		numSegments = 1
	}

	_, err = s.db.Exec(`
		INSERT INTO twilio_messages (sid, account_sid, from_number, to_number, body, status, direction, num_segments, price, price_unit)
		VALUES (?, ?, ?, ?, ?, 'queued', 'outbound-api', ?, ?, 'USD')
	`, sid, accountSid, from, to, body, numSegments, float64(numSegments)*0.0075)

	if err != nil {
		return nil, err
	}

	return s.GetMessage(sid)
}

func (s *TwilioStore) GetMessage(sid string) (*Message, error) {
	var msg Message
	var dateSent sql.NullTime

	err := s.db.QueryRow(`
		SELECT sid, account_sid, from_number, to_number, body, status, direction,
		       date_created, date_sent, date_updated, num_segments, price, price_unit
		FROM twilio_messages
		WHERE sid = ?
	`, sid).Scan(
		&msg.Sid, &msg.AccountSid, &msg.FromNumber, &msg.ToNumber, &msg.Body,
		&msg.Status, &msg.Direction, &msg.DateCreated, &dateSent, &msg.DateUpdated,
		&msg.NumSegments, &msg.Price, &msg.PriceUnit,
	)

	if err != nil {
		return nil, err
	}

	if dateSent.Valid {
		msg.DateSent = &dateSent.Time
	}

	return &msg, nil
}

func (s *TwilioStore) UpdateMessageStatus(sid, status string) error {
	now := time.Now()
	_, err := s.db.Exec(`
		UPDATE twilio_messages
		SET status = ?, date_updated = ?, date_sent = CASE WHEN ? IN ('sent', 'delivered') AND date_sent IS NULL THEN ? ELSE date_sent END
		WHERE sid = ?
	`, status, now, status, now, sid)
	return err
}

func (s *TwilioStore) ListMessages(accountSid string, limit int) ([]Message, error) {
	rows, err := s.db.Query(`
		SELECT sid, account_sid, from_number, to_number, body, status, direction,
		       date_created, date_sent, date_updated, num_segments, price, price_unit
		FROM twilio_messages
		WHERE account_sid = ?
		ORDER BY date_created DESC
		LIMIT ?
	`, accountSid, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var msg Message
		var dateSent sql.NullTime

		err := rows.Scan(
			&msg.Sid, &msg.AccountSid, &msg.FromNumber, &msg.ToNumber, &msg.Body,
			&msg.Status, &msg.Direction, &msg.DateCreated, &dateSent, &msg.DateUpdated,
			&msg.NumSegments, &msg.Price, &msg.PriceUnit,
		)
		if err != nil {
			return nil, err
		}

		if dateSent.Valid {
			msg.DateSent = &dateSent.Time
		}

		messages = append(messages, msg)
	}

	return messages, nil
}

type Call struct {
	Sid         string
	AccountSid  string
	FromNumber  string
	ToNumber    string
	Status      string
	Direction   string
	Duration    *int
	DateCreated time.Time
	DateUpdated time.Time
	AnsweredBy  string
}

func (s *TwilioStore) CreateCall(accountSid, from, to string) (*Call, error) {
	sid, err := generateSID("CA")
	if err != nil {
		return nil, err
	}

	_, err = s.db.Exec(`
		INSERT INTO twilio_calls (sid, account_sid, from_number, to_number, status, direction)
		VALUES (?, ?, ?, ?, 'initiated', 'outbound-api')
	`, sid, accountSid, from, to)

	if err != nil {
		return nil, err
	}

	return s.GetCall(sid)
}

func (s *TwilioStore) GetCall(sid string) (*Call, error) {
	var call Call
	var duration sql.NullInt64
	var answeredBy sql.NullString

	err := s.db.QueryRow(`
		SELECT sid, account_sid, from_number, to_number, status, direction,
		       duration, date_created, date_updated, answered_by
		FROM twilio_calls
		WHERE sid = ?
	`, sid).Scan(
		&call.Sid, &call.AccountSid, &call.FromNumber, &call.ToNumber,
		&call.Status, &call.Direction, &duration, &call.DateCreated,
		&call.DateUpdated, &answeredBy,
	)

	if err != nil {
		return nil, err
	}

	if duration.Valid {
		dur := int(duration.Int64)
		call.Duration = &dur
	}

	if answeredBy.Valid {
		call.AnsweredBy = answeredBy.String
	}

	return &call, nil
}

func (s *TwilioStore) UpdateCallStatus(sid, status string, duration *int) error {
	if duration != nil {
		_, err := s.db.Exec(`
			UPDATE twilio_calls
			SET status = ?, date_updated = ?, duration = ?
			WHERE sid = ?
		`, status, time.Now(), *duration, sid)
		return err
	}

	_, err := s.db.Exec(`
		UPDATE twilio_calls
		SET status = ?, date_updated = ?
		WHERE sid = ?
	`, status, time.Now(), sid)
	return err
}

func (s *TwilioStore) ListCalls(accountSid string, limit int) ([]Call, error) {
	rows, err := s.db.Query(`
		SELECT sid, account_sid, from_number, to_number, status, direction,
		       duration, date_created, date_updated, answered_by
		FROM twilio_calls
		WHERE account_sid = ?
		ORDER BY date_created DESC
		LIMIT ?
	`, accountSid, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var calls []Call
	for rows.Next() {
		var call Call
		var duration sql.NullInt64
		var answeredBy sql.NullString

		err := rows.Scan(
			&call.Sid, &call.AccountSid, &call.FromNumber, &call.ToNumber,
			&call.Status, &call.Direction, &duration, &call.DateCreated,
			&call.DateUpdated, &answeredBy,
		)
		if err != nil {
			return nil, err
		}

		if duration.Valid {
			dur := int(duration.Int64)
			call.Duration = &dur
		}

		if answeredBy.Valid {
			call.AnsweredBy = answeredBy.String
		}

		calls = append(calls, call)
	}

	return calls, nil
}

type PhoneNumber struct {
	Sid                  string
	AccountSid           string
	PhoneNumber          string
	FriendlyName         string
	VoiceURL             string
	VoiceMethod          string
	SmsURL               string
	SmsMethod            string
	StatusCallback       string
	StatusCallbackMethod string
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

func (s *TwilioStore) CreatePhoneNumber(accountSid, phoneNumber, friendlyName string) (*PhoneNumber, error) {
	sid, err := generateSID("PN")
	if err != nil {
		return nil, err
	}

	_, err = s.db.Exec(`
		INSERT INTO twilio_phone_numbers (sid, account_sid, phone_number, friendly_name)
		VALUES (?, ?, ?, ?)
	`, sid, accountSid, phoneNumber, friendlyName)

	if err != nil {
		return nil, err
	}

	return s.GetPhoneNumber(sid)
}

func (s *TwilioStore) GetPhoneNumber(sid string) (*PhoneNumber, error) {
	var pn PhoneNumber
	var friendlyName, voiceURL, voiceMethod, smsURL, smsMethod, statusCallback, statusCallbackMethod sql.NullString

	err := s.db.QueryRow(`
		SELECT sid, account_sid, phone_number, friendly_name, voice_url, voice_method,
		       sms_url, sms_method, status_callback, status_callback_method,
		       created_at, updated_at
		FROM twilio_phone_numbers
		WHERE sid = ?
	`, sid).Scan(
		&pn.Sid, &pn.AccountSid, &pn.PhoneNumber, &friendlyName,
		&voiceURL, &voiceMethod, &smsURL, &smsMethod,
		&statusCallback, &statusCallbackMethod,
		&pn.CreatedAt, &pn.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	if friendlyName.Valid {
		pn.FriendlyName = friendlyName.String
	}
	if voiceURL.Valid {
		pn.VoiceURL = voiceURL.String
	}
	if voiceMethod.Valid {
		pn.VoiceMethod = voiceMethod.String
	}
	if smsURL.Valid {
		pn.SmsURL = smsURL.String
	}
	if smsMethod.Valid {
		pn.SmsMethod = smsMethod.String
	}
	if statusCallback.Valid {
		pn.StatusCallback = statusCallback.String
	}
	if statusCallbackMethod.Valid {
		pn.StatusCallbackMethod = statusCallbackMethod.String
	}

	return &pn, nil
}

func (s *TwilioStore) ListPhoneNumbers(accountSid string) ([]PhoneNumber, error) {
	rows, err := s.db.Query(`
		SELECT sid, account_sid, phone_number, friendly_name, voice_url, voice_method,
		       sms_url, sms_method, status_callback, status_callback_method,
		       created_at, updated_at
		FROM twilio_phone_numbers
		WHERE account_sid = ?
		ORDER BY created_at DESC
	`, accountSid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var numbers []PhoneNumber
	for rows.Next() {
		var pn PhoneNumber
		var friendlyName, voiceURL, voiceMethod, smsURL, smsMethod, statusCallback, statusCallbackMethod sql.NullString

		err := rows.Scan(
			&pn.Sid, &pn.AccountSid, &pn.PhoneNumber, &friendlyName,
			&voiceURL, &voiceMethod, &smsURL, &smsMethod,
			&statusCallback, &statusCallbackMethod,
			&pn.CreatedAt, &pn.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		if friendlyName.Valid {
			pn.FriendlyName = friendlyName.String
		}
		if voiceURL.Valid {
			pn.VoiceURL = voiceURL.String
		}
		if voiceMethod.Valid {
			pn.VoiceMethod = voiceMethod.String
		}
		if smsURL.Valid {
			pn.SmsURL = smsURL.String
		}
		if smsMethod.Valid {
			pn.SmsMethod = smsMethod.String
		}
		if statusCallback.Valid {
			pn.StatusCallback = statusCallback.String
		}
		if statusCallbackMethod.Valid {
			pn.StatusCallbackMethod = statusCallbackMethod.String
		}

		numbers = append(numbers, pn)
	}

	return numbers, nil
}

type WebhookQueueItem struct {
	ID          int
	ResourceSid string
	WebhookURL  string
	Payload     string
	ScheduledAt time.Time
	DeliveredAt *time.Time
	Status      string
	Attempts    int
	CreatedAt   time.Time
}

func (s *TwilioStore) QueueWebhook(resourceSid, webhookURL, payload string, scheduledAt time.Time) error {
	_, err := s.db.Exec(`
		INSERT INTO twilio_webhook_queue (resource_sid, webhook_url, payload, scheduled_at)
		VALUES (?, ?, ?, ?)
	`, resourceSid, webhookURL, payload, scheduledAt)
	return err
}

func (s *TwilioStore) GetPendingWebhooks(now time.Time) ([]WebhookQueueItem, error) {
	rows, err := s.db.Query(`
		SELECT id, resource_sid, webhook_url, payload, scheduled_at, delivered_at, status, attempts, created_at
		FROM twilio_webhook_queue
		WHERE status = 'pending' AND scheduled_at <= ? AND attempts < 3
		ORDER BY scheduled_at ASC
		LIMIT 100
	`, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var webhooks []WebhookQueueItem
	for rows.Next() {
		var w WebhookQueueItem
		var deliveredAt sql.NullTime

		err := rows.Scan(&w.ID, &w.ResourceSid, &w.WebhookURL, &w.Payload,
			&w.ScheduledAt, &deliveredAt, &w.Status, &w.Attempts, &w.CreatedAt)
		if err != nil {
			return nil, err
		}

		if deliveredAt.Valid {
			w.DeliveredAt = &deliveredAt.Time
		}

		webhooks = append(webhooks, w)
	}

	return webhooks, nil
}

func (s *TwilioStore) MarkWebhookDelivered(id int) error {
	_, err := s.db.Exec(`
		UPDATE twilio_webhook_queue
		SET status = 'delivered', delivered_at = ?
		WHERE id = ?
	`, time.Now(), id)
	return err
}

func (s *TwilioStore) MarkWebhookFailed(id int) error {
	_, err := s.db.Exec(`
		UPDATE twilio_webhook_queue
		SET status = 'failed', attempts = attempts + 1
		WHERE id = ?
	`, id)
	return err
}

// ListAllMessages retrieves messages across all accounts for admin view
func (s *TwilioStore) ListAllMessages(limit, offset int) ([]Message, error) {
	rows, err := s.db.Query(`
		SELECT sid, account_sid, from_number, to_number, body, status, direction,
		       date_created, date_sent, date_updated, num_segments, price, price_unit
		FROM twilio_messages
		ORDER BY date_created DESC
		LIMIT ? OFFSET ?
	`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var msg Message
		var dateSent sql.NullTime

		err := rows.Scan(
			&msg.Sid, &msg.AccountSid, &msg.FromNumber, &msg.ToNumber, &msg.Body,
			&msg.Status, &msg.Direction, &msg.DateCreated, &dateSent, &msg.DateUpdated,
			&msg.NumSegments, &msg.Price, &msg.PriceUnit,
		)
		if err != nil {
			return nil, err
		}

		if dateSent.Valid {
			msg.DateSent = &dateSent.Time
		}

		messages = append(messages, msg)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return messages, nil
}

// ListAllCalls retrieves calls across all accounts for admin view
func (s *TwilioStore) ListAllCalls(limit, offset int) ([]Call, error) {
	rows, err := s.db.Query(`
		SELECT sid, account_sid, from_number, to_number, status, direction,
		       duration, date_created, date_updated, answered_by
		FROM twilio_calls
		ORDER BY date_created DESC
		LIMIT ? OFFSET ?
	`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var calls []Call
	for rows.Next() {
		var call Call
		var duration sql.NullInt64
		var answeredBy sql.NullString

		err := rows.Scan(
			&call.Sid, &call.AccountSid, &call.FromNumber, &call.ToNumber,
			&call.Status, &call.Direction, &duration, &call.DateCreated,
			&call.DateUpdated, &answeredBy,
		)
		if err != nil {
			return nil, err
		}

		if duration.Valid {
			dur := int(duration.Int64)
			call.Duration = &dur
		}

		if answeredBy.Valid {
			call.AnsweredBy = answeredBy.String
		}

		calls = append(calls, call)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return calls, nil
}

// ListAllAccounts retrieves all accounts for admin view
func (s *TwilioStore) ListAllAccounts(limit, offset int) ([]Account, error) {
	rows, err := s.db.Query(`
		SELECT account_sid, auth_token, friendly_name, status, created_at, updated_at
		FROM twilio_accounts
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accounts []Account
	for rows.Next() {
		var acct Account
		var friendlyName sql.NullString
		err := rows.Scan(
			&acct.AccountSid, &acct.AuthToken, &friendlyName,
			&acct.Status, &acct.CreatedAt, &acct.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		if friendlyName.Valid {
			acct.FriendlyName = friendlyName.String
		}
		accounts = append(accounts, acct)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return accounts, nil
}

// ListAllPhoneNumbers retrieves phone numbers across all accounts for admin view
func (s *TwilioStore) ListAllPhoneNumbers(limit, offset int) ([]PhoneNumber, error) {
	rows, err := s.db.Query(`
		SELECT sid, account_sid, phone_number, friendly_name, voice_url, voice_method,
		       sms_url, sms_method, status_callback, status_callback_method, created_at, updated_at
		FROM twilio_phone_numbers
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var phoneNumbers []PhoneNumber
	for rows.Next() {
		var pn PhoneNumber
		var friendlyName, voiceURL, smsURL, statusCallback sql.NullString
		err := rows.Scan(
			&pn.Sid, &pn.AccountSid, &pn.PhoneNumber, &friendlyName,
			&voiceURL, &pn.VoiceMethod, &smsURL, &pn.SmsMethod,
			&statusCallback, &pn.StatusCallbackMethod, &pn.CreatedAt, &pn.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		if friendlyName.Valid {
			pn.FriendlyName = friendlyName.String
		}
		if voiceURL.Valid {
			pn.VoiceURL = voiceURL.String
		}
		if smsURL.Valid {
			pn.SmsURL = smsURL.String
		}
		if statusCallback.Valid {
			pn.StatusCallback = statusCallback.String
		}
		phoneNumbers = append(phoneNumbers, pn)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return phoneNumbers, nil
}
