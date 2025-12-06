// ABOUTME: Request log storage operations.
// ABOUTME: Handles inserting and querying HTTP request logs.

package store

import "time"

// RequestLog represents an HTTP request log entry
type RequestLog struct {
	ID           int64
	Timestamp    time.Time
	PluginName   string
	Method       string
	Path         string
	StatusCode   int
	DurationMs   int
	UserID       string
	IPAddress    string
	UserAgent    string
	Error        string
	RequestBody  string
	ResponseBody string
}

// LogRequest inserts a request log entry
func (s *Store) LogRequest(log *RequestLog) error {
	_, err := s.db.Exec(`
		INSERT INTO request_logs (plugin_name, method, path, status_code, duration_ms, user_id, ip_address, user_agent, error, request_body, response_body)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, log.PluginName, log.Method, log.Path, log.StatusCode, log.DurationMs, log.UserID, log.IPAddress, log.UserAgent, log.Error, log.RequestBody, log.ResponseBody)
	return err
}

// RequestLogQuery represents filters for request logs
type RequestLogQuery struct {
	Limit      int
	Offset     int
	PluginName string
	Method     string
	PathPrefix string
	StatusCode int
	UserID     string
}

// RequestLogStats represents aggregate statistics
type RequestLogStats struct {
	TotalRequests    int
	TodayRequests    int
	ErrorRequests    int
	AvgDurationMs    int
	UniqueEndpoints  int
	UniqueUsers      int
}

// GetRequestLogs retrieves request logs with filtering
func (s *Store) GetRequestLogs(q *RequestLogQuery) ([]*RequestLog, error) {
	query := `SELECT id, timestamp, COALESCE(plugin_name, ''), method, path, status_code, duration_ms,
	          COALESCE(user_id, ''), COALESCE(ip_address, ''), COALESCE(user_agent, ''), COALESCE(error, ''),
	          COALESCE(request_body, ''), COALESCE(response_body, '')
	          FROM request_logs WHERE 1=1`
	args := []any{}

	if q.PluginName != "" {
		query += " AND plugin_name = ?"
		args = append(args, q.PluginName)
	}
	if q.Method != "" {
		query += " AND method = ?"
		args = append(args, q.Method)
	}
	if q.PathPrefix != "" {
		query += " AND path LIKE ?"
		args = append(args, q.PathPrefix+"%")
	}
	if q.StatusCode > 0 {
		query += " AND status_code = ?"
		args = append(args, q.StatusCode)
	}
	if q.UserID != "" {
		query += " AND user_id = ?"
		args = append(args, q.UserID)
	}

	query += " ORDER BY timestamp DESC LIMIT ? OFFSET ?"
	args = append(args, q.Limit, q.Offset)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []*RequestLog
	for rows.Next() {
		log := &RequestLog{}
		var timestamp string
		if err := rows.Scan(&log.ID, &timestamp, &log.PluginName, &log.Method, &log.Path, &log.StatusCode,
			&log.DurationMs, &log.UserID, &log.IPAddress, &log.UserAgent, &log.Error,
			&log.RequestBody, &log.ResponseBody); err != nil {
			return nil, err
		}
		log.Timestamp, _ = time.Parse("2006-01-02 15:04:05", timestamp)
		logs = append(logs, log)
	}
	return logs, nil
}

// GetRequestLogStats returns aggregate statistics
func (s *Store) GetRequestLogStats() (*RequestLogStats, error) {
	stats := &RequestLogStats{}

	// Total requests
	s.db.QueryRow("SELECT COUNT(*) FROM request_logs").Scan(&stats.TotalRequests)

	// Today's requests
	today := time.Now().Format("2006-01-02")
	s.db.QueryRow("SELECT COUNT(*) FROM request_logs WHERE date(timestamp) = ?", today).Scan(&stats.TodayRequests)

	// Error requests (4xx, 5xx)
	s.db.QueryRow("SELECT COUNT(*) FROM request_logs WHERE status_code >= 400").Scan(&stats.ErrorRequests)

	// Average duration
	s.db.QueryRow("SELECT COALESCE(AVG(duration_ms), 0) FROM request_logs").Scan(&stats.AvgDurationMs)

	// Unique endpoints
	s.db.QueryRow("SELECT COUNT(DISTINCT path) FROM request_logs").Scan(&stats.UniqueEndpoints)

	// Unique users
	s.db.QueryRow("SELECT COUNT(DISTINCT user_id) FROM request_logs WHERE user_id != ''").Scan(&stats.UniqueUsers)

	return stats, nil
}

// GetTopEndpoints returns the most frequently requested endpoints
func (s *Store) GetTopEndpoints(limit int) ([]map[string]any, error) {
	rows, err := s.db.Query(`
		SELECT path, COUNT(*) as count, AVG(duration_ms) as avg_ms
		FROM request_logs
		GROUP BY path
		ORDER BY count DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var endpoints []map[string]any
	for rows.Next() {
		var path string
		var count int
		var avgMs float64
		if err := rows.Scan(&path, &count, &avgMs); err != nil {
			return nil, err
		}
		endpoints = append(endpoints, map[string]any{
			"path":   path,
			"count":  count,
			"avg_ms": int(avgMs), // Round to int for display
		})
	}
	return endpoints, nil
}

// GetPluginRequestCount returns the number of requests for a plugin since a given time
func (s *Store) GetPluginRequestCount(pluginName string, since time.Time) (int, error) {
	var count int
	err := s.db.QueryRow(`
		SELECT COUNT(*)
		FROM request_logs
		WHERE plugin_name = ? AND timestamp >= ?
	`, pluginName, since).Scan(&count)
	return count, err
}

// GetPluginErrorRate returns the error rate percentage for a plugin since a given time
func (s *Store) GetPluginErrorRate(pluginName string, since time.Time) (float64, error) {
	var totalCount, errorCount int

	// Get total requests
	err := s.db.QueryRow(`
		SELECT COUNT(*)
		FROM request_logs
		WHERE plugin_name = ? AND timestamp >= ?
	`, pluginName, since).Scan(&totalCount)
	if err != nil {
		return 0, err
	}

	// No requests means 0% error rate
	if totalCount == 0 {
		return 0, nil
	}

	// Get error requests (status >= 400)
	err = s.db.QueryRow(`
		SELECT COUNT(*)
		FROM request_logs
		WHERE plugin_name = ? AND timestamp >= ? AND status_code >= 400
	`, pluginName, since).Scan(&errorCount)
	if err != nil {
		return 0, err
	}

	// Calculate percentage
	return (float64(errorCount) / float64(totalCount)) * 100.0, nil
}

// GetRecentRequests returns the most recent requests for a plugin
func (s *Store) GetRecentRequests(pluginName string, limit int) ([]*RequestLog, error) {
	query := `SELECT id, timestamp, COALESCE(plugin_name, ''), method, path, status_code, duration_ms,
	          COALESCE(user_id, ''), COALESCE(ip_address, ''), COALESCE(user_agent, ''), COALESCE(error, ''),
	          COALESCE(request_body, ''), COALESCE(response_body, '')
	          FROM request_logs
	          WHERE plugin_name = ?
	          ORDER BY timestamp DESC
	          LIMIT ?`

	rows, err := s.db.Query(query, pluginName, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []*RequestLog
	for rows.Next() {
		log := &RequestLog{}
		var timestamp string
		if err := rows.Scan(&log.ID, &timestamp, &log.PluginName, &log.Method, &log.Path, &log.StatusCode,
			&log.DurationMs, &log.UserID, &log.IPAddress, &log.UserAgent, &log.Error,
			&log.RequestBody, &log.ResponseBody); err != nil {
			return nil, err
		}
		log.Timestamp, _ = time.Parse("2006-01-02 15:04:05", timestamp)
		logs = append(logs, log)
	}
	return logs, nil
}
