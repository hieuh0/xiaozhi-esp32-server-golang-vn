<template>
  <div class="vp-docs">
    <aside class="vp-sidebar">
      <div class="vp-sidebar-title">{{ t('openapi_docs') }}OpenAPI 文档</div>
      <a v-for="item in nav" :key="item.id" :href="`#${item.id}`" class="vp-nav-item">{{ item.label }}</a>
    </aside>

    <main class="vp-content">
      <header class="vp-hero">
        <h1>{{ t('xiaozhi_openapi_docs') }}小智 OpenAPI 文档</h1>
        <p class="lead">{{ t('public_api_hint') }}公开可访问，按接口提供请求方式、参数、出参与示例。</p>
        <div class="hero-meta">
          <span>Base URL: <code>/api/open/v1</code></span>
          <span>Content-Type: <code>application/json</code></span>
          <el-button size="small" type="primary" plain @click="$router.push('/login')">返回登录</el-button>
        </div>
      </header>

      <section id="auth" class="vp-section">
        <h2>{{ t('auth_method') }}认证方式</h2>
        <pre><code>Authorization: Bearer &lt;jwt-or-api-token&gt;
X-API-Token: &lt;api-token&gt;</code></pre>
      </section>

      <section id="common" class="vp-section">
        <h2>{{ t('general_response_description') }}通用响应说明</h2>
        <ul>
          <li>{{ t('common_error_codes') }}常见错误码：<code>400</code> {{ t('params_error') }} 参数错误，<code>401</code> {{ t('auth_failed') }} 认证失败，<code>404</code> {{ t('resource_not_found') }} 资源不存在，<code>500</code> {{ t('server_error') }} 服务端异常。</li>
          <li>{{ t('paged_api_default') }}分页接口默认：<code>page=1</code>、<code>page_size=50</code>。</li>
        </ul>
      </section>

      <section id="profile" class="vp-section">
        <h2>{{ t('get_current_user_info') }}1. 获取当前用户信息</h2>
        <div class="api-line"><span class="method get">GET</span><code>/api/open/v1/profile</code></div>
        <h4>{{ t('input_params') }}入参</h4><p>{{ t('no_auth_required') }}无（仅需认证头）。</p>
        <h4>{{ t('output_example') }}出参示例</h4>
        <pre><code>{
  "user": {"id": 1, "username": "demo", "email": "demo@example.com", "role": "user"}
}</code></pre>
      </section>

      <section id="devices" class="vp-section">
        <h2>{{ t('device_api') }}2. 设备接口</h2>

        <h3>{{ t('get_device_list') }}2.1 获取设备列表</h3>
        <div class="api-line"><span class="method get">GET</span><code>/api/open/v1/devices</code></div>
        <h4>{{ t('input_params') }}入参</h4><p>{{ t('no_auth_required') }}无（仅需认证头）。</p>
        <h4>{{ t('output_example') }}出参示例</h4>
        <pre><code>{"data":[{"id":1,"device_name":"bedroom","device_code":"123456","agent_id":2,"activated":true}]}</code></pre>

        <h3>{{ t('create_device') }}2.2 创建设备</h3>
        <div class="api-line"><span class="method post">POST</span><code>/api/open/v1/devices</code></div>
        <h4>{{ t('body_params') }}Body 参数</h4>
        <table><thead><tr><th>{{ t('field') }}字段</th><th>{{ t('type') }}类型</th><th>{{ t('required') }}必填</th><th>{{ t('description_2') }}说明</th></tr></thead><tbody>
          <tr><td>device_name</td><td>string</td><td>是</td><td>{{ t('device_name_length') }}设备名称，2-50 字符</td></tr>
          <tr><td>agent_id</td><td>number</td><td>是</td><td>{{ t('bind_agent_id') }}绑定智能体 ID</td></tr>
        </tbody></table>
        <h4>{{ t('output_example') }}出参示例</h4>
        <pre><code>{"success":true,"message":"设备创建成功","data":{"device_code":"654321","device":{"id":8,"device_name":"bedroom"}}}</code></pre>
      </section>

      <section id="agents" class="vp-section">
        <h2>{{ t('agent_api') }}3. 智能体接口</h2>

        <h3>{{ t('get_agent_list') }}3.1 获取智能体列表</h3>
        <div class="api-line"><span class="method get">GET</span><code>/api/open/v1/agents</code></div>
        <h4>{{ t('input_params') }}入参</h4><p>{{ t('no_auth_required') }}无（仅需认证头）。</p>
        <h4>{{ t('output_example') }}出参示例</h4>
        <pre><code>{"data":[{"id":2,"name":"家庭助手","nickname":"小辉","llm_config_id":"llm_default"}]}</code></pre>

        <h3>{{ t('create_agent') }}3.2 创建智能体</h3>
        <div class="api-line"><span class="method post">POST</span><code>/api/open/v1/agents</code></div>
        <h4>{{ t('body_params') }}Body 参数</h4>
        <table><thead><tr><th>{{ t('field') }}字段</th><th>{{ t('type') }}类型</th><th>{{ t('required') }}必填</th><th>{{ t('description_2') }}说明</th></tr></thead><tbody>
          <tr><td>name</td><td>string</td><td>是</td><td>{{ t('name_length_hint') }}名称，2-50 字符</td></tr>
          <tr><td>nickname</td><td>string</td><td>否</td><td>{{ t('nickname_hint') }}昵称，给大模型/Prompt 使用；空则默认等于 name</td></tr>
          <tr><td>custom_prompt</td><td>string</td><td>否</td><td>{{ t('prompt') }}提示词</td></tr>
          <tr><td>llm_config_id</td><td>string</td><td>否</td><td>{{ t('llm_config_id') }}LLM 配置 ID</td></tr>
          <tr><td>tts_config_id</td><td>string</td><td>否</td><td>{{ t('tts_config_id') }}TTS 配置 ID</td></tr>
          <tr><td>voice</td><td>string</td><td>否</td><td>{{ t('timbre_id') }}音色标识</td></tr>
          <tr><td>asr_speed</td><td>string</td><td>否</td><td>{{ t('default_normal') }}默认 normal</td></tr>
          <tr><td>memory_mode</td><td>string</td><td>否</td><td>short/long/none</td></tr>
        </tbody></table>
        <h4>{{ t('output_example') }}出参示例</h4>
        <pre><code>{"success":true,"data":{"id":3,"name":"客厅助手","nickname":"小辉"}}</code></pre>

        <h3>{{ t('get_agent_detail') }}3.3 获取智能体详情</h3>
        <div class="api-line"><span class="method get">GET</span><code>/api/open/v1/agents/:id</code></div>
        <h4>{{ t('path_params') }}Path 参数</h4>
        <table><thead><tr><th>{{ t('params') }}参数</th><th>{{ t('type') }}类型</th><th>{{ t('required') }}必填</th><th>{{ t('description_2') }}说明</th></tr></thead><tbody>
          <tr><td>id</td><td>number</td><td>是</td><td>{{ t('agent_id') }}智能体 ID</td></tr>
        </tbody></table>
        <h4>{{ t('output_example') }}出参示例</h4>
        <pre><code>{"data":{"id":2,"name":"家庭助手","nickname":"小辉","custom_prompt":"..."}}</code></pre>

        <h3>{{ t('update_agent') }}3.4 更新智能体</h3>
        <div class="api-line"><span class="method put">PUT</span><code>/api/open/v1/agents/:id</code></div>
        <h4>{{ t('path_params') }}Path 参数</h4>
        <table><thead><tr><th>{{ t('params') }}参数</th><th>{{ t('type') }}类型</th><th>{{ t('required') }}必填</th><th>{{ t('description_2') }}说明</th></tr></thead><tbody>
          <tr><td>id</td><td>number</td><td>是</td><td>{{ t('agent_id') }}智能体 ID</td></tr>
        </tbody></table>
        <h4>{{ t('body_params') }}Body 参数</h4>
        <table><thead><tr><th>{{ t('field') }}字段</th><th>{{ t('type') }}类型</th><th>{{ t('required') }}必填</th><th>{{ t('description_2') }}说明</th></tr></thead><tbody>
          <tr><td>name</td><td>string</td><td>是</td><td>{{ t('name_length_hint') }}名称，2-50 字符</td></tr>
          <tr><td>nickname</td><td>string</td><td>否</td><td>{{ t('nickname_hint') }}昵称，给大模型/Prompt 使用；空则默认等于 name</td></tr>
          <tr><td>custom_prompt</td><td>string</td><td>否</td><td>{{ t('prompt') }}提示词</td></tr>
          <tr><td>llm_config_id</td><td>string</td><td>否</td><td>{{ t('llm_config_id_optional') }}LLM 配置 ID（可置空）</td></tr>
          <tr><td>tts_config_id</td><td>string</td><td>否</td><td>{{ t('tts_config_id_optional') }}TTS 配置 ID（可置空）</td></tr>
          <tr><td>voice</td><td>string</td><td>否</td><td>{{ t('timbre_id') }}音色标识</td></tr>
          <tr><td>asr_speed</td><td>string</td><td>否</td><td>{{ t('empty_use_normal') }}空则 normal</td></tr>
          <tr><td>memory_mode</td><td>string</td><td>否</td><td>short/long/none</td></tr>
        </tbody></table>
        <h4>{{ t('output_example') }}出参示例</h4>
        <pre><code>{"data":{"id":2,"name":"家庭助手-更新后","nickname":"小辉"}}</code></pre>

        <h3>{{ t('delete_agent') }}3.5 删除智能体</h3>
        <div class="api-line"><span class="method delete">DELETE</span><code>/api/open/v1/agents/:id</code></div>
        <h4>{{ t('path_params') }}Path 参数</h4>
        <table><thead><tr><th>{{ t('params') }}参数</th><th>{{ t('type') }}类型</th><th>{{ t('required') }}必填</th><th>{{ t('description_2') }}说明</th></tr></thead><tbody>
          <tr><td>id</td><td>number</td><td>是</td><td>{{ t('agent_id') }}智能体 ID</td></tr>
        </tbody></table>
        <h4>{{ t('output_example') }}出参示例</h4>
        <pre><code>{"message":"删除成功"}</code></pre>
      </section>

      <section id="history" class="vp-section">
        <h2>{{ t('chat_history_api') }}4. 聊天记录接口</h2>

        <h3>{{ t('query_messages_paged') }}4.1 查询消息（分页）</h3>
        <div class="api-line"><span class="method get">GET</span><code>/api/open/v1/history/messages</code></div>
        <h4>{{ t('query_params') }}Query 参数</h4>
        <table><thead><tr><th>{{ t('params') }}参数</th><th>{{ t('type') }}类型</th><th>{{ t('required') }}必填</th><th>{{ t('description_2') }}说明</th></tr></thead><tbody>
          <tr><td>agent_id</td><td>string</td><td>否</td><td>{{ t('agent_id') }}智能体 ID</td></tr>
          <tr><td>device_id</td><td>string</td><td>否</td><td>{{ t('device_identifier_field') }}设备标识（device_name）</td></tr>
          <tr><td>session_id</td><td>string</td><td>否</td><td>{{ t('session_id') }}会话 ID</td></tr>
          <tr><td>role</td><td>string</td><td>否</td><td>user/assistant</td></tr>
          <tr><td>page</td><td>number</td><td>否</td><td>{{ t('default_1') }}默认 1</td></tr>
          <tr><td>page_size</td><td>number</td><td>否</td><td>{{ t('default_50') }}默认 50</td></tr>
        </tbody></table>
        <h4>{{ t('output_example') }}出参示例</h4>
        <pre><code>{"total":120,"page":1,"page_size":50,"data":[{"id":1,"role":"user","content":"你好"}]}</code></pre>

        <h3>{{ t('export_messages') }}4.2 导出消息</h3>
        <div class="api-line"><span class="method get">GET</span><code>/api/open/v1/history/export</code></div>
        <h4>{{ t('query_params') }}Query 参数</h4>
        <table><thead><tr><th>{{ t('params') }}参数</th><th>{{ t('type') }}类型</th><th>{{ t('required') }}必填</th><th>{{ t('description_2') }}说明</th></tr></thead><tbody>
          <tr><td>agent_id</td><td>string</td><td>否</td><td>{{ t('agent_id') }}智能体 ID</td></tr>
          <tr><td>device_id</td><td>string</td><td>否</td><td>{{ t('device_identifier_field') }}设备标识（device_name）</td></tr>
          <tr><td>start_date</td><td>string</td><td>否</td><td>YYYY-MM-DD</td></tr>
          <tr><td>end_date</td><td>string</td><td>否</td><td>YYYY-MM-DD</td></tr>
        </tbody></table>
        <h4>{{ t('output_example') }}出参示例</h4>
        <pre><code>{"export_time":"2026-03-17 10:00:00","total":20,"messages":[...]}</code></pre>
      </section>

      <section id="inject" class="vp-section">
        <h2>{{ t('voice_push_api') }}5. 语音推送接口</h2>
        <div class="api-line"><span class="method post">POST</span><code>/api/open/v1/devices/inject-message</code></div>
        <h4>{{ t('body_params') }}Body 参数</h4>
        <table><thead><tr><th>{{ t('field') }}字段</th><th>{{ t('type') }}类型</th><th>{{ t('required') }}必填</th><th>{{ t('description_2') }}说明</th></tr></thead><tbody>
          <tr><td>device_id</td><td>string</td><td>是</td><td>{{ t('device_identifier_field') }}设备标识（device_name）</td></tr>
          <tr><td>message</td><td>string</td><td>是</td><td>{{ t('push_content') }}推送内容</td></tr>
          <tr><td>skip_llm</td><td>boolean</td><td>否</td><td>{{ t('skip_llm_default_false') }}是否跳过 LLM，默认 false</td></tr>
          <tr><td>auto_listen</td><td>boolean</td><td>否</td><td>{{ t('auto_listen_after_broadcast') }}播报完成后是否自动进入监听，默认 true</td></tr>
        </tbody></table>
        <h4>{{ t('output_example') }}出参示例</h4>
        <pre><code>{"success":true,"message":"语音推送请求已发送","data":{"device_id":"bedroom","message":"hello","skip_llm":false,"auto_listen":true}}</code></pre>
      </section>

      <section id="mcp" class="vp-section">
        <h2>{{ t('mcp_tool_api') }}6. MCP 工具接口</h2>

        <h3>{{ t('get_agent_tool_list') }}6.1 获取智能体工具列表</h3>
        <div class="api-line"><span class="method get">GET</span><code>/api/open/v1/agents/:id/mcp-tools</code></div>
        <h4>{{ t('path_params') }}Path 参数</h4>
        <table><thead><tr><th>{{ t('params') }}参数</th><th>{{ t('type') }}类型</th><th>{{ t('required') }}必填</th><th>{{ t('description_2') }}说明</th></tr></thead><tbody>
          <tr><td>id</td><td>number</td><td>是</td><td>{{ t('agent_id') }}智能体 ID</td></tr>
        </tbody></table>
        <h4>{{ t('output_example') }}出参示例</h4>
        <pre><code>{"data":{"tools":[{"name":"tool_a","description":"..."}]}}</code></pre>

        <h3>{{ t('call_agent_tool') }}6.2 调用智能体工具</h3>
        <div class="api-line"><span class="method post">POST</span><code>/api/open/v1/agents/:id/mcp-call</code></div>
        <h4>{{ t('path_params') }}Path 参数</h4>
        <table><thead><tr><th>{{ t('params') }}参数</th><th>{{ t('type') }}类型</th><th>{{ t('required') }}必填</th><th>{{ t('description_2') }}说明</th></tr></thead><tbody>
          <tr><td>id</td><td>number</td><td>是</td><td>{{ t('agent_id') }}智能体 ID</td></tr>
        </tbody></table>
        <h4>{{ t('body_params') }}Body 参数</h4>
        <table><thead><tr><th>{{ t('field') }}字段</th><th>{{ t('type') }}类型</th><th>{{ t('required') }}必填</th><th>{{ t('description_2') }}说明</th></tr></thead><tbody>
          <tr><td>tool_name</td><td>string</td><td>是</td><td>{{ t('tool_name') }}工具名称</td></tr>
          <tr><td>arguments</td><td>object</td><td>否</td><td>{{ t('tool_params_object') }}工具参数对象</td></tr>
        </tbody></table>
        <h4>{{ t('output_example') }}出参示例</h4>
        <pre><code>{"data":{"result":"ok"}}</code></pre>

        <h3>{{ t('get_device_tool_list') }}6.3 获取设备工具列表</h3>
        <div class="api-line"><span class="method get">GET</span><code>/api/open/v1/devices/:id/mcp-tools</code></div>
        <h4>{{ t('path_params') }}Path 参数</h4>
        <table><thead><tr><th>{{ t('params') }}参数</th><th>{{ t('type') }}类型</th><th>{{ t('required') }}必填</th><th>{{ t('description_2') }}说明</th></tr></thead><tbody>
          <tr><td>id</td><td>number</td><td>是</td><td>{{ t('device_id') }}设备 ID</td></tr>
        </tbody></table>
        <h4>{{ t('description_2') }}说明</h4>
        <ul>
          <li>仅返回当前在线 transport 对应的设备侧 IoT over MCP 工具。</li>
          <li>不会混入其它 transport 的历史工具，也不会混入智能体 ws-endpoint 工具。</li>
        </ul>
        <h4>{{ t('output_example') }}出参示例</h4>
        <pre><code>{"data":{"tools":[{"name":"device_tool","description":"..."}]}}</code></pre>

        <h3>{{ t('call_device_tool') }}6.4 调用设备工具</h3>
        <div class="api-line"><span class="method post">POST</span><code>/api/open/v1/devices/:id/mcp-call</code></div>
        <h4>{{ t('path_params') }}Path 参数</h4>
        <table><thead><tr><th>{{ t('params') }}参数</th><th>{{ t('type') }}类型</th><th>{{ t('required') }}必填</th><th>{{ t('description_2') }}说明</th></tr></thead><tbody>
          <tr><td>id</td><td>number</td><td>是</td><td>{{ t('device_id') }}设备 ID</td></tr>
        </tbody></table>
        <h4>{{ t('body_params') }}Body 参数</h4>
        <table><thead><tr><th>{{ t('field') }}字段</th><th>{{ t('type') }}类型</th><th>{{ t('required') }}必填</th><th>{{ t('description_2') }}说明</th></tr></thead><tbody>
          <tr><td>tool_name</td><td>string</td><td>是</td><td>{{ t('tool_name') }}工具名称</td></tr>
          <tr><td>arguments</td><td>object</td><td>否</td><td>{{ t('tool_params_object') }}工具参数对象</td></tr>
        </tbody></table>
        <h4>{{ t('description_2') }}说明</h4>
        <ul>
          <li>{{ t('prefer_device_tool_hint') }}优先按当前设备工具列表匹配调用。</li>
          <li>当工具暂未出现在列表中，但当前 runtime 仍可用时，服务端会自动尝试 raw call 兜底。</li>
        </ul>
        <h4>{{ t('output_example') }}出参示例</h4>
        <pre><code>{"data":{"device_id":"bedroom","tool_name":"device_tool","result":"ok"}}</code></pre>
      </section>
    </main>
  </div>
