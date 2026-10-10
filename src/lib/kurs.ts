/**
 * Rupiah dari harga Dollar dan kurs, dibulatkan ke rupiah terdekat.
 *
 * Cermin dari domain.TurunkanRupiah di backend. Dipakai halaman Pengaturan
 * untuk menampilkan Rupiah yang AKAN disimpan saat orang mengetik USD atau
 * kurs, sebelum ditekan Simpan. Yang disimpan tetap hasil hitungan server;
 * fungsi ini hanya memperlihatkannya lebih dulu, jadi aturan pembulatannya
 * harus sama persis.
 */
export function rupiahDari(usd: number, rate: number): number {
  return Math.round(usd * rate)
}
