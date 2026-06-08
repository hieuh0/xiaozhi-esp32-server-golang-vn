import { tl as _tl } from './i18n-helper'

const OPENCLAW_CHANNEL_NAME = 'xiaozhi'
const OPENCLAW_CHANNEL_CONFIG_PREFIX = `channels.${OPENCLAW_CHANNEL_NAME}`

const EMPTY_COMMAND_DATA = {
  ready: false,
  url: '',
  token: '',
  steps: [],
  commands: [],
  copyText: ''
}

export function buildOpenClawCommands(endpoint) {
  const trimmedEndpoint = String(endpoint || '').trim()
  if (!trimmedEndpoint) {
    return EMPTY_COMMAND_DATA
  }

  try {
    const parsed = new URL(trimmedEndpoint)
    const token = String(parsed.searchParams.get('token') || '').trim()
    parsed.search = ''
    parsed.hash = ''

    const url = parsed.toString()
    if (!url || !token) {
      return EMPTY_COMMAND_DATA
    }

    const steps = [
      {
        title: _tl('enable_channel'),
        command: `openclaw config set ${OPENCLAW_CHANNEL_CONFIG_PREFIX}.enabled true --strict-json`
      },
      {
        title: _tl('config_address'),
        command: `openclaw config set ${OPENCLAW_CHANNEL_CONFIG_PREFIX}.url "${url}"`
      },
      {
        title: _tl('config_token'),
        command: `openclaw config set ${OPENCLAW_CHANNEL_CONFIG_PREFIX}.token "${token}"`
      },
      {
        title: _tl('restart_gateway'),
        command: 'openclaw gateway restart'
      }
    ]
    const commands = steps.map((step) => step.command)

    return {
      ready: true,
      url,
      token,
      steps,
      commands,
      copyText: commands.join('\n')
    }
  } catch (error) {
    console.error('Failed to parse OpenClaw endpoint:', error)
    return EMPTY_COMMAND_DATA
  }
}
