package observability

import (
	"log/slog"
	"regexp"
	"strings"
)

const redacted = "[REDACTED]"

// Key atribut yang nilainya selalu disamarkan (dibandingkan lowercase).
var sensitiveKeys = map[string]struct{}{
	"password": {}, "passwd": {}, "pwd": {}, "secret": {},
	"session_id": {}, "session": {}, "session_token": {},
	"csrf_token": {}, "csrf": {}, "token": {}, "access_token": {},
	"cookie": {}, "set-cookie": {}, "authorization": {},
	"dsn": {}, "connection_string": {}, "conn_str": {},
}

var redactPatterns = []struct {
	re   *regexp.Regexp
	repl string
}{
	// key=value: password=abc, password='a b', session_id=xyz
	{regexp.MustCompile(`(?i)\b(password|passwd|pwd|session_id|csrf_token|access_token|token)\s*=\s*('(?:[^'\\]|\\.)*'|"[^"]*"|[^\s&;,]+)`), `${1}=` + redacted},
	// URL: postgres://user:pass@host
	{regexp.MustCompile(`(?i)([a-z][a-z0-9+.\-]*://[^/\s:@]+):[^@/\s]+@`), `${1}:` + redacted + `@`},
	// SQL: CREATE ROLE x PASSWORD 'abc', ALTER USER ... PASSWORD 'abc'
	{regexp.MustCompile(`(?i)\b(password)\s+'(?:[^'\\]|\\.|'')*'`), `${1} '` + redacted + `'`},
	// Header HTTP
	{regexp.MustCompile(`(?i)\b(cookie|authorization)\s*:\s*[^\r\n]+`), `${1}: ` + redacted},
}

func Redact(s string) string {
	if s == "" {
		return s
	}
	for _, p := range redactPatterns {
		s = p.re.ReplaceAllString(s, p.repl)
	}
	return s
}

func isSensitiveKey(key string) bool {
	k := strings.ToLower(key)
	if _, ok := sensitiveKeys[k]; ok {
		return true
	}
	return strings.Contains(k, "password") || strings.Contains(k, "secret")
}

// replaceAttr: (1) ganti key "msg" menjadi "event", (2) samarkan atribut sensitif.
func replaceAttr(groups []string, a slog.Attr) slog.Attr {
	if len(groups) == 0 && a.Key == slog.MessageKey {
		a.Key = "event"
		return a
	}
	if isSensitiveKey(a.Key) {
		return slog.String(a.Key, redacted)
	}
	switch a.Value.Kind() {
	case slog.KindString:
		a.Value = slog.StringValue(Redact(a.Value.String()))
	case slog.KindAny:
		if err, ok := a.Value.Any().(error); ok && err != nil {
			a.Value = slog.StringValue(Redact(err.Error()))
		}
	}
	return a
}
