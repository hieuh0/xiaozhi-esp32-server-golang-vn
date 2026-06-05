<template>
  <header class="app-header-shell">
    <div class="app-header-card apple-surface">
      <div class="app-header-copy">
        <p class="app-header-eyebrow">{{ props.eyebrow }}</p>
        <h1 class="app-header-title">{{ props.title }}</h1>
        <p v-if="props.showSubtitle && props.subtitle" class="app-header-subtitle">
          {{ props.subtitle }}
        </p>
      </div>

      <div class="app-header-actions">
        <slot name="actions" />

        <template v-if="props.showAdminShortcuts">
          <router-link to="/admin/config-wizard" custom v-slot="{ navigate, isActive }">
            <el-button
              class="header-nav-btn"
              :class="{ 'is-active': isActive }"
              plain
              @click="navigate"
            >
              <el-icon><Guide /></el-icon>
              <span>{{ t('config_wizard') }}</span>
            </el-button>
          </router-link>
          <router-link to="/admin/ota-config" custom v-slot="{ navigate, isActive }">
            <el-button
              class="header-nav-btn"
              :class="{ 'is-active': isActive }"
              plain
              @click="navigate"
            >
              <el-icon><Upload /></el-icon>
              <span>{{ t('ota_config') }}</span>
            </el-button>
          </router-link>
        </template>

        <!-- Language switcher -->
        <el-dropdown @command="setLang" trigger="click">
          <el-button text class="lang-btn">
            <span>{{ langLabel }}</span>
            <el-icon><ArrowDown /></el-icon>
          </el-button>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item command="vi" :class="{ 'is-active': lang === 'vi' }">🇻🇳 Tiếng Việt</el-dropdown-item>
              <el-dropdown-item command="en" :class="{ 'is-active': lang === 'en' }">🇬🇧 English</el-dropdown-item>
              <el-dropdown-item command="zh" :class="{ 'is-active': lang === 'zh' }">🇨🇳 Chinese</el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>

        <el-dropdown @command="emit('command', $event)">
          <button class="profile-button" type="button">
            <span class="profile-avatar">{{ props.initial }}</span>
            <span class="profile-copy">
              <strong>{{ props.username }}</strong>
              <small>{{ props.roleLabel }}</small>
            </span>
            <el-icon><ArrowDown /></el-icon>
          </button>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item v-if="!props.isAdmin" command="api-tokens">API Token</el-dropdown-item>
              <el-dropdown-item command="logout">{{ t('logout') }}</el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
      </div>
    </div>
  </header>
</template>

<script setup>
import { computed } from 'vue'
import { ArrowDown, Guide, Upload } from '@element-plus/icons-vue'
import { useLocale } from '../composables/useLocale'

const { t, lang, setLang } = useLocale()

const langLabel = computed(() => ({ vi: '🇻🇳 VI', en: '🇬🇧 EN', zh: '🇨🇳 ZH' })[lang.value] ?? 'VI')

const props = defineProps({
  title: {
    type: String,
    default: ''
  },
  eyebrow: {
    type: String,
    default: ''
  },
  subtitle: {
    type: String,
    default: ''
  },
  showSubtitle: {
    type: Boolean,
    default: false
  },
  username: {
    type: String,
    default: ''
  },
  roleLabel: {
    type: String,
    default: ''
  },
  initial: {
    type: String,
    default: 'U'
  },
  isAdmin: {
    type: Boolean,
    default: false
  },
  showAdminShortcuts: {
    type: Boolean,
    default: false
  }
})

const emit = defineEmits(['command'])
</script>

<style scoped>
.app-header-shell {
  padding: 0;
  height: auto;
}

.app-header-card {
  min-height: 78px;
  padding: 16px 20px;
  border-radius: 24px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 20px;
}

.app-header-copy {
  min-width: 0;
}

.app-header-eyebrow {
  margin: 0 0 4px;
  color: var(--apple-primary);
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.app-header-title {
  margin: 0;
  color: var(--apple-text);
  font-size: 26px;
  line-height: 1.08;
  letter-spacing: -0.04em;
}

.app-header-subtitle {
  margin: 6px 0 0;
  color: var(--apple-text-secondary);
  font-size: 13px;
  line-height: 1.55;
}

.app-header-actions {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  flex-wrap: nowrap;
  gap: 10px;
  flex: none;
}

.header-nav-btn {
  min-height: 40px;
  padding-inline: 14px;
  border-radius: 16px;
}

.header-nav-btn.is-active {
  border-color: transparent;
  background: rgba(0, 122, 255, 0.1);
  color: var(--apple-primary);
}

.profile-button {
  min-height: 44px;
  padding: 7px 12px 7px 7px;
  border-radius: 999px;
  border: 1px solid var(--apple-line);
  background: rgba(255, 255, 255, 0.86);
  box-shadow: var(--apple-shadow-sm);
  display: inline-flex;
  align-items: center;
  gap: 10px;
  cursor: pointer;
  color: var(--apple-text);
  transition: transform 0.2s ease, box-shadow 0.2s ease;
}

.profile-button:hover {
  transform: translateY(-1px);
  box-shadow: var(--apple-shadow-md);
}

.profile-avatar {
  width: 32px;
  height: 32px;
  border-radius: 50%;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(180deg, #eff6ff 0%, #dcebff 100%);
  color: var(--apple-primary);
  font-size: 13px;
  font-weight: 700;
  flex: none;
}

.profile-copy {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  line-height: 1.2;
}

.profile-copy strong {
  font-size: 13px;
}

.profile-copy small {
  color: var(--apple-text-secondary);
  font-size: 11px;
}

.lang-btn {
  padding: 6px 10px;
  border-radius: 12px;
  font-size: 13px;
  gap: 4px;
}

@media (max-width: 960px) {
  .app-header-card {
    align-items: flex-start;
    flex-direction: column;
  }

  .app-header-actions {
    width: 100%;
    justify-content: flex-start;
    flex-wrap: wrap;
  }
}
</style>
