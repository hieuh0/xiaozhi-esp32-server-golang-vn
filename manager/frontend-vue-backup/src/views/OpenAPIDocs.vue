<template>
  <div class="vp-docs">
    <aside class="vp-sidebar">
      <div class="vp-sidebar-title">{{ t('openapi_docs') }}</div>
      <a v-for="item in nav" :key="item.id" :href="`#${item.id}`" class="vp-nav-item">{{ item.label }}</a>
    </aside>

    <main class="vp-content">
      <header class="vp-hero">
        <h1>{{ t('xiaozhi_openapi_docs') }}</h1>
        <p class="lead">{{ t('public_api_hint') }}</p>
        <div class="hero-meta">
          <span>Base URL: <code>/api/open/v1</code></span>
          <span>Content-Type: <code>application/json</code></span>
          <Button size="sm" variant="outline" @click="$router.push('/login')">{{ t('back_to_login') }}</Button>
        </div>
      </header>

      <section id="auth" class="vp-section">
        <h2>{{ t('auth_method') }}</h2>
        <pre><code>Authorization: Bearer &lt;jwt-or-api-token&gt;
X-API-Token: &lt;api-token&gt;</code></pre>
      </section>

      <section id="common" class="vp-section">
        <h2>{{ t('general_response_description') }}</h2>
        <ul>
          <li>{{ t('common_error_codes') }}<code>400</code> {{ t('params_error') }}<code>401</code> {{ t('auth_failed') }}<code>404</code> {{ t('resource_not_found') }}<code>500</code> {{ t('server_error') }}</li>
          <li>{{ t('paged_api_default') }}<code>page=1</code>, <code>page_size=50</code>.</li>
        </ul>
      </section>

      <section id="profile" class="vp-section">
        <h2>{{ t('get_current_user_info') }}</h2>
        <div class="api-line"><span class="method get">GET</span><code>/api/open/v1/profile</code></div>
        <h4>{{ t('input_params') }}</h4><p>{{ t('no_auth_required') }}</p>
        <h4>{{ t('output_example') }}</h4>
        <pre><code>{
  "user": {"id": 1, "username": "demo", "email": "demo@example.com", "role": "user"}
}</code></pre>
      </section>

      <section id="devices" class="vp-section">
        <h2>{{ t('device_api') }}</h2>

        <h3>{{ t('get_device_list') }}</h3>
        <div class="api-line"><span class="method get">GET</span><code>/api/open/v1/devices</code></div>
        <h4>{{ t('input_params') }}</h4><p>{{ t('no_auth_required') }}</p>
        <h4>{{ t('output_example') }}</h4>
        <pre><code>{"data":[{"id":1,"device_name":"bedroom","device_code":"123456","agent_id":2,"activated":true}]}</code></pre>

        <h3>{{ t('create_device') }}</h3>
        <div class="api-line"><span class="method post">POST</span><code>/api/open/v1/devices</code></div>
        <h4>{{ t('body_params') }}</h4>
        <table><thead><tr><th>{{ t('field') }}</th><th>{{ t('type') }}</th><th>{{ t('required') }}</th><th>{{ t('description_2') }}</th></tr></thead><tbody>
          <tr><td>device_name</td><td>string</td><td>{{ t('yes_label') }}</td><td>{{ t('device_name_length') }}</td></tr>
          <tr><td>agent_id</td><td>number</td><td>{{ t('yes_label') }}</td><td>{{ t('bind_agent_id') }}</td></tr>
        </tbody></table>
        <h4>{{ t('output_example') }}</h4>
        <pre><code>{"success":true,"message":"Device created successfully","data":{"device_code":"654321","device":{"id":8,"device_name":"bedroom"}}}</code></pre>
      </section>

      <section id="agents" class="vp-section">
        <h2>{{ t('agent_api') }}</h2>

        <h3>{{ t('get_agent_list') }}</h3>
        <div class="api-line"><span class="method get">GET</span><code>/api/open/v1/agents</code></div>
        <h4>{{ t('input_params') }}</h4><p>{{ t('no_auth_required') }}</p>
        <h4>{{ t('output_example') }}</h4>
        <pre><code>{"data":[{"id":2,"name":"Home Assistant","nickname":"Xiao Hui","llm_config_id":"llm_default"}]}</code></pre>

        <h3>{{ t('create_agent') }}</h3>
        <div class="api-line"><span class="method post">POST</span><code>/api/open/v1/agents</code></div>
        <h4>{{ t('body_params') }}</h4>
        <table><thead><tr><th>{{ t('field') }}</th><th>{{ t('type') }}</th><th>{{ t('required') }}</th><th>{{ t('description_2') }}</th></tr></thead><tbody>
          <tr><td>name</td><td>string</td><td>{{ t('yes_label') }}</td><td>{{ t('name_length_hint') }}</td></tr>
          <tr><td>nickname</td><td>string</td><td>{{ t('no_label') }}</td><td>{{ t('nickname_hint') }}</td></tr>
          <tr><td>custom_prompt</td><td>string</td><td>{{ t('no_label') }}</td><td>{{ t('prompt') }}</td></tr>
          <tr><td>llm_config_id</td><td>string</td><td>{{ t('no_label') }}</td><td>{{ t('llm_config_id') }}</td></tr>
          <tr><td>tts_config_id</td><td>string</td><td>{{ t('no_label') }}</td><td>{{ t('tts_config_id') }}</td></tr>
          <tr><td>voice</td><td>string</td><td>{{ t('no_label') }}</td><td>{{ t('timbre_id') }}</td></tr>
          <tr><td>asr_speed</td><td>string</td><td>{{ t('no_label') }}</td><td>{{ t('default_normal') }}</td></tr>
          <tr><td>memory_mode</td><td>string</td><td>{{ t('no_label') }}</td><td>short/long/none</td></tr>
        </tbody></table>
        <h4>{{ t('output_example') }}</h4>
        <pre><code>{"success":true,"data":{"id":3,"name":"Living Room Assistant","nickname":"Xiao Hui"}}</code></pre>

        <h3>{{ t('get_agent_detail') }}</h3>
        <div class="api-line"><span class="method get">GET</span><code>/api/open/v1/agents/:id</code></div>
        <h4>{{ t('path_params') }}</h4>
        <table><thead><tr><th>{{ t('params') }}</th><th>{{ t('type') }}</th><th>{{ t('required') }}</th><th>{{ t('description_2') }}</th></tr></thead><tbody>
          <tr><td>id</td><td>number</td><td>{{ t('yes_label') }}</td><td>{{ t('agent_id') }}</td></tr>
        </tbody></table>
        <h4>{{ t('output_example') }}</h4>
        <pre><code>{"data":{"id":2,"name":"Home Assistant","nickname":"Xiao Hui","custom_prompt":"..."}}</code></pre>

        <h3>{{ t('update_agent') }}</h3>
        <div class="api-line"><span class="method put">PUT</span><code>/api/open/v1/agents/:id</code></div>
        <h4>{{ t('path_params') }}</h4>
        <table><thead><tr><th>{{ t('params') }}</th><th>{{ t('type') }}</th><th>{{ t('required') }}</th><th>{{ t('description_2') }}</th></tr></thead><tbody>
          <tr><td>id</td><td>number</td><td>{{ t('yes_label') }}</td><td>{{ t('agent_id') }}</td></tr>
        </tbody></table>
        <h4>{{ t('body_params') }}</h4>
        <table><thead><tr><th>{{ t('field') }}</th><th>{{ t('type') }}</th><th>{{ t('required') }}</th><th>{{ t('description_2') }}</th></tr></thead><tbody>
          <tr><td>name</td><td>string</td><td>{{ t('yes_label') }}</td><td>{{ t('name_length_hint') }}</td></tr>
          <tr><td>nickname</td><td>string</td><td>{{ t('no_label') }}</td><td>{{ t('nickname_hint') }}</td></tr>
          <tr><td>custom_prompt</td><td>string</td><td>{{ t('no_label') }}</td><td>{{ t('prompt') }}</td></tr>
          <tr><td>llm_config_id</td><td>string</td><td>{{ t('no_label') }}</td><td>{{ t('llm_config_id_optional') }}</td></tr>
          <tr><td>tts_config_id</td><td>string</td><td>{{ t('no_label') }}</td><td>{{ t('tts_config_id_optional') }}</td></tr>
          <tr><td>voice</td><td>string</td><td>{{ t('no_label') }}</td><td>{{ t('timbre_id') }}</td></tr>
          <tr><td>asr_speed</td><td>string</td><td>{{ t('no_label') }}</td><td>{{ t('empty_use_normal') }}</td></tr>
          <tr><td>memory_mode</td><td>string</td><td>{{ t('no_label') }}</td><td>short/long/none</td></tr>
        </tbody></table>
        <h4>{{ t('output_example') }}</h4>
        <pre><code>{"data":{"id":2,"name":"Home Assistant (updated)","nickname":"Xiao Hui"}}</code></pre>

        <h3>{{ t('delete_agent') }}</h3>
        <div class="api-line"><span class="method delete">DELETE</span><code>/api/open/v1/agents/:id</code></div>
        <h4>{{ t('path_params') }}</h4>
        <table><thead><tr><th>{{ t('params') }}</th><th>{{ t('type') }}</th><th>{{ t('required') }}</th><th>{{ t('description_2') }}</th></tr></thead><tbody>
          <tr><td>id</td><td>number</td><td>{{ t('yes_label') }}</td><td>{{ t('agent_id') }}</td></tr>
        </tbody></table>
        <h4>{{ t('output_example') }}</h4>
        <pre><code>{"message":"Deleted successfully"}</code></pre>
      </section>

      <section id="history" class="vp-section">
        <h2>{{ t('chat_history_api') }}</h2>

        <h3>{{ t('query_messages_paged') }}</h3>
        <div class="api-line"><span class="method get">GET</span><code>/api/open/v1/history/messages</code></div>
        <h4>{{ t('query_params') }}</h4>
        <table><thead><tr><th>{{ t('params') }}</th><th>{{ t('type') }}</th><th>{{ t('required') }}</th><th>{{ t('description_2') }}</th></tr></thead><tbody>
          <tr><td>agent_id</td><td>string</td><td>{{ t('no_label') }}</td><td>{{ t('agent_id') }}</td></tr>
          <tr><td>device_id</td><td>string</td><td>{{ t('no_label') }}</td><td>{{ t('device_identifier_field') }}</td></tr>
          <tr><td>session_id</td><td>string</td><td>{{ t('no_label') }}</td><td>{{ t('session_id') }}</td></tr>
          <tr><td>role</td><td>string</td><td>{{ t('no_label') }}</td><td>user/assistant</td></tr>
          <tr><td>page</td><td>number</td><td>{{ t('no_label') }}</td><td>{{ t('default_1') }}</td></tr>
          <tr><td>page_size</td><td>number</td><td>{{ t('no_label') }}</td><td>{{ t('default_50') }}</td></tr>
        </tbody></table>
        <h4>{{ t('output_example') }}</h4>
        <pre><code>{"total":120,"page":1,"page_size":50,"data":[{"id":1,"role":"user","content":"Hello"}]}</code></pre>

        <h3>{{ t('export_messages') }}</h3>
        <div class="api-line"><span class="method get">GET</span><code>/api/open/v1/history/export</code></div>
        <h4>{{ t('query_params') }}</h4>
        <table><thead><tr><th>{{ t('params') }}</th><th>{{ t('type') }}</th><th>{{ t('required') }}</th><th>{{ t('description_2') }}</th></tr></thead><tbody>
          <tr><td>agent_id</td><td>string</td><td>{{ t('no_label') }}</td><td>{{ t('agent_id') }}</td></tr>
          <tr><td>device_id</td><td>string</td><td>{{ t('no_label') }}</td><td>{{ t('device_identifier_field') }}</td></tr>
          <tr><td>start_date</td><td>string</td><td>{{ t('no_label') }}</td><td>YYYY-MM-DD</td></tr>
          <tr><td>end_date</td><td>string</td><td>{{ t('no_label') }}</td><td>YYYY-MM-DD</td></tr>
        </tbody></table>
        <h4>{{ t('output_example') }}</h4>
        <pre><code>{"export_time":"2026-03-17 10:00:00","total":20,"messages":[...]}</code></pre>
      </section>

      <section id="inject" class="vp-section">
        <h2>{{ t('voice_push_api') }}</h2>
        <div class="api-line"><span class="method post">POST</span><code>/api/open/v1/devices/inject-message</code></div>
        <h4>{{ t('body_params') }}</h4>
        <table><thead><tr><th>{{ t('field') }}</th><th>{{ t('type') }}</th><th>{{ t('required') }}</th><th>{{ t('description_2') }}</th></tr></thead><tbody>
          <tr><td>device_id</td><td>string</td><td>{{ t('yes_label') }}</td><td>{{ t('device_identifier_field') }}</td></tr>
          <tr><td>message</td><td>string</td><td>{{ t('yes_label') }}</td><td>{{ t('push_content') }}</td></tr>
          <tr><td>skip_llm</td><td>boolean</td><td>{{ t('no_label') }}</td><td>{{ t('skip_llm_default_false') }}</td></tr>
          <tr><td>auto_listen</td><td>boolean</td><td>{{ t('no_label') }}</td><td>{{ t('auto_listen_after_broadcast') }}</td></tr>
        </tbody></table>
        <h4>{{ t('output_example') }}</h4>
        <pre><code>{"success":true,"message":"Voice push request sent","data":{"device_id":"bedroom","message":"hello","skip_llm":false,"auto_listen":true}}</code></pre>
      </section>

      <section id="mcp" class="vp-section">
        <h2>{{ t('mcp_tool_api') }}</h2>

        <h3>{{ t('get_agent_tool_list') }}</h3>
        <div class="api-line"><span class="method get">GET</span><code>/api/open/v1/agents/:id/mcp-tools</code></div>
        <h4>{{ t('path_params') }}</h4>
        <table><thead><tr><th>{{ t('params') }}</th><th>{{ t('type') }}</th><th>{{ t('required') }}</th><th>{{ t('description_2') }}</th></tr></thead><tbody>
          <tr><td>id</td><td>number</td><td>{{ t('yes_label') }}</td><td>{{ t('agent_id') }}</td></tr>
        </tbody></table>
        <h4>{{ t('output_example') }}</h4>
        <pre><code>{"data":{"tools":[{"name":"tool_a","description":"..."}]}}</code></pre>

        <h3>{{ t('call_agent_tool') }}</h3>
        <div class="api-line"><span class="method post">POST</span><code>/api/open/v1/agents/:id/mcp-call</code></div>
        <h4>{{ t('path_params') }}</h4>
        <table><thead><tr><th>{{ t('params') }}</th><th>{{ t('type') }}</th><th>{{ t('required') }}</th><th>{{ t('description_2') }}</th></tr></thead><tbody>
          <tr><td>id</td><td>number</td><td>{{ t('yes_label') }}</td><td>{{ t('agent_id') }}</td></tr>
        </tbody></table>
        <h4>{{ t('body_params') }}</h4>
        <table><thead><tr><th>{{ t('field') }}</th><th>{{ t('type') }}</th><th>{{ t('required') }}</th><th>{{ t('description_2') }}</th></tr></thead><tbody>
          <tr><td>tool_name</td><td>string</td><td>{{ t('yes_label') }}</td><td>{{ t('tool_name') }}</td></tr>
          <tr><td>arguments</td><td>object</td><td>{{ t('no_label') }}</td><td>{{ t('tool_params_object') }}</td></tr>
        </tbody></table>
        <h4>{{ t('output_example') }}</h4>
        <pre><code>{"data":{"result":"ok"}}</code></pre>

        <h3>{{ t('get_device_tool_list') }}</h3>
        <div class="api-line"><span class="method get">GET</span><code>/api/open/v1/devices/:id/mcp-tools</code></div>
        <h4>{{ t('path_params') }}</h4>
        <table><thead><tr><th>{{ t('params') }}</th><th>{{ t('type') }}</th><th>{{ t('required') }}</th><th>{{ t('description_2') }}</th></tr></thead><tbody>
          <tr><td>id</td><td>number</td><td>{{ t('yes_label') }}</td><td>{{ t('device_id') }}</td></tr>
        </tbody></table>
        <h4>{{ t('description_2') }}</h4>
        <ul>
          <li>{{ t('device_mcp_tools_hint1') }}</li>
          <li>{{ t('device_mcp_tools_hint2') }}</li>
        </ul>
        <h4>{{ t('output_example') }}</h4>
        <pre><code>{"data":{"tools":[{"name":"device_tool","description":"..."}]}}</code></pre>

        <h3>{{ t('call_device_tool') }}</h3>
        <div class="api-line"><span class="method post">POST</span><code>/api/open/v1/devices/:id/mcp-call</code></div>
        <h4>{{ t('path_params') }}</h4>
        <table><thead><tr><th>{{ t('params') }}</th><th>{{ t('type') }}</th><th>{{ t('required') }}</th><th>{{ t('description_2') }}</th></tr></thead><tbody>
          <tr><td>id</td><td>number</td><td>{{ t('yes_label') }}</td><td>{{ t('device_id') }}</td></tr>
        </tbody></table>
        <h4>{{ t('body_params') }}</h4>
        <table><thead><tr><th>{{ t('field') }}</th><th>{{ t('type') }}</th><th>{{ t('required') }}</th><th>{{ t('description_2') }}</th></tr></thead><tbody>
          <tr><td>tool_name</td><td>string</td><td>{{ t('yes_label') }}</td><td>{{ t('tool_name') }}</td></tr>
          <tr><td>arguments</td><td>object</td><td>{{ t('no_label') }}</td><td>{{ t('tool_params_object') }}</td></tr>
        </tbody></table>
        <h4>{{ t('description_2') }}</h4>
        <ul>
          <li>{{ t('prefer_device_tool_hint') }}</li>
          <li>{{ t('device_tool_fallback_hint') }}</li>
        </ul>
        <h4>{{ t('output_example') }}</h4>
        <pre><code>{"data":{"device_id":"bedroom","tool_name":"device_tool","result":"ok"}}</code></pre>
      </section>
    </main>
  </div>
