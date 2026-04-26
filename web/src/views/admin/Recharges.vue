<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import * as rechargeApi from '@/api/recharge'
import { formatCredit, formatDateTime } from '@/utils/format'

const tab = ref<'packages' | 'codes' | 'orders'>('packages')

// ---------- packages ----------
const packages = ref<rechargeApi.Package[]>([])
const loadingPkg = ref(false)

async function loadPackages() {
  loadingPkg.value = true
  try {
    const d = await rechargeApi.adminListPackages()
    packages.value = d.items
  } finally {
    loadingPkg.value = false
  }
}

const pkgDialog = reactive({
  visible: false,
  mode: 'create' as 'create' | 'edit',
  form: {
    id: 0,
    name: '',
    price_cny: 100,
    credits: 1_000_000,
    bonus: 0,
    description: '',
    sort: 0,
    enabled: true,
  } as Partial<rechargeApi.Package>,
})

function openCreatePkg() {
  pkgDialog.mode = 'create'
  Object.assign(pkgDialog.form, {
    id: 0,
    name: '',
    price_cny: 100,
    credits: 1_000_000,
    bonus: 0,
    description: '',
    sort: 0,
    enabled: true,
  })
  pkgDialog.visible = true
}

function openEditPkg(p: rechargeApi.Package) {
  pkgDialog.mode = 'edit'
  Object.assign(pkgDialog.form, p)
  pkgDialog.visible = true
}

async function savePkg() {
  const f = pkgDialog.form
  if (!f.name || (f.price_cny ?? 0) <= 0) {
    ElMessage.warning('名称和金额不能为空')
    return
  }
  if (pkgDialog.mode === 'create') {
    await rechargeApi.adminCreatePackage(f)
    ElMessage.success('已创建')
  } else {
    await rechargeApi.adminUpdatePackage(f.id!, f)
    ElMessage.success('已保存')
  }
  pkgDialog.visible = false
  loadPackages()
}

async function deletePkg(p: rechargeApi.Package) {
  await ElMessageBox.confirm(`确认删除套餐【${p.name}】？该操作不可撤销`, '删除套餐', { type: 'warning' })
  await rechargeApi.adminDeletePackage(p.id)
  ElMessage.success('已删除')
  loadPackages()
}

// ---------- codes ----------
const codes = ref<rechargeApi.RedeemCode[]>([])
const codesTotal = ref(0)
const loadingCodes = ref(false)
const codeFilter = reactive({
  status: '' as '' | 'used' | 'unused',
  batch_no: '',
  code: '',
  limit: 20,
  offset: 0,
})

const createCodesDialog = reactive({
  visible: false,
  form: {
    count: 10,
    credits: 1_000_000,
    prefix: 'RC',
    remark: '',
  },
})

const generatedBatch = ref<{ batch_no: string; items: rechargeApi.RedeemCode[] } | null>(null)

async function loadCodes() {
  loadingCodes.value = true
  try {
    const d = await rechargeApi.adminListRedeemCodes({
      status: codeFilter.status || undefined,
      batch_no: codeFilter.batch_no || undefined,
      code: codeFilter.code || undefined,
      limit: codeFilter.limit,
      offset: codeFilter.offset,
    })
    codes.value = d.items
    codesTotal.value = d.total
  } finally {
    loadingCodes.value = false
  }
}

function openCreateCodes() {
  createCodesDialog.visible = true
}

async function generateCodes() {
  const f = createCodesDialog.form
  if (f.count <= 0 || f.credits <= 0) {
    ElMessage.warning('数量和积分必须大于 0')
    return
  }
  const d = await rechargeApi.adminGenerateRedeemCodes({
    count: f.count,
    credits: f.credits,
    prefix: f.prefix || undefined,
    remark: f.remark || undefined,
  })
  generatedBatch.value = { batch_no: d.batch_no, items: d.items }
  createCodesDialog.visible = false
  ElMessage.success(`已生成 ${d.total} 个兑换码`)
  codeFilter.offset = 0
  loadCodes()
}

async function deleteCode(row: rechargeApi.RedeemCode) {
  await ElMessageBox.confirm(`确认删除兑换码 ${row.code}？`, '删除兑换码', { type: 'warning' })
  await rechargeApi.adminDeleteRedeemCode(row.id)
  ElMessage.success('已删除')
  loadCodes()
}

async function copyBatchCodes() {
  if (!generatedBatch.value?.items?.length) return
  const text = generatedBatch.value.items.map((item) => item.code).join('\n')
  await navigator.clipboard.writeText(text)
  ElMessage.success('已复制本批次兑换码')
}

const codesCurrentPage = computed<number>({
  get() {
    return Math.floor(codeFilter.offset / codeFilter.limit) + 1
  },
  set(v) {
    codeFilter.offset = (v - 1) * codeFilter.limit
    loadCodes()
  },
})

