<template>
  <div>
    <el-row :gutter="16" style="margin-bottom:16px">
      <el-col :span="12"><h2 style="margin:0">实时监控</h2></el-col>
      <el-col :span="12" style="text-align:right">
        <el-tag v-if="connected" type="success" effect="dark">实时</el-tag>
        <el-tag v-else type="danger" effect="dark">断开</el-tag>
      </el-col>
    </el-row>

    <!-- Real-time Cards -->
    <el-row :gutter="16" class="cards-row">
      <el-col :xs="12" :md="6">
        <el-card shadow="hover" class="rt-card">
          <div class="rt-value" :class="qpsColor">{{ metrics.qps }}</div>
          <div class="rt-label">QPS</div>
        </el-card>
      </el-col>
      <el-col :xs="12" :md="6">
        <el-card shadow="hover" class="rt-card">
          <div class="rt-value" :class="latencyColor">{{ metrics.avg_latency_ms }}ms</div>
          <div class="rt-label">平均延迟</div>
          <div class="rt-sub">P95: {{ metrics.p95_latency_ms }}ms</div>
        </el-card>
      </el-col>
      <el-col :xs="12" :md="6">
        <el-card shadow="hover" class="rt-card">
          <div class="rt-value" :class="errorColor">{{ metrics.error_rate }}%</div>
          <div class="rt-label">错误率</div>
          <div class="rt-sub">{{ metrics.success_count }}成功 / {{ metrics.error_count }}失败</div>
        </el-card>
      </el-col>
      <el-col :xs="12" :md="6">
        <el-card shadow="hover" class="rt-card">
          <div class="rt-value">${{ formattedCost }}</div>
          <div class="rt-label">窗口内费用 (60s)</div>
          <div class="rt-sub">{{ metrics.total_tokens }} tokens</div>
        </el-card>
      </el-col>
    </el-row>

    <!-- QPS Chart -->
    <el-row :gutter="16" class="charts-row">
      <el-col :span="24">
        <el-card shadow="hover">
          <template #header><strong>QPS 实时曲线</strong></template>
          <div class="chart-container"><v-chart :option="qpsOption" autoresize /></div>
        </el-card>
      </el-col>
    </el-row>

    <!-- Latency + Error Rate Charts -->
    <el-row :gutter="16" class="charts-row">
      <el-col :xs="24" :md="12">
        <el-card shadow="hover">
          <template #header><strong>延迟趋势 (ms)</strong></template>
          <div class="chart-container"><v-chart :option="latencyOption" autoresize /></div>
        </el-card>
      </el-col>
      <el-col :xs="24" :md="12">
        <el-card shadow="hover">
          <template #header><strong>错误率趋势 (%)</strong></template>
          <div class="chart-container"><v-chart :option="errorOption" autoresize /></div>
        </el-card>
      </el-col>
    </el-row>

    <!-- Token Throughput -->
    <el-row :gutter="16" class="charts-row">
      <el-col :span="24">
        <el-card shadow="hover">
          <template #header><strong>Token 吞吐量 / 秒</strong></template>
          <div class="chart-container"><v-chart :option="tokenOption" autoresize /></div>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import VChart from 'vue-echarts'
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { LineChart, BarChart } from 'echarts/charts'
import { TooltipComponent, GridComponent } from 'echarts/components'
import { fetchRealtimeMetrics } from '../api/index.js'

use([CanvasRenderer, LineChart, BarChart, TooltipComponent, GridComponent])

const metrics = ref({
  qps: 0, avg_latency_ms: 0, p95_latency_ms: 0, p99_latency_ms: 0,
  error_rate: 0, total_requests: 0, error_count: 0, success_count: 0,
  total_tokens: 0, total_cost: 0, tokens_per_sec: 0,
  qps_history: [], latency_history: [], error_rate_history: [], token_history: [],
})
const connected = ref(true)

const qpsColor = computed(() => metrics.value.qps > 5 ? 'value-high' : metrics.value.qps > 1 ? 'value-mid' : 'value-low')
const latencyColor = computed(() => metrics.value.avg_latency_ms > 2000 ? 'value-high' : metrics.value.avg_latency_ms > 500 ? 'value-mid' : 'value-low')
const errorColor = computed(() => metrics.value.error_rate > 5 ? 'value-high' : metrics.value.error_rate > 1 ? 'value-mid' : 'value-low')
const formattedCost = computed(() => {
  const c = metrics.value.total_cost || 0
  return c > 0.001 ? c.toFixed(4) : c.toFixed(8)
})

function makeLineOption(history, name, color, yName) {
  const times = history.map(p => p.time)
  const values = history.map(p => p.value)
  return {
    tooltip: { trigger: 'axis', formatter: (params) => `${params[0].axisValue}: ${params[0].value}` },
    grid: { left: 60, right: 20, top: 10, bottom: 30 },
    xAxis: { type: 'category', data: times, axisLabel: { fontSize: 10 } },
    yAxis: { type: 'value', name: yName || name },
    series: [{ type: 'line', smooth: true, showSymbol: false, lineStyle: { color, width: 2 }, areaStyle: { color: { type: 'linear', x: 0, y: 0, x2: 0, y2: 1, colorStops: [{ offset: 0, color: color + '40' }, { offset: 1, color: color + '05' }] } }, data: values }],
  }
}

const qpsOption = computed(() => makeLineOption(metrics.value.qps_history, 'QPS', '#1890ff', 'req/s'))
const latencyOption = computed(() => makeLineOption(metrics.value.latency_history, 'Latency', '#fa8c16', 'ms'))
const errorOption = computed(() => makeLineOption(metrics.value.error_rate_history, 'Error Rate', '#f5222d', '%'))
const tokenOption = computed(() => makeLineOption(metrics.value.token_history, 'Tokens', '#52c41a', 'tokens/s'))

async function refresh() {
  try {
    const { data } = await fetchRealtimeMetrics()
    metrics.value = data
    connected.value = true
  } catch {
    connected.value = false
  }
}

let interval
onMounted(() => { refresh(); interval = setInterval(refresh, 2000) })
onUnmounted(() => clearInterval(interval))
</script>

<style scoped>
.cards-row, .charts-row { margin-bottom: 16px !important; }
.rt-card { text-align: center; }
.rt-value { font-size: 28px; font-weight: 700; }
.rt-label { font-size: 13px; color: #8c8c8c; margin: 4px 0; }
.rt-sub { font-size: 11px; color: #bfbfbf; }
.value-high { color: #f5222d; }
.value-mid { color: #fa8c16; }
.value-low { color: #52c41a; }
.chart-container { width: 100%; height: 250px; }
</style>
