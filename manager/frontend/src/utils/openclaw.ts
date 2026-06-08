const CHANNEL = 'xiaozhi'
const CHANNEL_PREFIX = `channels.${CHANNEL}`

interface OpenClawStep { title: string; command: string }
export interface OpenClawCommandData {
  ready: boolean
  url: string
  token: string
  steps: OpenClawStep[]
  commands: string[]
  copyText: string
}

const EMPTY: OpenClawCommandData = { ready: false, url: '', token: '', steps: [], commands: [], copyText: '' }

export function buildOpenClawCommands(endpoint: string): OpenClawCommandData {
  const trimmed = String(endpoint || '').trim()
  if (!trimmed) return EMPTY
  try {
    const parsed = new URL(trimmed)
    const token = String(parsed.searchParams.get('token') || '').trim()
    parsed.search = ''
    parsed.hash = ''
    const url = parsed.toString()
    if (!url || !token) return EMPTY
    const steps: OpenClawStep[] = [
      { title: 'Enable channel', command: `openclaw config set ${CHANNEL_PREFIX}.enabled true --strict-json` },
      { title: 'Set address', command: `openclaw config set ${CHANNEL_PREFIX}.url "${url}"` },
      { title: 'Set token', command: `openclaw config set ${CHANNEL_PREFIX}.token "${token}"` },
      { title: 'Restart gateway', command: 'openclaw gateway restart' },
    ]
    const commands = steps.map((s) => s.command)
    return { ready: true, url, token, steps, commands, copyText: commands.join('\n') }
  } catch {
    return EMPTY
  }
}
