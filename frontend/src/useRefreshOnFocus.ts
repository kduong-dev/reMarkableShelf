import { useEffect, useRef } from 'react'

// useRefreshOnFocus calls refresh whenever the tab comes back into view, so
// a page picks up what background tablet syncs changed while it was hidden.
export function useRefreshOnFocus(refresh: () => void) {
  const latest = useRef(refresh)
  useEffect(() => {
    latest.current = refresh
  })
  useEffect(() => {
    function onVisibilityChange() {
      if (document.visibilityState === 'visible') latest.current()
    }
    document.addEventListener('visibilitychange', onVisibilityChange)
    return () => document.removeEventListener('visibilitychange', onVisibilityChange)
  }, [])
}