</template>

<script setup>
import { useLocale } from '../composables/useLocale'
import { Button } from '@/components/ui/button'
const { t } = useLocale()
const nav = [
  { id: 'auth', label: t('auth_method') },
  { id: 'common', label: t('general_desc') },
  { id: 'profile', label: t('user_info_nav') },
  { id: 'devices', label: t('device_api') },
  { id: 'agents', label: t('agent_api') },
  { id: 'history', label: t('chat_history_nav') },
  { id: 'inject', label: t('voice_push_nav') },
  { id: 'mcp', label: t('mcp_tool_nav') }
]
</script>

<style scoped>
.vp-docs { display: flex; gap: 24px; max-width: 1280px; margin: 0 auto; padding: 24px 16px 40px; color: #213547; }
.vp-sidebar { position: sticky; top: 20px; height: calc(100vh - 40px); min-width: 220px; border-right: 1px solid #e5e7eb; padding-right: 14px; display: flex; flex-direction: column; gap: 8px; }
.vp-sidebar-title { font-weight: 700; margin-bottom: 8px; }
.vp-nav-item { color: #4b5563; text-decoration: none; font-size: 14px; }
.vp-nav-item:hover { color: #3b82f6; }
.vp-content { flex: 1; min-width: 0; }
.vp-hero h1 { margin: 0; font-size: 32px; }
.lead { margin: 10px 0; color: #4b5563; }
.hero-meta { display: flex; align-items: center; gap: 10px; flex-wrap: wrap; }
</style>
