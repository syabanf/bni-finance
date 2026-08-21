/**
 * Mock transport for the API console.
 *
 * The console lets you fire any documented endpoint. In Backend API mode those
 * are real HTTP calls; in Data Contoh mode they land here instead, so the whole
 * surface is explorable with no backend running.
 *
 * Answers come from the SAME in-memory store the mock app uses, so what the
 * console shows matches what the pages show — a separate set of canned replies
 * would drift and quietly lie.
 */

import { store, nowISO } from './store'
import { getMockAppSetting, setMockAppSetting } from './appSettings'
import { clearMockCalls, listMockCalls, recordMockCall } from './blackboxService'
import type { Invoice, Payment } from '@/types'

/** Paper.id memakai DD-MM-YYYY, bukan ISO. */
function ddmmyyyy(d: Date): string {
  return `${String(d.getDate()).padStart(2, '0')}-${String(d.getMonth() + 1).padStart(2, '0')}-${d.getFullYear()}`
}

function itemNameFor(type: Invoice['type']): string {
  return type === 'registration' ? 'Pendaftaran Keanggotaan BNI Grow' : 'Perpanjangan Keanggotaan BNI Grow'
}

export interface MockResponse {
  status: number
  body: unknown
}

/** Same envelope every list endpoint returns. */
function list<T>(items: T[], limit = 50, offset = 0): MockResponse {
  const page = items.slice(offset, offset + limit)
  return { status: 200, body: { data: page, meta: { total: items.length, limit, offset } } }
}

function ok(body: unknown): MockResponse {
  return { status: 200, body }
}

function created(body: unknown): MockResponse {
  return { status: 201, body }
}

function fail(status: number, message: string): MockResponse {
  return { status, body: { error: message } }
}

const notFound = () => fail(404, 'data tidak ditemukan')

function memberOf(invoice: Invoice) {
  return store.members.find((m) => m.id === invoice.memberId) ?? null
}

function chapterOf(id: string) {
  return store.chapters.find((c) => c.id === id) ?? null
}

/** Members carry their chapter, matching the real API's LEFT JOIN. */
function memberWithChapter(id: string) {
  const m = store.members.find((x) => x.id === id)
  if (!m) return null
  return { ...m, chapter: chapterOf(m.chapterId) }
}

function daysBetween(from: Date, to: Date): number {
  return Math.round((to.getTime() - from.getTime()) / 86_400_000)
}

/**
 * Routes one console request. `fullPath` is the path as written in the spec
 * (`/api/v1/invoices`, `/healthz`, …); `query` is already parsed.
 */
