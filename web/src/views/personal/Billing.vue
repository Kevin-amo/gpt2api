<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import * as rechargeApi from '@/api/recharge'
import { formatCredit, formatDateTime } from '@/utils/format'
import { useUserStore } from '@/stores/user'

const userStore = useUserStore()

const packages = ref<rechargeApi.Package[]>([])
const channelEnabled = ref(false)
const orders = ref<rechargeApi.Order[]>([])
const total = ref(0)
const paging = reactive({
  limit: 10,
  offset: 0,
  status: '' as '' | 'pending' | 'paid' | 'cancelled' | 'expired',
})

const loadingPkg = ref(false)
const loadingOrder = ref(false)
const redeemLoading = ref(false)
const redeemStatusLoading = ref(false)
const redeemed = ref(false)
const redeemRecord = ref<rechargeApi.RedeemCode | null>(null)
const redeemForm = reactive({ code: '' })

async function loadPackages() {
  loadingPkg.value = true
  try {
    const d = await rechargeApi.listMyPackages()
    packages.value = d.items
    channelEnabled.value = d.enabled
  } finally {
    loadingPkg.value = false
  }
}

async function loadOrders() {
  loadingOrder.value = true
  try {
    const d = await rechargeApi.listMyOrders({
      limit: paging.limit,
      offset: paging.offset,
      status: paging.status || undefined,
    })
    orders.value = d.items
    total.value = d.total
  } finally {
    loadingOrder.value = false
  }
}

async function loadRedeemStatus() {
  redeemStatusLoading.value = true
  try {
    const d = await rechargeApi.getMyRedeemStatus()
    redeemed.value = !!d.redeemed
    redeemRecord.value = d.record || null
  } finally {
    redeemStatusLoading.value = false
  }
}

async function submitRedeem() {
  const code = redeemForm.code.trim()
  if (!code) {
    ElMessage.warning('请输入兑换码')
    return
  }
  redeemLoading.value = true
  try {
    const row = await rechargeApi.redeemCode(code)
    redeemed.value = true
    redeemRecord.value = row
    redeemForm.code = ''
    await userStore.fetchMe()
    ElMessage.success(`兑换成功，已到账 ${formatCredit(row.credits)} 积分`)
  } finally {
    redeemLoading.value = false
  }
}

async function buy(pkg: rechargeApi.Package, payType?: string) {
  if (!channelEnabled.value) {
    ElMessage.warning('支付通道未配置，请使用兑换码或联系管理员')
    return
  }
  try {
    const order = await rechargeApi.createOrder(pkg.id, payType)
    if (!order.pay_url) {
      ElMessage.error('支付链接生成失败')
      return
    }
    window.open(order.pay_url, '_blank', 'noopener,noreferrer')
    ElMessageBox.alert(
      `订单号:${order.out_trade_no}\n\n支付完成后请返回本页并点击“刷新”按钮查看到账状态。`,
      '已跳转支付',
      {
        confirmButtonText: '去刷新订单',
        callback: () => {
          paging.offset = 0
          loadOrders()
        },
      },
    )
  } catch (e: any) {
    if (e?.message) ElMessage.error(e.message)
  }
}

async function cancel(o: rechargeApi.Order) {
  await ElMessageBox.confirm(`确认取消订单 ${o.out_trade_no}？`, '取消订单', { type: 'warning' })
  await rechargeApi.cancelMyOrder(o.id)
  ElMessage.success('已取消')
  loadOrders()
}

const statusColor: Record<string, 'success' | 'info' | 'warning' | 'danger'> = {
  paid: 'success',
  pending: 'warning',
  cancelled: 'info',
  expired: 'info',
  failed: 'danger',
}

const statusLabel: Record<string, string> = {
  paid: '已到账',
  pending: '待支付',
  cancelled: '已取消',
  expired: '已超时',
  failed: '失败',
}

const currentPage = computed<number>({
  get() {
    return Math.floor(paging.offset / paging.limit) + 1
  },
  set(v) {
    paging.offset = (v - 1) * paging.limit
    loadOrders()
  },
})

function priceYuan(fen: number) {
  return (fen / 100).toFixed(2)
}

function openPayUrl(url: string) {
  window.open(url, '_blank', 'noopener,noreferrer')
}

function continuePay(row: rechargeApi.Order) {
  if (!row.pay_url) return
  openPayUrl(row.pay_url)
}

function onStatusChange() {
  paging.offset = 0
  loadOrders()
}

function packageLabel(row: rechargeApi.Order) {
  return row.remark || `#${row.package_id}`
}

