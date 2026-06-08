const dateFormatter = new Intl.DateTimeFormat(undefined, {
  year: 'numeric', month: '2-digit', day: '2-digit',
  hour: '2-digit', minute: '2-digit', second: '2-digit'
})

export function useFormatDate() {
  const formatDate = (value) => {
    if (!value) return '—'
    try {
      return dateFormatter.format(new Date(value))
    } catch {
      return String(value)
    }
  }
  return { formatDate }
}
