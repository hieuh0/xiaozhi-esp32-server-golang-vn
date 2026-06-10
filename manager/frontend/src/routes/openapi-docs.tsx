import { createFileRoute, useRouter } from '@tanstack/react-router'
import { useLocale } from '@/hooks/use-locale'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'

const METHOD_STYLES: Record<string, string> = {
  GET: 'status-success',
  POST: 'status-primary',
  PUT: 'status-warning',
  DELETE: 'status-danger',
}

function ApiLine({ method, path }: { method: string; path: string }) {
  return (
    <div className="flex items-center gap-2 my-2">
      <span className={cn('inline-block px-2 py-0.5 rounded border text-xs font-bold uppercase', METHOD_STYLES[method])}>{method}</span>
      <code className="text-sm bg-[var(--color-surface-2)] px-2 py-0.5 rounded">{path}</code>
    </div>
  )
}

function ParamTable({ rows }: { rows: [string, string, string, string][] }) {
  return (
    <table className="w-full text-sm border-collapse my-2">
      <thead><tr className="border-b border-[var(--color-line)]">
        {['Field','Type','Required','Description'].map((h) => <th key={h} className="text-left py-1.5 pr-4 font-medium text-[var(--color-text-secondary)]">{h}</th>)}
      </tr></thead>
      <tbody>{rows.map(([f, t, r, d]) => (
        <tr key={f} className="border-b border-[var(--color-line)]/50">
          <td className="py-1.5 pr-4 font-mono text-xs">{f}</td>
          <td className="py-1.5 pr-4 text-[var(--color-text-secondary)]">{t}</td>
          <td className="py-1.5 pr-4 text-[var(--color-text-secondary)]">{r}</td>
          <td className="py-1.5 text-[var(--color-text-secondary)]">{d}</td>
        </tr>
      ))}</tbody>
    </table>
  )
}

function Section({ id, title, children }: { id: string; title: string; children: React.ReactNode }) {
  return (
    <section id={id} className="mb-10 scroll-mt-6">
      <h2 className="text-xl font-bold text-[var(--color-text)] mb-3 pb-2 border-b border-[var(--color-line)]">{title}</h2>
      {children}
    </section>
  )
}

function CodeBlock({ code }: { code: string }) {
  return <pre className="bg-[var(--color-surface-2)] rounded-lg p-3 text-xs overflow-x-auto my-2 border border-[var(--color-line)]"><code>{code}</code></pre>
}