onMounted(() => {
  loadPackages()
  loadOrders()
  loadRedeemStatus()
})
</script>

<template>
  <div class="page-container billing-page">
    <div class="card-block balance-card">
      <div class="flex-between">
        <div>
          <div class="section-hint">当前可用积分</div>
          <div class="balance-value">{{ formatCredit(userStore.user?.credit_balance) }}</div>
          <div class="section-hint">冻结 {{ formatCredit(userStore.user?.credit_frozen) }} 积分</div>
        </div>
        <el-button size="small" @click="userStore.fetchMe()">刷新余额</el-button>
      </div>
    </div>

    <div class="card-block redeem-card" v-loading="redeemStatusLoading">
      <div class="flex-between section-header">
        <div>
          <h3>兑换码兑换</h3>
          <div class="section-hint">每个用户仅可兑换一次，兑换成功后积分立即到账。</div>
        </div>
        <el-tag :type="redeemed ? 'success' : 'warning'" size="small">
          {{ redeemed ? '已兑换' : '未兑换' }}
        </el-tag>
      </div>

      <div class="redeem-panel">
        <el-input
          v-model="redeemForm.code"
          placeholder="请输入兑换码"
          :disabled="redeemed"
          clearable
          @keyup.enter="submitRedeem"
        ></el-input>
        <el-button type="primary" :loading="redeemLoading" :disabled="redeemed" @click="submitRedeem">
          立即兑换
        </el-button>
      </div>

      <div v-if="redeemRecord" class="redeem-result">
        <div class="result-title">兑换记录</div>
        <div class="result-grid">
          <div><span>兑换码</span><b>{{ redeemRecord.code }}</b></div>
          <div><span>到账积分</span><b>{{ formatCredit(redeemRecord.credits) }}</b></div>
          <div><span>兑换时间</span><b>{{ formatDateTime(redeemRecord.used_at) }}</b></div>
          <div><span>批次号</span><b>{{ redeemRecord.batch_no || '—' }}</b></div>
        </div>
        <div v-if="redeemRecord.remark" class="result-remark">备注：{{ redeemRecord.remark }}</div>
      </div>
    </div>

    <div class="card-block">
      <div class="flex-between section-header">
        <div>
          <h3>选择充值套餐</h3>
          <div class="section-hint">支付未开放时，可优先使用上方兑换码完成积分充值。</div>
        </div>
        <el-tag v-if="!channelEnabled" type="warning" size="small">支付通道未配置</el-tag>
      </div>

      <el-empty v-if="!loadingPkg && packages.length === 0" description="暂无可用套餐"></el-empty>
      <div v-else class="package-list" v-loading="loadingPkg">
        <el-card v-for="pkg in packages" :key="pkg.id" shadow="hover" class="pkg-card">
          <div class="pkg-name">{{ pkg.name }}</div>
          <div class="pkg-price">¥ <span>{{ priceYuan(pkg.price_cny) }}</span></div>
          <div class="pkg-credit">
            到账 <b>{{ formatCredit(pkg.credits) }}</b> 积分
            <span v-if="pkg.bonus > 0" class="bonus">+赠送 {{ formatCredit(pkg.bonus) }}</span>
          </div>
          <div class="pkg-desc">{{ pkg.description || '—' }}</div>
          <div class="pkg-actions">
            <el-button type="primary" :disabled="!channelEnabled" @click="buy(pkg, 'alipay')">支付宝</el-button>
            <el-button type="success" :disabled="!channelEnabled" @click="buy(pkg, 'wxpay')">微信</el-button>
          </div>
        </el-card>
      </div>
    </div>

    <div class="card-block">
      <div class="flex-between section-header">
        <div>
          <h3>我的订单</h3>
          <div class="section-hint">支持查看充值订单、继续支付或取消待支付订单。</div>
        </div>
        <div class="flex-wrap-gap">
          <el-select
            v-model="paging.status"
            placeholder="状态"
            clearable
            style="width: 130px"
            @change="onStatusChange"
          >
            <el-option label="全部" value=""></el-option>
            <el-option label="待支付" value="pending"></el-option>
            <el-option label="已到账" value="paid"></el-option>
            <el-option label="已取消" value="cancelled"></el-option>
            <el-option label="已超时" value="expired"></el-option>
          </el-select>
          <el-button :loading="loadingOrder" @click="loadOrders">刷新</el-button>
        </div>
      </div>

      <el-empty v-if="!loadingOrder && orders.length === 0" description="暂无订单"></el-empty>
      <div v-else class="orders-list" v-loading="loadingOrder">
        <div v-for="order in orders" :key="order.id" class="order-item">
          <div class="order-main">
            <div class="order-line">
              <span>订单号</span>
              <code>{{ order.out_trade_no }}</code>
            </div>
            <div class="order-line">
              <span>套餐</span>
              <b>{{ packageLabel(order) }}</b>
            </div>
            <div class="order-line">
              <span>金额</span>
              <b>¥ {{ priceYuan(order.price_cny) }}</b>
            </div>
            <div class="order-line">
              <span>积分</span>
              <b>{{ formatCredit(order.credits + order.bonus) }}</b>
            </div>
            <div class="order-line">
              <span>状态</span>
              <el-tag :type="statusColor[order.status] || 'info'" size="small">
                {{ statusLabel[order.status] || order.status }}
              </el-tag>
            </div>
            <div class="order-line">
              <span>创建时间</span>
              <b>{{ formatDateTime(order.created_at) }}</b>
            </div>
          </div>
          <div class="order-actions">
            <el-button v-if="order.status === 'pending' && order.pay_url" type="primary" link @click="continuePay(order)">
              继续支付
            </el-button>
            <el-button v-if="order.status === 'pending'" type="danger" link @click="cancel(order)">
              取消
            </el-button>
            <span v-if="order.status !== 'pending'" class="empty-action">—</span>
          </div>
        </div>
      </div>

      <el-pagination
        style="margin-top: 12px"
        background
        layout="total, prev, pager, next"
        :total="total"
        v-model:current-page="currentPage"
        :page-size="paging.limit"
      ></el-pagination>
    </div>
  </div>
