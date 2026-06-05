<template>
  <el-form ref="formRef" :model="model" :rules="rules" label-width="120px">
    <el-form-item :label="t('provider')" prop="provider">
      <el-select v-model="model.provider" :placeholder="t('select_provider')" style="width: 100%">
        <el-option
          v-for="provider in TTS_PROVIDER_OPTIONS"
          :key="provider.value"
          :label="provider.label"
          :value="provider.value"
        >
          <div style="display:flex; align-items:center; justify-content:space-between; gap:8px;">
            <span>{{ provider.label }}</span>
            <el-tag v-if="provider.supports_voice_clone" size="small" type="success" effect="plain">{{ t('support_clone') }}</el-tag>
          </div>
        </el-option>
      </el-select>
    </el-form-item>
    <el-form-item :label="t('config_name')" prop="name">
      <el-input v-model="model.name" :placeholder="t('enter_config_name')" />
    </el-form-item>
    <el-form-item :label="t('config_id')" prop="config_id">
      <el-input v-model="model.config_id" :placeholder="t('enter_unique_config_id')" />
    </el-form-item>

    <template v-if="model.provider === 'doubao_ws'">
      <el-form-item :label="t('app_id')" prop="doubao_ws.appid">
        <el-input v-model="model.doubao_ws.appid" :placeholder="t('enter_app_id')" />
      </el-form-item>
      <el-form-item :label="t('access_token')" prop="doubao_ws.access_token">
        <el-input v-model="model.doubao_ws.access_token" :placeholder="t('enter_access_token')" type="password" show-password />
      </el-form-item>
      <el-form-item :label="t('model')" prop="doubao_ws.model">
        <el-select v-model="model.doubao_ws.model" :placeholder="t('select_model')" style="width: 100%">
          <el-option v-for="option in DOUBAO_MODEL_OPTIONS" :key="option.value" :label="option.label" :value="option.value" />
        </el-select>
      </el-form-item>
      <el-form-item :label="t('resource_id')" prop="doubao_ws.resource_id">
        <el-input v-model="model.doubao_ws.resource_id" :placeholder="t('optional_instance_id_hint')" />
      </el-form-item>
      <el-form-item :label="t('voice_timbre')" prop="doubao_ws.voice">
        <el-select
          v-model="model.doubao_ws.voice"
          :placeholder="t('select_timbre')"
          style="width: 100%"
          filterable
          :loading="voiceLoading"
          :disabled="voiceLoading"
          allow-create
          default-first-option
        >
          <el-option v-for="option in voiceOptionsList" :key="option.value" :label="option.label" :value="option.value" />
        </el-select>
      </el-form-item>
      <el-form-item label="WebSocket URL" prop="doubao_ws.ws_url" label-width="unset">
        <el-input v-model="model.doubao_ws.ws_url" placeholder="wss://openspeech.bytedance.com/api/v3/tts/unidirectional/stream" />
      </el-form-item>
    </template>

    <template v-if="model.provider === 'edge'">
      <el-form-item :label="t('voice_timbre')" prop="edge.voice">
        <el-select
          v-model="model.edge.voice"
          :placeholder="t('select_timbre')"
          style="width: 100%"
          filterable
          :loading="voiceLoading"
          :disabled="voiceLoading"
          allow-create
          default-first-option
        >
          <el-option v-for="option in voiceOptionsList" :key="option.value" :label="option.label" :value="option.value" />
        </el-select>
      </el-form-item>
      <el-form-item :label="t('speech_rate')" prop="edge.rate">
        <el-input v-model="model.edge.rate" :placeholder="t('enter_speech_rate_placeholder')" />
      </el-form-item>
      <el-form-item :label="t('volume')" prop="edge.volume">
        <el-input v-model="model.edge.volume" :placeholder="t('enter_volume_placeholder')" />
      </el-form-item>
      <el-form-item :label="t('pitch')" prop="edge.pitch">
        <el-input v-model="model.edge.pitch" :placeholder="t('enter_pitch_placeholder')" />
      </el-form-item>
      <el-form-item :label="t('connection_timeout')" prop="edge.connect_timeout">
        <el-input-number v-model="model.edge.connect_timeout" :min="1" :max="60" style="width: 100%" />
      </el-form-item>
      <el-form-item :label="t('receive_timeout')" prop="edge.receive_timeout">
        <el-input-number v-model="model.edge.receive_timeout" :min="1" :max="300" style="width: 100%" />
      </el-form-item>
    </template>

    <template v-if="model.provider === 'edge_offline'">
      <el-form-item :label="t('server_url_label')" prop="edge_offline.server_url">
        <el-input v-model="model.edge_offline.server_url" :placeholder="t('enter_server_url')" />
      </el-form-item>
      <el-form-item :label="t('timeout_label')" prop="edge_offline.timeout">
        <el-input-number v-model="model.edge_offline.timeout" :min="1" :max="300" style="width: 100%" />
      </el-form-item>
      <el-form-item :label="t('sample_rate')" prop="edge_offline.sample_rate">
        <el-input-number v-model="model.edge_offline.sample_rate" :min="8000" :max="48000" style="width: 100%" />
      </el-form-item>
      <el-form-item :label="t('channel_count_label')" prop="edge_offline.channels">
        <el-input-number v-model="model.edge_offline.channels" :min="1" :max="8" style="width: 100%" />
      </el-form-item>
      <el-form-item :label="t('frame_duration')" prop="edge_offline.frame_duration">
        <el-input-number v-model="model.edge_offline.frame_duration" :min="1" :max="100" style="width: 100%" />
      </el-form-item>
    </template>

    <template v-if="model.provider === 'aliyun_qwen'">
      <el-form-item label="API Key" prop="qwen_tts.api_key">
        <el-input v-model="model.qwen_tts.api_key" :placeholder="t('enter_api_key')" type="password" show-password />
      </el-form-item>
      <el-form-item :label="t('region')" prop="qwen_tts.region">
        <el-select v-model="model.qwen_tts.region" :placeholder="t('select_region')" style="width: 100%">
          <el-option :label="t('beijing')" value="beijing" />
          <el-option :label="t('singapore')" value="singapore" />
        </el-select>
      </el-form-item>
      <el-form-item :label="t('model')" prop="qwen_tts.model">
        <el-input v-model="model.qwen_tts.model" placeholder="qwen3-tts-flash" />
      </el-form-item>
      <el-form-item :label="t('voice_timbre')" prop="qwen_tts.voice">
        <el-input v-model="model.qwen_tts.voice" placeholder="Cherry" />
      </el-form-item>
      <el-form-item :label="t('language_type')" prop="qwen_tts.language_type">
        <el-select v-model="model.qwen_tts.language_type" :placeholder="t('select_language_type')" style="width: 100%">
          <el-option :label="t('auto')" value="Auto" />
          <el-option :label="t('chinese_zh')" value="Chinese" />
          <el-option :label="t('english_en')" value="English" />
        </el-select>
      </el-form-item>
      <el-form-item :label="t('use_streaming')" prop="qwen_tts.stream">
        <el-switch v-model="model.qwen_tts.stream" />
      </el-form-item>
      <el-form-item :label="t('frame_duration')" prop="qwen_tts.frame_duration">
        <el-input-number v-model="model.qwen_tts.frame_duration" :min="1" :max="1000" style="width: 100%" />
      </el-form-item>
    </template>

    <template v-if="model.provider === 'zhipu'">
      <el-form-item label="API Key" prop="zhipu.api_key">
        <el-input v-model="model.zhipu.api_key" :placeholder="t('enter_api_key')" type="password" show-password />
      </el-form-item>
      <el-form-item label="API URL" prop="zhipu.api_url">
        <el-input v-model="model.zhipu.api_url" placeholder="https://open.bigmodel.cn/api/paas/v4/audio/speech" />
      </el-form-item>
      <el-form-item :label="t('model')" prop="zhipu.model">
        <el-input v-model="model.zhipu.model" placeholder="glm-tts" />
      </el-form-item>
      <el-form-item :label="t('voice_timbre')" prop="zhipu.voice">
        <el-select
          v-model="model.zhipu.voice"
          :placeholder="t('select_timbre')"
          style="width: 100%"
          filterable
          :loading="voiceLoading"
          :disabled="voiceLoading"
        >
          <el-option v-for="option in voiceOptionsList" :key="option.value" :label="option.label" :value="option.value" />
        </el-select>
      </el-form-item>
      <el-form-item :label="t('response_format_label')" prop="zhipu.response_format">
        <el-select v-model="model.zhipu.response_format" :placeholder="t('select_response_format')" style="width: 100%">
          <el-option label="WAV" value="wav" />
          <el-option label="PCM" value="pcm" />
        </el-select>
      </el-form-item>
      <el-form-item :label="t('volume')" prop="zhipu.volume">
        <el-input-number v-model="model.zhipu.volume" :min="0" :max="10" :step="0.1" style="width: 100%" placeholder="0-10 (default 1.0)" />
      </el-form-item>
      <el-form-item :label="t('speech_rate')" prop="zhipu.speed">
        <el-input-number v-model="model.zhipu.speed" :min="0.5" :max="2.0" :step="0.1" style="width: 100%" placeholder="0.5-2.0 (default 1.0)" />
      </el-form-item>
      <el-form-item :label="t('use_streaming')" prop="zhipu.stream">
        <el-switch v-model="model.zhipu.stream" />
      </el-form-item>
      <el-form-item v-if="model.zhipu.stream" :label="t('encoding_format')" prop="zhipu.encode_format">
        <el-select v-model="model.zhipu.encode_format" :placeholder="t('select_encoding_format')" style="width: 100%">
          <el-option label="Base64" value="base64" />
          <el-option label="Hex" value="hex" />
        </el-select>
      </el-form-item>
      <el-form-item :label="t('frame_duration')" prop="zhipu.frame_duration">
        <el-input-number v-model="model.zhipu.frame_duration" :min="1" :max="1000" style="width: 100%" :placeholder="t('milliseconds')" />
      </el-form-item>
    </template>

    <template v-if="model.provider === 'minimax'">
      <el-form-item label="API Key" prop="minimax.api_key">
        <el-input v-model="model.minimax.api_key" :placeholder="t('enter_api_key')" type="password" show-password />
      </el-form-item>
      <el-form-item :label="t('model')" prop="minimax.model">
        <el-input v-model="model.minimax.model" placeholder="speech-2.8-hd" />
      </el-form-item>
      <el-form-item :label="t('voice_timbre')" prop="minimax.voice">
        <el-select
          v-model="model.minimax.voice"
          :placeholder="t('select_timbre')"
          style="width: 100%"
          filterable
          :loading="voiceLoading"
          :disabled="voiceLoading"
          allow-create
          default-first-option
        >
          <el-option v-for="option in voiceOptionsList" :key="option.value" :label="option.label" :value="option.value" />
        </el-select>
      </el-form-item>
      <el-form-item :label="t('speech_rate')" prop="minimax.speed">
        <el-input-number v-model="model.minimax.speed" :min="0.5" :max="2.0" :step="0.1" style="width: 100%" placeholder="0.5-2.0 (default 1.0)" />
      </el-form-item>
      <el-form-item :label="t('volume')" prop="minimax.vol">
        <el-input-number v-model="model.minimax.vol" :min="0" :max="2" :step="0.1" style="width: 100%" placeholder="0-2 (default 1.0)" />
      </el-form-item>
      <el-form-item :label="t('pitch')" prop="minimax.pitch">
        <el-input-number v-model="model.minimax.pitch" :min="-12" :max="12" :step="1" style="width: 100%" placeholder="-12 to 12 (default 0)" />
      </el-form-item>
      <el-form-item :label="t('sample_rate')" prop="minimax.sample_rate">
        <el-input-number v-model="model.minimax.sample_rate" :min="8000" :max="48000" :step="1000" style="width: 100%" placeholder="default 32000" />
      </el-form-item>
      <el-form-item :label="t('bitrate')" prop="minimax.bitrate">
        <el-input-number v-model="model.minimax.bitrate" :min="32000" :max="320000" :step="16000" style="width: 100%" placeholder="default 128000" />
      </el-form-item>
      <el-form-item :label="t('audio_format')" prop="minimax.format">
        <el-select v-model="model.minimax.format" :placeholder="t('select_audio_format')" style="width: 100%">
          <el-option label="MP3" value="mp3" />
          <el-option label="WAV" value="wav" />
          <el-option label="PCM" value="pcm" />
        </el-select>
      </el-form-item>
      <el-form-item :label="t('channel_count_label')" prop="minimax.channel">
        <el-input-number v-model="model.minimax.channel" :min="1" :max="2" style="width: 100%" placeholder="default 1" />
      </el-form-item>
    </template>

    <template v-if="model.provider === 'openai'">
      <el-form-item label="API Key" prop="openai.api_key">
        <el-input v-model="model.openai.api_key" :placeholder="t('enter_api_key')" type="password" show-password />
      </el-form-item>
      <el-form-item label="API URL" prop="openai.api_url">
        <el-input v-model="model.openai.api_url" placeholder="API URL (default: https://api.openai.com/v1/audio/speech)" />
      </el-form-item>
      <el-form-item :label="t('model')" prop="openai.model">
        <el-input v-model="model.openai.model" :placeholder="t('enter_model_default_tts1')" />
      </el-form-item>
      <el-form-item :label="t('voice_timbre')" prop="openai.voice">
        <el-select
          v-model="model.openai.voice"
          :placeholder="t('select_timbre')"
          style="width: 100%"
          filterable
          :loading="voiceLoading"
          :disabled="voiceLoading"
        >
          <el-option v-for="option in voiceOptionsList" :key="option.value" :label="option.label" :value="option.value" />
        </el-select>
      </el-form-item>
      <el-form-item :label="t('response_format_label')" prop="openai.response_format">
        <el-select v-model="model.openai.response_format" :placeholder="t('select_response_format')" style="width: 100%">
          <el-option label="MP3" value="mp3" />
          <el-option label="Opus" value="opus" />
          <el-option label="AAC" value="aac" />
          <el-option label="FLAC" value="flac" />
          <el-option label="WAV" value="wav" />
          <el-option label="PCM" value="pcm" />
        </el-select>
      </el-form-item>
      <el-form-item :label="t('speech_rate')" prop="openai.speed">
        <el-input-number v-model="model.openai.speed" :min="0.25" :max="4.0" :step="0.1" style="width: 100%" placeholder="0.25-4.0 (default 1.0)" />
      </el-form-item>
      <el-form-item :label="t('use_streaming')" prop="openai.stream">
        <el-switch v-model="model.openai.stream" />
      </el-form-item>
      <el-form-item :label="t('frame_duration')" prop="openai.frame_duration">
        <el-input-number v-model="model.openai.frame_duration" :min="1" :max="1000" style="width: 100%" :placeholder="t('milliseconds')" />
      </el-form-item>
    </template>

    <template v-if="model.provider === 'xunfei'">
      <XunfeiCommonConfig :model-value="model" prefix="xunfei" default-ws-url="wss://tts-api.xfyun.cn/v2/tts" />
      <el-form-item :label="t('voice_timbre')" prop="xunfei.voice">
        <el-input v-model="model.xunfei.voice" :placeholder="t('enter_voice_eg_xiaoyan')" />
      </el-form-item>
      <el-form-item :label="t('audio_encoding_label')" prop="xunfei.audio_encoding">
        <el-select v-model="model.xunfei.audio_encoding" :placeholder="t('select_audio_encoding')" style="width: 100%">
          <el-option label="RAW" value="raw" />
          <el-option label="Opus" value="opus" />
        </el-select>
      </el-form-item>
      <el-form-item :label="t('sample_rate')" prop="xunfei.sample_rate">
        <el-select v-model="model.xunfei.sample_rate" :placeholder="t('select_sample_rate')" style="width: 100%">
          <el-option label="8000" :value="8000" />
          <el-option label="16000" :value="16000" />
        </el-select>
      </el-form-item>
      <el-form-item :label="t('speech_rate')" prop="xunfei.speed">
        <el-input-number v-model="model.xunfei.speed" :min="0" :max="100" style="width: 100%" />
      </el-form-item>
      <el-form-item :label="t('volume')" prop="xunfei.volume">
        <el-input-number v-model="model.xunfei.volume" :min="0" :max="100" style="width: 100%" />
      </el-form-item>
      <el-form-item :label="t('pitch')" prop="xunfei.pitch">
        <el-input-number v-model="model.xunfei.pitch" :min="0" :max="100" style="width: 100%" />
      </el-form-item>
      <el-form-item :label="t('text_encoding')" prop="xunfei.tte">
        <el-select v-model="model.xunfei.tte" :placeholder="t('select_text_encoding')" style="width: 100%">
          <el-option label="UTF8" value="UTF8" />
          <el-option label="UNICODE" value="UNICODE" />
          <el-option label="GB2312" value="GB2312" />
        </el-select>
      </el-form-item>
      <el-form-item :label="t('digit_pronunciation')" prop="xunfei.reg">
        <el-input-number v-model="model.xunfei.reg" :min="0" :max="2" style="width: 100%" />
      </el-form-item>
      <el-form-item :label="t('digit_reading')" prop="xunfei.rdn">
        <el-input-number v-model="model.xunfei.rdn" :min="0" :max="2" style="width: 100%" />
      </el-form-item>
    </template>

    <template v-if="model.provider === 'xunfei_super_tts'">
      <XunfeiCommonConfig :model-value="model" prefix="xunfei_super_tts" default-ws-url="wss://cbm01.cn-huabei-1.xf-yun.com/v1/private/mcd9m97e6" />
      <el-form-item :label="t('voice_timbre')" prop="xunfei_super_tts.voice">
        <el-select
          v-model="model.xunfei_super_tts.voice"
          :placeholder="t('select_or_enter_voice_select')"
          style="width: 100%"
          filterable
          :loading="voiceLoading"
          :disabled="voiceLoading"
          allow-create
          default-first-option
        >
          <el-option v-for="option in voiceOptionsList" :key="option.value" :label="option.label" :value="option.value" />
        </el-select>
      </el-form-item>
      <el-form-item :label="t('audio_encoding_label')" prop="xunfei_super_tts.audio_encoding">
        <el-select v-model="model.xunfei_super_tts.audio_encoding" :placeholder="t('select_audio_encoding')" style="width: 100%">
          <el-option label="RAW" value="raw" />
          <el-option label="Opus" value="opus" />
        </el-select>
      </el-form-item>
      <el-form-item :label="t('sample_rate')" prop="xunfei_super_tts.sample_rate">
        <el-select v-model="model.xunfei_super_tts.sample_rate" :placeholder="t('select_sample_rate')" style="width: 100%">
          <el-option label="8000" :value="8000" />
          <el-option label="16000" :value="16000" />
          <el-option label="24000" :value="24000" />
        </el-select>
      </el-form-item>
      <el-form-item :label="t('speech_rate')" prop="xunfei_super_tts.speed">
        <el-input-number v-model="model.xunfei_super_tts.speed" :min="0" :max="100" style="width: 100%" />
      </el-form-item>
      <el-form-item :label="t('volume')" prop="xunfei_super_tts.volume">
        <el-input-number v-model="model.xunfei_super_tts.volume" :min="0" :max="100" style="width: 100%" />
      </el-form-item>
      <el-form-item :label="t('pitch')" prop="xunfei_super_tts.pitch">
        <el-input-number v-model="model.xunfei_super_tts.pitch" :min="0" :max="100" style="width: 100%" />
      </el-form-item>
      <el-form-item :label="t('background_sound')" prop="xunfei_super_tts.bgs">
        <el-input-number v-model="model.xunfei_super_tts.bgs" :min="0" :max="10" style="width: 100%" />
      </el-form-item>
      <el-form-item :label="t('digit_pronunciation')" prop="xunfei_super_tts.reg">
        <el-input-number v-model="model.xunfei_super_tts.reg" :min="0" :max="2" style="width: 100%" />
      </el-form-item>
      <el-form-item :label="t('digit_reading')" prop="xunfei_super_tts.rdn">
        <el-input-number v-model="model.xunfei_super_tts.rdn" :min="0" :max="2" style="width: 100%" />
      </el-form-item>
      <el-form-item :label="t('prosody_enhance')" prop="xunfei_super_tts.rhy">
        <el-input-number v-model="model.xunfei_super_tts.rhy" :min="0" :max="1" style="width: 100%" />
      </el-form-item>
      <el-form-item :label="t('oral_level')" prop="xunfei_super_tts.oral_level">
        <el-select v-model="model.xunfei_super_tts.oral_level" :placeholder="t('select_oral_level')" style="width: 100%">
          <el-option label="low" value="low" />
          <el-option label="mid" value="mid" />
          <el-option label="high" value="high" />
        </el-select>
      </el-form-item>
      <el-form-item label="Spark Assist" prop="xunfei_super_tts.spark_assist">
        <el-input-number v-model="model.xunfei_super_tts.spark_assist" :min="0" :max="1" style="width: 100%" />
      </el-form-item>
      <el-form-item :label="t('pause_split')" prop="xunfei_super_tts.stop_split">
        <el-input-number v-model="model.xunfei_super_tts.stop_split" :min="0" :max="1" style="width: 100%" />
      </el-form-item>
      <el-form-item :label="t('retain_oral')" prop="xunfei_super_tts.remain">
        <el-input-number v-model="model.xunfei_super_tts.remain" :min="0" :max="1" style="width: 100%" />
      </el-form-item>
      <el-form-item label-width="0">
        <div class="indextts-help">
          <div class="indextts-help-subtitle">{{ t('x4_series_hint') }}</div>
        </div>
      </el-form-item>
    </template>



    <template v-if="model.provider === 'indextts_vllm'">
      <el-form-item label-width="0" class="indextts-help-item">
        <div class="indextts-help">
          <div class="indextts-help-head">
            <div class="indextts-help-title">{{ t('indextts_guide_title') }}</div>
            <div class="indextts-help-subtitle">{{ t('indextts_guide_subtitle') }}</div>
          </div>
          <div class="indextts-help-links">
            <el-link :href="indexTTSDocURL" target="_blank" type="primary" :underline="false">{{ t('indextts_project_docs') }}</el-link>
            <span class="indextts-help-divider">|</span>
            <el-link :href="indexTTSReferenceURL" target="_blank" type="info" :underline="false">{{ t('indextts_reference_api') }}</el-link>
          </div>
          <div class="indextts-help-tags">
            <el-tag size="small" effect="plain" type="success">/audio/speech</el-tag>
            <el-tag size="small" effect="plain" type="warning">/audio/voices</el-tag>
            <el-tag size="small" effect="plain" type="info">/audio/clone</el-tag>
          </div>
        </div>
      </el-form-item>
      <el-form-item label="API URL" prop="indextts_vllm.api_url">
        <el-input v-model="model.indextts_vllm.api_url" placeholder="IndexTTS service URL (e.g. http://127.0.0.1:7860)" />
      </el-form-item>
      <el-form-item label="API Key" prop="indextts_vllm.api_key">
        <el-input v-model="model.indextts_vllm.api_key" :placeholder="t('optional_fill_as_needed')" type="password" show-password />
      </el-form-item>
      <el-form-item :label="t('model')" prop="indextts_vllm.model">
        <el-input v-model="model.indextts_vllm.model" placeholder="indextts-vllm" />
      </el-form-item>
      <el-form-item :label="t('voice_timbre')" prop="indextts_vllm.voice">
        <el-select
          v-model="model.indextts_vllm.voice"
          :placeholder="t('select_timbre')"
          style="width: 100%"
          filterable
          :loading="voiceLoading"
          :disabled="voiceLoading"
          allow-create
          default-first-option
          @visible-change="handleIndexTTSVoiceVisibleChange"
        >
          <el-option v-for="option in voiceOptionsList" :key="option.value" :label="option.label" :value="option.value" />
        </el-select>
      </el-form-item>
      <el-form-item :label="t('frame_duration')" prop="indextts_vllm.frame_duration">
        <el-input-number v-model="model.indextts_vllm.frame_duration" :min="1" :max="1000" style="width: 100%" />
      </el-form-item>
    </template>

    <template v-if="model.provider === 'cosyvoice'">
      <el-form-item label="API URL" prop="cosyvoice.api_url">
        <el-input v-model="model.cosyvoice.api_url" :placeholder="t('enter_api_url')" />
      </el-form-item>
      <el-form-item :label="t('speaker_id')" prop="cosyvoice.spk_id">
        <el-input v-model="model.cosyvoice.spk_id" :placeholder="t('enter_speaker_id')" />
      </el-form-item>
      <el-form-item :label="t('frame_duration')" prop="cosyvoice.frame_duration">
        <el-input-number v-model="model.cosyvoice.frame_duration" :min="1" :max="1000" style="width: 100%" />
      </el-form-item>
      <el-form-item :label="t('target_sample_rate')" prop="cosyvoice.target_sr">
        <el-input-number v-model="model.cosyvoice.target_sr" :min="8000" :max="48000" style="width: 100%" />
      </el-form-item>
      <el-form-item :label="t('audio_format')" prop="cosyvoice.audio_format">
        <el-select v-model="model.cosyvoice.audio_format" :placeholder="t('select_audio_format')" style="width: 100%">
          <el-option label="MP3" value="mp3" />
          <el-option label="WAV" value="wav" />
          <el-option label="PCM" value="pcm" />
        </el-select>
      </el-form-item>
      <el-form-item :label="t('instruct_text')" prop="cosyvoice.instruct_text">
        <el-input v-model="model.cosyvoice.instruct_text" :placeholder="t('enter_instruct_text_opt')" />
      </el-form-item>
    </template>
  </el-form>
