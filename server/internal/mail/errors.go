package mail

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

var ErrSessionMissing = errors.New("尚未导入 iCloud session")

type UpstreamError struct {
	Status int
	Body   string
}

func (err *UpstreamError) Error() string {
	return fmt.Sprintf("HTTP %d: %s", err.Status, err.Body)
}

type AppleError struct {
	Payload any
}

func (err *AppleError) Error() string {
	return "iCloud returned success=false: " + safeValue(err.Payload)
}

func (err *AppleError) CodeAndRetryAfter() (string, time.Duration) {
	payload, _ := err.Payload.(map[string]any)
	nested, _ := payload["error"].(map[string]any)
	code := strings.TrimSpace(fmt.Sprint(nested["errorCode"]))
	if code == "<nil>" {
		code = "ICLOUD_ERROR"
	}
	retryAfter, _ := nested["retryAfter"].(float64)
	if retryAfter <= 0 {
		retryAfter, _ = payload["retryAfter"].(float64)
	}
	return code, time.Duration(retryAfter * float64(time.Second))
}

func isSessionExpired(err error) bool {
	var upstream *UpstreamError
	if errors.As(err, &upstream) {
		return upstream.Status == 401 || upstream.Status == 403 || upstream.Status == 421
	}
	message := err.Error()
	return strings.Contains(message, "HTTP 401") || strings.Contains(message, "HTTP 403") || strings.Contains(message, "HTTP 421")
}

func safeValue(value any) string {
	data, err := json.Marshal(sanitizeValue(value))
	if err != nil {
		return "<redacted>"
	}
	text := sanitizeText(string(data))
	if len(text) > 500 {
		return text[:500]
	}
	return text
}

func safeErrorText(err error) string {
	if err == nil {
		return ""
	}
	var upstream *UpstreamError
	if errors.As(err, &upstream) {
		return fmt.Sprintf("HTTP %d: upstream request failed", upstream.Status)
	}
	text := sanitizeText(err.Error())
	if len(text) > 500 {
		return text[:500]
	}
	return text
}

var sensitiveKeyPattern = regexp.MustCompile(`(?i)(authorization|cookie|set-cookie|password|secret|api[_-]?key|token|x-apple-[a-z0-9_-]+)`)
var sensitiveTextPattern = regexp.MustCompile(`(?i)(["']?(?:authorization|cookie|set-cookie|password|secret|api[_-]?key|token|x-apple-[a-z0-9_-]+)["']?\s*[:=]\s*)("[^"]*"|'[^']*'|[^\s,;}\]]+)`)

func sanitizeValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		clean := make(map[string]any, len(typed))
		for key, item := range typed {
			if sensitiveKeyPattern.MatchString(key) {
				clean[key] = "<redacted>"
				continue
			}
			clean[key] = sanitizeValue(item)
		}
		return clean
	case []any:
		clean := make([]any, len(typed))
		for index, item := range typed {
			clean[index] = sanitizeValue(item)
		}
		return clean
	case string:
		return sanitizeText(typed)
	default:
		return value
	}
}

func sanitizeText(value string) string {
	value = sensitiveTextPattern.ReplaceAllString(value, `${1}<redacted>`)
	for _, name := range []string{
		"X-APPLE-DS-WEB-SESSION-TOKEN",
		"X-APPLE-WEBAUTH-TOKEN",
		"X-APPLE-WEBAUTH-LOGIN",
		"X-APPLE-WEBAUTH-VALIDATE",
	} {
		value = strings.ReplaceAll(value, name, name+"<redacted>")
	}
	return value
}
