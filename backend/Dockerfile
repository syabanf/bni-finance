# Backend BNI Finance Hub — Go 1.25, satu binary statis.
#
# Dibangun dua tahap supaya toolchain Go tidak ikut ke image akhir: hasilnya
# ~20 MB, bukan ~800 MB.

# --- tahap bangun ------------------------------------------------------------
FROM golang:1.25-alpine AS build

WORKDIR /src

# Dependensi disalin lebih dulu supaya lapisan ini tetap ter-cache selama go.mod
# dan go.sum tidak berubah — build ulang tidak perlu mengunduh apa pun.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# CGO_ENABLED=0 menghasilkan binary yang tidak bergantung libc sama sekali,
# sehingga bisa dijalankan di image apa pun. Aman di sini karena satu-satunya
# dependensi langsung adalah pgx, yang murni Go.
#
# -trimpath membuang jalur absolut mesin pembangun dari binary; tanpa itu,
# setiap stack trace di produksi memuat struktur direktori laptop seseorang.
ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=linux go build \
      -trimpath \
      -ldflags="-s -w -X main.version=${VERSION}" \
      -o /out/api ./cmd/api

# Dijalankan di dalam build supaya image yang rusak tidak pernah sampai ke
# registry. Tes yang butuh Postgres melewatkan dirinya sendiri tanpa
# TEST_DATABASE_URL, jadi tahap ini tidak memerlukan basis data.
RUN go vet ./... && go test ./... -count=1

# --- image akhir -------------------------------------------------------------
FROM alpine:3.22

# ca-certificates dipasang eksplisit sebagai PENGUNCI, bukan karena alpine
# kekurangannya — diuji, alpine:3.22 polos sudah bisa melakukan TLS keluar:
#
#   alpine:3.22 tanpa paket apa pun  -> HTTP 401 dari Paper.id (handshake sukses)
#   alpine:3.22 + ca-certificates    -> sama
#   scratch                          -> x509: certificate signed by unknown authority
#
# Baris terakhir itu alasan penguncinya ada. Bundel sertifikat di alpine datang
# lewat paket yang kebetulan ikut terbawa base image; kalau suatu saat tidak,
# atau kalau image akhir dipindah ke base yang lebih minimal, kegagalannya SUNYI:
# aplikasi tetap start, /healthz tetap hijau karena hanya menyentuh Postgres, dan
# yang mati hanya panggilan keluar ke Paper.id dan BNI Visitor Management.
#
# wget dipakai HEALTHCHECK di bawah; itu alasan image akhir memakai alpine
# alih-alih scratch, yang tidak punya alat apa pun untuk memeriksa dirinya
# sendiri.
RUN apk add --no-cache ca-certificates wget \
 && adduser -D -u 10001 -h /app app

WORKDIR /app
COPY --from=build /out/api /app/api

# Bukti pembayaran ditulis saat runtime. Dibuat dan dimiliki lebih dulu oleh
# pengguna non-root, karena prosesnya tidak akan bisa membuat direktori ini
# sendiri setelah turun hak.
RUN mkdir -p /app/uploads && chown -R app:app /app
VOLUME ["/app/uploads"]

USER app
EXPOSE 8080

ENV PORT=8080 \
    UPLOAD_DIR=/app/uploads

# /healthz ikut menyentuh basis data, jadi kontainer dilaporkan sehat hanya
# kalau Postgres benar-benar terjangkau — bukan sekadar prosesnya hidup.
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD wget -qO- "http://127.0.0.1:${PORT}/healthz" >/dev/null || exit 1

ENTRYPOINT ["/app/api"]
