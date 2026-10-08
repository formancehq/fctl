package httpdebug

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime"
	"net/http"
	"net/url"
	"slices"
	"strings"
)

const redacted = "[REDACTED]"

func sensitive(key string) bool {
	key = strings.NewReplacer("_", "", "-", "", " ", "").Replace(strings.ToLower(key))
	for _, part := range []string{"authorization", "cookie", "secret", "token", "password", "assertion", "devicecode", "usercode", "credential", "apikey"} {
		if strings.Contains(key, part) {
			return true
		}
	}
	// Error text can echo arbitrary submitted secrets, even on OAuth failures.
	return strings.HasPrefix(key, "error") || slices.Contains([]string{"clear", "code", "message", "detail", "details", "value", "terraformstate", "tfstate"}, key)
}

func safeURL(u *url.URL) string {
	if u == nil {
		return "<nil>"
	}
	copyURL := *u
	copyURL.User, copyURL.Fragment, copyURL.RawFragment = nil, "", ""
	query, err := url.ParseQuery(copyURL.RawQuery)
	if err != nil {
		copyURL.RawQuery = "omitted"
	} else {
		redactValues(query)
		copyURL.RawQuery = query.Encode()
	}
	return copyURL.String()
}

func redactValues(values url.Values) {
	for key, entries := range values {
		for i, value := range entries {
			if sensitive(key) && !safeOAuthError(key, value) {
				entries[i] = redacted
			} else {
				entries[i] = safeString(value)
			}
		}
	}
}

func safeOAuthError(key, value string) bool {
	return key == "error" && slices.Contains([]string{
		"invalid_grant", "invalid_client", "invalid_scope", "invalid_request",
		"access_denied", "expired_token", "unauthorized_client", "unsupported_grant_type",
		"authorization_pending", "slow_down",
		"server_error", "temporarily_unavailable",
	}, value)
}

func safeString(value string) string {
	u, err := url.Parse(value)
	if err == nil && u.IsAbs() && (u.Scheme == "http" || u.Scheme == "https") {
		return safeURL(u)
	}
	return value
}

func headerTrace(headers http.Header) string {
	var trace strings.Builder
	keys := make([]string, 0, len(headers))
	for key := range headers {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	for _, key := range keys {
		values := slices.Clone(headers[key])
		if sensitive(key) {
			values = []string{redacted}
		} else {
			for i, value := range values {
				values[i] = safeString(value)
			}
		}
		if _, err := fmt.Fprintf(&trace, "header %q: %q\n", key, values); err != nil {
			return trace.String()
		}
	}
	return trace.String()
}

func bodyTrace(contentType string, data []byte, size int64, complete bool) string {
	kind, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		kind = "unknown"
	}
	omitted := fmt.Sprintf("omitted (type=%q; observed-size=%d; preview-limit=%d)", kind, size, previewLimit)
	if !complete {
		return omitted
	}
	switch {
	case kind == "application/json" || strings.HasSuffix(kind, "+json"):
		return jsonTrace(data, omitted)
	case kind == "application/x-www-form-urlencoded":
		values, err := url.ParseQuery(string(data))
		if err != nil {
			return omitted
		}
		redactValues(values)
		return fmt.Sprintf("%q", values.Encode())
	default:
		return omitted
	}
}

func jsonTrace(data []byte, omitted string) string {
	if !json.Valid(data) {
		return omitted
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return omitted
	}
	encoded, err := json.Marshal(redactJSON(value))
	if err != nil {
		return omitted
	}
	return string(encoded)
}

func redactJSON(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			typed[key] = redactField(key, child)
		}
	case []any:
		for i, child := range typed {
			typed[i] = redactJSON(child)
		}
	case string:
		return safeString(typed)
	}
	return value
}

func redactField(key string, value any) any {
	if text, ok := value.(string); ok && safeOAuthError(key, text) {
		return text
	}
	if sensitive(key) {
		return redacted
	}
	return redactJSON(value)
}
