import { tl as _tl } from './i18n-helper'

const parseJSONSafe = (text) => {
  if (typeof text !== 'string' || text.trim() === '') {
    return null
  }
  try {
    return JSON.parse(text)
  } catch (_) {
    return null
  }
}

const buildResponseError = (status, payload) => {
  const message =
    (payload && typeof payload === 'object' && payload.error) ||
    (payload && typeof payload === 'object' && payload.message) ||
    _tl('request_failed_status', { status })
  return new Error(String(message))
}

const normalizeSSEDataLine = (line) => {
  let data = line.slice(5)
  if (data.startsWith(' ')) {
    data = data.slice(1)
  }
  return data
}

const readSSE = async (response, onEvent) => {
  const reader = response.body?.getReader()
  if (!reader) {
    throw new Error(_tl('browser_no_stream'))
  }

  const decoder = new TextDecoder('utf-8')
  let buffer = ''
  let eventName = 'message'
  let dataLines = []
  let lastEvent = ''
  let lastPayload = null

  const dispatchEvent = () => {
    if (dataLines.length === 0) {
      eventName = 'message'
      return
    }
    const raw = dataLines.join('\n')
    const payload = parseJSONSafe(raw)
    const data = payload !== null ? payload : raw

    lastEvent = eventName || 'message'
    lastPayload = data
    if (typeof onEvent === 'function') {
      onEvent(lastEvent, data)
    }

    eventName = 'message'
    dataLines = []
  }

  const consumeBuffer = () => {
    for (;;) {
      const lineEnd = buffer.indexOf('\n')
      if (lineEnd < 0) {
        break
      }

      let line = buffer.slice(0, lineEnd)
      buffer = buffer.slice(lineEnd + 1)
      if (line.endsWith('\r')) {
        line = line.slice(0, -1)
      }

      if (line === '') {
        dispatchEvent()
        continue
      }
      if (line.startsWith(':')) {
        continue
      }
      if (line.startsWith('event:')) {
        eventName = line.slice(6).trim() || 'message'
        continue
      }
      if (line.startsWith('data:')) {
        dataLines.push(normalizeSSEDataLine(line))
      }
    }
  }

  for (;;) {
    const { value, done } = await reader.read()
    if (done) {
      break
    }
    buffer += decoder.decode(value, { stream: true })
    consumeBuffer()
  }

  buffer += decoder.decode()
  if (buffer !== '') {
    buffer += '\n'
    consumeBuffer()
  }
  dispatchEvent()

  return {
    lastEvent,
    lastPayload
  }
}

export const postJSONWithSSE = async ({
  url,
  body,
  timeoutMs = 0,
  token = '',
  onEvent
}) => {
  if (!url) {
    throw new Error(_tl('request_url_empty'))
  }

  const controller = new AbortController()
  const timer =
    timeoutMs > 0
      ? setTimeout(() => {
          controller.abort()
        }, timeoutMs)
      : null

  try {
    const headers = {
      'Content-Type': 'application/json',
      Accept: 'text/event-stream'
    }
    if (token) {
      headers.Authorization = `Bearer ${token}`
    }

    const response = await fetch(url, {
      method: 'POST',
      headers,
      body: JSON.stringify(body || {}),
      signal: controller.signal
    })

    const contentType = String(response.headers.get('content-type') || '').toLowerCase()
    if (contentType.includes('text/event-stream')) {
      const streamResult = await readSSE(response, onEvent)
      if (!response.ok) {
        throw new Error(_tl('request_failed_status', { status: response.status }))
      }
      return {
        mode: 'sse',
        status: response.status,
        ...streamResult
      }
    }

    const text = await response.text()
    const payload = parseJSONSafe(text)
    if (!response.ok) {
      throw buildResponseError(response.status, payload)
    }

    return {
      mode: 'json',
      status: response.status,
      payload
    }
  } catch (error) {
    if (error && error.name === 'AbortError') {
      throw new Error(_tl('request_timeout'))
    }
    throw error
  } finally {
    if (timer) {
      clearTimeout(timer)
    }
  }
}
