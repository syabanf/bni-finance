#!/usr/bin/env bash
# Enkripsi env produksi supaya bisa ikut ter-commit tanpa membocorkan isinya.
#
# KENAPA INI ADA
#
# Repo ini PUBLIK. Meletakkan backend/.env.production apa adanya berarti
# menyerahkan DATABASE_URL — beserta seluruh data pribadi member di dalamnya —
# dan JWT_SECRET yang bisa dipakai siapa pun memalsukan sesi admin. Riwayat git
# tidak bisa ditarik kembali, dan pemindai kredensial menemukan berkas seperti
# itu dalam hitungan menit.
#
# Yang ikut ter-commit hanyalah .env.production.gpg. Tanpa kata kunci, isinya
# tidak berarti apa-apa bagi siapa pun yang mengunduh repo ini.
#
# gpg simetris dipilih karena dua alasan: ia BERAUTENTIKASI (perubahan pada
# berkas terenkripsi ketahuan saat dibuka, bukan diam-diam menghasilkan sampah),
# dan ia sudah ada di macOS maupun Raspberry Pi tanpa memasang apa pun.
#
#   ./secrets.sh lock      .env.production  -> .env.production.gpg   (aman di-commit)
#   ./secrets.sh unlock    .env.production.gpg -> .env               (di server)
#
# Kata kuncinya dibaca dari BNI_SECRETS_KEY, atau ditanyakan bila kosong. Kunci
# itu SATU-SATUNYA hal yang tidak boleh lewat git — kirim sekali lewat pengelola
# kata sandi, lalu simpan di server.
set -euo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")"

plain=".env.production"
cipher=".env.production.gpg"

key() {
  if [ -n "${BNI_SECRETS_KEY:-}" ]; then printf '%s' "$BNI_SECRETS_KEY"; return; fi
  # -s: tidak menggemakan ketikan. Tanpa ini kata kuncinya tertinggal di layar
  # dan di scrollback terminal.
  read -rsp "kata kunci: " k </dev/tty; echo >&2
  printf '%s' "$k"
}

case "${1:-}" in
  lock)
    [ -f "$plain" ] || { echo "tidak ada $plain" >&2; exit 1; }
    gpg --batch --yes --symmetric --cipher-algo AES256 \
        --s2k-mode 3 --s2k-count 65011712 \
        --passphrase-fd 0 -o "$cipher" "$plain" <<< "$(key)"
    echo "terkunci -> backend/$cipher  ($(wc -c < "$cipher" | tr -d ' ') byte)" >&2
    ;;
  unlock)
    [ -f "$cipher" ] || { echo "tidak ada $cipher" >&2; exit 1; }
    # Menulis ke .env, bukan menimpa .env.production: di server, .env adalah
    # berkas yang benar-benar dibaca compose.
    gpg --batch --yes --decrypt --passphrase-fd 0 -o ".env" "$cipher" <<< "$(key)" 2>/dev/null
    chmod 600 .env
    echo "terbuka -> backend/.env  ($(grep -c '^[A-Z]' .env) variabel)" >&2
    ;;
  *)
    echo "pakai: ./secrets.sh lock | unlock" >&2; exit 2 ;;
esac
