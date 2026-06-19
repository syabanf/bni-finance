# EPIC-001 — Xendit Self Payment Mode

**Status**: `done`

## Goal

Implementasi Self Payment Mode: member dapat bayar invoice secara mandiri via Xendit
(Virtual Account BCA/BNI/Mandiri/BRI atau QRIS) tanpa login, dengan otomasi Lunas +
renewal +1 tahun via webhook.

## Scope

**IN:**
- Toggle Self Payment Mode (ON = Xendit, OFF = Paper.id)
- Halaman publik `/pay/:id` (no auth): info invoice → download PDF → pilih metode
- Edge Function `xendit-create-payment` (VA & QRIS)
- Edge Function `xendit-webhook` (auto-Lunas + renewal)
- VA bank picker: BCA / BNI / Mandiri / BRI
- QRIS (disabled bila amount > Rp 10.000.000)
- "Ganti metode / bank" kembali ke picker penuh
- Logo BNI di sidebar, topbar, invoice

**OUT:**
- Deploy ke produksi dengan QRIS untuk amount > 10M (Xendit limit)
- BNI VM sync di produksi (dev proxy only)

## Acceptance Criteria

- [x] Self Payment Mode toggle tersimpan di `app_settings.self_payment_mode`
- [x] `/pay/:id` accessible tanpa login
- [x] VA (BCA/BNI/Mandiri/BRI) bisa dibuat via Xendit test API
- [x] QRIS dinonaktifkan bila amount > 10M dengan pesan jelas
- [x] Webhook tandai invoice Lunas + insert payments + audit log
- [x] Renewal date member diupdate ke period_end saat pembayaran sukses
- [x] Callback "FVA created" diabaikan (hanya proses pembayaran nyata)
- [x] Tombol "Ganti metode / bank" warna merah, kembali ke picker penuh
- [x] Halaman publik: info invoice → download PDF → pilih metode
- [x] Tidak ada flicker auto-refresh di halaman publik
- [x] Logo BNI SVG di sidebar, topbar, header invoice

## Key Files

| File | Fungsi |
|---|---|
| `src/features/pay/PublicPaymentPage.tsx` | Halaman publik `/pay/:id` |
| `src/features/invoices/components/PaymentPanel.tsx` | Bank picker + VA/QRIS display |
| `src/features/invoices/components/BankLogo.tsx` | Wordmark SVG BCA/BNI/Mandiri/BRI |
| `src/components/ui/BniLogo.tsx` | Logo BNI merah SVG |
| `src/services/supabase/paymentGateway.ts` | `createXenditPayment()`, `isSelfPaymentMode()` |
| `src/features/settings/PaymentModePage.tsx` | Toggle ON/OFF + perbandingan mode |
| `supabase/functions/xendit-create-payment/index.ts` | Edge Function buat VA/QRIS |
| `supabase/functions/xendit-webhook/index.ts` | Edge Function terima callback |
| `supabase/migrations/0002_xendit_self_payment.sql` | Kolom Xendit di invoices/payments |

## Automation Log

| Date | Agent | Action | Result |
| --- | --- | --- | --- |
| 2026-06-19 | Claude Code | Implementasi penuh Self Payment Mode | done |
| 2026-06-19 | Claude Code | UX fixes: picker-first, red button, page order, no flicker | done |
| 2026-06-19 | Claude Code | Logo BNI SVG di dashboard + invoice | done |
| 2026-06-19 | Claude Code | Webhook guard FVA created events | done |
| 2026-06-19 | Claude Code | Deploy ke Vercel (bni-finance-five.vercel.app) | done |
