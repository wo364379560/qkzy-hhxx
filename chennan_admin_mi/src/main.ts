import '@fontsource-variable/inter/index.css'
import { createApp } from 'vue'
import { createPinia } from 'pinia'
import ElementPlus from 'element-plus'
import 'element-plus/dist/index.css'
import zhCn from 'element-plus/es/locale/lang/zh-cn'
import type { Language } from 'element-plus/es/locale'
import {
  Avatar, Check, ChatDotRound, ChatSquare, DataBoard, Download, List,
  Plus, Promotion, Search, Setting, Ticket, User, UserFilled, Warning,
} from '@element-plus/icons-vue'
import App from './App.vue'
import router from './router'

const app = createApp(App)

const icons = {
  Avatar, Check, ChatDotRound, ChatSquare, DataBoard, Download, List,
  Plus, Promotion, Search, Setting, Ticket, User, UserFilled, Warning,
}
for (const [name, icon] of Object.entries(icons)) {
  app.component(name, icon)
}

app.use(createPinia())
app.use(router)
app.use(ElementPlus, { locale: zhCn as unknown as Language })
app.mount('#app')
