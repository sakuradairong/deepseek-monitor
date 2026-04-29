<template>
  <div class="dashboard">
    <!-- Connection status -->
    <el-row :gutter="20" style="margin-bottom:12px">
      <el-col :span="24">
        <el-alert
          v-if="errorMsg"
          :title="errorMsg"
          type="error"
          show-icon
          :closable="true"
          @close="errorMsg = ''"
        />
      </el-col>
    </el-row>

    <!-- Overview Cards -->
    <el-row :gutter="20" class="cards-row">
      <el-col :xs="24" :sm="12" :md="6">
        <el-card shadow="hover" class="stat-card">
          <div class="card-content">
            <div class="card-icon" style="background:#e6f7ff;color:#1890ff">
              <el-icon :size="24"><Wallet /></el-icon>
            </div>
            <div class="card-info">
              <div class="card-label">账户余额</div>
              <div class="card-value" :class="balanceStatus">
                ¥{{ currentBalance }}
              </div>
              <div class="card-sub">总额度</div>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :xs="24" :sm="12" :md="6">
        <el-card shadow="hover" class="stat-card">
          <div class="card-content">
            <div class="card-icon" style="background:#f0f5ff;color:#2f54eb">
              <el-icon :size="24"><TrendingUp /></el-icon>
            </div>
            <div class="card-info">
              <div class="card-label">今日 Token 用量</div>
              <div class="card-value">{{ formatNumber(todayTokens) }}</div>
              <div class="card-sub">请求: {{ todayRequests }}</div>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :xs="24" :sm="12" :md="6">
        <el-card shadow="hover" class="stat-card">
          <div class="card-content">
            <div class="card-icon" style="background:#fff7e6;color:#fa8c16">
              <el-icon :size="24"><Coin /></el-icon>
            </div>
            <div class="card-info">
              <div class="card-label">本月费用</div>
              <div class="card-value">¥{{ monthCost }}</div>
              <div class="card-sub">今日: ¥{{ todayCost }}</div>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :xs="24" :sm="12" :md="6">
        <el-card shadow="hover" class="stat-card">
          <div class="card-content">
            <div class="card-icon" style="background:#f6ffed;color:#52c41a">
              <el-icon :size="24"><DataBoard /></el-icon>
            </div>
            <div class="card-info">
              <div class="card-label">速率限制</div>
              <div class="card-value">{{ rateRemaining }} / {{ rateLimit }}</div>
              <div class="card-sub" v-if="ratePct !== null">剩余 {{ ratePct }}%</div>
              <div class="card-sub" v-else>暂无数据</div>
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- Charts -->
    <el-row :gutter="20" class="charts-row">
      <el-col :xs="24" :lg="16">
        <el-card shadow="hover">
          <template #header>
            <div class="card-header">
              <span><strong>Token 用量趋势</strong></span>
              <el-radio-group v-model="trendDays" size="small" @change="loadTrend">
                <el-radio-button value="7">7天</el-radio-button>
                <el-radio-button value="14">14天</el-radio-button>
                <el-radio-button value="30">30天</el-radio-button>
              </el-radio-group>
            </div>
          </template>
          <div class="chart-container"><v-chart :option="trendOption" autoresize /></div>
        </el-card>
      </el-col>
      <el-col :xs="24" :lg="8">
        <el-card shadow="hover">
          <template #header>
            <div class="card-header">
              <span><strong>模型分布</strong></span>
            </div>
          </template>
          <div class="chart-container"><v-chart :option="modelDistOption" autoresize /></div>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="20" class="charts-row">
      <el-col :span="24">
        <el-card shadow="hover">
          <template #header>
            <div class="card-header">
              <span><strong>余额变化趋势</strong></span>
              <el-radio-group v-model="balanceDays" size="small" @change="loadBalanceHistory">
                <el-radio-button value="7">7天</el-radio-button>
                <el-radio-button value="30">30天</el-radio-button>
                <el-radio-button value="90">90天</el-radio-button>
              </el-radio-group>
            </div>
          </template>
          <div class="chart-container"><v-chart :option="balanceOption" autoresize /></div>
        </el-card>
      </el-col>
    </el-row>

    <!-- Errors -->
    <el-row :gutter="20" class="charts-row" v-if="recentErrors.length > 0">
      <el-col :span="24">
        <el-card shadow="hover">
          <template #header>
            <div class="card-header">
              <span><strong>近期 API 错误</strong></span>
              <el-tag type="danger" size="small">{{ recentErrors.length }} 条</el-tag>
            </div>
          </template>
          <el-table :data="recentErrors" size="small" style="width:100%">
            <el-table-column prop="collected_at" label="时间" width="180">
              <template #default="{row}">{{ formatTime(row.collected_at) }}</template>
            </el-table-column>
            <el-table-column prop="error_type" label="类型" width="120">
              <template #default="{row}">
                <el-tag :type="row.error_type === 'balance' ? 'warning' : 'danger'" size="small">
                  {{ row.error_type }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="message" label="错误信息" />
          </el-table>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import VChart from 'vue-echarts'
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { LineChart, PieChart } from 'echarts/charts'
import { TooltipComponent, LegendComponent, GridComponent } from 'echarts/components'
import { fetchOverview, fetchUsageTrend, fetchBalanceHistory, fetchModelDistribution, fetchRecentErrors } from '../api/index.js'

use([CanvasRenderer, LineChart, PieChart, TooltipComponent, LegendComponent, GridComponent])

const errorMsg = ref('')
const trendDays = ref('7')
const balanceDays = ref('30')
const currentBalance = ref('0.00')
const todayTokens = ref(0)
const todayRequests = ref(0)
const monthCost = ref('0.00')
const todayCost = ref('0.00')
const rateLimit = ref(0)
const rateRemaining = ref(0)
const ratePct = ref(null)
const recentErrors = ref([])

const balanceStatus = ref('normal')

const trendOption = ref({
  tooltip: { trigger: 'axis' },
  legend: { show: true, bottom: 0 },
  grid: { left: 60, right: 20, top: 10, bottom: 40 },
  xAxis: { type: 'category', axisLabel: { rotate: 30 } },
  yAxis: { type: 'value', name: 'Tokens' },
  series: [],
})

const modelDistOption = ref({
  tooltip: { trigger: 'item', formatter: '{b}: {c} ({d}%)' },
  legend: { show: true, bottom: 0, type: 'scroll' },
  series: [{ type: 'pie', radius: ['40%', '65%'], center: ['50%', '45%'], data: [] }],
})

const balanceOption = ref({
  tooltip: { trigger: 'axis' },
  grid: { left: 60, right: 20, top: 10, bottom: 30 },
  xAxis: { type: 'category', axisLabel: { rotate: 30 } },
  yAxis: { type: 'value', name: 'USD' },
  series: [{ type: 'line', smooth: true, showSymbol: false, areaStyle: { color: { type: 'linear', x: 0, y: 0, x2: 0, y2: 1, colorStops: [{ offset: 0, color: 'rgba(24,144,255,0.3)' }, { offset: 1, color: 'rgba(24,144,255,0.02)' }] } }, lineStyle: { color: '#1890ff', width: 2 }, data: [] }],
})

function formatNumber(n) { n = Number(n || 0); if (n >= 1_000_000) return (n/1_000_000).toFixed(2)+'M'; if (n >= 1_000) return (n/1_000).toFixed(1)+'K'; return n.toLocaleString() }
function formatTime(t) { return t ? new Date(t).toLocaleString() : '-' }

async function loadOverview() {
  try {
    const { data } = await fetchOverview()
    const bal = data.current_balance
    if (bal) {
      currentBalance.value = (bal.total_balance || 0).toFixed(4)
      balanceStatus.value = parseFloat(currentBalance.value) <= 0 ? 'danger' : parseFloat(currentBalance.value) < 5 ? 'warning' : 'normal'
    }
    const t = data.today_usage
    if (t) { todayTokens.value = t.total_tokens || 0; todayRequests.value = t.total_requests || 0; todayCost.value = (t.estimated_cost || 0).toFixed(4) }
    const m = data.month_usage
    if (m) { monthCost.value = (m.estimated_cost || 0).toFixed(4) }
    const rl = data.latest_rate_limit
    if (rl) { rateLimit.value = rl.requests_limit || 0; rateRemaining.value = rl.requests_remaining || 0; ratePct.value = rl.requests_limit > 0 ? Math.round((rl.requests_remaining / rl.requests_limit) * 100) : null }
    recentErrors.value = data.recent_errors || []
  } catch (err) { errorMsg.value = '加载数据失败: ' + (err.response?.data?.error || err.message) }
}

async function loadTrend() {
  try {
    const { data } = await fetchUsageTrend(parseInt(trendDays.value))
    const dateMap = {}, modelSet = new Set()
    data.forEach(item => { const d = item.date; if (!dateMap[d]) dateMap[d] = {}; dateMap[d][item.model || 'unknown'] = (item.total_tokens || 0); modelSet.add(item.model) })
    const dates = Object.keys(dateMap).sort(), models = Array.from(modelSet)
    const colors = ['#1890ff','#52c41a','#fa8c16','#eb2f96','#722ed1','#13c2c2','#f5222d','#faad14']
    const series = models.map((model,i) => ({ name: model, type: 'line', smooth: true, showSymbol: false, lineStyle: { color: colors[i%colors.length], width: 2 }, data: dates.map(d => dateMap[d][model] || 0) }))
    trendOption.value = { ...trendOption.value, xAxis: { ...trendOption.value.xAxis, data: dates }, series }
  } catch (e) {}
}

async function loadBalanceHistory() {
  try {
    const { data } = await fetchBalanceHistory(parseInt(balanceDays.value))
    const dates = data.map(d => { const t = new Date(d.collected_at); return `${t.getMonth()+1}/${t.getDate()} ${t.getHours()}:${String(t.getMinutes()).padStart(2,'0')}` })
    const values = data.map(d => d.total_balance || 0)
    balanceOption.value = { ...balanceOption.value, xAxis: { ...balanceOption.value.xAxis, data: dates }, series: [{ ...balanceOption.value.series[0], data: values }] }
  } catch (e) {}
}

async function loadModelDist() {
  try {
    const { data } = await fetchModelDistribution(parseInt(trendDays.value))
    const colors = ['#1890ff','#52c41a','#fa8c16','#eb2f96','#722ed1','#13c2c2','#f5222d','#faad14']
    const pieData = (data.models || []).map((m,i) => ({ name: m.model, value: m.total_tokens || 0, itemStyle: { color: colors[i%colors.length] } }))
    modelDistOption.value = { ...modelDistOption.value, series: [{ ...modelDistOption.value.series[0], data: pieData }] }
  } catch (e) {}
}

async function refreshAll() { await Promise.all([loadOverview(), loadTrend(), loadBalanceHistory(), loadModelDist()]) }

let interval
onMounted(() => { refreshAll(); interval = setInterval(refreshAll, 5*60*1000) })
onUnmounted(() => clearInterval(interval))
</script>

<style scoped>
.cards-row, .charts-row { margin-bottom: 20px !important; }
.stat-card { border-radius: 8px; }
.card-content { display: flex; align-items: center; gap: 16px; }
.card-icon { width: 48px; height: 48px; border-radius: 12px; display: flex; align-items: center; justify-content: center; flex-shrink: 0; }
.card-info { flex: 1; }
.card-label { font-size: 13px; color: #8c8c8c; margin-bottom: 4px; }
.card-value { font-size: 24px; font-weight: 700; }
.card-value.normal { color: #1890ff; }
.card-value.warning { color: #fa8c16; }
.card-value.danger { color: #f5222d; }
.card-sub { font-size: 12px; color: #bfbfbf; }
.card-header { display: flex; align-items: center; justify-content: space-between; }
.chart-container { width: 100%; height: 320px; }
</style>
