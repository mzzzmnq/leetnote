<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { NButton, NSpin } from 'naive-ui'
import { useUserStore } from '@/stores/user'

const route = useRoute()
const router = useRouter()
const userStore = useUserStore()

const failed = ref(false)
const errorMessage = ref('')

const ERROR_MESSAGES: Record<string, string> = {
  state_invalid: '安全校验失败，请重新发起登录',
  github_denied: '你在 GitHub 上取消了授权',
  code_missing: 'GitHub 未返回授权码，请重试',
  exchange_failed: '与 GitHub 通信失败，请稍后重试',
  login_failed: '登录失败，请稍后重试',
}

onMounted(async () => {
  // GitHub 侧出错时，后端会带上 status=error 跳回来
  if (route.query.status === 'error') {
    failed.value = true
    const code = typeof route.query.error === 'string' ? route.query.error : ''
    errorMessage.value = ERROR_MESSAGES[code] ?? '登录失败，请稍后重试'
    return
  }

  // 后端已经下发 refresh Cookie，这里用它换 access_token。
  // 【注意】token 没有出现在 URL 里——这正是设计意图。
  const ok = await userStore.restore()
  if (!ok) {
    failed.value = true
    errorMessage.value = '登录状态获取失败，请重试'
    return
  }

  const raw = route.query.redirect
  const target = typeof raw === 'string' && raw.startsWith('/') && !raw.startsWith('//') ? raw : '/'
  await router.replace(target)
})
</script>

<template>
  <div class="auth-page">
    <div class="auth-card" style="text-align: center">
      <template v-if="!failed">
        <n-spin size="large" />
        <p style="margin-top: 16px; color: var(--ln-text-muted)">正在完成登录…</p>
      </template>

      <template v-else>
        <h1 class="auth-card__title">登录失败</h1>
        <p class="auth-card__subtitle">{{ errorMessage }}</p>
        <n-button type="primary" block @click="router.push({ name: 'login' })">
          返回登录
        </n-button>
      </template>
    </div>
  </div>
</template>
