# Frontend BNI Finance Hub: bundel Vite disajikan nginx, yang juga meneruskan
# /api dan /uploads ke kontainer api. Satu origin, jadi bundel tidak perlu tahu
# alamat backend dan cookie maupun CSP tetap sederhana.

# --- tahap bangun ------------------------------------------------------------
FROM node:22-alpine AS build
WORKDIR /src

# Lockfile lebih dulu supaya lapisan npm ci tetap ter-cache selama dependensi
# tidak berubah.
COPY package.json package-lock.json ./
RUN npm ci --no-audit --no-fund

COPY . .

# Variabel lingkungan menang atas .env.production. Di belakang nginx ini
# backend ada di origin yang sama, jadi mode awalnya Backend API dan alamatnya
# kosong. .env.local sengaja dikecualikan lewat .dockerignore: isinya alamat
# backend di mesin pengembang.
ENV VITE_USE_MOCK=false \
    VITE_API_URL=
RUN npm run build

# --- image akhir -------------------------------------------------------------
FROM nginx:1.27-alpine
COPY docker/nginx.conf /etc/nginx/conf.d/default.conf
COPY docker/keamanan.conf /etc/nginx/snippets/keamanan.conf
COPY --from=build /src/dist /usr/share/nginx/html

HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
  CMD wget -qO- http://127.0.0.1/ >/dev/null || exit 1
