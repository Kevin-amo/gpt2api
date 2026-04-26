<script setup lang="ts">
import { computed, ref, onMounted, onBeforeUnmount } from 'vue'
import { storeToRefs } from 'pinia'
import { useRouter } from 'vue-router'
import {
  Refresh, Wallet, Key, ChatLineRound, PictureFilled,
  DataAnalysis, Collection, Promotion,
} from '@element-plus/icons-vue'
import { useUserStore } from '@/stores/user'
import { listKeys } from '@/api/apikey'
import * as meApi from '@/api/me'
import { formatCredit, formatDateTime, formatErrorCode } from '@/utils/format'

const store = useUserStore()
const { user } = storeToRefs(store)
const router = useRouter()

// ---------- 基本信息 ----------
const balance = computed(() => formatCredit(user.value?.credit_balance))
const frozen = computed(() => formatCredit(user.value?.credit_frozen))
const greeting = computed(() => {
  const h = new Date().getHours()
  if (h < 6) return '夜深了'
  if (h < 12) return '早上好'
  if (h < 14) return '中午好'
  if (h < 18) return '下午好'
  return '晚上好'
})

// ---------- 数据:API Keys / 模型 / 统计 / 最近日志 / 账变 ----------
const loading = ref(false)
const keyTotal = ref(0)
const keyActive = ref(0)
const modelCount = ref(0)
const stats14 = ref<meApi.MyStatsResp | null>(null)
const stats1 = ref<meApi.MyStatsResp | null>(null)
const recentLogs = ref<meApi.UsageItem[]>([])
const recentCredits = ref<meApi.MyCreditLog[]>([])

async function loadAll() {
  loading.value = true
  try {
    await store.fetchMe()
    const [keys, models, s14, s1, logs, credits] = await Promise.all([
      listKeys(1, 100),
      meApi.listMyModels(),
      meApi.getMyUsageStats({ days: 14, top_n: 3 }),
      meApi.getMyUsageStats({ days: 1, top_n: 1 }),
      meApi.listMyUsageLogs({ limit: 6, offset: 0 }),
      meApi.listMyCreditLogs({ limit: 3, offset: 0 }),
    ])
    keyTotal.value = keys.total
    keyActive.value = keys.list.filter((k) => k.enabled).length
    modelCount.value = models.total
    stats14.value = s14
    stats1.value = s1
    recentLogs.value = logs.items
    recentCredits.value = credits.items
  } finally {
    loading.value = false
  }
}

// ---------- 派生指标 ----------
const todayOverall = computed(() => stats1.value?.overall)
const monthOverall = computed(() => stats14.value?.overall) // 近 14 天,作为"近期"展示
const daily = computed(() => stats14.value?.daily || [])
const topModels = computed(() => (stats14.value?.by_model || []).slice(0, 3))

function successRate(o?: meApi.UsageOverall | null): string {
  if (!o || o.requests === 0) return '—'
  return `${(((o.requests - o.failures) / o.requests) * 100).toFixed(1)}%`
}

// ---------- 趋势图(SVG)—— 复用 Usage 页的思路,改成面积图+折线 ----------
const chartWrap = ref<HTMLElement | null>(null)
const chartW = ref(640)
const chartH = 140
const padT = 10
const padB = 24
const padL = 32
const padR = 10
const hoverIdx = ref(-1)

let ro: ResizeObserver | null = null
onMounted(async () => {
  if (chartWrap.value && typeof ResizeObserver !== 'undefined') {
    ro = new ResizeObserver((entries) => {
      for (const e of entries) {
        const w = e.contentRect.width
        if (w > 0) chartW.value = Math.floor(w)
      }
    })
    ro.observe(chartWrap.value)
  }
  await loadAll()
})
onBeforeUnmount(() => { ro?.disconnect() })

function niceMax(v: number): number {
  if (v <= 1) return 1
  const exp = Math.pow(10, Math.floor(Math.log10(v)))
  const n = v / exp
  let m = 10
  if (n <= 1) m = 1
  else if (n <= 2) m = 2
  else if (n <= 5) m = 5
  return m * exp
}
const yMax = computed(() => niceMax(daily.value.reduce((x, r) => Math.max(x, r.requests), 0) || 1))
const yTicks = computed(() => {
  const m = yMax.value
  return [0, m / 2, m].map((v) => Math.round(v))
})
const cellW = computed(() => {
  const n = Math.max(daily.value.length, 1)
  return (chartW.value - padL - padR) / n
})
function xCenter(i: number) { return padL + cellW.value * (i + 0.5) }
function pointY(v: number) {
  const innerH = chartH - padT - padB
  return chartH - padB - (v / yMax.value) * innerH
}