export async function mockApiFetch(
  method: string,
  fullPath: string,
  query: URLSearchParams,
  body: unknown,
): Promise<MockResponse> {
  const m = method.toUpperCase()
  const path = fullPath.replace(/^\/api\/v1/, '')
  const seg = path.split('/').filter(Boolean)
  const num = (key: string, fallback: number) => {
    const raw = query.get(key)
    const n = raw === null ? NaN : Number(raw)
    return Number.isFinite(n) ? n : fallback
  }
  const limit = num('limit', 50)
  const offset = num('offset', 0)

  // Simulate a little latency so the console's timing display is meaningful.
  await new Promise((r) => setTimeout(r, 60 + Math.round(Math.random() * 120)))

  // --- auth ----------------------------------------------------------------
  if (seg[0] === 'auth') {
    if (seg[1] === 'login' && m === 'POST') {
      const b = (body ?? {}) as { email?: string; password?: string }
      if (!b.email || !b.password) return fail(400, 'email dan kata sandi wajib diisi')
      return ok({
        token: 'mock.jwt.token',
        expiresAt: new Date(Date.now() + 12 * 3600_000).toISOString(),
        user: { id: 'mock-admin', name: 'Administrator', email: b.email, role: 'admin' },
      })
    }
    if (seg[1] === 'me' && m === 'GET') {
      return ok({ id: 'mock-admin', name: 'Administrator', email: 'admin@bni-finance.com', role: 'admin' })
    }
    if (seg[1] === 'me' && m === 'PATCH') {
      const b = (body ?? {}) as { name?: string }
      if (!b.name?.trim()) return fail(400, 'nama tidak boleh kosong')
      return ok({ id: 'mock-admin', name: b.name, email: 'admin@bni-finance.com', role: 'admin' })
    }
    // Masuk cepat. Di server ini dikunci daftar izin AUTH_QUICK_LOGIN dan
    // menjawab 404 saat kosong; pada Data Contoh akunnya memang akun contoh,
    // jadi selalu tersedia — sekaligus membuat endpointnya bisa dicoba dari
    // Konsol API alih-alih muncul sebagai "belum tersedia".
    if (seg[1] === 'quick-login') {
      const demo = [
        { id: 'mock-admin', name: 'Administrator', email: 'admin@bni-finance.com', role: 'admin' },
        { id: 'mock-user', name: 'Staf Keuangan', email: 'staf@bni-finance.com', role: 'user' },
      ]
      if (m === 'GET') return ok({ data: demo })
      if (m === 'POST') {
        const b = (body ?? {}) as { email?: string }
        const user = demo.find((u) => u.email === (b.email ?? '').toLowerCase())
        if (!user) return fail(403, 'akun ini tidak terdaftar untuk quick login')
        return ok({
          token: 'mock.jwt.token',
          expiresAt: new Date(Date.now() + 12 * 3600_000).toISOString(),
          user,
        })
      }
    }
    if (seg[1] === 'password' && m === 'PUT') {
      const b = (body ?? {}) as { currentPassword?: string; newPassword?: string }
      if (!b.currentPassword) return fail(401, 'kata sandi saat ini salah')
      if ((b.newPassword ?? '').length < 6) return fail(400, 'kata sandi minimal 6 karakter')
      return { status: 204, body: null }
    }
  }

  // --- users ---------------------------------------------------------------
  if (seg[0] === 'users') {
    const users = [
      { id: 'mock-admin', name: 'Administrator', email: 'admin@bni-finance.com', role: 'admin' },
      { id: 'mock-user', name: 'Staf Keuangan', email: 'staf@bni-finance.com', role: 'user' },
    ]
    if (seg.length === 1 && m === 'GET') return list(users, users.length)
    if (seg.length === 1 && m === 'POST') {
      const b = (body ?? {}) as { email?: string; name?: string; password?: string; role?: string }
      if (!b.email || !b.name) return fail(400, 'email dan nama wajib diisi')
      if ((b.password ?? '').length < 6) return fail(400, 'kata sandi minimal 6 karakter')
      return created({ id: 'mock-baru', name: b.name, email: b.email, role: b.role ?? 'user' })
    }
    if (seg[2] === 'role' && m === 'PATCH') {
      const b = (body ?? {}) as { role?: string }
      if (b.role !== 'admin' && b.role !== 'user') return fail(400, "role harus 'admin' atau 'user'")
      return ok({ id: seg[1], name: 'Staf Keuangan', email: 'staf@bni-finance.com', role: b.role })
    }
    if (seg[2] === 'password' && m === 'PUT') return { status: 204, body: null }
    if (seg.length === 2 && m === 'DELETE') return { status: 204, body: null }
  }

  // --- invoices ------------------------------------------------------------
  if (seg[0] === 'invoices') {
    if (seg.length === 1 && m === 'GET') {
      let rows = [...store.invoices]
      const status = query.get('status')
      if (status === 'outstanding') rows = rows.filter((i) => i.status === 'sent' || i.status === 'overdue')
      else if (status) rows = rows.filter((i) => i.status === status)
      const type = query.get('type')
      if (type) rows = rows.filter((i) => i.type === type)
      const chapterId = query.get('chapterId')
      if (chapterId) rows = rows.filter((i) => i.chapterId === chapterId)
      const memberId = query.get('memberId')
      if (memberId) rows = rows.filter((i) => i.memberId === memberId)
      const q = query.get('q')?.toLowerCase()
      if (q) rows = rows.filter((i) => i.number.toLowerCase().includes(q))
      rows.sort((a, b) => b.createdAt.localeCompare(a.createdAt))
      return list(rows, limit, offset)
    }
    if (seg.length === 1 && m === 'POST') {
      const b = (body ?? {}) as Partial<Invoice>
      if (!b.memberId || !b.chapterId) return fail(400, 'memberId dan chapterId wajib diisi')
      if (!b.amount || b.amount <= 0) return fail(400, 'amount harus lebih besar dari 0')
      const now = nowISO()
      const invoice: Invoice = {
        id: `mock-inv-${store.invoices.length + 1}`,
        number: b.number ?? `INV-2026-${String(store.invoices.length + 1).padStart(3, '0')}`,
        memberId: b.memberId,
        chapterId: b.chapterId,
        type: b.type ?? 'renewal',
        amount: b.amount,
        currency: 'IDR',
        dueDate: b.dueDate ?? now.slice(0, 10),
        periodStart: b.periodStart ?? now.slice(0, 10),
        periodEnd: b.periodEnd ?? now.slice(0, 10),
        status: 'draft',
        createdAt: now,
        updatedAt: now,
      }
      return created(invoice)
    }

    const invoice = store.invoices.find((i) => i.id === seg[1])

    if (seg[2] === 'audit') {
      if (!invoice) return notFound()
      const rows = store.auditLog.filter((a) => a.invoiceId === invoice.id)
      if (m === 'GET') return list(rows, num('limit', 50))
      if (m === 'POST') {
        const b = (body ?? {}) as { notes?: string; actorName?: string }
        if (!b.notes) return fail(400, 'notes wajib diisi untuk catatan manual')
        return created({
          id: `mock-aud-${rows.length + 1}`,
          invoiceId: invoice.id,
          action: 'updated',
          actorName: b.actorName,
          notes: b.notes,
          createdAt: nowISO(),
        })
      }
    }

    if (seg[2] === 'send' && m === 'POST') {
      if (!invoice) return notFound()
      if (invoice.status !== 'draft') return fail(409, 'hanya invoice draft yang bisa dikirim ke Paper.id')
      // Mirrors the server: an unset channel falls back to the operational
      // setting rather than defaulting to "don't deliver".
      const b = (body ?? {}) as { sendEmail?: boolean; sendWhatsApp?: boolean }
      const member = memberOf(invoice)
      // Cermin server: hanya 'false' yang mematikan kanal.
      const setting = async (key: string) => (await getMockAppSetting(key)) !== 'false'
      const sendEmail = (b.sendEmail ?? (await setting('paperid_send_email'))) && Boolean(member?.email)
      const sendWhatsApp = b.sendWhatsApp ?? (await setting('paperid_send_whatsapp'))

      const paperInvoiceID = `mock-paper-${invoice.id.slice(-6)}`
      const paymentURL = `https://stg-v2.paper.id/MOCK${invoice.number.slice(-3)}`
      // Payload wire yang sama bentuknya dengan yang dikirim server sungguhan,
      // supaya halaman Blackbox pada demo memperlihatkan hal yang sama.
      recordMockCall({
        integration: 'paper_id',
        direction: 'outbound',
        method: 'POST',
        url: 'https://open-api.stag-v2.paper.id/api/v1/store-invoice',
        status: 201,
        durationMs: 812,
        success: true,
        request: {
          invoice_date: ddmmyyyy(new Date()),
          due_date: ddmmyyyy(new Date(invoice.dueDate)),
          number: invoice.number,
          customer: {
            id: `${invoice.memberId}-mock`,
            name: member?.name ?? '',
            email: member?.email ?? '',
            phone: member?.phone ?? '',
          },
          items: [{ name: itemNameFor(invoice.type), quantity: 1, price: invoice.amount }],
          send: { email: sendEmail, whatsapp: sendWhatsApp, sms: false },
        },
        response: {
          data: {
            id: paperInvoiceID,
            number: invoice.number,
            payper_url: paymentURL,
            status_send: { email: sendEmail, whatsapp: sendWhatsApp, sms: false },
          },
        },
      })

      return ok({
        ...invoice,
        status: 'sent',
        paymentProvider: 'paper_id',
        paperIdInvoiceId: paperInvoiceID,
        paperIdPaymentUrl: paymentURL,
        paperIdInvoiceUrl: 'https://storage.googleapis.com/mock/INV.pdf',
        paperIdSentAt: nowISO(),
        deliveredVia: [sendEmail && 'email', sendWhatsApp && 'whatsapp'].filter(Boolean),
      })
    }

    if (seg[2] === 'remind' && m === 'POST') {
      if (!invoice) return notFound()
      if (invoice.status === 'draft') {
        return fail(409, 'invoice belum pernah dikirim — pakai Kirim, bukan Pengingat')
      }
      if (invoice.status !== 'sent' && invoice.status !== 'overdue') {
        return fail(409, `invoice berstatus ${invoice.status} tidak bisa diingatkan`)
      }
      const b = (body ?? {}) as { sendEmail?: boolean; sendWhatsApp?: boolean }
      const member = memberOf(invoice)
      const setting = async (key: string) => (await getMockAppSetting(key)) !== 'false'
      const sendEmail = (b.sendEmail ?? (await setting('paperid_send_email'))) && Boolean(member?.email)
      const sendWhatsApp = b.sendWhatsApp ?? (await setting('paperid_send_whatsapp'))

      // Nomor turunan, cermin server: Paper.id membakar nomor secara permanen,
      // jadi pengingat tidak bisa memakai nomor yang sama.
      const n = (invoice.paperIdReminderCount ?? 0) + 1
      const docNumber = `${invoice.number}-R${n}`
      const paperInvoiceID = `mock-paper-r${n}-${invoice.id.slice(-6)}`
      const paymentURL = `https://stg-v2.paper.id/MOCKR${n}${invoice.number.slice(-3)}`

      recordMockCall({
        integration: 'paper_id',
        direction: 'outbound',
        method: 'POST',
        url: 'https://open-api.stag-v2.paper.id/api/v1/store-invoice',
        status: 201,
        durationMs: 774,
        success: true,
        request: {
          invoice_date: ddmmyyyy(new Date()),
          // Jatuh tempo tetap tanggal asli — memundurkannya membuat tunggakan
          // tampak belum jatuh tempo.
          due_date: ddmmyyyy(new Date(invoice.dueDate)),
          number: docNumber,
          customer: {
            id: `${invoice.memberId}-mock`,
            name: member?.name ?? '',
            email: member?.email ?? '',
            phone: member?.phone ?? '',
          },
          items: [{ name: itemNameFor(invoice.type), quantity: 1, price: invoice.amount }],
          send: { email: sendEmail, whatsapp: sendWhatsApp, sms: false },
          notes: `Pengingat pembayaran untuk invoice ${invoice.number}`,
        },
        response: {
          data: {
            id: paperInvoiceID,
            number: docNumber,
            payper_url: paymentURL,
            status_send: { email: sendEmail, whatsapp: sendWhatsApp, sms: false },
          },
        },
      })

      // Status TIDAK berubah: overdue tetap overdue.
      return ok({
        ...invoice,
        paperIdInvoiceId: paperInvoiceID,
        paperIdPaymentUrl: paymentURL,
        paperIdSentAt: nowISO(),
        paperIdReminderCount: n,
        deliveredVia: [sendEmail && 'email', sendWhatsApp && 'whatsapp'].filter(Boolean),
      })
    }

    if (seg.length === 2) {
      if (!invoice) return notFound()
      if (m === 'GET') return ok(invoice)
      if (m === 'PATCH') return ok({ ...invoice, ...(body as object), updatedAt: nowISO() })
      if (m === 'DELETE') {
        const paid = store.payments.some((p) => p.invoiceId === invoice.id)
        if (paid) return fail(409, 'invoice masih punya data pembayaran — batalkan invoice alih-alih menghapusnya')
        return { status: 204, body: null }
      }
    }
  }

  // --- payments ------------------------------------------------------------
  if (seg[0] === 'payments') {
    if (seg.length === 1 && m === 'GET') {
      let rows = [...store.payments]
      const invoiceId = query.get('invoiceId')
      if (invoiceId) rows = rows.filter((p) => p.invoiceId === invoiceId)
      const method_ = query.get('method')
      if (method_) rows = rows.filter((p) => p.paymentMethod === method_)
      rows.sort((a, b) => b.paidAt.localeCompare(a.paidAt))
      return list(rows, limit, offset)
    }
    if (seg.length === 1 && m === 'POST') {
      const b = (body ?? {}) as Partial<Payment>
      if (!b.invoiceId) return fail(400, 'invoiceId wajib diisi')
      if (!b.amount || b.amount <= 0) return fail(400, 'amount harus lebih besar dari 0')
      const target = store.invoices.find((i) => i.id === b.invoiceId)
      if (!target) return fail(404, 'invoice tidak ditemukan')
      if (target.status === 'cancelled') return fail(409, 'invoice sudah dibatalkan — tidak bisa menerima pembayaran')
      return created({
        id: `mock-pay-${store.payments.length + 1}`,
        invoiceId: b.invoiceId,
        amount: b.amount,
        paidAt: b.paidAt ?? nowISO(),
        paymentMethod: b.paymentMethod ?? 'bank_transfer',
        createdAt: nowISO(),
      })
    }
    const payment = store.payments.find((p) => p.id === seg[1])
    if (seg.length === 2) {
      if (!payment) return notFound()
      if (m === 'GET') return ok(payment)
      if (m === 'PATCH') return ok({ ...payment, ...(body as object) })
      if (m === 'DELETE') return { status: 204, body: null }
    }
  }

  // --- members -------------------------------------------------------------
  if (seg[0] === 'members') {
    if (seg[1] === 'renewal-due' && m === 'GET') {
      const days = num('days', 30)
      const today = new Date()
      const rows = store.members
        .filter((x) => x.status === 'active' && x.renewalDate)
        .map((x) => ({ ...x, chapter: chapterOf(x.chapterId), daysUntilDue: daysBetween(today, new Date(x.renewalDate!)) }))
        .filter((x) => x.daysUntilDue >= 0 && x.daysUntilDue <= days)
        .sort((a, b) => a.daysUntilDue - b.daysUntilDue)
      return list(rows, num('limit', 100))
    }
    if (seg.length === 1 && m === 'GET') {
      let rows = store.members.map((x) => ({ ...x, chapter: chapterOf(x.chapterId) }))
      const chapterId = query.get('chapterId')
      if (chapterId) rows = rows.filter((x) => x.chapterId === chapterId)
      const status = query.get('status')
      if (status) {
        if (!['active', 'inactive', 'pending'].includes(status)) {
          return fail(400, "status harus 'active', 'inactive', atau 'pending'")
        }
        rows = rows.filter((x) => x.status === status)
      }
      const q = query.get('q')?.toLowerCase()
      if (q) {
        rows = rows.filter(
          (x) =>
            x.name.toLowerCase().includes(q) ||
            (x.email ?? '').toLowerCase().includes(q) ||
            (x.company ?? '').toLowerCase().includes(q),
        )
      }
      rows.sort((a, b) => a.name.localeCompare(b.name))
      return list(rows, limit, offset)
    }
    if (seg.length === 1 && m === 'POST') {
      const b = (body ?? {}) as { chapterId?: string; name?: string }
      if (!b.chapterId) return fail(400, 'chapterId wajib diisi')
      if (!b.name) return fail(400, 'name wajib diisi')
      return created({
        id: `mock-mem-${store.members.length + 1}`,
        ...b,
        status: 'active',
        joinedDate: null,
        renewalDate: null,
        syncedAt: nowISO(),
        chapter: chapterOf(b.chapterId),
      })
    }
    if (seg.length === 2) {
      const member = memberWithChapter(seg[1])
      if (!member) return notFound()
      if (m === 'GET') return ok(member)
      if (m === 'PATCH') return ok({ ...member, ...(body as object) })
      if (m === 'DELETE') {
        const hasInvoices = store.invoices.some((i) => i.memberId === member.id)
        if (hasInvoices) {
          return fail(409, "member masih punya invoice — ubah statusnya menjadi 'inactive' alih-alih menghapus")
        }
        return { status: 204, body: null }
      }
    }
  }

  // --- chapters ------------------------------------------------------------
  if (seg[0] === 'chapters') {
    if (seg.length === 1 && m === 'GET') {
      let rows = [...store.chapters]
      const city = query.get('cityName')
      if (city) rows = rows.filter((c) => c.cityName === city)
      const q = query.get('q')?.toLowerCase()
      if (q) rows = rows.filter((c) => c.name.toLowerCase().includes(q) || c.displayName.toLowerCase().includes(q))
      rows.sort((a, b) => a.displayName.localeCompare(b.displayName))
      return list(rows, num('limit', 100), offset)
    }
    if (seg.length === 1 && m === 'POST') {
      const b = (body ?? {}) as { id?: string; name?: string; displayName?: string }
      if (!b.name) return fail(400, 'name wajib diisi')
      return created({
        id: b.id ?? `mock-ch-${store.chapters.length + 1}`,
        name: b.name,
        displayName: b.displayName || b.name,
        syncedAt: nowISO(),
      })
    }
    if (seg.length === 2) {
      const chapter = chapterOf(seg[1])
      if (!chapter) return notFound()
      if (m === 'GET') return ok(chapter)
      if (m === 'PATCH') return ok({ ...chapter, ...(body as object) })
      if (m === 'DELETE') {
        const used = store.members.filter((x) => x.chapterId === chapter.id).length
        const inv = store.invoices.filter((i) => i.chapterId === chapter.id).length
        if (used || inv) {
          return fail(409, `chapter masih dipakai oleh ${used} member dan ${inv} invoice — pindahkan dulu sebelum menghapus`)
        }
        return { status: 204, body: null }
      }
    }
  }

  // --- settings ------------------------------------------------------------
  if (seg[0] === 'fee-settings') {
    if (m === 'GET') return ok(store.feeSettings)
    if (m === 'PATCH') {
      const b = (body ?? {}) as Record<string, unknown>
      if (Object.keys(b).length === 0) return fail(400, 'tidak ada field yang diubah')
      if (typeof b.registrationFee === 'number' && b.registrationFee < 0) {
        return fail(400, 'registrationFee tidak boleh negatif')
      }
      if (typeof b.renewalFee === 'number' && b.renewalFee < 0) {
        return fail(400, 'renewalFee tidak boleh negatif')
      }
      return ok({ ...store.feeSettings, ...b, updatedAt: nowISO() })
    }
  }

  if (seg[0] === 'app-settings') {
    const KEYS = [
      'invoice_draft_days_before',
      'invoice_due_days_after',
      'paperid_send_email',
      'paperid_send_whatsapp',
      'bni_vm_token',
    ]
    if (seg.length === 1 && m === 'GET') {
      const rows = await Promise.all(
        KEYS.map(async (key) => {
          const value = (await getMockAppSetting(key)) ?? ''
          const secret = /token|secret|password|apikey|credential|private/i.test(key)
          return secret && value
            ? { key, value: '••••••', updatedAt: nowISO(), masked: true }
            : { key, value, updatedAt: nowISO() }
        }),
      )
      return list(rows.filter((r) => r.value !== ''), rows.length)
    }
    if (seg.length === 2) {
      const key = seg[1]
      const secret = /token|secret|password|apikey|credential|private/i.test(key)
      if (m === 'GET') {
        const value = await getMockAppSetting(key)
        if (value === null) return notFound()
        return ok(secret ? { key, value: '••••••', updatedAt: nowISO(), masked: true } : { key, value, updatedAt: nowISO() })
      }
      if (m === 'PUT') {
        const b = (body ?? {}) as { value?: string }
        if (b.value === '••••••') return fail(400, 'value masih berupa nilai tersamar — kirim nilai sebenarnya')
        await setMockAppSetting(key, b.value ?? '')
        return ok(secret ? { key, value: '••••••', updatedAt: nowISO(), masked: true } : { key, value: b.value, updatedAt: nowISO() })
      }
      if (m === 'DELETE') return { status: 204, body: null }
    }
  }

  // --- dashboard -----------------------------------------------------------
  if (seg[0] === 'dashboard' && seg[1] === 'summary' && m === 'GET') {
    const months = num('months', 6)
    const active = store.invoices.filter((i) => i.status !== 'cancelled')
    const paid = store.invoices.filter((i) => i.status === 'paid')
    const outstanding = store.invoices.filter((i) => i.status === 'sent' || i.status === 'overdue')
    const overdue = store.invoices.filter((i) => i.status === 'overdue')
    const sum = (rows: Invoice[]) => rows.reduce((t, i) => t + i.amount, 0)

    const now = new Date()
    const monthly = Array.from({ length: months }, (_, idx) => {
      const d = new Date(now.getFullYear(), now.getMonth() - (months - 1 - idx), 1)
      const key = `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}`
      return {
        month: key,
        issued: sum(active.filter((i) => i.createdAt.startsWith(key))),
        paid: store.payments.filter((p) => p.paidAt.startsWith(key)).reduce((t, p) => t + p.amount, 0),
      }
    })

    return ok({
      total: { count: active.length, amount: sum(active), trend: 0 },
      paid: { count: paid.length, amount: sum(paid), trend: 0 },
      outstanding: { count: outstanding.length, amount: sum(outstanding), trend: 0 },
      overdue: { count: overdue.length, amount: sum(overdue), trend: 0 },
      renewalDue: { count: store.members.filter((x) => x.renewalDate).length, trend: 0 },
      statusBreakdown: ['draft', 'sent', 'paid', 'overdue', 'cancelled'].map((status) => ({
        status,
        count: store.invoices.filter((i) => i.status === status).length,
      })),
      monthly,
      chapterStats: store.chapters.map((c) => {
        const rows = store.invoices.filter((i) => i.chapterId === c.id)
        return {
          chapterId: c.id,
          chapterName: c.displayName,
          total: rows.filter((i) => i.status !== 'cancelled').length,
          paid: rows.filter((i) => i.status === 'paid').length,
          outstanding: rows.filter((i) => i.status === 'sent' || i.status === 'overdue').length,
          overdue: rows.filter((i) => i.status === 'overdue').length,
          totalAmount: sum(rows.filter((i) => i.status !== 'cancelled')),
        }
      }),
    })
  }

  // --- sync ----------------------------------------------------------------
  if (seg[0] === 'sync' && m === 'POST') {
    const result = {
      chapters: store.chapters.length,
      members: store.members.length,
      deactivated: 0,
      syncedAt: nowISO(),
    }
    recordMockCall({
      integration: 'bni_vm',
      direction: 'outbound',
      method: 'GET',
      url: 'https://www.bni-vh.com/api/external/v1/members?page=1',
      status: 200,
      durationMs: 1_240,
      success: true,
      request: { page: 1, perPage: 100 },
      response: { total: store.members.length, data: '(dipangkas — daftar member)' },
    })
    return ok(result)
  }

  // --- uploads -------------------------------------------------------------
  if (seg[0] === 'uploads' && m === 'POST') {
    return created({ url: '/uploads/20260727-mockbukti.png' })
  }

  // --- blackbox ------------------------------------------------------------
  if (seg[0] === 'blackbox') {
    if (m === 'GET') {
      const calls = listMockCalls({
        integration: query.get('integration') ?? undefined,
        direction: query.get('direction') ?? undefined,
        status: query.get('status') ?? undefined,
      })
      return list(calls, num('limit', 100))
    }
    if (m === 'DELETE') {
      clearMockCalls()
      return { status: 204, body: null }
    }
  }

  // --- Paper.id console ----------------------------------------------------
  if (seg[0] === 'paperid') {
    if (seg[1] === 'status' && m === 'GET') {
      return ok({
        configured: false,
        baseUrl: 'https://open-api.stag-v2.paper.id',
        callbackConfigured: false,
      })
    }
    if (seg[1] === 'test-invoice' && m === 'POST') {
      const b = (body ?? {}) as Record<string, unknown>
      const now = new Date()
      const dd = (d: Date) =>
        `${String(d.getDate()).padStart(2, '0')}-${String(d.getMonth() + 1).padStart(2, '0')}-${d.getFullYear()}`
      const due = new Date(now)
      due.setDate(due.getDate() + 30)
      return ok({
        dryRun: b.dryRun !== false,
        method: 'POST',
        url: 'https://open-api.stag-v2.paper.id/api/v1/store-invoice',
        request: {
          invoice_date: dd(now),
          due_date: dd(due),
          number: `TEST-MOCK-${Date.now()}`,
          customer: {
            id: 'console-test',
            name: (b.customerName as string) || 'Uji Coba BNI Finance',
            // Nomor milik tim, bukan angka karangan: bila uji ini dijalankan
            // dengan kanal menyala, pesannya harus mendarat di ponsel kita —
            // bukan di ponsel orang lain yang kebetulan memiliki nomor itu.
            phone: (b.customerPhone as string) || '082240274833',
          },
          items: [
            {
              name: 'Uji Koneksi Paper.id',
              description: 'Invoice uji dari konsol — bukan tagihan sungguhan',
              quantity: 1,
              price: (b.amount as number) || 1_500_000,
            },
          ],
          send: { email: b.sendEmail === true, whatsapp: b.sendWhatsApp === true, sms: false },
        },
        success: true,
        durationMs: 0,
      })
    }
    if (seg[1] === 'test-callback' && m === 'POST') {
      const b = (body ?? {}) as { invoiceId?: string }
      if (!b.invoiceId) return fail(400, 'invoiceId wajib diisi')
      return ok({
        dryRun: true,
        url: '/api/v1/webhooks/paperid',
        request: {
          payment_date: nowISO().slice(0, 10),
          payment_info: { method: 'bank_transfer', channel: 'bni', status: 'PAID' },
          additional_info: { invoices: [{ uuid: 'mock-paper-uuid', number: 'INV-MOCK' }] },
        },
        settled: false,
        success: true,
      })
    }
  }

  // --- public --------------------------------------------------------------

  if (seg[0] === 'webhooks' && m === 'POST') {
    // Both webhooks reject an unsigned call — the mock keeps that boundary
    // visible rather than pretending callbacks are open.
    //
    // Direkam sebagai panggilan MASUK yang gagal, sehingga demo memperlihatkan
    // kedua arah di Blackbox — dan filter "Gagal saja" punya isi.
    recordMockCall({
      integration: 'paper_id',
      direction: 'inbound',
      method: 'POST',
      url: `/api/v1/webhooks/${seg[1] ?? ''}`,
      status: 401,
      durationMs: 2,
      success: false,
      request: body ?? {},
      error: 'token callback tidak valid',
    })
    return fail(401, 'token callback tidak valid')
  }

  // --- routes outside /api/v1 ---------------------------------------------
  if (fullPath === '/healthz' && m === 'GET') return ok({ status: 'ok' })
  if (fullPath === '/metrics' && m === 'GET') {
    // Metrik menghitung lalu lintas HTTP, pool database, dan panggilan
    // integrasi milik SERVER. Tidak ada satu pun di browser, jadi angka
    // karangan di sini akan lebih menyesatkan daripada jujur mengatakan
    // tempatnya bukan di sini.
    return {
      status: 200,
      body:
        '# Metrik dihitung di server; mode Data Contoh tidak punya proses yang\n' +
        '# bisa diukur. Beralihlah ke Backend API, lalu scrape /metrics di sana.\n',
    }
  }
  if (fullPath === '/openapi.json' && m === 'GET') {
    return ok({ note: 'Spesifikasi lengkap hanya dilayani backend — jalankan mode Backend API.' })
  }
  if (fullPath === '/openapi.yaml' || fullPath === '/docs') {
    return { status: 200, body: '(dokumen non-JSON — buka langsung di browser saat mode Backend API)' }
  }
  if (fullPath.startsWith('/uploads/')) {
    return fail(404, 'berkas tidak ada pada Data Contoh — unggahan disimpan di disk server')
  }

  return fail(404, `endpoint mock belum tersedia: ${m} ${fullPath}`)
}