</template>

<script setup>
import { ref, computed } from 'vue'
import { getTTSProviderOptions } from './ttsProviderOptions'
import XunfeiCommonConfig from './XunfeiCommonConfig.vue'
import { useLocale } from '../../../composables/useLocale'
const { t } = useLocale()

const TTS_PROVIDER_OPTIONS = computed(() => getTTSProviderOptions(t))

const DOUBAO_MODEL_OPTIONS = [
  { label: t('doubao_tts_v11'), value: 'seed-tts-1.1' },
  { label: t('doubao_tts_v2_standard'), value: 'seed-tts-2.0-standard' },
  { label: t('doubao_tts_v2_expressive'), value: 'seed-tts-2.0-expressive' },
  { label: t('doubao_clone_v1'), value: 'seed-icl-1.0' },
  { label: t('doubao_clone_v2_standard'), value: 'seed-icl-2.0-standard' },
  { label: t('doubao_clone_v2_expressive'), value: 'seed-icl-2.0-expressive' }
]

const props = defineProps({
  model: { type: Object, required: true },
  rules: { type: Object, default: () => ({}) },
  voiceOptions: { type: Array, default: () => [] },
  voiceLoading: { type: Boolean, default: false }
})
const emit = defineEmits(['request-voice-options'])

const formRef = ref()
// 保证音色选项始终为数组且响应式，供下拉使用
const voiceOptionsList = computed(() => Array.isArray(props.voiceOptions) ? props.voiceOptions : [])
const indexTTSDocURL = 'https://github.com/hackers365/xiaozhi-esp32-server-golang/blob/main/doc/indextts_vllm_api.md'
const indexTTSReferenceURL = 'https://github.com/hackers365/index-tts-vllm/blob/master/api_server.py'

