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
#   ./secrets.sh check     periksa .env sebelum menyalakan server
#
# `check` ada karena berkas env yang SALAH tidak terlihat seperti berkas env
# yang salah. Ia tetap terbaca, servernya tetap menyala, dan yang gagal
# muncul jauh kemudian di tempat lain: tautan reset yang menunjuk localhost,
# CORS yang menolak setiap panggilan, atau kunci email yang namanya sudah
# berganti. `check` menanyakan semuanya sekaligus, sebelum ada yang menyala.
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

# nilai membaca satu variabel dari berkas env, tanpa menjalankan isinya.
#
# `source` akan MENJALANKAN berkasnya: nilai yang memuat spasi, tanda kutip,
# atau $(…) berubah jadi perintah. Kata sandi memang sering memuat ketiganya.
nilai() {
  sed -n "s/^$2=//p" "$1" | head -1 | sed -e 's/^"//' -e 's/"$//' -e "s/^'//" -e "s/'\$//"
}

ada() { grep -qE "^$2=" "$1"; }

# --- daftar variabel ---------------------------------------------------------
#
# DIJAGA AGAR TIDAK BASI: internal/config/config_secrets_test.go membandingkan
# daftar ini dengan yang benar-benar dibaca config.go, dan memerah bila ada
# variabel baru yang lupa didaftarkan di sini. Tanpa penjaga itu, `check` akan
# perlahan jadi hiasan yang selalu hijau.
WAJIB="DATABASE_URL JWT_SECRET"
PENTING="ALLOWED_ORIGINS APP_BASE_URL RESEND_API_KEY MAIL_FROM PAPER_ID_BASE_URL PAPER_ID_CLIENT_ID PAPER_ID_CLIENT_SECRET PAPER_ID_CALLBACK_TOKEN"
SANTAI="PORT TOKEN_TTL UPLOAD_DIR MAX_UPLOAD_SIZE SEED_ADMIN_EMAIL SEED_ADMIN_PASSWORD SEED_ADMIN_NAME AUTH_QUICK_LOGIN DB_MAX_CONNS METRICS_TOKEN BNI_VM_URL BNI_VM_TOKEN BLACKBOX_SIZE BLACKBOX_RETAIN"

