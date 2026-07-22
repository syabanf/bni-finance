package domain

import (
	"fmt"
	"strings"
	"time"
)

// FeeSettings is the singleton row (`id = 'default'`) holding the current
// registration and renewal fees used when an invoice is created.
type FeeSettings struct {
	ID              string    `json:"id"`
	RegistrationFee int64     `json:"registrationFee"`
	RenewalFee      int64     `json:"renewalFee"`
	Currency        string    `json:"currency"`
	Notes           *string   `json:"notes,omitempty"`
	UpdatedBy       *string   `json:"updatedBy,omitempty"`
	UpdatedAt       time.Time `json:"updatedAt"`
	CreatedAt       time.Time `json:"createdAt"`
}

type UpdateFeeSettingsInput struct {
	RegistrationFee *int64  `json:"registrationFee"`
	RenewalFee      *int64  `json:"renewalFee"`
	Currency        *string `json:"currency"`
	Notes           *string `json:"notes"`
	UpdatedBy       *string `json:"updatedBy"`
}

func (in UpdateFeeSettingsInput) Validate() error {
	switch {
	case in.RegistrationFee != nil && *in.RegistrationFee < 0:
		return fmt.Errorf("registrationFee tidak boleh negatif")
	case in.RenewalFee != nil && *in.RenewalFee < 0:
		return fmt.Errorf("renewalFee tidak boleh negatif")
	case in.Currency != nil && *in.Currency == "":
		return fmt.Errorf("currency tidak boleh kosong")
	}
	return nil
}

// FeeFor returns the fee that applies to an invoice type.
func (f FeeSettings) FeeFor(t InvoiceType) int64 {
	if t == TypeRegistration {
		return f.RegistrationFee
	}
	return f.RenewalFee
}

// AppSetting is one row of the `app_settings` key/value table.
type AppSetting struct {
	Key       string     `json:"key"`
	Value     string     `json:"value"`
	UpdatedAt *time.Time `json:"updatedAt,omitempty"`
	// Masked is true when Value was redacted because the key looks like a
	// credential (see IsSecretKey).
	Masked bool `json:"masked,omitempty"`
}

// MaskedValue is what a secret setting reads back as.
const MaskedValue = "••••••"

var secretKeyParts = []string{"token", "secret", "password", "apikey", "api_key", "credential", "private"}

// IsSecretKey reports whether a settings key holds a credential. `app_settings`
// stores the BNI VM token alongside harmless flags like self_payment_mode, so
// reads of those keys are redacted — writes still work, they're just write-only.
func IsSecretKey(key string) bool {
	k := strings.ToLower(key)
	for _, part := range secretKeyParts {
		if strings.Contains(k, part) {
			return true
		}
	}
	return false
}

// Redact returns a copy safe to send over the wire.
func (s AppSetting) Redact() AppSetting {
	if IsSecretKey(s.Key) && s.Value != "" {
		s.Value = MaskedValue
		s.Masked = true
	}
	return s
}

type SetAppSettingInput struct {
	Value string `json:"value"`
}
