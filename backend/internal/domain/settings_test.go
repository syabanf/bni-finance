package domain

import "testing"

func TestIsSecretKey(t *testing.T) {
	secret := []string{
		"bni_vm_token", "BNI_VM_TOKEN", "xendit_secret", "smtp_password",
		"paper_id_apikey", "paper_id_api_key", "webhook_credential", "private_key",
	}
	for _, k := range secret {
		if !IsSecretKey(k) {
			t.Errorf("%q harusnya dianggap rahasia", k)
		}
	}

	public := []string{
		"self_payment_mode", "invoice_draft_days_before", "app_origin", "locale",
	}
	for _, k := range public {
		if IsSecretKey(k) {
			t.Errorf("%q bukan rahasia, tidak boleh disamarkan", k)
		}
	}
}

func TestRedact(t *testing.T) {
	got := AppSetting{Key: "bni_vm_token", Value: "abc123"}.Redact()
	if got.Value != MaskedValue || !got.Masked {
		t.Errorf("token harus tersamar, dapat %+v", got)
	}

	plain := AppSetting{Key: "self_payment_mode", Value: "true"}.Redact()
	if plain.Value != "true" || plain.Masked {
		t.Errorf("flag biasa tidak boleh disamarkan, dapat %+v", plain)
	}

	// An empty secret stays empty — masking a blank value would imply one is set.
	empty := AppSetting{Key: "bni_vm_token", Value: ""}.Redact()
	if empty.Value != "" || empty.Masked {
		t.Errorf("nilai kosong tidak perlu disamarkan, dapat %+v", empty)
	}
}

func TestFeeFor(t *testing.T) {
	f := FeeSettings{RegistrationFee: 2_000_000, RenewalFee: 1_500_000}
	if got := f.FeeFor(TypeRegistration); got != 2_000_000 {
		t.Errorf("biaya pendaftaran: dapat %d", got)
	}
	if got := f.FeeFor(TypeRenewal); got != 1_500_000 {
		t.Errorf("biaya perpanjangan: dapat %d", got)
	}
}