// 折线 path
const linePath = computed(() => {
  if (!daily.value.length) return ''
  return daily.value
    .map((p, i) => `${i === 0 ? 'M' : 'L'}${xCenter(i).toFixed(1)},${pointY(p.requests).toFixed(1)}`)
    .join(' ')
})
// 面积 path(折线 + 底边)
const areaPath = computed(() => {
  if (!daily.value.length) return ''
  const n = daily.value.length
  const baseY = chartH - padB
  let d = `M${xCenter(0).toFixed(1)},${baseY.toFixed(1)}`
  daily.value.forEach((p, i) => {
    d += ` L${xCenter(i).toFixed(1)},${pointY(p.requests).toFixed(1)}`
  })
  d += ` L${xCenter(n - 1).toFixed(1)},${baseY.toFixed(1)} Z`
  return d
})

const labelStep = computed(() => (daily.value.length > 10 ? 3 : 2))
function shouldShowLabel(i: number) {
  const n = daily.value.length
  if (i === 0 || i === n - 1) return true
  return i % labelStep.value === 0
}

const tipX = computed(() => (hoverIdx.value >= 0 ? xCenter(hoverIdx.value) : 0))
const tipY = computed(() => (hoverIdx.value >= 0 ? pointY(daily.value[hoverIdx.value]?.requests || 0) : 0))
const tipSide = computed<'left' | 'right'>(() => (tipX.value > chartW.value / 2 ? 'left' : 'right'))

// ---------- TOP 模型横向条 ----------
const maxTop = computed(() => topModels.value.reduce((x, r) => Math.max(x, r.requests), 0) || 1)

// ---------- 最近请求/账变 辅助 ----------
const statusMap: Record<string, { tag: 'success' | 'danger' | 'warning' | 'info'; label: string }> = {
  success: { tag: 'success', label: '成功' },
  failed: { tag: 'danger', label: '失败' },
  partial: { tag: 'warning', label: '部分' },
}
function statusTag(s: string) { return statusMap[s]?.tag || 'info' }
function statusLabel(s: string) { return statusMap[s]?.label || s || '-' }

const creditTypeLabel: Record<string, string> = {
  recharge: '充值', consume: '消费', refund: '退款',
  adjust: '调账', bonus: '赠送', freeze: '冻结', unfreeze: '解冻',
}

// ---------- 导航 CTA ----------
function go(p: string) { router.push(p) }
</script>

