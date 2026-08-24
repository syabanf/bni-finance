#!/usr/bin/env bash
# Cetak blok env Paper.id dari backend/.env untuk ditempel ke host produksi.
#
# MENCETAK RAHASIA KE LAYAR. Jalankan hanya di mesin Anda sendiri, dan jangan
# menyalurkan keluarannya ke berkas yang ikut ter-commit.
#
# Dipakai untuk menyamakan produksi dengan dev — artinya produksi ikut memakai
# Paper.id STAGING, karena itulah yang ada di backend/.env. Konsekuensinya nyata:
# invoice dari produksi hanya ada di staging, dan tidak ada uang yang benar-benar
# masuk. PAPER_ID_BASE_URL sengaja tidak ikut dicetak — nilai bawaannya di
# internal/config/config.go sudah menunjuk staging, jadi membiarkannya kosong
# memberi hasil yang sama dan satu variabel lebih sedikit untuk salah diisi.
set -euo pipefail

env_file="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/.env"
[ -f "$env_file" ] || { echo "tidak ada $env_file" >&2; exit 1; }

echo "# --- tempel ke konfigurasi env backend produksi, lalu RESTART prosesnya ---"
echo "# Tanpa restart, tidak ada yang berubah: kredensial dibaca sekali saat start."
grep -E '^PAPER_ID_(CLIENT_ID|CLIENT_SECRET|CALLBACK_TOKEN)=' "$env_file"
echo
echo "# URL webhook yang didaftarkan di dashboard Paper.id:"
token="$(grep -E '^PAPER_ID_CALLBACK_TOKEN=' "$env_file" | cut -d= -f2-)"
for p in payment-in invoice-paid static-va payment-out supplier-payment paylater disbursement invoice-amount-due; do
  echo "#   https://bni-finance.reddie.id/api/v1/webhooks/paperid/${p}?token=${token}"
done
