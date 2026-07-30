#!/usr/bin/env bash
#
# Menyiapkan berkas env kerja untuk pengembangan lokal.
#
#   ./scripts/setup-env.sh
#
# Membuat .env.local (frontend) dan backend/.env dari berkas .example, lalu
# mengisi yang bisa diisi sendiri:
#
#   - JWT_SECRET dibangkitkan acak. Ini alasan utama skrip ini ada: satu-satunya
#     cara menaruh rahasia yang benar-benar dipakai ke dalam env tanpa pernah
#     menulisnya ke berkas yang ikut ter-commit.
#   - PAPER_ID_CALLBACK_TOKEN juga dibangkitkan — tanpa itu SETIAP callback
#     Paper.id ditolak, termasuk yang asli.
#   - Kata sandi admin awal dibangkitkan, lalu dicetak sekali di akhir.
#
# Tidak pernah menimpa berkas yang sudah ada. Kredensial pihak ketiga (Paper.id,
# Xendit, BNI VM) dibiarkan kosong: hanya Anda yang punya, dan menebaknya akan
# menghasilkan konfigurasi yang gagal dengan cara membingungkan.

set -euo pipefail

cd "$(dirname "$0")/.."

green() { printf '\033[32m%s\033[0m\n' "$1"; }
dim()   { printf '\033[2m%s\033[0m\n' "$1"; }
warn()  { printf '\033[33m%s\033[0m\n' "$1"; }

# openssl ada di macOS dan hampir semua Linux; /dev/urandom sebagai cadangan.
rand() {
  if command -v openssl >/dev/null 2>&1; then
    openssl rand -base64 "${1:-36}" | tr -d '\n/+=' | cut -c1-"${2:-40}"
  else
    LC_ALL=C tr -dc 'A-Za-z0-9' < /dev/urandom | head -c "${2:-40}"
  fi
}

# set_var berkas KUNCI NILAI — mengganti baris KUNCI=... di tempat.
set_var() {
  local file=$1 key=$2 value=$3
  if grep -qE "^${key}=" "$file"; then
    # Nilai bisa memuat / dan &, jadi jangan pakai sed dengan pemisah /.
    python3 - "$file" "$key" "$value" <<'PY'
import sys
path, key, value = sys.argv[1], sys.argv[2], sys.argv[3]
lines = open(path).read().splitlines(keepends=True)
out = []
for line in lines:
    if line.startswith(key + '='):
        out.append(f'{key}={value}\n')
    else:
        out.append(line)
open(path, 'w').writelines(out)
PY
  else
    printf '%s=%s\n' "$key" "$value" >> "$file"
  fi
}

made_any=false

# --- frontend -----------------------------------------------------------------

if [ -f .env.local ]; then
  dim "· .env.local sudah ada — dilewati"
else
  cp .env.example .env.local
  set_var .env.local VITE_USE_MOCK true
  set_var .env.local VITE_API_URL http://localhost:8080
  green "✓ .env.local dibuat"
  made_any=true
fi

# --- backend ------------------------------------------------------------------

ADMIN_PASSWORD=""

if [ -f backend/.env ]; then
  dim "· backend/.env sudah ada — dilewati"
else
  cp backend/.env.example backend/.env

  JWT_SECRET=$(rand 48 60)
  CALLBACK_TOKEN=$(rand 24 32)
  ADMIN_PASSWORD=$(rand 18 24)

  set_var backend/.env JWT_SECRET "$JWT_SECRET"
  set_var backend/.env PAPER_ID_CALLBACK_TOKEN "$CALLBACK_TOKEN"
  set_var backend/.env SEED_ADMIN_PASSWORD "$ADMIN_PASSWORD"
  set_var backend/.env DATABASE_URL "postgres://postgres@localhost:5432/bni_finance_dev?sslmode=disable"

  green "✓ backend/.env dibuat — JWT_SECRET, callback token, dan kata sandi admin dibangkitkan"
  made_any=true
fi

# --- ringkasan ----------------------------------------------------------------

if [ "$made_any" = false ]; then
  dim "Tidak ada yang dibuat. Hapus berkasnya lebih dulu bila ingin menyusun ulang."
  exit 0
fi

echo
green "Siap dijalankan:"
cat <<'STEPS'

    cd backend && make db-reset && make run     # http://localhost:8080
    npm run dev                                 # http://localhost:5173

STEPS

if [ -n "$ADMIN_PASSWORD" ]; then
  ADMIN_EMAIL=$(grep -E '^SEED_ADMIN_EMAIL=' backend/.env | cut -d= -f2-)
  echo "Akun admin awal (dibuat saat tabel users masih kosong):"
  echo "    $ADMIN_EMAIL"
  echo "    $ADMIN_PASSWORD"
  echo
  dim "Kata sandi ini hanya dicetak sekali. Ada juga di backend/.env."
fi

warn "Belum diisi — hanya Anda yang punya nilainya:"
cat <<'TODO'
    PAPER_ID_CLIENT_ID / PAPER_ID_CLIENT_SECRET   penerbitan invoice
    XENDIT_SECRET_KEY / XENDIT_CALLBACK_TOKEN     pembayaran mandiri
    BNI_VM_TOKEN                                  sinkronisasi member

Semuanya opsional: fitur yang bersangkutan menjawab 503 dengan pesan yang
jelas selama kosong, sisanya tetap jalan.
TODO

echo
dim ".env.local dan backend/.env keduanya di-gitignore. Jangan pernah di-commit."
