package domain

import (
	"fmt"
	"math"
	"strings"
	"time"
)

// FeeSettings is the singleton row (`id = 'default'`) holding the current
// registration and renewal fees used when an invoice is created.
type FeeSettings struct {
	ID string `json:"id"`
	// RegistrationFee dan RenewalFee adalah angka yang DITAGIHKAN, dalam
	// Rupiah. Keduanya diturunkan server dari USD × kurs; lihat
	// TurunkanRupiah.
	RegistrationFee int64  `json:"registrationFee"`
	RenewalFee      int64  `json:"renewalFee"`
	Currency        string `json:"currency"`

	// Harga dasar dalam Dollar, seperti yang ditetapkan BNI, beserta kurs
	// yang dipakai mengubahnya ke Rupiah. Kursnya diisi manual oleh admin.
	RegistrationFeeUSD float64 `json:"registrationFeeUsd"`
	RenewalFeeUSD      float64 `json:"renewalFeeUsd"`
	UsdRate            int64   `json:"usdRate"`

	Notes     *string   `json:"notes,omitempty"`
	UpdatedBy *string   `json:"updatedBy,omitempty"`
	UpdatedAt time.Time `json:"updatedAt"`
	CreatedAt time.Time `json:"createdAt"`
}

type UpdateFeeSettingsInput struct {
	RegistrationFee    *int64   `json:"registrationFee"`
	RenewalFee         *int64   `json:"renewalFee"`
	RegistrationFeeUSD *float64 `json:"registrationFeeUsd"`
	RenewalFeeUSD      *float64 `json:"renewalFeeUsd"`
	UsdRate            *int64   `json:"usdRate"`
	Currency           *string  `json:"currency"`
	Notes              *string  `json:"notes"`
	UpdatedBy          *string  `json:"updatedBy"`
}

// MenyentuhDollar melaporkan permintaan ini mengubah USD atau kurs, yang
// berarti Rupiahnya harus dihitung ulang.
func (in UpdateFeeSettingsInput) MenyentuhDollar() bool {
	return in.RegistrationFeeUSD != nil || in.RenewalFeeUSD != nil || in.UsdRate != nil
}

func (in UpdateFeeSettingsInput) Validate() error {
	switch {
	case in.RegistrationFee != nil && *in.RegistrationFee < 0:
		return fmt.Errorf("registrationFee tidak boleh negatif")
	case in.RenewalFee != nil && *in.RenewalFee < 0:
		return fmt.Errorf("renewalFee tidak boleh negatif")
	case in.RegistrationFeeUSD != nil && *in.RegistrationFeeUSD < 0:
		return fmt.Errorf("registrationFeeUsd tidak boleh negatif")
	case in.RenewalFeeUSD != nil && *in.RenewalFeeUSD < 0:
		return fmt.Errorf("renewalFeeUsd tidak boleh negatif")
	case in.UsdRate != nil && *in.UsdRate <= 0:
		return fmt.Errorf("usdRate harus lebih dari nol")
	case in.Currency != nil && *in.Currency == "":
		return fmt.Errorf("currency tidak boleh kosong")
	}
	return nil
}

// TurunkanRupiah mengubah harga Dollar ke Rupiah dengan kurs yang diberikan,
// dibulatkan ke rupiah terdekat.
//
// Dibulatkan, bukan dipotong: 793,75 × 16.000 memang bulat, tapi kurs yang
// diketik orang jarang serapi itu, dan pemotongan selalu merugikan satu pihak
// yang sama. Hasilnya adalah angka yang tercetak di invoice member.
func TurunkanRupiah(usd float64, rate int64) int64 {
	return int64(math.Round(usd * float64(rate)))
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

// secretKeyParts adalah potongan nama yang menandai sebuah setting berisi
// kredensial.
//
// Daftar ini sengaja BERLEBIHAN, dan arah kesalahannya dipilih sadar: menyamarkan
// setting yang sebenarnya tidak rahasia hanya membuatnya terbaca "••••••" di UI
// — nilainya tetap bisa ditulis, karena penyamaran ini hanya berlaku saat dibaca.
// Melewatkan satu yang benar-benar rahasia membocorkannya apa adanya ke SETIAP
// pengguna yang sudah masuk, termasuk yang bukan admin.
//
// Versi pertama hanya berisi tujuh potongan dan meloloskan lima dari enam nama
// kredensial yang paling lazim. Dibuktikan dengan menulisnya lalu membacanya
// sebagai pengguna biasa:
//
//	service_role_key           -> RAHASIA-BOCOR-12345
//	signing_key                -> RAHASIA-BOCOR-12345
//	encryption_key             -> RAHASIA-BOCOR-12345
//	db_pass                    -> RAHASIA-BOCOR-12345
//	connection_string          -> RAHASIA-BOCOR-12345
//
// Semuanya lolos karena "key", "pass", dan "conn" tidak ada di daftar — hanya
// "apikey" dan "api_key", yang tidak cocok dengan "service_role_key".
var secretKeyParts = []string{
	"token", "secret", "password", "passwd", "pass",
	"apikey", "api_key", "key",
	"credential", "private", "priv",
	"conn", "dsn",
	"sign", "cert", "salt", "hash",
	"jwt", "bearer", "auth",
}

// secretKeyNames menutup nama yang berbahaya tapi tidak punya potongan penanda.
//
// "url" sengaja TIDAK dimasukkan ke secretKeyParts: base URL adalah konfigurasi
// yang justru harus terlihat oleh admin — menyamarkannya membuat orang tidak
// tahu sistem sedang menunjuk ke mana. Tapi beberapa nama berakhiran _url
// memang memuat sandi di dalamnya, dan itu ditutup di sini satu per satu.
var secretKeyNames = map[string]bool{
	"database_url": true, "db_url": true, "postgres_url": true,
	"redis_url": true, "amqp_url": true, "smtp_url": true,
}

// IsSecretKey reports whether a settings key holds a credential. `app_settings`
// stores the BNI VM token alongside harmless flags like self_payment_mode, so
// reads of those keys are redacted — writes still work, they're just write-only.
func IsSecretKey(key string) bool {
	k := strings.ToLower(strings.TrimSpace(key))
	if secretKeyNames[k] {
		return true
	}
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
