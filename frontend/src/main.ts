import { createApp } from 'vue'
import { createPinia } from 'pinia'
import App from './App.vue'
import router from './router'
import { setUnauthorizedHandler } from './api/client'
import { useThemeStore } from './stores/theme'
import './styles/main.css'

const app = createApp(App)

// 注意顺序：Pinia 必须先于 router 安装，
// 因为路由守卫里会用到 store。
const pinia = createPinia()
app.use(pinia)
app.use(router)

// 主题要在挂载前初始化：读 localStorage、应用 <html> 的类名、监听系统配色。
// 首屏那一次是 index.html 里的内联脚本做的（避免闪白），这里只做后续同步。
useThemeStore(pinia).init()

// 刷新 token 也失败时（refresh token 过期/被撤销），
// 说明会话彻底结束，统一跳登录页。
setUnauthorizedHandler(() => {
  const current = router.currentRoute.value
  if (current.name === 'login' || current.name === 'oauth-callback') {
    return
  }
  void router.push({ name: 'login', query: { redirect: current.fullPath } })
})

app.mount('#app')
