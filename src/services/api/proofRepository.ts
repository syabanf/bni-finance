import { api, fileUrl, query, type ListResponse } from '@/lib/apiClient'
import type { ProofRepository } from '@/services/types'
import type { PaymentProof } from '@/types'

/** Path /uploads/... milik backend diberi alamatnya, sama seperti bukti pembayaran. */
const tampilkan = (p: PaymentProof): PaymentProof => ({ ...p, proofUrl: fileUrl(p.proofUrl) ?? p.proofUrl })

/**
 * Bukti bayar lewat backend Go. Daftarnya sudah dibatasi chapter di server,
 * dan hanya admin yang boleh menyetujui atau menolak.
 */
export const apiProofRepository: ProofRepository = {
  async list(params) {
    const semua: PaymentProof[] = []
    for (;;) {
      const res = await api.get<ListResponse<PaymentProof>>(
        `/payment-proofs${query({ status: params?.status, invoiceId: params?.invoiceId, limit: 200, offset: semua.length })}`,
      )
      semua.push(...res.data)
      if (res.data.length === 0 || semua.length >= res.meta.total) return semua.map(tampilkan)
    }
  },

  async submit({ file, ...input }) {
    // Dua langkah, seperti faktur pajak: berkasnya disimpan dulu, lalu
    // path-nya dikirim bersama data pembayaran.
    const { url } = await api.upload('/uploads', file)
    return tampilkan(
      await api.post<PaymentProof>('/payment-proofs', {
        ...input,
        paymentMethod: input.paymentMethod || null,
        note: input.note?.trim() || null,
        proofUrl: url,
      }),
    )
  },

  async approve(id, note) {
    return tampilkan(
      await api.post<PaymentProof>(`/payment-proofs/${encodeURIComponent(id)}/approve`, { note: note?.trim() || null }),
    )
  },

  async reject(id, note) {
    return tampilkan(await api.post<PaymentProof>(`/payment-proofs/${encodeURIComponent(id)}/reject`, { note }))
  },

  async withdraw(id) {
    await api.delete(`/payment-proofs/${encodeURIComponent(id)}`)
  },
}
