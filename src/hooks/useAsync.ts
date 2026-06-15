import { useCallback, useEffect, useState } from 'react'

interface AsyncState<T> {
  data: T | null
  loading: boolean
  error: string | null
}

/**
 * Run an async function on mount (and when `deps` change), tracking
 * loading/error state. Returns a `reload` for manual refresh.
 */
export function useAsync<T>(fn: () => Promise<T>, deps: unknown[] = []) {
  const [state, setState] = useState<AsyncState<T>>({
    data: null,
    loading: true,
    error: null,
  })

  const run = useCallback(() => {
    let active = true
    setState((s) => ({ ...s, loading: true, error: null }))
    fn()
      .then((data) => {
        if (active) setState({ data, loading: false, error: null })
      })
      .catch((err: unknown) => {
        if (active)
          setState({
            data: null,
            loading: false,
            error: err instanceof Error ? err.message : 'Terjadi kesalahan.',
          })
      })
    return () => {
      active = false
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, deps)

  useEffect(run, [run])

  const reload = useCallback(() => {
    run()
  }, [run])

  return { ...state, reload }
}
