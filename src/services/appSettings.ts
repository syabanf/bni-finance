/**
 * Konfigurasi key/value (`app_settings`) — titik komposisi yang sama seperti
 * `services/index.ts`.
 *
 * Halaman TIDAK boleh mengimpor dari `services/mock/` atau `services/api/`
 * secara langsung. Dulu tiga halaman Pengaturan melakukannya, sehingga mode
 * mock ikut menembak backend dan gagal — itulah alasan modul ini ada.
 */

import { isMockMode } from './dataSource'
import { getMockAppSetting, setMockAppSetting } from './mock/appSettings'
import { getAppSetting as getApiAppSetting, setAppSetting as setApiAppSetting } from './api/settingsRepository'

const useMock = isMockMode()

/**
 * Membaca satu konfigurasi; `null` bila belum diatur.
 *
 * Catatan mode API: key yang namanya berbau kredensial (mengandung `token`,
 * `secret`, `password`, …) terbaca **tersamar** sebagai `••••••` — server
 * sengaja menutup jalur keluarnya. Penulisan tetap berfungsi.
 */
export const getAppSetting = useMock ? getMockAppSetting : getApiAppSetting

export const setAppSetting = useMock ? setMockAppSetting : setApiAppSetting