// ---------- orders ----------
const orders = ref<rechargeApi.Order[]>([])
const total = ref(0)
const loadingOrd = ref(false)
const filter = reactive({
  user_id: undefined as number | undefined,
  status: '' as '' | 'pending' | 'paid' | 'cancelled' | 'expired' | 'failed',
  limit: 20,
  offset: 0,
})

async function loadOrders() {
  loadingOrd.value = true
  try {
    const d = await rechargeApi.adminListOrders({
      user_id: filter.user_id || undefined,
      status: filter.status || undefined,
      limit: filter.limit,
      offset: filter.offset,
    })
    orders.value = d.items
    total.value = d.total
  } finally {
    loadingOrd.value = false
  }
}

async function forcePaid(o: rechargeApi.Order) {
  if (o.status !== 'pending') {
    ElMessage.warning('只有 pending 状态可以手工入账')
    return
  }
  const { value: pwd } = await ElMessageBox.prompt(
    `请输入管理员密码以确认为订单 ${o.out_trade_no} 强制入账。`,
    '手工入账',
    { type: 'warning', inputType: 'password', confirmButtonText: '确认入账', cancelButtonText: '取消' },
  )
  if (!pwd) return
  await rechargeApi.adminForcePaid(o.id, pwd)
  ElMessage.success('已入账')
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
    return Math.floor(filter.offset / filter.limit) + 1
  },
  set(v) {
    filter.offset = (v - 1) * filter.limit
    loadOrders()
  },
})

function resetCodesAndLoad() {
  codeFilter.offset = 0
  loadCodes()
}

function resetOrdersAndLoad() {
  filter.offset = 0
  loadOrders()
}

function onCodesPageSizeChange() {
  codeFilter.offset = 0
  loadCodes()
}

function onOrdersPageSizeChange() {
  filter.offset = 0
  loadOrders()
}

function priceYuan(fen: number) {
  return (fen / 100).toFixed(2)
}

onMounted(() => {
  loadPackages()
  loadCodes()
  loadOrders()
})
</script>