</template>

<script setup>
import { useLocale } from '../composables/useLocale'
const { t } = useLocale()
const nav = [
  { id: 'auth', label: t('auth_method') },
  { id: 'common', label: '通用说明' },
  { id: 'profile', label: '1. 用户信息' },
  { id: 'devices', label: t('device_api') },
  { id: 'agents', label: t('agent_api') },
  { id: 'history', label: '4. 聊天记录' },
  { id: 'inject', label: '5. 语音推送' },
  { id: 'mcp', label: '6. MCP 工具' }
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
.vp-section { margin-top: 26px; padding-top: 6px; border-top: 1px solid #f0f2f5; }
.vp-section h2 { margin: 0 0 10px; }
.vp-section h3 { margin: 20px 0 8px; }
.vp-section h4 { margin: 14px 0 6px; font-size: 14px; color: #374151; }
.api-line { display: flex; align-items: center; gap: 10px; margin: 8px 0; }
.method { color: #fff; font-size: 12px; border-radius: 6px; padding: 2px 8px; font-weight: 600; }
.method.get { background: #10b981; }
.method.post { background: #3b82f6; }
.method.put { background: #f59e0b; }
.method.delete { background: #ef4444; }
pre { margin: 6px 0; background: #f6f8fb; color: #111827; border: 1px solid #d1d5db; border-radius: 8px; padding: 12px; overflow: auto; font-size: 12px; }
code { background: #f3f4f6; padding: 2px 6px; border-radius: 8px; }
table { width: 100%; border-collapse: collapse; font-size: 14px; }
th, td { border: 1px solid #e5e7eb; padding: 8px; text-align: left; }
thead { background: #f8fafc; }
@media (max-width: 960px) {
  .vp-docs { display: block; }
  .vp-sidebar { position: static; height: auto; min-width: auto; border-right: none; border-bottom: 1px solid #e5e7eb; padding-bottom: 12px; margin-bottom: 16px; }
}
</style>