</template>

<style scoped lang="scss">
.billing-page {
  .section-header {
    margin-bottom: 12px;
    align-items: flex-start;

    h3 {
      margin: 0 0 4px;
      font-size: 15px;
    }
  }

  .section-hint {
    font-size: 13px;
    color: var(--el-text-color-secondary);
  }

  .balance-card {
    .balance-value {
      margin: 6px 0;
      font-size: 32px;
      font-weight: 700;
      color: #409eff;
    }
  }

  .redeem-panel {
    display: flex;
    gap: 12px;
    align-items: center;

    :deep(.el-input) {
      flex: 1;
    }
  }

  .redeem-result {
    margin-top: 14px;
    padding: 14px 16px;
    border-radius: 14px;
    background: var(--el-fill-color-light);

    .result-title {
      margin-bottom: 10px;
      font-size: 13px;
      font-weight: 600;
      color: var(--el-text-color-primary);
    }

    .result-grid {
      display: grid;
      grid-template-columns: repeat(2, minmax(0, 1fr));
      gap: 10px 16px;

      span {
        margin-right: 8px;
        color: var(--el-text-color-secondary);
      }

      b {
        font-weight: 600;
        color: var(--el-text-color-primary);
        word-break: break-all;
      }
    }

    .result-remark {
      margin-top: 10px;
      font-size: 13px;
      color: var(--el-text-color-secondary);
    }
  }

  .pkg-col {
    margin-bottom: 16px;
  }

  .pkg-card {
    border-radius: 10px;
    transition: transform .15s;

    &:hover {
      transform: translateY(-2px);
    }

    .pkg-name {
      font-size: 16px;
      font-weight: 600;
    }

    .pkg-price {
      margin: 8px 0 4px;
      font-size: 14px;
      color: #f56c6c;

      span {
        font-size: 28px;
        font-weight: 700;
      }
    }

    .pkg-credit {
      font-size: 14px;
      color: var(--el-text-color-primary);

      .bonus {
        margin-left: 6px;
        font-weight: 600;
        color: #67c23a;
      }
    }

    .pkg-desc {
      min-height: 36px;
      margin: 10px 0;
      font-size: 12px;
      color: var(--el-text-color-secondary);
    }

    .pkg-actions {
      display: flex;
      gap: 8px;
      flex-wrap: wrap;
    }
  }

  code {
    padding: 1px 6px;
    border-radius: 4px;
    background: #f2f3f5;
    font-size: 12px;
  }

  .empty-action {
    color: var(--el-text-color-placeholder);
  }
}

:global(html.dark) .billing-page code {
  background: #1d2026;
}

@media (max-width: 767px) {
  .billing-page {
    .redeem-panel {
      flex-direction: column;
      align-items: stretch;
    }

    .redeem-result .result-grid {
      grid-template-columns: 1fr;
    }
  }
}
</style>