function OpenAPIDocsPage() {
  const { t } = useLocale()
  const router = useRouter()

  const nav = [
    { id: 'auth', label: t('auth_method') },
    { id: 'common', label: t('general_desc') },
    { id: 'profile', label: t('user_info_nav') },
    { id: 'devices', label: t('device_api') },
    { id: 'agents', label: t('agent_api') },
    { id: 'history', label: t('chat_history_nav') },
    { id: 'inject', label: t('voice_push_nav') },
    { id: 'mcp', label: t('mcp_tool_nav') },
  ]

  return (
    <div className="flex gap-6 max-w-7xl mx-auto px-4 py-6 pb-10 text-[var(--color-text)]">
      <aside className="sticky top-5 h-[calc(100vh-40px)] min-w-[220px] border-r border-[var(--color-line)] pr-4 flex flex-col gap-1.5 shrink-0 overflow-y-auto">
        <p className="font-bold text-sm mb-2">{t('openapi_docs')}</p>
        {nav.map((item) => (
          <a key={item.id} href={`#${item.id}`} className="text-sm text-[var(--color-text-secondary)] hover:text-[var(--color-primary)] transition-colors py-0.5">{item.label}</a>
        ))}
      </aside>

      <main className="flex-1 min-w-0">
        <header className="mb-8">
          <h1 className="text-3xl font-bold text-[var(--color-text)] mb-2">{t('xiaozhi_openapi_docs')}</h1>
          <p className="text-[var(--color-text-secondary)] mb-3">{t('public_api_hint')}</p>
          <div className="flex items-center gap-3 flex-wrap text-sm text-[var(--color-text-secondary)]">
            <span>Base URL: <code className="bg-[var(--color-surface-2)] px-1.5 py-0.5 rounded text-xs">/api/open/v1</code></span>
            <span>Content-Type: <code className="bg-[var(--color-surface-2)] px-1.5 py-0.5 rounded text-xs">application/json</code></span>
            <Button size="sm" variant="outline" onClick={() => router.navigate({ to: '/login' })}>{t('back_to_login')}</Button>
          </div>
        </header>

        <Section id="auth" title={t('auth_method')}>
          <CodeBlock code={`Authorization: Bearer <jwt-or-api-token>\nX-API-Token: <api-token>`} />
        </Section>

        <Section id="common" title={t('general_response_description')}>
          <ul className="text-sm text-[var(--color-text-secondary)] list-disc pl-5 space-y-1">
            <li>{t('common_error_codes')} <code>400</code> {t('params_error')} <code>401</code> {t('auth_failed')} <code>404</code> {t('resource_not_found')} <code>500</code> {t('server_error')}</li>
            <li>{t('paged_api_default')} <code>page=1</code>, <code>page_size=50</code>.</li>
          </ul>
        </Section>

        <Section id="profile" title={t('get_current_user_info')}>
          <ApiLine method="GET" path="/api/open/v1/profile" />
          <p className="text-sm text-[var(--color-text-secondary)] my-2">{t('no_auth_required')}</p>
          <CodeBlock code={`{"user": {"id": 1, "username": "demo", "email": "demo@example.com", "role": "user"}}`} />
        </Section>

        <Section id="devices" title={t('device_api')}>
          <h3 className="font-semibold mt-4 mb-1">{t('get_device_list')}</h3>
          <ApiLine method="GET" path="/api/open/v1/devices" />
          <p className="text-sm text-[var(--color-text-secondary)] my-1">{t('no_auth_required')}</p>
          <CodeBlock code={`{"data":[{"id":1,"device_name":"bedroom","device_code":"123456","agent_id":2,"activated":true}]}`} />
          <h3 className="font-semibold mt-6 mb-1">{t('create_device')}</h3>
          <ApiLine method="POST" path="/api/open/v1/devices" />
          <ParamTable rows={[
            ['device_name', 'string', t('yes_label'), t('device_name_length')],
            ['agent_id', 'number', t('yes_label'), t('bind_agent_id')],
          ]} />
          <CodeBlock code={`{"success":true,"message":"Device created successfully","data":{"device_code":"654321","device":{"id":8,"device_name":"bedroom"}}}`} />
        </Section>

        <Section id="agents" title={t('agent_api')}>
          <h3 className="font-semibold mt-4 mb-1">{t('get_agent_list')}</h3>
          <ApiLine method="GET" path="/api/open/v1/agents" />
          <CodeBlock code={`{"data":[{"id":2,"name":"Home Assistant","nickname":"Xiao Hui","llm_config_id":"llm_default"}]}`} />
          <h3 className="font-semibold mt-6 mb-1">{t('create_agent')}</h3>
          <ApiLine method="POST" path="/api/open/v1/agents" />
          <ParamTable rows={[
            ['name','string',t('yes_label'),t('name_length_hint')],
            ['nickname','string',t('no_label'),t('nickname_hint')],
            ['custom_prompt','string',t('no_label'),t('prompt')],
            ['llm_config_id','string',t('no_label'),t('llm_config_id')],
            ['tts_config_id','string',t('no_label'),t('tts_config_id')],
            ['voice','string',t('no_label'),t('timbre_id')],
            ['asr_speed','string',t('no_label'),t('default_normal')],
            ['memory_mode','string',t('no_label'),'short/long/none'],
          ]} />
          <CodeBlock code={`{"success":true,"data":{"id":3,"name":"Living Room Assistant","nickname":"Xiao Hui"}}`} />
          <h3 className="font-semibold mt-6 mb-1">{t('get_agent_detail')}</h3>
          <ApiLine method="GET" path="/api/open/v1/agents/:id" />
          <h3 className="font-semibold mt-6 mb-1">{t('update_agent')}</h3>
          <ApiLine method="PUT" path="/api/open/v1/agents/:id" />
          <h3 className="font-semibold mt-6 mb-1">{t('delete_agent')}</h3>
          <ApiLine method="DELETE" path="/api/open/v1/agents/:id" />
          <CodeBlock code={`{"message":"Deleted successfully"}`} />
        </Section>

        <Section id="history" title={t('chat_history_api')}>
          <h3 className="font-semibold mt-4 mb-1">{t('query_messages_paged')}</h3>
          <ApiLine method="GET" path="/api/open/v1/history/messages" />
          <ParamTable rows={[
            ['agent_id','string',t('no_label'),t('agent_id')],
            ['device_id','string',t('no_label'),t('device_identifier_field')],
            ['session_id','string',t('no_label'),t('session_id')],
            ['role','string',t('no_label'),'user/assistant'],
            ['page','number',t('no_label'),t('default_1')],
            ['page_size','number',t('no_label'),t('default_50')],
          ]} />
          <CodeBlock code={`{"total":120,"page":1,"page_size":50,"data":[{"id":1,"role":"user","content":"Hello"}]}`} />
          <h3 className="font-semibold mt-6 mb-1">{t('export_messages')}</h3>
          <ApiLine method="GET" path="/api/open/v1/history/export" />
        </Section>

        <Section id="inject" title={t('voice_push_api')}>
          <ApiLine method="POST" path="/api/open/v1/devices/inject-message" />
          <ParamTable rows={[
            ['device_id','string',t('yes_label'),t('device_identifier_field')],
            ['message','string',t('yes_label'),t('push_content')],
            ['skip_llm','boolean',t('no_label'),t('skip_llm_default_false')],
            ['auto_listen','boolean',t('no_label'),t('auto_listen_after_broadcast')],
          ]} />
          <CodeBlock code={`{"success":true,"message":"Voice push request sent","data":{"device_id":"bedroom","message":"hello","skip_llm":false,"auto_listen":true}}`} />
        </Section>

        <Section id="mcp" title={t('mcp_tool_api')}>
          <h3 className="font-semibold mt-4 mb-1">{t('get_agent_tool_list')}</h3>
          <ApiLine method="GET" path="/api/open/v1/agents/:id/mcp-tools" />
          <h3 className="font-semibold mt-6 mb-1">{t('call_agent_tool')}</h3>
          <ApiLine method="POST" path="/api/open/v1/agents/:id/mcp-call" />
          <ParamTable rows={[
            ['tool_name','string',t('yes_label'),t('tool_name')],
            ['arguments','object',t('no_label'),t('tool_params_object')],
          ]} />
          <h3 className="font-semibold mt-6 mb-1">{t('get_device_tool_list')}</h3>
          <ApiLine method="GET" path="/api/open/v1/devices/:id/mcp-tools" />
          <h3 className="font-semibold mt-6 mb-1">{t('call_device_tool')}</h3>
          <ApiLine method="POST" path="/api/open/v1/devices/:id/mcp-call" />
          <CodeBlock code={`{"data":{"device_id":"bedroom","tool_name":"device_tool","result":"ok"}}`} />
        </Section>
      </main>
    </div>
  )
}

export const Route = createFileRoute('/openapi-docs')({
  component: OpenAPIDocsPage,
})
