const units: Array<[Intl.RelativeTimeFormatUnit, number]> = [
  ['day', 86_400_000],
  ['hour', 3_600_000],
  ['minute', 60_000],
]

const format = new Intl.RelativeTimeFormat(undefined, { numeric: 'auto' })

// timeAgo describes an ISO timestamp relative to now, e.g. "5 minutes ago".
export function timeAgo(iso: string): string {
  const elapsed = Date.now() - new Date(iso).getTime()
  for (const [unit, milliseconds] of units) {
    if (elapsed >= milliseconds) return format.format(-Math.floor(elapsed / milliseconds), unit)
  }
  return 'just now'
}
