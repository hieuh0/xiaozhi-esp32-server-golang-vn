import { createApp } from 'vue'
import { createPinia } from 'pinia'
import App from './App.vue'
import router from './router'
import { setRouter } from './utils/api'
import { useThemeStore } from './stores/theme'
import './styles/globals.css'

const app = createApp(App)

app.use(createPinia())
app.use(router)

setRouter(router)

app.mount('#app')

useThemeStore().init()
