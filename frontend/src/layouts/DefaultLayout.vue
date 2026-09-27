<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink, RouterView, useRouter } from 'vue-router'
import { NAvatar, NDropdown, useDialog, useMessage } from 'naive-ui'
import ThemeToggle from '@/components/ThemeToggle.vue'
import { useUserStore } from '@/stores/user'

const router = useRouter()
const userStore = useUserStore()
const message = useMessage()
const dialog = useDialog()

const navItems = [
  { name: 'dashboard', label: '概览' },
  { name: 'notes', label: '笔记' },
  { name: 'review', label: '复习' },
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

        <div class="app-actions">
          <ThemeToggle />

          <n-dropdown :options="userOptions" trigger="click" @select="handleUserSelect">
            <button class="app-user" type="button">
              <n-avatar round :size="28" :src="userStore.user?.avatar_url ?? undefined">
                {{ initial }}
              </n-avatar>
              <span class="app-user__name">{{ userStore.user?.username }}</span>
            </button>
          </n-dropdown>
        </div>
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
  background: var(--ln-header-bg);
  border-bottom: 1px solid var(--ln-border);
  /* 滚动时内容从顶栏底下透出来，比纯色顶栏更有层次 */
  backdrop-filter: saturate(160%) blur(10px);
  -webkit-backdrop-filter: saturate(160%) blur(10px);
}

.app-header__inner {
  display: flex;
  gap: 28px;
  align-items: center;
  max-width: 1080px;
  height: 56px;
  margin: 0 auto;
  padding: 0 24px;
}

.app-brand {
  font-size: 17px;
  font-weight: 700;
  color: var(--ln-text);
  letter-spacing: -0.02em;
}

.app-brand:hover {
  text-decoration: none;
}

.app-brand__accent {
  color: var(--ln-primary-text);
}

.app-nav {
  display: flex;
  flex: 1;
  gap: 2px;
  min-width: 0;
  overflow-x: auto;
  /* 导航项少，不会真的溢出；这里只是防止极窄屏把布局撑破 */
  scrollbar-width: none;
}

.app-nav::-webkit-scrollbar {
  display: none;
}

.app-nav__link {
  flex-shrink: 0;
  padding: 6px 12px;
  font-size: 14px;
  color: var(--ln-text-muted);
  white-space: nowrap;
  border-radius: var(--ln-radius-sm);
}

.app-nav__link:hover {
  color: var(--ln-text);
  background: var(--ln-surface-active);
  text-decoration: none;
}

.app-nav__link--active {
  font-weight: 600;
  color: var(--ln-primary-text);
  background: var(--ln-primary-soft);
}

.app-actions {
  display: flex;
  gap: 6px;
  align-items: center;
  flex-shrink: 0;
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
  background: var(--ln-surface-active);
}

.app-user__name {
  font-size: 14px;
}

/* 窄屏下收起用户名，只留头像 —— 顶栏不至于挤成一团 */
@media (max-width: 720px) {
  .app-header__inner {
    gap: 14px;
    padding: 0 16px;
  }

  .app-user__name {
    display: none;
  }
}

.app-main {
  flex: 1;
}
</style>
