import { createApp } from 'vue'
import { createPinia } from 'pinia'
import piniaPersist from 'pinia-plugin-persistedstate'
import ElementPlus from 'element-plus'
import 'element-plus/dist/index.css'
import 'element-plus/theme-chalk/dark/css-vars.css'
import zhCn from 'element-plus/es/locale/lang/zh-cn'
import * as ElementIcons from '@element-plus/icons-vue'

import App from './App.vue'
import router from './router'
import './styles/global.scss'
import { useSiteStore } from './stores/site'

const app = createApp(App)

const pinia = createPinia()
pinia.use(piniaPersist)
app.use(pinia)
app.use(router)
app.use(ElementPlus, { size: 'default', locale: zhCn })

for (const [name, comp] of Object.entries(ElementIcons)) {
  app.component(name, comp as never)
}

useSiteStore(pinia).refresh()

app.mount('#app')