<template>
  <div class="page-container admin-recharges-page">
    <div class="card-block">
      <el-tabs v-model="tab">
        <el-tab-pane label="套餐管理" name="packages">
          <div class="flex-between section-header">
            <div>
              <h3>充值套餐管理</h3>
              <div class="section-hint">普通用户在“账单与充值”页看到的是启用中的套餐。价格单位：分，积分单位：厘。</div>
            </div>
            <el-button type="primary" @click="openCreatePkg">
              <el-icon><Plus /></el-icon>
              新增套餐
            </el-button>
          </div>

          <el-table :data="packages" stripe v-loading="loadingPkg">
            <el-table-column prop="id" label="ID" width="70" />
            <el-table-column prop="name" label="名称" min-width="160" />
            <el-table-column label="价格" width="110">
              <template #default="{ row }">
                ¥ {{ priceYuan(row.price_cny) }}
              </template>
            </el-table-column>
            <el-table-column label="基础积分" width="120">
              <template #default="{ row }">
                {{ formatCredit(row.credits) }}
              </template>
            </el-table-column>
            <el-table-column label="赠送" width="110">
              <template #default="{ row }">
                {{ formatCredit(row.bonus) }}
              </template>
            </el-table-column>
            <el-table-column prop="sort" label="排序" width="80" />
            <el-table-column label="状态" width="90">
              <template #default="{ row }">
                <el-tag :type="row.enabled ? 'success' : 'info'" size="small">{{ row.enabled ? '启用' : '停用' }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="description" label="描述" show-overflow-tooltip />
            <el-table-column label="操作" width="140" fixed="right">
              <template #default="{ row }">
                <el-button size="small" link @click="openEditPkg(row)">编辑</el-button>
                <el-button size="small" link type="danger" @click="deletePkg(row)">删除</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-tab-pane>

        <el-tab-pane label="兑换码管理" name="codes">
          <div class="flex-between section-header">
            <div>
              <h3>兑换码管理</h3>
              <div class="section-hint">支持批量生成、按批次筛选、查看使用情况。每个用户仅能兑换一次。</div>
            </div>
            <el-button type="primary" @click="openCreateCodes">
              <el-icon><Plus /></el-icon>
              批量生成兑换码
            </el-button>
          </div>

          <div class="filters-panel">
            <el-select v-model="codeFilter.status" placeholder="使用状态" clearable style="width: 140px">
              <el-option label="全部" value="" />
              <el-option label="未使用" value="unused" />
              <el-option label="已使用" value="used" />
            </el-select>
            <el-input v-model="codeFilter.batch_no" placeholder="批次号" clearable style="width: 180px" />
            <el-input v-model="codeFilter.code" placeholder="兑换码关键字" clearable style="width: 220px" />
            <el-button type="primary" :loading="loadingCodes" @click="resetCodesAndLoad">
              <el-icon><Search /></el-icon>
              查询
            </el-button>
          </div>

          <div v-if="generatedBatch" class="generated-batch">
            <div class="generated-head">
              <div>
                <div class="batch-title">最近生成批次：{{ generatedBatch.batch_no }}</div>
                <div class="section-hint">共 {{ generatedBatch.items.length }} 个兑换码，可直接复制发放。</div>
              </div>
              <el-button size="small" @click="copyBatchCodes">复制本批次兑换码</el-button>
            </div>
            <el-scrollbar max-height="180px">
              <div class="batch-codes">
                <code v-for="item in generatedBatch.items" :key="item.code">{{ item.code }}</code>
              </div>
            </el-scrollbar>
          </div>

          <el-table :data="codes" stripe v-loading="loadingCodes">
            <el-table-column prop="id" label="ID" width="70" />
            <el-table-column label="兑换码" min-width="200">
              <template #default="{ row }">
                <code>{{ row.code }}</code>
              </template>
            </el-table-column>
            <el-table-column label="积分" width="120">
              <template #default="{ row }">
                {{ formatCredit(row.credits) }}
              </template>
            </el-table-column>
            <el-table-column prop="batch_no" label="批次号" width="160" />
            <el-table-column label="状态" width="100">
              <template #default="{ row }">
                <el-tag :type="row.used_by ? 'success' : 'info'" size="small">{{ row.used_by ? '已使用' : '未使用' }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="使用用户" width="100">
              <template #default="{ row }">{{ row.used_by || '—' }}</template>
            </el-table-column>
            <el-table-column label="使用时间" width="180">
              <template #default="{ row }">{{ formatDateTime(row.used_at) }}</template>
            </el-table-column>
            <el-table-column label="备注" min-width="180" show-overflow-tooltip>
              <template #default="{ row }">{{ row.remark || '—' }}</template>
            </el-table-column>
            <el-table-column label="创建时间" width="180">
              <template #default="{ row }">{{ formatDateTime(row.created_at) }}</template>
            </el-table-column>
            <el-table-column label="操作" width="120" fixed="right">
              <template #default="{ row }">
                <el-button size="small" link type="danger" :disabled="!!row.used_by" @click="deleteCode(row)">删除</el-button>
              </template>
            </el-table-column>
          </el-table>

          <el-pagination
            style="margin-top: 12px"
            background
            layout="total, prev, pager, next, sizes"
            :total="codesTotal"
            v-model:current-page="codesCurrentPage"
            :page-sizes="[20, 50, 100]"
            v-model:page-size="codeFilter.limit"
            @size-change="onCodesPageSizeChange"
          />
        </el-tab-pane>

        <el-tab-pane label="订单流水" name="orders">
          <div class="flex-between section-header">
            <div>
              <h3>充值订单流水</h3>
              <div class="section-hint">支持按用户和状态筛选，并可对待支付订单执行手工入账。</div>
            </div>
          </div>

          <div class="filters-panel">
            <el-input-number v-model="filter.user_id" :min="1" placeholder="用户 ID" style="width: 140px" />
            <el-select v-model="filter.status" placeholder="状态" clearable style="width: 130px">
              <el-option label="全部" value="" />
              <el-option label="待支付" value="pending" />
              <el-option label="已到账" value="paid" />
              <el-option label="已取消" value="cancelled" />
              <el-option label="已超时" value="expired" />
              <el-option label="失败" value="failed" />
            </el-select>
            <el-button type="primary" @click="resetOrdersAndLoad" :loading="loadingOrd">
              <el-icon><Search /></el-icon>
              查询
            </el-button>
          </div>

          <el-table :data="orders" stripe v-loading="loadingOrd">
            <el-table-column label="订单号" min-width="180">
              <template #default="{ row }"><code>{{ row.out_trade_no }}</code></template>
            </el-table-column>
            <el-table-column prop="user_id" label="用户 ID" width="90" />
            <el-table-column label="金额" width="100">
              <template #default="{ row }">¥ {{ priceYuan(row.price_cny) }}</template>
            </el-table-column>
            <el-table-column label="积分" width="140">
              <template #default="{ row }">{{ formatCredit(row.credits) }} + {{ formatCredit(row.bonus) }}</template>
            </el-table-column>
            <el-table-column prop="pay_method" label="方式" width="90" />
            <el-table-column label="状态" width="90">
              <template #default="{ row }">
                <el-tag :type="statusColor[row.status] || 'info'" size="small">{{ statusLabel[row.status] || row.status }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="上游单号" min-width="160">
              <template #default="{ row }"><span class="mono-text">{{ row.trade_no || '—' }}</span></template>
            </el-table-column>
            <el-table-column label="支付时间" width="180">
              <template #default="{ row }">{{ formatDateTime(row.paid_at) }}</template>
            </el-table-column>
            <el-table-column label="创建时间" width="180">
              <template #default="{ row }">{{ formatDateTime(row.created_at) }}</template>
            </el-table-column>
            <el-table-column label="操作" width="130" fixed="right">
              <template #default="{ row }">
                <el-button v-if="row.status === 'pending'" size="small" link type="warning" @click="forcePaid(row)">手工入账</el-button>
                <span v-else class="empty-action">—</span>
              </template>
            </el-table-column>
          </el-table>

          <el-pagination
            style="margin-top: 12px"
            background
            layout="total, prev, pager, next, sizes"
            :total="total"
            v-model:current-page="currentPage"
            :page-sizes="[20, 50, 100]"
            v-model:page-size="filter.limit"
            @size-change="onOrdersPageSizeChange"
          />
        </el-tab-pane>
      </el-tabs>
    </div>

    <el-dialog v-model="pkgDialog.visible" :title="pkgDialog.mode === 'create' ? '新增套餐' : '编辑套餐'" width="520px">
      <el-form label-width="110px">
        <el-form-item label="名称">
          <el-input v-model="pkgDialog.form.name" />
        </el-form-item>
        <el-form-item label="售价(分)">
          <el-input-number v-model="pkgDialog.form.price_cny" :min="1" style="width: 220px" />
          <span class="form-hint">= ¥ {{ ((pkgDialog.form.price_cny || 0) / 100).toFixed(2) }}</span>
        </el-form-item>
        <el-form-item label="基础积分(厘)">
          <el-input-number v-model="pkgDialog.form.credits" :min="0" style="width: 220px" />
        </el-form-item>
        <el-form-item label="赠送积分(厘)">
          <el-input-number v-model="pkgDialog.form.bonus" :min="0" style="width: 220px" />
        </el-form-item>
        <el-form-item label="排序">
          <el-input-number v-model="pkgDialog.form.sort" :min="0" />
        </el-form-item>
        <el-form-item label="状态">
          <el-switch v-model="pkgDialog.form.enabled" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="pkgDialog.form.description" type="textarea" :rows="2" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="pkgDialog.visible = false">取消</el-button>
        <el-button type="primary" @click="savePkg">保存</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="createCodesDialog.visible" title="批量生成兑换码" width="520px">
      <el-form label-width="110px">
        <el-form-item label="生成数量">
          <el-input-number v-model="createCodesDialog.form.count" :min="1" :max="1000" style="width: 220px" />
        </el-form-item>
        <el-form-item label="兑换积分(厘)">
          <el-input-number v-model="createCodesDialog.form.credits" :min="1" style="width: 220px" />
        </el-form-item>
        <el-form-item label="前缀">
          <el-input v-model="createCodesDialog.form.prefix" maxlength="8" placeholder="默认 RC" style="width: 220px" />
        </el-form-item>
        <el-form-item label="备注">
          <el-input v-model="createCodesDialog.form.remark" type="textarea" :rows="2" placeholder="例如：活动赠送 / 补偿批次" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="createCodesDialog.visible = false">取消</el-button>
        <el-button type="primary" @click="generateCodes">生成</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped lang="scss">
.admin-recharges-page {
  .section-header {
    margin-bottom: 12px;
    align-items: flex-start;

    h3 {
      margin: 0 0 4px;
      font-size: 15px;
    }
  }

  .section-hint,
  .form-hint {
    font-size: 13px;
    color: var(--el-text-color-secondary);
  }

  .form-hint {
    margin-left: 8px;
  }

  .filters-panel {
    display: flex;
    gap: 12px;
    flex-wrap: wrap;
    margin-bottom: 12px;
  }

  .generated-batch {
    margin-bottom: 14px;
    padding: 14px 16px;
    border-radius: 14px;
    background: var(--el-fill-color-light);

    .generated-head {
      display: flex;
      justify-content: space-between;
      gap: 12px;
      margin-bottom: 12px;
      align-items: flex-start;
    }

    .batch-title {
      margin-bottom: 4px;
      font-size: 14px;
      font-weight: 600;
    }

    .batch-codes {
      display: flex;
      flex-wrap: wrap;
      gap: 8px;
    }
  }

  code,
  .mono-text {
    font-family: ui-monospace, Menlo, Consolas, monospace;
  }

  code {
    display: inline-block;
    padding: 2px 8px;
    border-radius: 6px;
    background: #f2f3f5;
    font-size: 12px;
  }

  .empty-action {
    color: var(--el-text-color-placeholder);
  }
}

:global(html.dark) .admin-recharges-page code {
  background: #1d2026;
}

@media (max-width: 767px) {
  .admin-recharges-page {
    .generated-batch .generated-head {
      flex-direction: column;
    }
  }
}
</style>