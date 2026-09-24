import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'
import { useUserStore } from '@/stores/user'

// 给路由 meta 加类型，避免在守卫里到处 as 断言
declare module 'vue-router' {
  interface RouteMeta {
    /** 需要登录才能访问 */
    requiresAuth?: boolean
    /** 仅未登录时可访问（登录/注册页） */
    guestOnly?: boolean
    /** 页面标题 */
    title?: string
  }
}

const routes: RouteRecordRaw[] = [
  {
    path: '/login',
    name: 'login',
    component: () => import('@/views/LoginView.vue'),
    meta: { guestOnly: true, title: '登录' },
  },
  {
    path: '/register',
    name: 'register',
    component: () => import('@/views/RegisterView.vue'),
    meta: { guestOnly: true, title: '注册' },
  },
  {
    // OAuth 回调页刻意【不套主布局】——它只是中转，不该闪出导航栏
    path: '/auth/callback',
    name: 'oauth-callback',
    component: () => import('@/views/OAuthCallbackView.vue'),
    meta: { title: '登录中' },
  },
  {
    path: '/',
    component: () => import('@/layouts/DefaultLayout.vue'),
    children: [
      {
        path: '',
        name: 'dashboard',
        component: () => import('@/views/DashboardView.vue'),
        meta: { requiresAuth: true, title: '概览' },
      },
      {
        path: 'notes',
        name: 'notes',
        component: () => import('@/views/NotesView.vue'),
        meta: { requiresAuth: true, title: '笔记' },
      },
      {
        path: 'settings',
        name: 'settings',
        component: () => import('@/views/SettingsView.vue'),
        meta: { requiresAuth: true, title: '设置' },
      },
    ],
  },
  {
    path: '/:pathMatch(.*)*',
    name: 'not-found',
    component: () => import('@/views/NotFoundView.vue'),
    meta: { title: '页面不存在' },
  },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
  scrollBehavior: () => ({ top: 0 }),
})

router.beforeEach(async (to) => {
  const userStore = useUserStore()

  // 首次导航时尝试用 refresh Cookie 恢复会话。
  // 这一步让「刷新页面后仍是登录态」成立。
  await userStore.ensureInitialized()

  if (to.meta.requiresAuth === true && !userStore.isLoggedIn) {
    // 记住原本要去哪，登录后跳回去
    return { name: 'login', query: { redirect: to.fullPath } }
  }

  if (to.meta.guestOnly === true && userStore.isLoggedIn) {
    return { name: 'dashboard' }
  }

  return true
})

router.afterEach((to) => {
  document.title = to.meta.title ? `${to.meta.title} · LeetNote` : 'LeetNote · 算法练习笔记'
})

export default router
