# BNI Finance Hub — Codex Instructions

@~/.agentic-workflows/claude-codex-docs-workflow.md

## Role

Codex handles workflow hygiene in this repo:
- Update status vocabulary in docs
- Normalize stale automation logs
- Review diffs and verify agent handoffs
- Tighten runbooks after Claude Code development

Do NOT run Claude Code slash commands. Translate them into concrete repo steps
or tell the user to run them in Claude Code.

## Stack Notes

- **Frontend**: Vite 5 + React 18 + TypeScript + Tailwind CSS 3 + React Router 6
- **Backend**: REST API Go 1.25 (stdlib + pgx) di atas PostgreSQL — `backend/`
- **Payment**: Paper.id (invoice, pengingat, dan callback pembayaran)
- **Hosting**: Vercel (frontend); backend Go dijalankan terpisah
- **Data source**: BNI Visitor Management API — disinkronkan `POST /api/v1/sync`
- **Auth**: Tabel `users` + JWT HS256 (mock localStorage di dev)

## Test Gate

```bash
npm run typecheck && npm run build
```

> No unit test suite yet. typecheck + build is the gate.

## Penamaan berkas

Diturunkan dari apa yang sudah dominan di repo ini, bukan dikarang. Yang
menyimpang sudah diseragamkan; penjaganya ada di
`backend/internal/apidocs/apidocs_naming_test.go`.

### Backend (Go)

| Pola | Contoh |
| --- | --- |
| Satu berkas per peran, tiga serangkai per sumber daya | `handler.go` · `service.go` · `repository.go` |
| Tes menamai sumbernya | `phone.go` → `phone_test.go` |
| Tes beraspek khusus | `repository.go` → `repository_live_test.go` |

Awalan berkas tes **wajib** menamai berkas sumber di paket yang sama. Nama yang
berdiri sendiri (`statusof_test.go`, `persist_test.go`) memaksa membuka
berkasnya dulu untuk tahu apa yang diuji, dan seiring waktu tes berhenti punya
hubungan yang bisa ditelusuri dengan kode yang dijaganya.

`internal/api` dikecualikan: paket itu harness integrasi yang menguji API
sebagai kotak hitam. Satu-satunya sumbernya `router.go`, sementara tesnya
berbicara tentang autentikasi, konkurensi, beban, dan perjalanan end-to-end —
memaksakan awalan `router_` akan menghasilkan nama yang lebih buruk.

### Frontend (TypeScript)

| Jenis | Pola | Contoh |
| --- | --- | --- |
| Komponen & context | `PascalCase.tsx` | `InvoiceTable.tsx` · `AuthContext.tsx` |
| Halaman | `<Nama>Page.tsx` | `InvoiceDetailPage.tsx` |
| Hook | `use<Nama>.ts` | `useAsync.ts` |
| Utilitas & data | `camelCase.ts` | `apiClient.ts` · `nav.ts` |
| Repository/service | `<domain>Repository.ts` · `<domain>Service.ts` | `invoiceRepository.ts` · `syncService.ts` |

Berkas di `services/mock/` harus **sepadan namanya** dengan pasangannya di
`services/api/` — keduanya implementasi kontrak yang sama, dan nama yang beda
menyembunyikan fakta itu.

### Dokumen & skrip

| Lokasi | Pola | Contoh |
| --- | --- | --- |
| `docs/` | `UPPERCASE-KEBAB.md` | `OVERVIEW.md` · `QA-E2E.md` · `SYSTEM-PLAN.md` |
| `docs/features/` | `NN-nama.md` berurut | `02-invoice.md` |
| `docs/epics/` | `EPIC-NNN-nama.md` | `EPIC-001-xendit-self-payment.md` |
| `scripts/` | `kebab-case.<ext>` | `build-api-collection.mjs` · `setup-env.sh` |
