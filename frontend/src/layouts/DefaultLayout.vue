<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink, RouterView, useRouter } from 'vue-router'
import { NAvatar, NDropdown, useDialog, useMessage } from 'naive-ui'
import { useUserStore } from '@/stores/user'

const router = useRouter()
const userStore = useUserStore()
const message = useMessage()
const dialog = useDialog()

const navItems = [
  { name: 'dashboard', label: '概览' },
  { name: 'notes', label: '笔记' },
  { name: 'problems', label: '题目库' },
  { name: 'search', label: '搜索' },
  { name: 'settings', label: '设置' },
] as const

const initial = computed(() => userStore.user?.username.charAt(0).toUpperCase() ?? '?')

const userOptions = [
  { label: '账号设置', key: 'settings' },
  { type: 'divider', key: 'd1' },
  { label: '退出登录', key: 'logout' },
]

function handleUserSelect(key: string): void {
  if (key === 'settings') {
    void router.push({ name: 'settings' })
    return
  }

  if (key === 'logout') {
    dialog.warning({
      title: '退出登录',
      content: '确定要退出当前账号吗？',
      positiveText: '退出',
      negativeText: '取消',
      onPositiveClick: async () => {
        await userStore.logout()
        message.success('已退出登录')
        await router.push({ name: 'login' })
      },
    })
  }
}
</script>

<template>
  <div class="app-shell">
    <header class="app-header">
      <div class="app-header__inner">
        <RouterLink :to="{ name: 'dashboard' }" class="app-brand">
          Leet<span class="app-brand__accent">Note</span>
        </RouterLink>

        <nav class="app-nav">
          <RouterLink
            v-for="item in navItems"
            :key="item.name"
            :to="{ name: item.name }"
            class="app-nav__link"
            active-class="app-nav__link--active"
          >
            {{ item.label }}
          </RouterLink>
        </nav>

        <n-dropdown :options="userOptions" trigger="click" @select="handleUserSelect">
          <button class="app-user" type="button">
            <n-avatar round :size="28" :src="userStore.user?.avatar_url ?? undefined">
              {{ initial }}
            </n-avatar>
            <span class="app-user__name">{{ userStore.user?.username }}</span>
          </button>
        </n-dropdown>
      </div>
    </header>

    <main class="app-main">
      <RouterView />
    </main>
  </div>
</template>

<style scoped>
.app-shell {
  display: flex;
  flex-direction: column;
  min-height: 100vh;
}

.app-header {
  position: sticky;
  top: 0;
  z-index: 10;
  background: #fff;
  border-bottom: 1px solid var(--ln-border);
}

.app-header__inner {
  display: flex;
  gap: 32px;
  align-items: center;
  max-width: 1080px;
  height: 56px;
  margin: 0 auto;
  padding: 0 24px;
}

.app-brand {
  font-size: 18px;
  font-weight: 700;
  color: var(--ln-text);
  letter-spacing: -0.02em;
}

.app-brand:hover {
  text-decoration: none;
}

.app-brand__accent {
  color: var(--ln-primary);
}

.app-nav {
  display: flex;
  flex: 1;
  gap: 4px;
}

.app-nav__link {
  padding: 6px 12px;
  font-size: 14px;
  color: var(--ln-text-muted);
  border-radius: 6px;
}

.app-nav__link:hover {
  color: var(--ln-text);
  background: var(--ln-bg);
  text-decoration: none;
}

.app-nav__link--active {
  font-weight: 600;
  color: var(--ln-primary);
  background: rgb(47 111 237 / 8%);
}

.app-user {
  display: flex;
  gap: 8px;
  align-items: center;
  padding: 4px 8px;
  font: inherit;
  color: var(--ln-text);
  cursor: pointer;
  background: none;
  border: 0;
  border-radius: 8px;
}

.app-user:hover {
  background: var(--ln-bg);
}

.app-user__name {
  font-size: 14px;
}

.app-main {
  flex: 1;
}
</style>
