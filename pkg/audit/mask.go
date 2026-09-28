package audit

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"strings"
)

// hmacVersion prefixes every subject so the HMAC key can rotate without losing
// the ability to match historical events.
const hmacVersion = "v1"

var nonDigits = regexp.MustCompile(`\D`)

// Subject returns a keyed HMAC of a CPF/CNPJ, formatted "hmac:v1:<hex>", so the
// audit trail can answer "who accessed document X" without storing the number.
// Digits are extracted first so formatting differences hash the same. An empty
// input (or no key) returns "".
func (e *Emitter) Subject(cpfOrCNPJ string) string {
	digits := nonDigits.ReplaceAllString(cpfOrCNPJ, "")
	if digits == "" || len(e.hmacKey) == 0 {
		return ""
	}
	mac := hmac.New(sha256.New, e.hmacKey)
	mac.Write([]byte(digits))
	return "hmac:" + hmacVersion + ":" + hex.EncodeToString(mac.Sum(nil))
}

// sensitiveKey matches metadata/changes keys whose values must never be stored.
var sensitiveKey = regexp.MustCompile(`(?i)pass|secret|token|authorization|credential|api[_-]?key|private`)

// redact returns a copy of m with sensitive values replaced by "[redacted]",
// recursively. Used on changes/metadata so a caller passing a whole object does
// not leak secrets into the trail.
func redact(m map[string]any) map[string]any {
	if m == nil {
		return nil
	}
	out := make(map[string]any, len(m))
	for k, v := range m {
		if sensitiveKey.MatchString(k) {
			out[k] = "[redacted]"
			continue
		}
		switch vv := v.(type) {
		case map[string]any:
			out[k] = redact(vv)
		default:
			out[k] = v
		}
	}
	return out
}

// MaskEmail keeps the first character and the domain: "jo***@zarv.com".
func MaskEmail(email string) string {
	at := strings.LastIndexByte(email, '@')
	if at <= 0 {
		return "***"
	}
	local, domain := email[:at], email[at:]
	if len(local) <= 1 {
		return "*" + domain
	}
	return local[:1] + "***" + domain
}
