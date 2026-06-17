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
import { SyncPage } from '@/features/settings/SyncPage'
import { NotFoundPage } from '@/features/misc/NotFoundPage'
import { UrgentPage } from '@/features/urgent/UrgentPage'
import { PaymentModePage } from '@/features/settings/PaymentModePage'
import { PublicPaymentPage } from '@/features/pay/PublicPaymentPage'
import { NotificationsPage } from '@/features/notifications/NotificationsPage'
import { ProfilePage } from '@/features/profile/ProfilePage'
import { ReportPage } from '@/features/reports/ReportPage'

export const router = createBrowserRouter([
  {
    element: <Providers />,
    children: [
      { path: '/login', element: <LoginPage /> },
      { path: '/pay/:id', element: <PublicPaymentPage /> },
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
          { path: '/invoices/new', element: <InvoiceNewPage /> },
          { path: '/invoices/renewal-due', element: <RenewalDuePage /> },
          { path: '/invoices/:id', element: <InvoiceDetailPage /> },

          { path: '/members', element: <MemberListPage /> },
          { path: '/members/:id', element: <MemberDetailPage /> },

          { path: '/chapters', element: <ChapterListPage /> },
          { path: '/payments', element: <PaymentListPage /> },
          { path: '/reports', element: <ReportPage /> },

          { path: '/settings', element: <SettingsPage /> },
          { path: '/settings/payment', element: <PaymentModePage /> },
          { path: '/settings/sync', element: <SyncPage /> },

          { path: '*', element: <NotFoundPage /> },
        ],
      },
    ],
  },
])
