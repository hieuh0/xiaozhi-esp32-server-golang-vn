import { createApp } from 'vue'
import { createPinia } from 'pinia'
import ElementPlus from 'element-plus'
import 'element-plus/dist/index.css'
// Vant 4 on-demand imports to reduce bundle size
import { 
  NavBar, 
  Tabbar, 
  TabbarItem, 
  Form, 
  Field, 
  CellGroup, 
  Button, 
  Tabs, 
  Tab, 
  Cell,
  Popup,
  Icon
} from 'vant'
import 'vant/lib/index.css'
import * as ElementPlusIconsVue from '@element-plus/icons-vue'
import App from './App.vue'
import router from './router'
import './styles/apple-light.css'

const app = createApp(App)

// Register all Element Plus icons
for (const [key, component] of Object.entries(ElementPlusIconsVue)) {
  app.component(key, component)
}

// Register Vant components (on-demand)
app.use(NavBar)
app.use(Tabbar)
app.use(TabbarItem)
app.use(Form)
app.use(Field)
app.use(CellGroup)
app.use(Button)
app.use(Tabs)
app.use(Tab)
app.use(Cell)
app.use(Popup)
app.use(Icon)

app.use(createPinia())
app.use(router)
app.use(ElementPlus)  // Desktop use

app.mount('#app')
