<template>
  <div>
    <h2 style="margin-bottom:20px">系统设置</h2>

    <el-row :gutter="20">
      <el-col :xs="24" :md="12">
        <el-card shadow="hover">
          <template #header><strong>监控设置</strong></template>
          <el-form label-position="top">
            <el-form-item label="数据采集间隔">
              <el-select v-model="settings['monitor.collect_interval']" style="width:100%">
                <el-option label="1 分钟" value="1m" />
                <el-option label="5 分钟" value="5m" />
                <el-option label="10 分钟" value="10m" />
                <el-option label="30 分钟" value="30m" />
                <el-option label="1 小时" value="1h" />
              </el-select>
              <div style="font-size:12px;color:#8c8c8c;margin-top:4px">
                建议 5 分钟以上，避免触发 DeepSeek 频率限制
              </div>
            </el-form-item>

            <el-form-item label="数据保留天数">
              <el-input-number v-model="settings['monitor.retention_days']" :min="7" :max="365" style="width:100%" />
              <div style="font-size:12px;color:#8c8c8c;margin-top:4px">
                超过此天数的原始数据会被自动清理
              </div>
            </el-form-item>

            <el-form-item label="余额告警阈值 ($)">
              <el-input-number v-model="settings['alert.balance_threshold']" :min="0.1" :max="1000" :step="0.5" :precision="1" style="width:100%" />
              <div style="font-size:12px;color:#8c8c8c;margin-top:4px">
                余额低于此值时仪表盘会显示警告
              </div>
            </el-form-item>

            <el-form-item label="错误通知">
              <el-switch
                v-model="settings['alert.error_enabled']"
                active-value="true"
                inactive-value="false"
                active-text="启用"
                inactive-text="关闭"
              />
            </el-form-item>
          </el-form>
        </el-card>
      </el-col>

      <el-col :xs="24" :md="12">
        <el-card shadow="hover">
          <template #header><strong>账户信息</strong></template>
          <el-descriptions :column="1" border>
            <el-descriptions-item label="用户名">{{ user.username }}</el-descriptions-item>
            <el-descriptions-item label="角色">
              <el-tag size="small">{{ user.role }}</el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="注册时间">{{ user.created_at }}</el-descriptions-item>
          </el-descriptions>

          <el-divider />

          <h4 style="margin-bottom:12px">注册新用户</h4>
          <el-form label-position="top" @submit.prevent="handleRegister">
            <el-form-item label="用户名">
              <el-input v-model="regForm.username" placeholder="3-64 字符" />
            </el-form-item>
            <el-form-item label="密码">
              <el-input v-model="regForm.password" type="password" show-password placeholder="至少 6 位" />
            </el-form-item>
            <el-form-item>
              <el-button type="primary" native-type="submit" :loading="regLoading">创建用户</el-button>
            </el-form-item>
          </el-form>
          <el-alert v-if="regMsg" :title="regMsg" :type="regOk ? 'success' : 'error'" show-icon :closable="true" @close="regMsg=''" />
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="20" style="margin-top:20px">
      <el-col :span="24">
        <el-card shadow="hover">
          <template #header><strong>系统信息</strong></template>
          <el-descriptions :column="3" border>
            <el-descriptions-item label="DeepSeek API 地址">https://api.deepseek.com</el-descriptions-item>
            <el-descriptions-item label="数据库">{{ dbInfo.driver }}</el-descriptions-item>
            <el-descriptions-item label="服务端口">8080</el-descriptions-item>
          </el-descriptions>
        </el-card>
      </el-col>
    </el-row>

    <div style="text-align:right;margin-top:16px">
      <el-button type="primary" @click="handleSave" :loading="saveLoading">保存设置</el-button>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { fetchSettings, updateSettings, fetchMe, register } from '../api/index.js'
import { useAuth } from '../store/auth.js'
import { ElMessage } from 'element-plus'

const auth = useAuth()

const settings = reactive({
  'monitor.collect_interval': '5m',
  'monitor.retention_days': 90,
  'alert.balance_threshold': 5.0,
  'alert.error_enabled': 'true',
})

const dbInfo = reactive({ driver: 'SQLite' })
const user = reactive({ username: '', role: '', created_at: '' })
const regForm = reactive({ username: '', password: '' })
const regMsg = ref('')
const regOk = ref(false)
const regLoading = ref(false)
const saveLoading = ref(false)

async function loadSettings() {
  try {
    const { data } = await fetchSettings()
    Object.assign(settings, data)
    settings['monitor.retention_days'] = parseInt(settings['monitor.retention_days'] || '90')
    settings['alert.balance_threshold'] = parseFloat(settings['alert.balance_threshold'] || '5.0')
  } catch (e) {}
}

async function loadUser() {
  try {
    const { data } = await fetchMe()
    Object.assign(user, data)
  } catch (e) {}
}

async function handleSave() {
  saveLoading.value = true
  try {
    await updateSettings({
      'monitor.collect_interval': settings['monitor.collect_interval'],
      'monitor.retention_days': String(settings['monitor.retention_days']),
      'alert.balance_threshold': String(settings['alert.balance_threshold']),
      'alert.error_enabled': settings['alert.error_enabled'],
    })
    ElMessage.success('设置已保存 (下次采集生效)')
  } catch (e) {
    ElMessage.error('保存失败: ' + (e.response?.data?.error || e.message))
  }
  saveLoading.value = false
}

async function handleRegister() {
  if (!regForm.username || regForm.password.length < 6) {
    regMsg.value = '用户名不能为空，密码至少 6 位'
    regOk.value = false
    return
  }
  regLoading.value = true
  regMsg.value = ''
  try {
    await register(regForm.username, regForm.password)
    regMsg.value = `用户 ${regForm.username} 创建成功！`
    regOk.value = true
    regForm.username = ''
    regForm.password = ''
  } catch (e) {
    regMsg.value = e.response?.data?.error || '创建失败'
    regOk.value = false
  }
  regLoading.value = false
}

onMounted(() => { loadSettings(); loadUser() })
</script>