periksa() {
  local f="$1" galat=0 peringatan=0
  echo "memeriksa $f" >&2
  echo >&2

  for v in $WAJIB; do
    if [ -z "$(nilai "$f" "$v")" ]; then
      echo "  GALAT  $v kosong — server tidak akan menyala" >&2; galat=$((galat+1))
    fi
  done

  local jwt; jwt="$(nilai "$f" JWT_SECRET)"
  if [ -n "$jwt" ] && [ "${#jwt}" -lt 32 ]; then
    echo "  GALAT  JWT_SECRET hanya ${#jwt} karakter — minimal 32" >&2; galat=$((galat+1))
  fi

  for v in $PENTING; do
    if [ -z "$(nilai "$f" "$v")" ]; then
      echo "  KOSONG $v — fitur yang memakainya menjawab 503" >&2; peringatan=$((peringatan+1))
    fi
  done

  # --- yang terlihat benar tapi salah ---------------------------------------

  local base; base="$(nilai "$f" APP_BASE_URL)"
  case "$base" in
    *localhost*|*127.0.0.1*)
      echo "  BAHAYA APP_BASE_URL=$base — tautan reset di email akan menunjuk" >&2
      echo "         mesin PENERIMANYA. Emailnya terkirim, lognya bersih, dan" >&2
      echo "         yang gagal hanya orang yang menekan tautannya." >&2
      galat=$((galat+1)) ;;
  esac

  local origins; origins="$(nilai "$f" ALLOWED_ORIGINS)"
  case "$origins" in
    *"*"*)
      echo "  BAHAYA ALLOWED_ORIGINS memuat \"*\" — situs mana pun boleh memanggil API ini" >&2
      galat=$((galat+1)) ;;
    *localhost*|*127.0.0.1*)
      echo "  BAHAYA ALLOWED_ORIGINS masih memuat localhost: $origins" >&2
      peringatan=$((peringatan+1)) ;;
  esac

  if [ -n "$(nilai "$f" AUTH_QUICK_LOGIN)" ]; then
    echo "  BAHAYA AUTH_QUICK_LOGIN terisi — akun itu bisa masuk TANPA kata sandi" >&2
    galat=$((galat+1))
  fi

  local dari; dari="$(nilai "$f" MAIL_FROM)"
  if [ -n "$dari" ] && ! printf '%s' "$dari" | grep -q '@'; then
    echo "  GALAT  MAIL_FROM=\"$dari\" tidak memuat alamat email" >&2; galat=$((galat+1))
  fi

  # Nama variabel email BERUBAH saat pengiriman pindah dari SMTP ke REST API
  # Resend. Berkas yang belum ikut berubah kehilangan kemampuan mengirim email
  # tanpa satu pun galat: kredensialnya masih ada, isinya masih benar, hanya
  # namanya yang tidak dibaca lagi.
  if ada "$f" SMTP_PASSWORD || ada "$f" SMTP_HOST; then
    echo "  BASI   SMTP_* masih ada tapi tidak dibaca lagi:" >&2
    echo "         SMTP_PASSWORD -> RESEND_API_KEY, SMTP_FROM -> MAIL_FROM" >&2
    peringatan=$((peringatan+1))
  fi

  # Variabel yang tidak dikenal siapa pun: salah ketik terlihat persis seperti
  # ini, dan salah ketik pada nama variabel tidak pernah memunculkan galat.
  # SMTP_* sengaja ikut "dikenal": ia sudah dilaporkan sebagai BASI di atas,
  # dengan penggantinya disebutkan. Melaporkannya dua kali dengan dua nama
  # masalah yang berbeda membuat orang berhenti membaca keluaran ini.
  local dikenal=" $WAJIB $PENTING $SANTAI SMTP_HOST SMTP_PORT SMTP_USER SMTP_PASSWORD SMTP_FROM "
  while read -r v; do
    case "$dikenal" in *" $v "*) ;; *)
      echo "  ASING  $v tidak dibaca kode mana pun — salah ketik, atau sisa lama" >&2
      peringatan=$((peringatan+1)) ;;
    esac
  done < <(grep -oE '^[A-Z_0-9]+=' "$f" | tr -d '=' | sort -u)

  echo >&2
  if [ "$galat" -gt 0 ]; then
    echo "TIDAK LULUS — $galat galat, $peringatan peringatan" >&2
    exit 1
  fi
  if [ "$peringatan" -gt 0 ]; then
    echo "LULUS dengan $peringatan peringatan" >&2
  else
    echo "LULUS — semua variabel terisi dan tidak ada yang mencurigakan" >&2
  fi
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
    #
    # Yang lama disalin dulu. Di server .env memang selalu hasil unlock, tapi di
    # mesin pengembang ia berisi konfigurasi lokal — DATABASE_URL ke basis data
    # dev, APP_BASE_URL ke localhost — dan menimpanya diam-diam berarti
    # kehilangannya tanpa satu pun pertanyaan. (Saya menabraknya sendiri saat
    # menyegel berkas ini.)
    if [ -f ".env" ]; then
      cp -p ".env" ".env.sebelum-unlock"
      echo "salinan .env lama -> backend/.env.sebelum-unlock" >&2
    fi
    gpg --batch --yes --decrypt --passphrase-fd 0 -o ".env" "$cipher" <<< "$(key)" 2>/dev/null
    chmod 600 .env
    echo "terbuka -> backend/.env  ($(grep -c '^[A-Z]' .env) variabel)" >&2
    ;;
  check)
    berkas="${2:-.env}"
    [ -f "$berkas" ] || { echo "tidak ada $berkas" >&2; exit 1; }
    periksa "$berkas"
    ;;
  *)
    echo "pakai: ./secrets.sh lock | unlock | check [berkas]" >&2; exit 2 ;;
esac
