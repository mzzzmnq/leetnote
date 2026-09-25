import { createApp } from 'vue'
import { createPinia } from 'pinia'
import App from './App.vue'
import router from './router'
import { setUnauthorizedHandler } from './api/client'
import './styles/main.css'
// highlight.js 的代码配色（github 亮色主题）
import 'highlight.js/styles/github.css'

const app = createApp(App)

// 注意顺序：Pinia 必须先于 router 安装，
// 因为路由守卫里会用到 store。
app.use(createPinia())
app.use(router)

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
