/**
 * Membuat invoice untuk banyak member, satu per satu, TANPA berhenti di
 * kegagalan pertama.
 *
 * Versi sebelumnya membungkus seluruh perulangan dalam satu try: member pertama
 * terbuat, member kedua ditolak server, lalu yang tampil hanya pesan gagal.
 * Orangnya tidak tahu ada invoice yang sudah terbuat, dan mengulang dari daftar
 * yang sama berarti menagih member pertama dua kali.
 */
export async function buatMassal<T extends { id: string; name: string }>(
  targets: T[],
  buat: (t: T) => Promise<unknown>,
): Promise<{ dibuat: string[]; gagal: { nama: string; sebab: string }[] }> {
  const dibuat: string[] = []
  const gagal: { nama: string; sebab: string }[] = []
  for (const t of targets) {
    try {
      await buat(t)
      dibuat.push(t.id)
    } catch (err) {
      gagal.push({ nama: t.name, sebab: err instanceof Error ? err.message : 'gagal dibuat' })
    }
  }
  return { dibuat, gagal }
}

/** Pesan toast untuk hasil buatMassal; `jenis` misalnya "renewal". */
export function pesanMassal(
  hasil: { dibuat: string[]; gagal: { nama: string; sebab: string }[] },
  jenis: string,
): { pesan: string; tone: 'success' | 'error' } {
  const terbuat = `${hasil.dibuat.length} invoice ${jenis} dibuat sebagai draft`
  if (hasil.gagal.length === 0) return { pesan: `${terbuat}.`, tone: 'success' }
  const rincian = hasil.gagal.map((g) => `${g.nama}: ${g.sebab}`).join(' · ')
  return { pesan: `${terbuat}, ${hasil.gagal.length} gagal. ${rincian}`, tone: 'error' }
}
