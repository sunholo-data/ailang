// Package claudegateway admits Anthropic inference only after durable credit reservation.
package claudegateway

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

// Capability is a bearer credential scoped to one admitted execution attempt.
// It contains no provider key and is useful only at this gateway.
type Capability struct {
	AccountID   string
	TaskID      string
	AttemptID   string
	JobIdentity string
	ExpiresAt   time.Time
}

func SignCapability(key []byte, c Capability) (string, error) {
	if len(key) < 32 || c.AccountID == "" || c.TaskID == "" || c.AttemptID == "" || c.JobIdentity == "" || c.ExpiresAt.IsZero() {
		return "", errors.New("invalid capability contract")
	}
	b, err := json.Marshal(c)
	if err != nil {
		return "", err
	}
	payload := base64.RawURLEncoding.EncodeToString(b)
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(payload))
	return payload + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil)), nil
}
func VerifyCapability(key []byte, token string, now time.Time) (Capability, error) {
	var c Capability
	parts := strings.Split(token, ".")
	if len(key) < 32 || len(parts) != 2 || len(token) > 4096 {
		return c, errors.New("invalid capability")
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return c, errors.New("invalid capability")
	}
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(parts[0]))
	if !hmac.Equal(sig, mac.Sum(nil)) {
		return c, errors.New("invalid capability")
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return c, errors.New("invalid capability")
	}
	if err = json.Unmarshal(raw, &c); err != nil || c.AccountID == "" || c.TaskID == "" || c.AttemptID == "" || c.JobIdentity == "" || !now.Before(c.ExpiresAt) {
		return Capability{}, errors.New("invalid or expired capability")
	}
	return c, nil
}
