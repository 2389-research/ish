// ABOUTME: Plugin detection for request logging.
// ABOUTME: Determines which plugin a request belongs to based on URL path.

package logging

import "strings"

// GetPluginFromPath determines which plugin handles a given path
func GetPluginFromPath(path string) string {
	// Google APIs
	if strings.HasPrefix(path, "/gmail/") {
		return "google"
	}
	if strings.HasPrefix(path, "/calendar/") || strings.HasPrefix(path, "/calendars/") {
		return "google"
	}
	if strings.HasPrefix(path, "/people/") || strings.HasPrefix(path, "/v1/people") {
		return "google"
	}
	if strings.HasPrefix(path, "/tasks/") {
		return "google"
	}
	if strings.HasPrefix(path, "/oauth/google/") {
		return "google"
	}

	// Unknown
	return "unknown"
}
