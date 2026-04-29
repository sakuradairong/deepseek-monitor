<template>
  <div>
    <el-row :gutter="16" style="margin-bottom:16px">
      <el-col :span="12"><h2 style="margin:0">API 调用日志</h2></el-col>
      <el-col :span="12" style="text-align:right">
        <el-button size="small" @click="refresh">
          <el-icon><Refresh /></el-icon> 刷新
        </el-button>
      </el-col>
    </el-row>

    <!-- Filters -->
    <el-card shadow="hover" style="margin-bottom:16px">
      <el-row :gutter="16">
        <el-col :span="6">
          <el-select v-model="filter.model" clearable placeholder="模型过滤" style="width:100%" @change="refresh">
            <el-option v-for="m in filterModels" :key="m" :label="m" :value="m" />
          </el-select>
        </el-col>
        <el-col :span="6">
          <el-select v-model="filter.error_type" clearable placeholder="错误类型" style="width:100%" @change="refresh">
            <el-option label="全部 (含成功)" value="" />
            <el-option label="4xx 错误" value="4xx" />
            <el-option label="5xx 错误" value="5xx" />
          </el-select>
        </el-col>
        <el-col :span="6">
          <el-select v-model="filter.min_status" clearable placeholder="状态码范围" style="width:100%" @change="refresh">
            <el-option label="全部" :value="0" />
            <el-option label="仅错误 (>=400)" :value="400" />
            <el-option label="仅服务端错误 (>=500)" :value="500" />
          </el-select>
        </el-col>
        <el-col :span="6">
          <el-button type="primary" @click="refresh">
            <el-icon><Search /></el-icon> 查询
          </el-button>
        </el-col>
      </el-row>
    </el-card>

    <!-- Log Table -->
    <el-card shadow="hover">
      <el-table :data="logs" style="width:100%" v-loading="loading" :max-height="600" size="small">
        <el-table-column label="时间" width="170">
          <template #default="{row}">{{ formatTime(row.created_at) }}</template>
        </el-table-column>
        <el-table-column prop="model" label="模型" width="140" />
        <el-table-column label="状态" width="80">
          <template #default="{row}">
            <el-tag :type="row.status_code >= 500 ? 'danger' : row.status_code >= 400 ? 'warning' : 'success'" size="small">
              {{ row.status_code }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="延迟" width="90">
          <template #default="{row}">{{ row.latency_ms }}ms</template>
        </el-table-column>
        <el-table-column label="Tokens" width="150">
          <template #default="{row}">
            P={{ row.prompt_tokens }} C={{ row.completion_tokens }} T={{ row.total_tokens }}
          </template>
        </el-table-column>
        <el-table-column prop="error_type" label="错误" width="80">
          <template #default="{row}">
            <el-tag v-if="row.error_type" type="danger" size="small">{{ row.error_type }}</el-tag>
            <span v-else class="no-error">-</span>
          </template>
        </el-table-column>
        <el-table-column label="Prompt 预览" min-width="200">
          <template #default="{row}">
            <el-tooltip :content="row.prompt_preview || '-'" placement="top" :disabled="!row.prompt_preview">
              <span class="preview-text">{{ row.prompt_preview ? row.prompt_preview.substring(0, 50) + '...' : '-' }}</span>
            </el-tooltip>
          </template>
        </el-table-column>
        <el-table-column label="响应预览" min-width="200">
          <template #default="{row}">
            <el-tooltip :content="row.response_preview || '-'" placement="top" :disabled="!row.response_preview">
              <span class="preview-text">{{ row.response_preview ? row.response_preview.substring(0, 50) + '...' : '-' }}</span>
            </el-tooltip>
          </template>
        </el-table-column>
      </el-table>

      <!-- Pagination -->
      <div style="text-align:right;margin-top:16px">
        <el-pagination
          v-model:current-page="page"
          :page-size="pageSize"
          :total="total"
          layout="total, prev, pager, next"
          @current-change="handlePageChange"
          background
          small
        />
      </div>
    </el-card>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { fetchProxyLogs } from '../api/index.js'

const logs = ref([])
const loading = ref(false)
const total = ref(0)
const page = ref(1)
const pageSize = 30
const filterModels = ref(['deepseek-v4-flash', 'deepseek-v4-pro', 'deepseek-chat', 'deepseek-reasoner'])

const filter = reactive({
  model: '',
  error_type: '',
  min_status: 0,
})

function formatTime(t) {
  if (!t) return '-'
  const d = new Date(t)
  return d.toLocaleString()
}

async function refresh() {
  loading.value = true
  try {
    const offset = (page.value - 1) * pageSize
    const params = {}
    if (filter.model) params.model = filter.model
    if (filter.error_type) params.error_type = filter.error_type
    if (filter.min_status) params.min_status = filter.min_status
    const { data } = await fetchProxyLogs(offset, pageSize, params)
    logs.value = data.logs || []
    total.value = data.total || 0
  } catch (e) {}
  loading.value = false
}

function handlePageChange() {
  refresh()
}

onMounted(refresh)
</script>

<style scoped>
.no-error { color: #bfbfbf; }
.preview-text { font-size: 12px; color: #595959; cursor: pointer; }
</style>
