.PHONY: deploy deploy-check help

help: ## Daftar target
	@grep -hE '^[a-z-]+:.*?##' $(MAKEFILE_LIST) | sort | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-14s\033[0m %s\n", $$1, $$2}'

## Menyalakan layanan di server, dengan env yang sudah dibuka DAN diperiksa.
##
## Tiga langkah yang dulu diketik terpisah, dan langkah tengahnya yang paling
## mudah hilang: orang membuka env lalu langsung menyalakan server, dan
## `secrets.sh check` hanya dijalankan yang sudah curiga ada yang salah. Justru
## saat tidak ada yang curiga itulah env yang salah lolos.
##
## `&&`, bukan `;`. Compose tidak boleh menyala kalau pemeriksaannya gagal.
deploy: ## Buka env, periksa, lalu nyalakan layanan (dipakai di server)
	./backend/secrets.sh deploy && docker compose up -d --build

## Path relatif terhadap backend/, bukan terhadap akar repo: secrets.sh
## melakukan cd ke direktorinya sendiri di baris pertama supaya .env dan
## .env.production.gpg selalu ketemu dari mana pun ia dipanggil.
deploy-check: ## Periksa env produksi tanpa membuka atau menyalakan apa pun
	./backend/secrets.sh check .env.production
