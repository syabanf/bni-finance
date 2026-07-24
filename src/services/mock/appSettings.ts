/**
 * app_settings versi mock — disimpan di localStorage.
 *
 * Mode demo harus berdiri sendiri tanpa backend. Sebelumnya halaman Pengaturan,
 * Sync, dan Metode Pembayaran memanggil endpoint API secara langsung, sehingga
 * setiap penyimpanan gagal saat mode mock. Di sini nilainya bertahan antar
 * reload, jadi demo terasa seperti aplikasi sungguhan.
 */

const PREFIX = 'mock.app_settings.'

/** Nilai awal, mencerminkan default di db/schema.sql. */
const DEFAULTS: Record<string, string> = {
  self_payment_mode: 'false',
  invoice_draft_days_before: '30',
  invoice_due_days_after: '30',
}

export async function getMockAppSetting(key: string): Promise<string | null> {
  try {
    const stored = localStorage.getItem(PREFIX + key)
    if (stored !== null) return stored
  } catch {
    // Private browsing bisa membuat localStorage melempar, bukan mengembalikan null.
  }
  return DEFAULTS[key] ?? null
}

export async function setMockAppSetting(key: string, value: string): Promise<void> {
  try {
    localStorage.setItem(PREFIX + key, value)
  } catch {
    // Penyimpanan tidak tersedia — nilainya sekadar tidak bertahan setelah reload.
  }
}