<template>
  <div class="page-container" v-loading="loading">
    <!-- ====== 顶部横幅 ====== -->
    <div class="hero-card">
      <div class="hero-main">
        <div class="hero-greet">
          {{ greeting }}，{{ user?.nickname || user?.email?.split('@')[0] || '同学' }}
        </div>
        <div class="hero-sub">
          欢迎回到个人控制台 · 当前角色
          <el-tag size="small" effect="plain">{{ store.role || '-' }}</el-tag>
          <span v-if="user?.last_login_at" class="muted hero-meta">
            上次登录 {{ formatDateTime(user?.last_login_at) }}
          </span>
        </div>
        <div class="hero-actions">
          <el-button type="primary" :icon="Wallet" @click="go('/personal/billing')">
            充值积分
          </el-button>
          <el-button :icon="Key" @click="go('/personal/keys')">
            管理 API Key
          </el-button>
          <el-button :icon="ChatLineRound" @click="go('/personal/play')">
            在线体验
          </el-button>
          <el-button text :icon="Refresh" :loading="loading" @click="loadAll">刷新</el-button>
        </div>
      </div>
      <div class="hero-balance">
        <div class="balance-label">可用积分</div>
        <div class="balance-value">{{ balance }}</div>
        <div class="balance-sub">冻结中 {{ frozen }}</div>
      </div>
    </div>

    <!-- ====== 关键指标(6 张小卡) ====== -->
    <el-row :gutter="16" class="kpis">
      <el-col :lg="4" :md="8" :sm="12" :xs="12">
        <div class="kpi primary">
          <div class="kpi-icon"><el-icon><DataAnalysis /></el-icon></div>
          <div class="kpi-body">
            <div class="kpi-label">今日请求</div>
            <div class="kpi-value">{{ todayOverall?.requests ?? 0 }}</div>
            <div class="kpi-sub">成功率 {{ successRate(todayOverall) }}</div>
          </div>
        </div>
      </el-col>
      <el-col :lg="4" :md="8" :sm="12" :xs="12">
        <div class="kpi success">
          <div class="kpi-icon"><el-icon><Collection /></el-icon></div>
          <div class="kpi-body">
            <div class="kpi-label">近 14 天请求</div>
            <div class="kpi-value">{{ monthOverall?.requests ?? 0 }}</div>
            <div class="kpi-sub">失败 {{ monthOverall?.failures ?? 0 }}</div>
          </div>
        </div>
      </el-col>
      <el-col :lg="4" :md="8" :sm="12" :xs="12">
        <div class="kpi info">
          <div class="kpi-icon"><el-icon><ChatLineRound /></el-icon></div>
          <div class="kpi-body">
            <div class="kpi-label">Token 合计</div>
            <div class="kpi-value">
              {{ (monthOverall?.input_tokens ?? 0) + (monthOverall?.output_tokens ?? 0) }}
            </div>
            <div class="kpi-sub">
              入 {{ monthOverall?.input_tokens ?? 0 }} / 出 {{ monthOverall?.output_tokens ?? 0 }}
            </div>
          </div>
        </div>
      </el-col>
      <el-col :lg="4" :md="8" :sm="12" :xs="12">
        <div class="kpi warning">
          <div class="kpi-icon"><el-icon><PictureFilled /></el-icon></div>
          <div class="kpi-body">
            <div class="kpi-label">生图张数</div>
            <div class="kpi-value">{{ monthOverall?.image_images ?? 0 }}</div>
            <div class="kpi-sub">近 14 天</div>
          </div>
        </div>
      </el-col>
      <el-col :lg="4" :md="8" :sm="12" :xs="12">
        <div class="kpi danger">
          <div class="kpi-icon"><el-icon><Wallet /></el-icon></div>
          <div class="kpi-body">
            <div class="kpi-label">近 14 天扣费</div>
            <div class="kpi-value">{{ formatCredit(monthOverall?.credit_cost) }}</div>
            <div class="kpi-sub">单位:积分</div>
          </div>
        </div>
      </el-col>
      <el-col :lg="4" :md="8" :sm="12" :xs="12">
        <div class="kpi purple">
          <div class="kpi-icon"><el-icon><Key /></el-icon></div>
          <div class="kpi-body">
            <div class="kpi-label">API Key</div>
            <div class="kpi-value">{{ keyTotal }}</div>
            <div class="kpi-sub">启用 {{ keyActive }} · 可用模型 {{ modelCount }}</div>
          </div>
        </div>
      </el-col>
    </el-row>

    <!-- ====== 趋势 + 热门模型 ====== -->
    <el-row :gutter="16">
      <el-col :lg="16" :md="14" :sm="24">
        <div class="card-block">
          <div class="flex-between" style="margin-bottom:10px">
            <div>
              <h3 class="page-title" style="margin:0;font-size:16px">请求趋势(近 14 天)</h3>
              <div class="muted" style="margin-top:4px">悬停查看每日详情</div>
            </div>
            <el-button text :icon="Promotion" @click="go('/personal/usage')">
              查看详情
            </el-button>
          </div>
          <div ref="chartWrap" class="chart-wrap">
            <el-empty
              v-if="!loading && daily.length === 0"
              description="暂无数据,先在「在线体验」发一条请求试试"
              :image-size="80"
              style="padding:24px 0"
            />
            <svg
              v-else
              class="chart-svg"
              :viewBox="`0 0 ${chartW} ${chartH}`"
              :style="{ height: chartH + 'px' }"
              @mouseleave="hoverIdx = -1"
            >
              <defs>
                <linearGradient id="dashAreaGrad" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="0%" stop-color="var(--el-color-primary)" stop-opacity="0.32" />
                  <stop offset="100%" stop-color="var(--el-color-primary)" stop-opacity="0" />
                </linearGradient>
              </defs>

              <!-- 网格 + y 刻度 -->
              <g class="chart-axis">
                <line
                  v-for="(t, ti) in yTicks" :key="'ty' + ti"
                  :x1="padL" :x2="chartW - padR"
                  :y1="pointY(t)" :y2="pointY(t)"
                  class="grid-line"
                  :class="{ 'grid-zero': ti === 0 }"
                />
                <text
                  v-for="(t, ti) in yTicks" :key="'tl' + ti"
                  :x="padL - 6" :y="pointY(t) + 4"
                  text-anchor="end" class="axis-tick"
                >{{ t }}</text>
              </g>

              <!-- 面积 + 折线 -->
              <path :d="areaPath" fill="url(#dashAreaGrad)" stroke="none" />
              <path
                :d="linePath"
                fill="none"
                stroke="var(--el-color-primary)"
                stroke-width="2"
                stroke-linejoin="round"
                stroke-linecap="round"
              />

              <!-- 交互矩形 + 圆点 + 日期 -->
              <g v-for="(p, i) in daily" :key="p.day">
                <rect
                  :x="padL + cellW * i" :y="padT"
                  :width="cellW" :height="chartH - padT - padB"
                  fill="transparent"
                  @mouseenter="hoverIdx = i"
                />
                <circle
                  :cx="xCenter(i)" :cy="pointY(p.requests)"
                  :r="hoverIdx === i ? 4.5 : 2.5"
                  class="dot"
                  :class="{ active: hoverIdx === i }"
                />
                <text
                  v-if="shouldShowLabel(i)"
                  :x="xCenter(i)" :y="chartH - padB + 14"
                  text-anchor="middle" class="axis-date"
                >{{ p.day.slice(5) }}</text>
              </g>

              <!-- hover 指示 + tooltip -->
              <line
                v-if="hoverIdx >= 0"
                :x1="tipX" :x2="tipX"
                :y1="padT" :y2="chartH - padB"
                class="hover-guide"
              />
              <foreignObject
                v-if="hoverIdx >= 0"
                :x="tipSide === 'right' ? tipX + 10 : tipX - 170"
                :y="Math.max(padT, tipY - 60)"
                width="160" height="72"
              >
                <div class="chart-tip">
                  <div class="tip-day">{{ daily[hoverIdx]?.day }}</div>
                  <div class="tip-row">
                    <span class="tip-dot primary"></span>请求
                    <b>{{ daily[hoverIdx]?.requests || 0 }}</b>
                  </div>
                  <div class="tip-row">
                    <span class="tip-dot danger"></span>失败
                    <b>{{ daily[hoverIdx]?.failures || 0 }}</b>
                  </div>
                </div>
              </foreignObject>
            </svg>
          </div>
        </div>
      </el-col>

      <el-col :lg="8" :md="10" :sm="24">
        <div class="card-block">
          <div class="flex-between" style="margin-bottom:10px">
            <h3 class="page-title" style="margin:0;font-size:16px">热门模型</h3>
            <span class="muted">近 14 天</span>
          </div>
          <div v-if="topModels.length === 0" class="muted" style="padding:16px 0;text-align:center">
            暂无调用记录
          </div>
          <div v-else class="top-list">
            <div v-for="(m, idx) in topModels" :key="m.model_id" class="top-row">
              <div class="top-head">
                <span class="top-rank" :class="'r' + (idx + 1)">{{ idx + 1 }}</span>
                <code class="top-slug">{{ m.model_slug || `#${m.model_id}` }}</code>
                <el-tag
                  size="small"
                  :type="m.type === 'image' ? 'warning' : 'primary'"
                  effect="plain"
                >{{ m.type || '-' }}</el-tag>
                <span class="top-val">{{ m.requests }}</span>
              </div>
              <div class="top-bar">
                <div
                  class="top-bar-inner"
                  :class="{ img: m.type === 'image' }"
                  :style="{ width: ((m.requests / maxTop) * 100) + '%' }"
                />
              </div>
              <div class="top-foot muted">
                扣费 {{ formatCredit(m.credit_cost) }} · 平均 {{ m.avg_dur_ms || 0 }} ms
              </div>
            </div>
          </div>
        </div>
      </el-col>
    </el-row>

    <!-- ====== 最近请求 + 最近账变 ====== -->
    <el-row :gutter="16">
      <el-col :lg="14" :md="14" :sm="24">
        <div class="card-block">
          <div class="flex-between" style="margin-bottom:10px">
            <h3 class="page-title" style="margin:0;font-size:16px">最近请求</h3>
            <el-button text @click="go('/personal/usage')">查看全部</el-button>
          </div>
          <el-table
            :data="recentLogs"
            stripe
            size="small"
            empty-text="暂无请求记录"
            :show-header="recentLogs.length > 0"
          >
            <el-table-column prop="created_at" label="时间" width="150">
              <template #default="{ row }">{{ formatDateTime(row.created_at) }}</template>
            </el-table-column>
            <el-table-column label="模型" min-width="140">
              <template #default="{ row }">
                <code>{{ row.model_slug || `#${row.model_id}` }}</code>
              </template>
            </el-table-column>
            <el-table-column label="类型" width="72">
              <template #default="{ row }">
                <el-tag size="small" :type="row.type === 'image' ? 'warning' : 'primary'">
                  {{ row.type || '-' }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column label="状态" width="80">
              <template #default="{ row }">
                <el-tag size="small" :type="statusTag(row.status)">
                  {{ statusLabel(row.status) }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column label="扣费" width="100">
              <template #default="{ row }">
                <span class="cost">{{ formatCredit(row.credit_cost) }}</span>
              </template>
            </el-table-column>
            <el-table-column label="错误" min-width="120">
              <template #default="{ row }">
                <el-tooltip v-if="row.error_code" :content="row.error_code" placement="top">
                  <span class="err">{{ formatErrorCode(row.error_code) }}</span>
                </el-tooltip>
                <span v-else class="muted">-</span>
              </template>
            </el-table-column>
          </el-table>
        </div>
      </el-col>

      <el-col :lg="10" :md="10" :sm="24">
        <div class="card-block">
          <div class="flex-between" style="margin-bottom:10px">
            <h3 class="page-title" style="margin:0;font-size:16px">最近账变</h3>
            <el-button text @click="go('/personal/usage')">查看全部</el-button>
          </div>
          <div v-if="recentCredits.length === 0" class="muted" style="padding:16px 0;text-align:center">
            暂无账变记录
          </div>
          <div v-else class="credit-list">
            <div v-for="c in recentCredits" :key="c.id" class="credit-row">
              <div class="cr-left">
                <el-tag size="small" effect="plain">
                  {{ creditTypeLabel[c.type] || c.type }}
                </el-tag>
                <div class="cr-remark">{{ c.remark || '-' }}</div>
                <div class="cr-time muted">{{ formatDateTime(c.created_at) }}</div>
              </div>
              <div class="cr-right">
                <div :class="['cr-amt', c.amount >= 0 ? 'in' : 'out']">
                  {{ c.amount >= 0 ? '+' : '' }}{{ formatCredit(c.amount) }}
                </div>
                <div class="cr-bal muted">余额 {{ formatCredit(c.balance_after) }}</div>
              </div>
            </div>
          </div>
        </div>
      </el-col>
    </el-row>

  </div>
</template>

<style scoped lang="scss">
.page-title { font-size: 18px; font-weight: 600; }
.muted { color: var(--el-text-color-secondary); font-size: 13px; }

/* 行与行之间留 12px 间距;每个 col 内部的 card-block 自身 padding 已够 */
:deep(.el-row) { margin-bottom: 12px; }
:deep(.el-row:last-child) { margin-bottom: 0; }
:deep(.el-row .card-block) { margin: 0; padding: 14px 16px; height: 100%; box-sizing: border-box; }
code {
  background: var(--el-fill-color-light);
  padding: 1px 6px;
  border-radius: 4px;
  font-family: ui-monospace, Menlo, Consolas, monospace;
  font-size: 12px;
}

/* ==== Hero ==== */
.hero-card {
  position: relative;
  border-radius: 12px;
  padding: 16px 22px;
  color: #fff;
  background:
    radial-gradient(circle at 85% 20%, rgba(255, 255, 255, 0.18), transparent 60%),
    linear-gradient(135deg, #4c6ef5 0%, #7c3aed 55%, #db2777 100%);
  box-shadow: 0 6px 24px rgba(76, 110, 245, 0.25);
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 24px;
  flex-wrap: wrap;
  margin-bottom: 12px;
  overflow: hidden;
}
.hero-card::after {
  content: '';
  position: absolute;
  right: -60px; top: -60px;
  width: 220px; height: 220px;
  border-radius: 50%;
  background: rgba(255, 255, 255, 0.08);
  pointer-events: none;
}
.hero-main { flex: 1 1 360px; min-width: 0; }
.hero-greet {
  font-size: 20px;
  font-weight: 600;
  letter-spacing: 0.5px;
  .wave { display: inline-block; transform: translateY(-1px); margin-right: 4px; }
}
.hero-sub {
  margin-top: 6px;
  font-size: 13px;
  color: rgba(255, 255, 255, 0.82);
  b { color: #fff; }
  :deep(.el-tag) {
    margin: 0 2px;
    background: rgba(255, 255, 255, 0.18);
    border-color: rgba(255, 255, 255, 0.35);
    color: #fff;
  }
  .muted { color: rgba(255, 255, 255, 0.7); font-size: 12.5px; }
}
.hero-actions {
  margin-top: 12px;
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  :deep(.el-button) { font-weight: 500; }
  :deep(.el-button.is-text) {
    color: rgba(255, 255, 255, 0.9);
    &:hover { background: rgba(255, 255, 255, 0.12); color: #fff; }
  }
}
.hero-balance {
  min-width: 180px;
  padding: 12px 20px;
  background: rgba(255, 255, 255, 0.14);
  border: 1px solid rgba(255, 255, 255, 0.25);
  border-radius: 10px;
  backdrop-filter: blur(6px);
  text-align: right;
  .balance-label { font-size: 12px; color: rgba(255, 255, 255, 0.8); }
  .balance-value {
    font-size: 26px;
    font-weight: 700;
    margin: 2px 0;
    line-height: 1.15;
    letter-spacing: 0.5px;
  }
  .balance-sub { font-size: 12px; color: rgba(255, 255, 255, 0.72); }
}

/* ==== KPI 卡 ==== */
.kpis { margin-bottom: 12px; }
.kpis .el-col { margin-bottom: 10px; }
.kpi {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 14px;
  border-radius: 10px;
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-lighter);
  transition: transform 0.15s, box-shadow 0.15s, border-color 0.15s;
  position: relative;
  overflow: hidden;
}
.kpi:hover {
  transform: translateY(-2px);
  box-shadow: 0 6px 18px rgba(0, 0, 0, 0.06);
  border-color: var(--el-border-color);
}
.kpi .kpi-icon {
  width: 36px; height: 36px;
  border-radius: 9px;
  display: flex; align-items: center; justify-content: center;
  font-size: 18px;
  flex-shrink: 0;
}
.kpi .kpi-body { flex: 1; min-width: 0; }
.kpi .kpi-label { font-size: 12px; color: var(--el-text-color-secondary); }
.kpi .kpi-value {
  font-size: 21px;
  font-weight: 700;
  line-height: 1.2;
  margin: 1px 0;
  color: var(--el-text-color-primary);
}
.kpi .kpi-sub {
  font-size: 11.5px;
  color: var(--el-text-color-secondary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.kpi.primary .kpi-icon { background: rgba(64, 158, 255, 0.12); color: var(--el-color-primary); }
.kpi.success .kpi-icon { background: rgba(103, 194, 58, 0.12); color: var(--el-color-success); }
.kpi.info .kpi-icon    { background: rgba(144, 147, 153, 0.14); color: var(--el-color-info); }
.kpi.warning .kpi-icon { background: rgba(230, 162, 60, 0.14); color: var(--el-color-warning); }
.kpi.danger .kpi-icon  { background: rgba(245, 108, 108, 0.14); color: var(--el-color-danger); }
.kpi.purple .kpi-icon  { background: rgba(124, 58, 237, 0.14); color: #7c3aed; }

/* ==== 趋势图 ==== */
.chart-wrap { width: 100%; min-height: 140px; }
.chart-svg {
  width: 100%;
  display: block;
  font-family: ui-sans-serif, system-ui, -apple-system, 'Segoe UI', sans-serif;
  user-select: none;
}
.chart-svg .grid-line {
  stroke: var(--el-border-color-lighter);
  stroke-width: 1;
  stroke-dasharray: 3 4;
}
.chart-svg .grid-line.grid-zero { stroke: var(--el-border-color); stroke-dasharray: none; }
.chart-svg .axis-tick { fill: var(--el-text-color-placeholder); font-size: 10.5px; }
.chart-svg .axis-date { fill: var(--el-text-color-secondary); font-size: 10.5px; }
.chart-svg .dot {
  fill: #fff;
  stroke: var(--el-color-primary);
  stroke-width: 2;
  transition: r 0.15s;
}
.chart-svg .dot.active {
  fill: var(--el-color-primary);
  stroke: #fff;
}
.chart-svg .hover-guide {
  stroke: var(--el-color-primary);
  stroke-width: 1;
  stroke-dasharray: 3 3;
  opacity: 0.45;
  pointer-events: none;
}

.chart-tip {
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-light);
  border-radius: 8px;
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.08);
  padding: 8px 10px;
  font-size: 12px;
  line-height: 1.6;
  color: var(--el-text-color-primary);
  pointer-events: none;
}
.chart-tip .tip-day { font-weight: 600; margin-bottom: 2px; }
.chart-tip .tip-row { display: flex; align-items: center; }
.chart-tip .tip-row b { margin-left: auto; font-weight: 600; }
.chart-tip .tip-dot {
  width: 8px; height: 8px; border-radius: 50%;
  display: inline-block; margin-right: 6px;
}
.chart-tip .tip-dot.primary { background: var(--el-color-primary); }
.chart-tip .tip-dot.danger { background: var(--el-color-danger); }

/* ==== 热门模型横向条 ==== */
.top-list { display: flex; flex-direction: column; gap: 12px; }
.top-row { min-width: 0; }
.top-head {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 6px;
  font-size: 13px;
}
.top-rank {
  width: 20px; height: 20px;
  border-radius: 50%;
  font-size: 11px;
  font-weight: 700;
  display: flex; align-items: center; justify-content: center;
  background: var(--el-fill-color);
  color: var(--el-text-color-secondary);
  flex-shrink: 0;
}
.top-rank.r1 { background: #ffd43b; color: #7a5a00; }
.top-rank.r2 { background: #ced4da; color: #495057; }
.top-rank.r3 { background: #ffc9a5; color: #8a4708; }
.top-slug {
  flex: 1 1 auto;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.top-val { font-weight: 600; color: var(--el-text-color-primary); }
.top-bar {
  height: 6px;
  background: var(--el-fill-color-light);
  border-radius: 4px;
  overflow: hidden;
}
.top-bar-inner {
  height: 100%;
  background: linear-gradient(90deg, var(--el-color-primary), #7c3aed);
  border-radius: 4px;
  transition: width 0.3s;
}
.top-bar-inner.img {
  background: linear-gradient(90deg, var(--el-color-warning), var(--el-color-danger));
}
.top-foot { font-size: 11.5px; margin-top: 4px; }

/* ==== 最近请求表格 ==== */
.cost { font-weight: 600; color: var(--el-color-danger); }
.err { color: var(--el-color-danger); font-size: 12px; }

/* ==== 最近账变列表 ==== */
.credit-list { display: flex; flex-direction: column; }
.credit-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
  padding: 8px 2px;
  border-bottom: 1px solid var(--el-border-color-lighter);
}
.credit-row:last-child { border-bottom: none; }
.cr-left { min-width: 0; flex: 1; }
.cr-remark {
  margin-top: 4px;
  font-size: 13px;
  color: var(--el-text-color-regular);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-width: 260px;
}
.cr-time { font-size: 11.5px; margin-top: 2px; }
.cr-right { text-align: right; flex-shrink: 0; }
.cr-amt {
  font-weight: 700;
  font-size: 15px;
  font-variant-numeric: tabular-nums;
}
.cr-amt.in { color: var(--el-color-success); }
.cr-amt.out { color: var(--el-color-danger); }
.cr-bal { font-size: 11.5px; margin-top: 2px; }

</style>
