<template>
  <div>
    <el-row :gutter="20" style="margin-bottom:16px">
      <el-col :span="12">
        <h2 style="margin:0">API Key 管理</h2>
      </el-col>
      <el-col :span="12" style="text-align:right">
        <el-button type="primary" @click="showAddDialog = true">
          <el-icon><Plus /></el-icon> 添加 Key
        </el-button>
      </el-col>
    </el-row>

    <el-alert
      title="多个 API Key 会自动轮转使用，优先使用 Priority 较高且调用次数较少的 Key。"
      type="info" show-icon :closable="false" style="margin-bottom:16px"
    />

    <!-- Key List -->
    <el-table :data="keys" style="width:100%" v-loading="loading">
      <el-table-column prop="name" label="名称" width="180" />
      <el-table-column prop="key_prefix" label="Key (前缀)" width="180" />
      <el-table-column prop="priority" label="优先级" width="80" />
      <el-table-column prop="usage_count" label="已使用" width="80" />
      <el-table-column label="状态" width="100">
        <template #default="{row}">
          <el-tag :type="row.is_active ? 'success' : 'danger'" size="small">
            {{ row.is_active ? '启用' : '停用' }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="last_error" label="最近错误" min-width="200">
        <template #default="{row}">
          <el-tooltip :content="row.last_error" placement="top" :disabled="!row.last_error">
            <span :style="{ color: row.last_error ? '#f5222d' : '#8c8c8c' }">
              {{ row.last_error ? row.last_error.substring(0, 40) + (row.last_error.length > 40 ? '...' : '') : '无' }}
            </span>
          </el-tooltip>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="240" fixed="right">
        <template #default="{row}">
          <el-button size="small" @click="testKey(row)">测试</el-button>
          <el-button size="small" @click="editKey(row)">编辑</el-button>
          <el-popconfirm title="确定删除此 Key？" @confirm="removeKey(row.id)">
            <template #reference>
              <el-button size="small" type="danger">删除</el-button>
            </template>
          </el-popconfirm>
        </template>
      </el-table-column>
    </el-table>

    <!-- Test Result Dialog -->
    <el-dialog v-model="testResult.show" title="Key 测试结果" width="400px">
      <el-result v-if="testResult.success" icon="success" title="Key 有效" :sub-title="`余额: ¥${testResult.balance}`">
        <template #extra>
          <el-tag type="success">可用</el-tag>
        </template>
      </el-result>
      <el-result v-else icon="error" title="Key 测试失败" :sub-title="testResult.error">
        <template #extra>
          <el-tag type="danger">不可用</el-tag>
        </template>
      </el-result>
    </el-dialog>

    <!-- Add/Edit Dialog -->
    <el-dialog v-model="showAddDialog" :title="editingKey ? '编辑 Key' : '添加 API Key'" width="500px">
      <el-form :model="keyForm" label-position="top">
        <el-form-item label="名称" required>
          <el-input v-model="keyForm.name" placeholder="例如: 主账号 Key" />
        </el-form-item>
        <el-form-item label="API Key" required>
          <el-input v-model="keyForm.key_value" type="password" show-password
            placeholder="sk-..." :disabled="!!editingKey" />
          <div v-if="editingKey" style="font-size:12px;color:#8c8c8c;margin-top:4px">
            编辑时不显示原有 Key 值，如需更改请重新输入
          </div>
        </el-form-item>
        <el-row :gutter="20">
          <el-col :span="12">
            <el-form-item label="优先级">
              <el-input-number v-model="keyForm.priority" :min="0" :max="100" />
              <div style="font-size:12px;color:#8c8c8c">数值越高的 Key 优先使用</div>
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="状态">
              <el-switch v-model="keyForm.is_active" active-text="启用" inactive-text="停用" />
            </el-form-item>
          </el-col>
        </el-row>
      </el-form>
      <template #footer>
        <el-button @click="showAddDialog = false">取消</el-button>
        <el-button type="primary" @click="saveKey" :loading="saving">
          {{ editingKey ? '保存' : '添加' }}
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { fetchKeys, createKey, updateKey, deleteKey, testKey as testKeyApi } from '../api/index.js'

const keys = ref([])
const loading = ref(false)
const saving = ref(false)
const showAddDialog = ref(false)
const editingKey = ref(null)

const keyForm = ref({ name: '', key_value: '', priority: 0, is_active: true })

const testResult = ref({ show: false, success: false, balance: '0', error: '' })

async function loadKeys() {
  loading.value = true
  try {
    const { data } = await fetchKeys()
    keys.value = data
  } catch (e) {}
  loading.value = false
}

function editKey(row) {
  editingKey.value = row
  keyForm.value = { name: row.name, key_value: '', priority: row.priority, is_active: row.is_active }
  showAddDialog.value = true
}

function resetForm() {
  editingKey.value = null
  keyForm.value = { name: '', key_value: '', priority: 0, is_active: true }
}

async function saveKey() {
  if (!keyForm.value.name || (!editingKey.value && !keyForm.value.key_value)) return
  saving.value = true
  try {
    if (editingKey.value) {
      const payload = { name: keyForm.value.name, priority: keyForm.value.priority, is_active: keyForm.value.is_active }
      if (keyForm.value.key_value) payload.key_value = keyForm.value.key_value
      await updateKey(editingKey.value.id, payload)
    } else {
      await createKey({ name: keyForm.value.name, key_value: keyForm.value.key_value, priority: keyForm.value.priority, is_active: keyForm.value.is_active })
    }
    showAddDialog.value = false
    resetForm()
    await loadKeys()
  } catch (e) {}
  saving.value = false
}

async function removeKey(id) {
  try {
    await deleteKey(id)
    await loadKeys()
  } catch (e) {}
}

async function testKey(row) {
  try {
    const { data } = await testKeyApi(row.id)
    testResult.value = { show: true, success: data.success, balance: data.total_balance || 'N/A', error: data.error || '' }
    await loadKeys()
  } catch (e) {
    testResult.value = { show: true, success: false, balance: '0', error: e.message }
  }
}

onMounted(loadKeys)
</script>
