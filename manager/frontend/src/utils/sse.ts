export type SSEEventHandler = (event: string, data: unknown) => void

interface SSEResult {
  mode: 'sse' | 'json'
  status: number
  lastEvent?: string
  lastPayload?: unknown
  payload?: unknown
}

const parseJSONSafe = (text: string): unknown | null => {
  if (typeof text !== 'string' || text.trim() === '') return null
  try { return JSON.parse(text) } catch { return null }
}

const buildResponseError = (status: number, payload: unknown): Error => {
  const p = payload as Record<string, string> | null
  const message = p?.error || p?.message || `Request failed (${status})`
  return new Error(String(message))
}

const normalizeSSEDataLine = (line: string): string => {
  const data = line.slice(5)
  return data.startsWith(' ') ? data.slice(1) : data
}

const readSSE = async (response: Response, onEvent?: SSEEventHandler) => {
  const reader = response.body?.getReader()
  if (!reader) throw new Error('Browser does not support streaming')

  const decoder = new TextDecoder('utf-8')
  let buffer = ''
  let eventName = 'message'
  let dataLines: string[] = []
  let lastEvent = ''
  let lastPayload: unknown = null

  const dispatchEvent = () => {
    if (dataLines.length === 0) { eventName = 'message'; return }
    const raw = dataLines.join('\n')
    const payload = parseJSONSafe(raw)
    lastEvent = eventName || 'message'
    lastPayload = payload !== null ? payload : raw
    onEvent?.(lastEvent, lastPayload)
    eventName = 'message'
    dataLines = []
  }

  const consumeBuffer = () => {
    for (;;) {
      const lineEnd = buffer.indexOf('\n')
      if (lineEnd < 0) break
      let line = buffer.slice(0, lineEnd)
      buffer = buffer.slice(lineEnd + 1)
      if (line.endsWith('\r')) line = line.slice(0, -1)
      if (line === '') { dispatchEvent(); continue }
      if (line.startsWith(':')) continue
      if (line.startsWith('event:')) { eventName = line.slice(6).trim() || 'message'; continue }
      if (line.startsWith('data:')) dataLines.push(normalizeSSEDataLine(line))
    }
  }

  for (;;) {
    const { value, done } = await reader.read()
    if (done) break
    buffer += decoder.decode(value, { stream: true })
    consumeBuffer()
  }

  buffer += decoder.decode()
  if (buffer !== '') { buffer += '\n'; consumeBuffer() }
  dispatchEvent()

  return { lastEvent, lastPayload }
}

export const postJSONWithSSE = async ({
  url,
  body,
  timeoutMs = 0,
  token = '',
  onEvent,
}: {
  url: string
  body?: unknown
  timeoutMs?: number
  token?: string
  onEvent?: SSEEventHandler
}): Promise<SSEResult> => {
  if (!url) throw new Error('Request URL is empty')

  const controller = new AbortController()
  const timer = timeoutMs > 0 ? setTimeout(() => controller.abort(), timeoutMs) : null

  try {
    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
      Accept: 'text/event-stream',
    }
    if (token) headers.Authorization = `Bearer ${token}`

    const response = await fetch(url, {
      method: 'POST',
      headers,
      body: JSON.stringify(body ?? {}),
      signal: controller.signal,
    })

    const contentType = String(response.headers.get('content-type') ?? '').toLowerCase()
    if (contentType.includes('text/event-stream')) {
      const streamResult = await readSSE(response, onEvent)
      if (!response.ok) throw new Error(`Request failed (${response.status})`)
      return { mode: 'sse', status: response.status, ...streamResult }
    }

    const text = await response.text()
    const payload = parseJSONSafe(text)
    if (!response.ok) throw buildResponseError(response.status, payload)
    return { mode: 'json', status: response.status, payload }
  } catch (error) {
    if (error instanceof Error && error.name === 'AbortError') throw new Error('Request timed out')
    throw error
  } finally {
    if (timer) clearTimeout(timer)
  }
}