function handleIndexTTSVoiceVisibleChange(visible) {
  if (visible) {
    emit('request-voice-options', 'indextts_vllm')
  }
}

function getJsonData() {
  const form = props.model
  const config = {}
  switch (form.provider) {
    case 'cosyvoice':
      config.api_url = form.cosyvoice?.api_url
      config.spk_id = form.cosyvoice?.spk_id
      config.frame_duration = form.cosyvoice?.frame_duration
      config.target_sr = form.cosyvoice?.target_sr
      config.audio_format = form.cosyvoice?.audio_format
      config.instruct_text = form.cosyvoice?.instruct_text
      break
    case 'doubao_ws':
      config.appid = form.doubao_ws?.appid
      config.access_token = form.doubao_ws?.access_token
      config.model = form.doubao_ws?.model || 'seed-tts-2.0-standard'
      config.resource_id = form.doubao_ws?.resource_id
      config.voice = form.doubao_ws?.voice
      config.ws_url = form.doubao_ws?.ws_url || 'wss://openspeech.bytedance.com/api/v3/tts/unidirectional/stream'
      break
    case 'edge':
      config.voice = form.edge?.voice
      config.rate = form.edge?.rate
      config.volume = form.edge?.volume
      config.pitch = form.edge?.pitch
      config.connect_timeout = form.edge?.connect_timeout
      config.receive_timeout = form.edge?.receive_timeout
      break
    case 'edge_offline':
      config.server_url = form.edge_offline?.server_url
      config.timeout = form.edge_offline?.timeout
      config.sample_rate = form.edge_offline?.sample_rate
      config.channels = form.edge_offline?.channels
      config.frame_duration = form.edge_offline?.frame_duration
      break
    case 'aliyun_qwen':
      config.provider = 'aliyun_qwen'
      config.api_key = form.qwen_tts?.api_key
      config.api_url = form.qwen_tts?.api_url
      config.region = form.qwen_tts?.region
      config.model = form.qwen_tts?.model || 'qwen3-tts-flash'
      config.voice = form.qwen_tts?.voice || 'Cherry'
      config.language_type = form.qwen_tts?.language_type || 'Chinese'
      config.stream = form.qwen_tts?.stream
      config.frame_duration = form.qwen_tts?.frame_duration || 60
      break
    case 'openai':
      config.api_key = form.openai?.api_key
      config.api_url = form.openai?.api_url
      config.model = form.openai?.model
      config.voice = form.openai?.voice
      config.response_format = form.openai?.response_format
      config.speed = form.openai?.speed
      config.stream = form.openai?.stream
      config.frame_duration = form.openai?.frame_duration
      break
    case 'xunfei':
      config.provider = 'xunfei'
      config.app_id = form.xunfei?.app_id
      config.api_key = form.xunfei?.api_key
      config.api_secret = form.xunfei?.api_secret
      config.ws_url = form.xunfei?.ws_url
      config.voice = form.xunfei?.voice
      config.audio_encoding = form.xunfei?.audio_encoding || 'raw'
      config.sample_rate = form.xunfei?.sample_rate || 16000
      config.speed = form.xunfei?.speed ?? 50
      config.volume = form.xunfei?.volume ?? 50
      config.pitch = form.xunfei?.pitch ?? 50
      config.tte = form.xunfei?.tte || 'UTF8'
      config.reg = form.xunfei?.reg ?? 0
      config.rdn = form.xunfei?.rdn ?? 0
      config.frame_duration = form.xunfei?.frame_duration || 60
      config.connect_timeout = form.xunfei?.connect_timeout || 10
      config.read_timeout = form.xunfei?.read_timeout || 30
      break
    case 'xunfei_super_tts':
      config.provider = 'xunfei_super_tts'
      config.double_stream = true
      config.app_id = form.xunfei_super_tts?.app_id
      config.api_key = form.xunfei_super_tts?.api_key
      config.api_secret = form.xunfei_super_tts?.api_secret
      config.ws_url = form.xunfei_super_tts?.ws_url
      config.voice = form.xunfei_super_tts?.voice
      config.audio_encoding = form.xunfei_super_tts?.audio_encoding || 'raw'
      config.sample_rate = form.xunfei_super_tts?.sample_rate || 24000
      config.speed = form.xunfei_super_tts?.speed ?? 50
      config.volume = form.xunfei_super_tts?.volume ?? 50
      config.pitch = form.xunfei_super_tts?.pitch ?? 50
      config.bgs = form.xunfei_super_tts?.bgs ?? 0
      config.reg = form.xunfei_super_tts?.reg ?? 0
      config.rdn = form.xunfei_super_tts?.rdn ?? 0
      config.rhy = form.xunfei_super_tts?.rhy ?? 0
      config.oral_level = form.xunfei_super_tts?.oral_level || 'mid'
      config.spark_assist = form.xunfei_super_tts?.spark_assist ?? 1
      config.stop_split = form.xunfei_super_tts?.stop_split ?? 0
      config.remain = form.xunfei_super_tts?.remain ?? 0
      config.frame_duration = form.xunfei_super_tts?.frame_duration || 60
      config.connect_timeout = form.xunfei_super_tts?.connect_timeout || 10
      config.read_timeout = form.xunfei_super_tts?.read_timeout || 30
      break
    case 'indextts_vllm':
      config.provider = 'indextts_vllm'
      config.api_url = form.indextts_vllm?.api_url
      config.api_key = form.indextts_vllm?.api_key
      config.model = form.indextts_vllm?.model || 'indextts-vllm'
      config.voice = form.indextts_vllm?.voice
      config.response_format = 'wav'
      config.stream = false
      config.frame_duration = form.indextts_vllm?.frame_duration || 60
      break
    case 'zhipu':
      config.provider = 'zhipu'
      config.api_key = form.zhipu?.api_key
      config.api_url = form.zhipu?.api_url || 'https://open.bigmodel.cn/api/paas/v4/audio/speech'
      config.model = form.zhipu?.model || 'glm-tts'
      config.voice = form.zhipu?.voice
      config.response_format = form.zhipu?.response_format
      config.speed = form.zhipu?.speed
      config.volume = form.zhipu?.volume || 1.0
      config.stream = form.zhipu?.stream
      config.encode_format = form.zhipu?.encode_format || 'base64'
      config.frame_duration = form.zhipu?.frame_duration
      break
    case 'minimax':
      config.provider = 'minimax'
      config.api_key = form.minimax?.api_key
      config.model = form.minimax?.model || 'speech-2.8-hd'
      config.voice = form.minimax?.voice || 'male-qn-qingse'
      config.speed = form.minimax?.speed || 1.0
      config.vol = form.minimax?.vol || 1.0
      config.pitch = form.minimax?.pitch || 0
      config.sample_rate = form.minimax?.sample_rate || 32000
      config.bitrate = form.minimax?.bitrate || 128000
      config.format = form.minimax?.format || 'mp3'
      config.channel = form.minimax?.channel || 1
      break
  }
  return JSON.stringify(config)
}

function validate(callback) {
  return formRef.value?.validate(callback)
}

function resetFields() {
  formRef.value?.resetFields()
}

defineExpose({ validate, getJsonData, resetFields })
</script>

<style scoped>
.indextts-help-item :deep(.el-form-item__content) {
  margin-left: 0 !important;
}

.indextts-help {
  width: 100%;
  border: 1px solid rgba(255, 255, 255, 0.92);
  border-radius: 18px;
  padding: 12px 14px;
  background: rgba(248, 250, 252, 0.9);
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.7);
}

.indextts-help-head {
  margin-bottom: 8px;
}

.indextts-help-title {
  font-size: 14px;
  font-weight: 700;
  color: var(--apple-text);
}

.indextts-help-subtitle {
  margin-top: 4px;
  font-size: 12px;
  color: var(--apple-text-secondary);
}

.indextts-help-links {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 10px;
  flex-wrap: wrap;
}

.indextts-help-divider {
  color: var(--apple-text-tertiary);
  font-size: 12px;
}

.indextts-help-tags {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}

.form-item-tip {
  margin-left: 8px;
  font-size: 12px;
  color: var(--apple-text-secondary);
}
</style>
