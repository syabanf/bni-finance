import { useEffect, useState } from 'react'
import { dashboardService } from '@/services'

interface UrgentCount {
  overdue: number
  renewalDue: number
  total: number
}

/** Polls dashboard summary to keep the nav badge count up to date. */
export function useUrgentCount(): UrgentCount {
  const [count, setCount] = useState<UrgentCount>({ overdue: 0, renewalDue: 0, total: 0 })

  useEffect(() => {
    let cancelled = false
    const fetchCount = async () => {
      try {
        const summary = await dashboardService.summary()
        if (cancelled) return
        const overdue = summary.overdue.count
        const renewalDue = summary.renewalDue.count
        setCount({ overdue, renewalDue, total: overdue + renewalDue })
      } catch {
        // silently ignore — badge is non-critical
      }
    }
    fetchCount()
    const t = setInterval(fetchCount, 60_000)
    return () => {
      cancelled = true
      clearInterval(t)
    }
  }, [])

  return count
}
