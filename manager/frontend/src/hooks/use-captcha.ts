import { useState } from 'react'
import api from '@/utils/api'
import { useLocale } from './use-locale'

export interface CaptchaState {
  id: string
  prompt: string
  answer: string
  loading: boolean
  setAnswer: (v: string) => void
  fetch: () => Promise<void>
  clear: () => void
}

export function useCaptcha(): CaptchaState {
  const { t } = useLocale()
  const [id, setId] = useState('')
  const [prompt, setPrompt] = useState('')
  const [answer, setAnswer] = useState('')
  const [loading, setLoading] = useState(false)

  const fetch = async () => {
    setLoading(true)
    try {
      const { data } = await api.get<{ captchaId: string; prompt: string }>('/captcha/challenge')
      setId(data.captchaId)
      setAnswer('')
      setPrompt(data.prompt)
    } catch {
      setId('')
      setAnswer('')
      setPrompt(t('question_load_failed'))
    } finally {
      setLoading(false)
    }
  }

  const clear = () => { setId(''); setAnswer(''); setPrompt('') }

  return { id, prompt, answer, loading, setAnswer, fetch, clear }
}
