import { useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { ArrowLeft, Calendar, Mail, Pencil, Phone, Plus, Trash2 } from 'lucide-react'
import type { Invoice, MemberWithChapter } from '@/types'
import {
  Avatar,
  Button,
  Card,
  CardBody,
  CardHeader,
  EmptyState,
  InvoiceStatusBadge,
  Modal,
  InvoiceTypeBadge,
  LoadingState,
  MemberStatusBadge,
  PageHeader,
  Table,
  TBody,
  Td,
  Th,
  THead,
  Tr,
} from '@/components/ui'
import { useAsync } from '@/hooks/useAsync'
import { invoiceService, memberService } from '@/services'
import { useToast } from '@/components/ui'
import { formatCurrency, formatDate } from '@/lib/format'
import { bolehDitagih } from '@/lib/status'
import { useCan } from '@/features/auth/usePermission'
import { UbahMemberModal } from './components/UbahMemberModal'

export function MemberDetailPage() {
  const { id = '' } = useParams()
  const navigate = useNavigate()

  // Mengubah member admin-saja di server, sama seperti Tambah Member.
  const bolehUbah = useCan('settings:manage')
  const [ubahBuka, setUbahBuka] = useState(false)
  const [hapusBuka, setHapusBuka] = useState(false)
  const [menghapus, setMenghapus] = useState(false)
  const { toast } = useToast()

  const { data: member, loading, reload } = useAsync<MemberWithChapter | null>(
    () => memberService.getById(id),
    [id],
  )
  const { data: invoices } = useAsync<Invoice[]>(() => invoiceService.listByMember(id), [id])

  if (loading) return <LoadingState label="Memuat member…" />
  if (!member)
    return (
      <Card data-tour="member-detail-info">
        <CardBody>
          <p className="py-10 text-center text-ink-500">Member tidak ditemukan.</p>
        </CardBody>
      </Card>
    )

  const totalPaid = (invoices ?? [])
    .filter((i) => i.status === 'paid')
    .reduce((acc, i) => acc + i.amount, 0)
  const outstanding = (invoices ?? []).filter((i) => i.status === 'sent' || i.status === 'overdue').length
  const activePeriodEnd = (invoices ?? [])
    .filter((i) => i.status !== 'cancelled')
    .sort((a, b) => b.periodEnd.localeCompare(a.periodEnd))[0]?.periodEnd

  return (
    <div>
      <PageHeader
        title={member.name}
        breadcrumb={
          <button
            onClick={() => navigate('/members')}
            className="inline-flex items-center gap-1.5 text-sm text-ink-500 hover:text-ink-800"
          >
            <ArrowLeft className="h-4 w-4" />
            Kembali ke Member
          </button>
        }
        description={<MemberStatusBadge status={member.status} />}
        action={
          // Tipe mengikuti status: yang belum anggota (termasuk New Member yang
          // baru ditambahkan) ditagih pendaftaran, member aktif ditagih renewal.
          <div className="flex flex-wrap items-center gap-2">
            {bolehUbah && (
              <Button variant="outline" onClick={() => setUbahBuka(true)}>
                <Pencil className="h-4 w-4" />
                Ubah Data
              </Button>
            )}
            {bolehUbah && (
              <Button
                variant="outline"
                onClick={() => setHapusBuka(true)}
                disabled={(invoices ?? []).length > 0}
                title={
                  (invoices ?? []).length > 0
                    ? 'Member yang punya invoice tidak bisa dihapus; ubah statusnya ke Inactive.'
                    : undefined
                }
              >
                <Trash2 className="h-4 w-4" />
                Hapus
              </Button>
            )}
            <Button
              onClick={() =>
                navigate(
                  `/invoices/new?type=${bolehDitagih(member.status, 'registration') ? 'registration' : 'renewal'}&member=${member.id}`,
                )
              }
            >
              <Plus className="h-4 w-4" />
              Buat Invoice
            </Button>
          </div>
        }
      />

      <UbahMemberModal open={ubahBuka} onClose={() => setUbahBuka(false)} onDisimpan={reload} member={member} />
      <Modal
        open={hapusBuka}
        onClose={() => setHapusBuka(false)}
        title={`Hapus ${member.name}?`}
        description="Permintaan konfirmasi renewal member ini ikut terhapus. Penghapusan tidak bisa dibatalkan."
        footer={
          <div className="flex justify-end gap-2">
            <Button variant="outline" onClick={() => setHapusBuka(false)} disabled={menghapus}>
              Batal
            </Button>
            <Button
              variant="danger"
              loading={menghapus}
              onClick={async () => {
                setMenghapus(true)
                try {
                  await memberService.remove(member.id)
                  toast(`${member.name} dihapus.`)
                  navigate('/members')
                } catch (err) {
                  toast(err instanceof Error ? err.message : 'Gagal menghapus member.', 'error')
                  setMenghapus(false)
                }
              }}
            >
              <Trash2 className="h-4 w-4" />
              Hapus
            </Button>
          </div>
        }
      />

      <div className="grid grid-cols-1 gap-5 lg:grid-cols-3">
        {/* Profile */}
        <div className="space-y-5">
          <Card>
            <CardBody className="flex flex-col items-center p-6 text-center">
              <Avatar name={member.name} size="lg" className="h-16 w-16 text-xl" />
              <h2 className="mt-3 text-lg font-bold text-ink-900">{member.name}</h2>
              <p className="text-sm text-ink-400">{member.id}</p>
              <div className="mt-3">
                <MemberStatusBadge status={member.status} />
              </div>
            </CardBody>
            <div className="space-y-3 border-t border-ink-100 px-5 py-4 text-sm">
              <InfoRow icon={Mail} value={member.email || '—'} />
              <InfoRow icon={Phone} value={member.phone || '—'} />
              {member.joinedDate && <InfoRow icon={Calendar} value={`Bergabung ${formatDate(member.joinedDate)}`} />}
            </div>
            <div className="border-t border-ink-100 px-5 py-4">
              <div className="text-xs font-medium uppercase tracking-wide text-ink-400">Chapter</div>
              <div className="mt-1 font-medium text-ink-800">{member.chapter?.displayName ?? '—'}</div>
              {member.chapter?.cityName && (
                <div className="text-xs text-ink-400">{member.chapter.cityName}</div>
              )}
            </div>
          </Card>

          <div className="grid grid-cols-2 gap-4">
            <Card>
              <CardBody className="p-4">
                <div className="text-xs text-ink-400">Total Dibayar</div>
                <div className="mt-1 text-lg font-bold text-ink-900">{formatCurrency(totalPaid)}</div>
              </CardBody>
            </Card>
            <Card>
              <CardBody className="p-4">
                <div className="text-xs text-ink-400">Outstanding</div>
                <div className="mt-1 text-lg font-bold text-ink-900">{outstanding}</div>
              </CardBody>
            </Card>
          </div>
        </div>

        {/* Invoice history */}
        <Card data-tour="member-detail-invoices" className="lg:col-span-2">
          <CardHeader
            title="Riwayat Invoice"
            subtitle={
              activePeriodEnd
                ? `Keanggotaan aktif s/d ${formatDate(activePeriodEnd)}`
                : 'Belum ada keanggotaan aktif'
            }
          />
          {!invoices || invoices.length === 0 ? (
            <EmptyState title="Belum ada invoice" description="Member ini belum memiliki invoice." />
          ) : (
            <>
              {/* Mobile cards */}
              <div className="divide-y divide-ink-100 lg:hidden">
                {invoices.map((inv) => (
                  <div
                    key={inv.id}
                    onClick={() => navigate(`/invoices/${inv.id}`)}
                    className="flex items-start gap-3 px-4 py-3.5 active:bg-ink-50"
                  >
                    <div className="min-w-0 flex-1">
                      <div className="flex items-start justify-between gap-2">
                        <span className="font-mono text-[13px] text-ink-700">{inv.number}</span>
                        <div className="shrink-0 text-right">
                          <div className="font-semibold text-ink-900 text-sm">{formatCurrency(inv.amount)}</div>
                        </div>
                      </div>
                      <div className="mt-1.5 flex flex-wrap items-center gap-1.5">
                        <InvoiceStatusBadge status={inv.status} />
                        <InvoiceTypeBadge type={inv.type} />
                      </div>
                      <div className="mt-1 text-xs text-ink-400">
                        {formatDate(inv.periodStart)} – {formatDate(inv.periodEnd)}
                      </div>
                    </div>
                  </div>
                ))}
              </div>
              {/* Desktop table */}
              <div className="hidden lg:block">
                <Table>
                  <THead>
                    <Tr>
                      <Th>No. Invoice</Th>
                      <Th>Tipe</Th>
                      <Th>Nominal</Th>
                      <Th>Status</Th>
                      <Th>Periode</Th>
                    </Tr>
                  </THead>
                  <TBody>
                    {invoices.map((inv) => (
                      <Tr key={inv.id} onClick={() => navigate(`/invoices/${inv.id}`)}>
                        <Td>
                          <span className="font-mono text-[13px] text-ink-700">{inv.number}</span>
                        </Td>
                        <Td>
                          <InvoiceTypeBadge type={inv.type} />
                        </Td>
                        <Td className="font-medium text-ink-900">{formatCurrency(inv.amount)}</Td>
                        <Td>
                          <InvoiceStatusBadge status={inv.status} />
                        </Td>
                        <Td className="whitespace-nowrap text-xs text-ink-500">
                          {formatDate(inv.periodStart)} – {formatDate(inv.periodEnd)}
                        </Td>
                      </Tr>
                    ))}
                  </TBody>
                </Table>
              </div>
            </>
          )}
        </Card>
      </div>
    </div>
  )
}

function InfoRow({ icon: Icon, value }: { icon: typeof Mail; value: string }) {
  return (
    <div className="flex items-center gap-2.5 text-ink-600">
      <Icon className="h-4 w-4 flex-shrink-0 text-ink-400" />
      <span className="truncate">{value}</span>
    </div>
  )
}
