import type { ProofRepository } from '@/services/types'
import type { PaymentProof } from '@/types'
import { mockAuthRepository } from './authRepository'
import { mockInvoiceRepository } from './invoiceRepository'
import { delay, nextId, nowISO, store } from './store'

/**
 * Bukti bayar versi contoh, dengan aturan yang sama dengan backend: satu bukti
 * menunggu per invoice, hanya invoice terkirim yang belum lunas, alasan wajib
 * saat menolak, dan bukti yang sudah diperiksa tidak bisa ditarik.
 */
const proofs: PaymentProof[] = []

const pelaku = () => mockAuthRepository.getCurrentUser()?.name ?? 'Pengguna Demo'

/** Dipakai Konfirmasi Renewal untuk tahap "bukti menunggu". */
export function buktiMenunggu(invoiceId: string): boolean {
  return proofs.some((p) => p.invoiceId === invoiceId && p.status === 'pending')
}

function isiRelasi(p: PaymentProof): PaymentProof {
  const inv = store.invoices.find((i) => i.id === p.invoiceId)
  const m = store.members.find((x) => x.id === inv?.memberId)
  const c = store.chapters.find((x) => x.id === p.chapterId)
  return {
    ...p,
    invoiceNumber: inv?.number ?? '',
    invoiceAmount: inv?.amount ?? 0,
    invoiceStatus: inv?.status ?? 'draft',
    memberName: m?.name ?? null,
    chapterName: c?.displayName ?? null,
  }
}

function cari(id: string): PaymentProof {
  const p = proofs.find((x) => x.id === id)
  if (!p) throw new Error('Bukti bayar tidak ditemukan.')
  return p
}

export const mockProofRepository: ProofRepository = {
  async list(params) {
    const out = proofs
      .filter((p) => !params?.status || p.status === params.status)
      .filter((p) => !params?.invoiceId || p.invoiceId === params.invoiceId)
      .map(isiRelasi)
      .sort(
        (a, b) =>
          Number(b.status === 'pending') - Number(a.status === 'pending') ||
          b.submittedAt.localeCompare(a.submittedAt),
      )
    return delay(out, 200)
  },

  async submit({ file, ...input }) {
    const inv = store.invoices.find((i) => i.id === input.invoiceId)
    if (!inv) throw new Error('invoice tidak ditemukan')
    if (inv.status !== 'sent' && inv.status !== 'overdue') {
      throw new Error(`bukti bayar hanya untuk invoice yang sudah dikirim dan belum lunas (status sekarang ${inv.status})`)
    }
    if (buktiMenunggu(inv.id)) {
      throw new Error('invoice ini sudah punya bukti bayar yang menunggu verifikasi finance')
    }
    const p: PaymentProof = {
      id: nextId('prf'),
      invoiceId: inv.id,
      chapterId: inv.chapterId,
      amount: input.amount,
      paidAt: input.paidAt,
      paymentMethod: input.paymentMethod || null,
      // Mock tidak punya penyimpanan; object URL cukup untuk dilihat selama sesi.
      proofUrl: URL.createObjectURL(file),
      note: input.note?.trim() || null,
      status: 'pending',
      submittedBy: pelaku(),
      submittedAt: nowISO(),
      invoiceNumber: '',
      invoiceAmount: 0,
      invoiceStatus: inv.status,
    }
    proofs.unshift(p)
    return delay(isiRelasi(p), 300)
  },

  async approve(id, note) {
    const p = cari(id)
    if (p.status !== 'pending') throw new Error('bukti ini sudah diperiksa')
    const inv = store.invoices.find((i) => i.id === p.invoiceId)
    if (inv?.status === 'paid') throw new Error('invoice ini sudah lunas; tolak bukti ini bila pembayarannya sama')
    await mockInvoiceRepository.recordManualPayment(p.invoiceId, {
      amount: p.amount,
      paidAt: p.paidAt,
      method: p.paymentMethod ?? 'bank_transfer',
      note: p.note ?? undefined,
      proofUrl: p.proofUrl,
    })
    Object.assign(p, {
      status: 'approved',
      reviewedBy: pelaku(),
      reviewedAt: nowISO(),
      reviewNote: note?.trim() || null,
      paymentId: store.payments.find((x) => x.invoiceId === p.invoiceId)?.id ?? null,
    })
    return delay(isiRelasi(p), 200)
  },

  async reject(id, note) {
    if (!note.trim()) throw new Error('tulis alasan penolakan, supaya ST tahu apa yang harus diperbaiki')
    const p = cari(id)
    if (p.status !== 'pending') throw new Error('bukti ini sudah diperiksa')
    Object.assign(p, { status: 'rejected', reviewedBy: pelaku(), reviewedAt: nowISO(), reviewNote: note.trim() })
    return delay(isiRelasi(p), 200)
  },

  async withdraw(id) {
    const p = cari(id)
    if (p.status !== 'pending') throw new Error('bukti yang sudah diperiksa finance tidak bisa ditarik')
    proofs.splice(proofs.indexOf(p), 1)
    return delay(undefined, 200)
  },
}
