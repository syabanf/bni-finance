import { createBrowserRouter, Navigate } from 'react-router-dom'
import { Providers } from './Providers'
import { AppLayout } from '@/components/layout/AppLayout'
import { LoginPage } from '@/features/auth/LoginPage'
import { DashboardPage } from '@/features/dashboard/DashboardPage'
import { InvoiceListPage } from '@/features/invoices/InvoiceListPage'
import { InvoiceNewPage } from '@/features/invoices/InvoiceNewPage'
import { RenewalDuePage } from '@/features/invoices/RenewalDuePage'
import { InvoiceDetailPage } from '@/features/invoices/InvoiceDetailPage'
import { MemberListPage } from '@/features/members/MemberListPage'
import { MemberDetailPage } from '@/features/members/MemberDetailPage'
import { ChapterListPage } from '@/features/chapters/ChapterListPage'
import { PaymentListPage } from '@/features/payments/PaymentListPage'
import { SettingsPage } from '@/features/settings/SettingsPage'
import { ImportPage } from '@/features/import/ImportPage'
import { RenewalPage } from '@/features/renewal/RenewalPage'
import { UsersPage } from '@/features/users/UsersPage'
import { SyncPage } from '@/features/settings/SyncPage'
import { BlackboxPage } from '@/features/blackbox/BlackboxPage'
import { ApiConsolePage } from '@/features/apiconsole/ApiConsolePage'
import { NotFoundPage } from '@/features/misc/NotFoundPage'
import { RouteErrorPage } from '@/features/misc/RouteErrorPage'
import { RequirePermission } from '@/features/auth/RequirePermission'
import { UrgentPage } from '@/features/urgent/UrgentPage'
import { NotificationsPage } from '@/features/notifications/NotificationsPage'
import { ProfilePage } from '@/features/profile/ProfilePage'
import { ReportPage } from '@/features/reports/ReportPage'

export const router = createBrowserRouter([
  {
    element: <Providers />,
    // Catches render/loader errors anywhere below instead of showing React
    // Router's raw "Unexpected Application Error!" stack trace.
    errorElement: <RouteErrorPage />,
    children: [
      { path: '/login', element: <LoginPage /> },
      {
        element: <AppLayout />,
        children: [
          { path: '/', element: <Navigate to="/dashboard" replace /> },
          { path: '/dashboard', element: <DashboardPage /> },
          { path: '/urgent', element: <UrgentPage /> },
          { path: '/notifications', element: <NotificationsPage /> },
          { path: '/profile', element: <ProfilePage /> },

          // Invoices — order matters: static segments before the :id param.
          { path: '/invoices', element: <InvoiceListPage /> },
          {
            path: '/invoices/new',
            element: (
              <RequirePermission permission="invoice:create">
                <InvoiceNewPage />
              </RequirePermission>
            ),
          },
          { path: '/invoices/renewal-due', element: <RenewalDuePage /> },
          { path: '/invoices/:id', element: <InvoiceDetailPage /> },

          { path: '/members', element: <MemberListPage /> },
          { path: '/members/:id', element: <MemberDetailPage /> },

          { path: '/chapters', element: <ChapterListPage /> },
          { path: '/payments', element: <PaymentListPage /> },
          { path: '/reports', element: <ReportPage /> },

          {
            path: '/settings',
            element: (
              <RequirePermission permission="settings:manage">
                <SettingsPage />
              </RequirePermission>
            ),
          },
          {
            // Tanpa permission khusus: MC harus bisa membukanya untuk menjawab,
            // dan MC memang tidak punya izin menulis apa pun yang lain. Siapa
            // yang boleh MENJAWAB diperiksa di server, bukan di rute.
            path: '/renewal',
            element: <RenewalPage />,
          },
          {
            // Admin saja. Impor menulis lintas chapter sekaligus — mengizinkan
            // ST berarti memberi jalan mengubah data chapter lain lewat satu
            // berkas, melewati seluruh batas chapter yang dijaga di server.
            path: '/settings/import',
            element: (
              <RequirePermission permission="settings:manage">
                <ImportPage />
              </RequirePermission>
            ),
          },
          {
            // Pengelolaan akun butuh settings:manage — hanya admin. Peran ST
            // dan MC yang dibuat di sini adalah batas keamanan sistem, jadi
            // yang boleh membuatnya harus lebih sempit daripada yang memakainya.
            path: '/settings/users',
            element: (
              <RequirePermission permission="settings:manage">
                <UsersPage />
              </RequirePermission>
            ),
          },
          {
            path: '/settings/sync',
            element: (
              <RequirePermission permission="sync:run">
                <SyncPage />
              </RequirePermission>
            ),
          },

          {
            path: '/blackbox',
            element: (
              <RequirePermission permission="settings:manage">
                <BlackboxPage />
              </RequirePermission>
            ),
          },
          {
            path: '/api-console',
            element: (
              <RequirePermission permission="settings:manage">
                <ApiConsolePage />
              </RequirePermission>
            ),
          },

          { path: '*', element: <NotFoundPage /> },
        ],
      },
    ],
  },
])
