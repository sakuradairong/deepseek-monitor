<template>
  <!-- Login Page -->
  <div v-if="currentView === 'login'" class="login-page">
    <el-card class="login-card" shadow="always">
      <template #header>
        <div class="login-header">
          <el-icon :size="28"><Monitor /></el-icon>
          <h2>DeepSeek API Monitor</h2>
        </div>
      </template>

      <el-form @submit.prevent="handleLogin" label-position="top">
        <el-form-item label="用户名">
          <el-input v-model="loginForm.username" placeholder="输入用户名" :prefix-icon="User" />
        </el-form-item>
        <el-form-item label="密码">
          <el-input v-model="loginForm.password" type="password" placeholder="输入密码" :prefix-icon="Lock" show-password />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" native-type="submit" :loading="loginLoading" style="width:100%">
            {{ isFirstUser ? '创建管理员账号' : '登录' }}
          </el-button>
        </el-form-item>
        <el-alert v-if="loginError" :title="loginError" type="error" show-icon :closable="false" />
      </el-form>
    </el-card>
  </div>

  <!-- Main App Layout -->
  <div v-else class="app-layout">
    <el-container style="min-height: 100vh;">
      <el-aside :width="sidebarCollapsed ? '64px' : '220px'" class="app-sidebar">
        <div class="sidebar-header" @click="sidebarCollapsed = !sidebarCollapsed">
          <el-icon :size="22"><Monitor /></el-icon>
          <span v-show="!sidebarCollapsed" class="sidebar-title">DeepSeek Monitor</span>
        </div>

        <el-menu
          :default-active="currentView"
          :collapse="sidebarCollapsed"
          background-color="#001529"
          text-color="#fff"
          active-text-color="#1890ff"
          @select="(index) => currentView = index"
        >
          <el-menu-item index="dashboard">
            <el-icon><DataBoard /></el-icon>
            <template #title>仪表盘</template>
          </el-menu-item>
          <el-menu-item index="realtime">
            <el-icon><Odometer /></el-icon>
            <template #title>实时监控</template>
          </el-menu-item>
          <el-menu-item index="logs">
            <el-icon><Document /></el-icon>
            <template #title>调用日志</template>
          </el-menu-item>
          <el-menu-item index="keys">
            <el-icon><Key /></el-icon>
            <template #title>API Keys</template>
          </el-menu-item>
          <el-menu-item index="settings">
            <el-icon><Setting /></el-icon>
            <template #title>系统设置</template>
          </el-menu-item>
        </el-menu>

        <div class="sidebar-footer">
          <el-tag size="small" type="info" effect="dark" v-show="!sidebarCollapsed">
            {{ auth.username }}
          </el-tag>
          <el-button text size="small" style="color:#fff" @click="handleLogout">
            <el-icon><SwitchButton /></el-icon>
            <span v-show="!sidebarCollapsed">退出</span>
          </el-button>
        </div>
      </el-aside>

      <el-main class="app-main">
        <Dashboard v-if="currentView === 'dashboard'" />
        <Realtime v-else-if="currentView === 'realtime'" />
        <Logs v-else-if="currentView === 'logs'" />
        <KeysView v-else-if="currentView === 'keys'" />
        <SettingsView v-else-if="currentView === 'settings'" />
      </el-main>
    </el-container>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { User, Lock, Monitor, SwitchButton } from '@element-plus/icons-vue'
import { login, register } from './api/index.js'
import { useAuth } from './store/auth.js'
import Dashboard from './views/Dashboard.vue'
import Realtime from './views/Realtime.vue'
import Logs from './views/Logs.vue'
import KeysView from './views/Keys.vue'
import SettingsView from './views/Settings.vue'

const auth = useAuth()
const currentView = ref('login')
const sidebarCollapsed = ref(false)
const isFirstUser = ref(true)
const loginLoading = ref(false)
const loginError = ref('')
const loginForm = reactive({ username: '', password: '' })

async function checkFirstUser() {
  try {
    const { data } = await login('check_if_first_user', 'check')
  } catch (err) {
    // If 401, user exists but wrong password → not first user
    // If 4xx "invalid username", first user
    const msg = err.response?.data?.error || ''
    isFirstUser.value = msg.includes('invalid username') || msg.includes('not found')
  }
}

async function handleLogin() {
  loginLoading.value = true
  loginError.value = ''
  try {
    const { data } = await login(loginForm.username, loginForm.password)
    auth.setAuth(data.token, data.user)
    currentView.value = 'dashboard'
    window.location.hash = '#/dashboard'
  } catch (err) {
    const msg = err.response?.data?.error || '登录失败'
    // If "invalid username", try registering as first user
    if (isFirstUser.value && msg.includes('invalid')) {
      try {
        const regData = await register(loginForm.username, loginForm.password)
        loginError.value = ''
        // Auto login after register
        const loginData = await login(loginForm.username, loginForm.password)
        auth.setAuth(loginData.data.token, loginData.data.user)
        currentView.value = 'dashboard'
        window.location.hash = '#/dashboard'
      } catch (regErr) {
        loginError.value = regErr.response?.data?.error || '注册失败'
      }
    } else {
      loginError.value = msg
    }
  } finally {
    loginLoading.value = false
  }
}

function handleLogout() {
  auth.logout()
  currentView.value = 'login'
  window.location.hash = '#/login'
}

// Check hash on mount for deep linking
onMounted(async () => {
  const hash = window.location.hash.replace('#/', '')
  if (hash && hash !== 'login') {
    const ok = await auth.verifySession()
    if (ok) {
      currentView.value = hash
      return
    }
  }

  // Auto-login if token exists
  if (auth.state.token) {
    const ok = await auth.verifySession()
    if (ok) {
      currentView.value = 'dashboard'
      return
    }
  }

  await checkFirstUser()
  currentView.value = 'login'
})
</script>

<style>
* { margin: 0; padding: 0; box-sizing: border-box; }
body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #f0f2f5; }

/* Login */
.login-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #001529 0%, #003a70 100%);
}
.login-card { width: 400px; }
.login-header {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 10px;
}
.login-header h2 { font-size: 18px; margin: 0; }

/* Sidebar */
.app-layout { min-height: 100vh; }
.app-sidebar {
  background: #001529;
  display: flex;
  flex-direction: column;
  transition: width 0.3s;
  position: fixed;
  left: 0;
  top: 0;
  bottom: 0;
  z-index: 100;
  overflow: hidden;
}
.sidebar-header {
  height: 56px;
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 0 16px;
  color: #fff;
  cursor: pointer;
  border-bottom: 1px solid rgba(255,255,255,0.1);
}
.sidebar-title { font-size: 15px; font-weight: 600; white-space: nowrap; }
.sidebar-footer {
  margin-top: auto;
  padding: 12px 16px;
  display: flex;
  align-items: center;
  gap: 8px;
  border-top: 1px solid rgba(255,255,255,0.1);
}
.sidebar-footer .el-button { color: rgba(255,255,255,0.7); }

/* Override el-menu to be vertical full height */
.app-sidebar .el-menu { border-right: none; flex: 1; }

/* Main content area */
.app-main {
  margin-left: 220px;
  padding: 20px;
  min-height: 100vh;
  transition: margin-left 0.3s;
}

@media (max-width: 768px) {
  .app-sidebar { width: 64px !important; }
  .app-main { margin-left: 64px !important; }
}
</style>
