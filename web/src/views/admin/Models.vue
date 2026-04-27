<script setup lang="ts">
import { ref, reactive, onMounted, computed } from 'vue'
import { ElMessage, ElMessageBox, type FormInstance, type FormRules } from 'element-plus'
import { Plus } from '@element-plus/icons-vue'
import * as statsApi from '@/api/stats'
import { CREDITS_PER_UNIT, formatCredit } from '@/utils/format'
import { useUserStore } from '@/stores/user'
import { ENABLE_CHAT_MODEL } from '@/config/feature'

const userStore = useUserStore()
const canWrite = computed(() => userStore.hasPerm('model:write'))

const loading = ref(false)
const rows = ref<statsApi.Model[]>([])
// feature flag 关掉时默认把筛选固定到 image,表格、弹窗里也就看不到 chat 模型;
// 库里如果已经有 chat 模型历史数据,仍然保留,只是 UI 入口不暴露。
const filterType = ref<string>(ENABLE_CHAT_MODEL ? '' : 'image')

async function load() {
  loading.value = true
  try {
    const d = await statsApi.listModels()
    rows.value = d.items
  } finally { loading.value = false }
}

const filteredRows = computed(() =>
  filterType.value ? rows.value.filter((r) => r.type === filterType.value) : rows.value,
)

/**
 * 价目说明:
 *   chat  模型: input_price_per_1m / output_price_per_1m / cache_read_price_per_1m
 *              后端存储单位是"厘",管理端按"积分"录入和展示。
 *   image 模型: image_price_per_call 每张图价格,后端存"厘",管理端按"积分"录入和展示。
 */
function perMillion(c: number) { return formatCredit(c) }
function perImage(c: number)   { return formatCredit(c) }
function fromCredits(c: number) { return Number((c / CREDITS_PER_UNIT).toFixed(4)) }
function toCredits(v: number) { return Math.round((Number(v) || 0) * CREDITS_PER_UNIT) }

// ---------- 新增 / 编辑 ----------
type EditState = 'create' | 'edit'
const dlgVisible = ref(false)
const dlgState = ref<EditState>('create')
const dlgLoading = ref(false)
const formRef = ref<FormInstance>()

const emptyForm = (): statsApi.ModelUpsert & { id: number } => ({
  id: 0,
  slug: '',
  type: ENABLE_CHAT_MODEL ? 'chat' : 'image',
  upstream_model_slug: '',
  input_price_per_1m: 0,
  output_price_per_1m: 0,
  cache_read_price_per_1m: 0,
  image_price_per_call: 0,
  description: '',
  enabled: true,
})
const form = reactive(emptyForm())

const rules: FormRules = {
  slug: [
    { required: true, message: '请输入 slug', trigger: 'blur' },
    { pattern: /^[A-Za-z][A-Za-z0-9._\-]{1,63}$/, message: '字母开头,2-64 位字母/数字/点/下划线/短横', trigger: 'blur' },
  ],
  type: [{ required: true, message: '请选择类型', trigger: 'change' }],
  upstream_model_slug: [{ required: true, message: '上游 slug 必填', trigger: 'blur' }],
}

function openCreate() {
  Object.assign(form, emptyForm())
  dlgState.value = 'create'
  dlgVisible.value = true
}
function openEdit(row: statsApi.Model) {
  Object.assign(form, {
    id: row.id,
    slug: row.slug,
    type: row.type as 'chat' | 'image',
    upstream_model_slug: row.upstream_model_slug,
    input_price_per_1m: fromCredits(row.input_price_per_1m),
    output_price_per_1m: fromCredits(row.output_price_per_1m),
    cache_read_price_per_1m: fromCredits(row.cache_read_price_per_1m),
    image_price_per_call: fromCredits(row.image_price_per_call),
    description: row.description,
    enabled: row.enabled,
  })
  dlgState.value = 'edit'
  dlgVisible.value = true
}

async function submit() {
  if (!formRef.value) return
  const valid = await formRef.value.validate().catch(() => false)
  if (!valid) return
  const payload: statsApi.ModelUpsert = {
    slug: form.slug,
    type: form.type,
    upstream_model_slug: form.upstream_model_slug,
    input_price_per_1m: toCredits(form.input_price_per_1m),
    output_price_per_1m: toCredits(form.output_price_per_1m),
    cache_read_price_per_1m: toCredits(form.cache_read_price_per_1m),
    image_price_per_call: toCredits(form.image_price_per_call),
    description: form.description,
    enabled: form.enabled,
  }
  dlgLoading.value = true
  try {
    if (dlgState.value === 'create') {
      await statsApi.createModel(payload)
      ElMessage.success('新增成功')
    } else {
      await statsApi.updateModel(form.id, payload)
      ElMessage.success('保存成功')
    }
    dlgVisible.value = false
    load()
  } finally { dlgLoading.value = false }
}

async function onToggleEnabled(row: statsApi.Model) {
  try {
    await statsApi.setModelEnabled(row.id, !row.enabled)
    row.enabled = !row.enabled
    ElMessage.success(row.enabled ? '已启用' : '已禁用')
  } catch { /* 拦截器已提示 */ }
}

async function onDelete(row: statsApi.Model) {
  await ElMessageBox.confirm(
    `确定删除模型 "${row.slug}" 吗?已发生的用量日志不会被清除。`,
    '删除确认',
    { type: 'warning', confirmButtonText: '删除', cancelButtonText: '取消' },
  ).catch(() => null).then(async (r) => {
    if (!r) return
    await statsApi.deleteModel(row.id)
    ElMessage.success('已删除')
    load()
  })
}

onMounted(load)
</script>

<template>
  <div class="page-container">
    <div class="card-block">
      <div class="flex-between" style="margin-bottom:12px">
        <div>
          <h2 class="page-title" style="margin:0">模型配置</h2>
          <div style="color:var(--el-text-color-secondary);font-size:13px;margin-top:4px">
            定义对外 slug 与上游模型的映射关系,设置每张图 / 每 1M tokens 的计费倍率;列表按积分展示,编辑时也按积分录入,提交时自动换算成后端存储的厘。
          </div>
        </div>
        <div class="flex-wrap-gap">
          <el-radio-group v-model="filterType" size="small">
            <el-radio-button v-if="ENABLE_CHAT_MODEL" label="">全部</el-radio-button>
            <el-radio-button v-if="ENABLE_CHAT_MODEL" label="chat">对话</el-radio-button>
            <el-radio-button label="image">生图</el-radio-button>
          </el-radio-group>
          <el-button
            v-if="canWrite"
            type="primary"
            :icon="Plus"
            @click="openCreate"
          >新增模型</el-button>
        </div>
      </div>

      <el-table v-loading="loading" :data="filteredRows" stripe>
        <el-table-column prop="id" label="ID" width="70" />
        <el-table-column prop="slug" label="对外 slug" min-width="170">
          <template #default="{ row }">
            <code>{{ row.slug }}</code>
          </template>
        </el-table-column>
        <el-table-column label="类型" width="90">
          <template #default="{ row }">
            <el-tag :type="row.type === 'image' ? 'warning' : 'primary'" size="small">
              {{ row.type === 'image' ? '生图' : '对话' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="upstream_model_slug" label="上游 slug" min-width="170" show-overflow-tooltip>
          <template #default="{ row }"><code>{{ row.upstream_model_slug }}</code></template>
        </el-table-column>
        <el-table-column v-if="ENABLE_CHAT_MODEL" label="价目(chat)" min-width="220">
          <template #default="{ row }">
            <div v-if="row.type === 'chat'" style="font-size:12px;line-height:1.5">
              <div>输入 <b>{{ perMillion(row.input_price_per_1m) }}</b> / 1M tok</div>
              <div>输出 <b>{{ perMillion(row.output_price_per_1m) }}</b> / 1M tok</div>
              <div v-if="row.cache_read_price_per_1m > 0">
                缓存读 <b>{{ perMillion(row.cache_read_price_per_1m) }}</b> / 1M tok
              </div>
            </div>
            <span v-else style="color:var(--el-text-color-secondary)">-</span>
          </template>
        </el-table-column>
        <el-table-column label="价目(image)" width="160">
          <template #default="{ row }">
            <span v-if="row.type === 'image'">
              <b>{{ perImage(row.image_price_per_call) }}</b> / 张
            </span>
            <span v-else style="color:var(--el-text-color-secondary)">-</span>
          </template>
        </el-table-column>
        <el-table-column prop="description" label="说明" min-width="180" show-overflow-tooltip />
        <el-table-column label="状态" width="90">
          <template #default="{ row }">
            <el-switch
              v-if="canWrite"
              :model-value="row.enabled"
              inline-prompt
              active-text="启"
              inactive-text="停"
              @change="onToggleEnabled(row)"
            />
            <el-tag v-else :type="row.enabled ? 'success' : 'info'" size="small">
              {{ row.enabled ? '启用' : '禁用' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column v-if="canWrite" label="操作" width="140" fixed="right">
          <template #default="{ row }">
            <el-button size="small" link type="primary" @click="openEdit(row)">编辑</el-button>
            <el-button size="small" link type="danger" @click="onDelete(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <el-dialog
      v-model="dlgVisible"
      :title="dlgState === 'create' ? '新增模型' : `编辑模型 · ${form.slug}`"
      width="640px"
      destroy-on-close
    >
      <el-form ref="formRef" :model="form" :rules="rules" label-width="160px">
        <el-form-item label="对外 slug" prop="slug">
          <el-input
            v-model="form.slug"
            :disabled="dlgState === 'edit'"
            placeholder="例如 gpt-5.1 / gpt-image-2"
          />
          <div class="hint">
            创建后不可修改;这是
            <template v-if="ENABLE_CHAT_MODEL">/v1/chat/completions 与 </template>
            /v1/images/generations 中 model 字段传入的值。
          </div>
        </el-form-item>
        <el-form-item label="类型" prop="type">
          <el-radio-group v-model="form.type">
            <el-radio-button v-if="ENABLE_CHAT_MODEL" label="chat">对话 chat</el-radio-button>
            <el-radio-button label="image">生图 image</el-radio-button>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="上游 slug" prop="upstream_model_slug">
          <el-input v-model="form.upstream_model_slug" placeholder="上游 chatgpt.com 实际模型名" />
        </el-form-item>

        <template v-if="form.type === 'chat'">
          <el-form-item label="输入 / 1M tok(积分)">
            <el-input-number v-model="form.input_price_per_1m" :min="0" :step="0.1" :precision="4" style="width:100%" />
          </el-form-item>
          <el-form-item label="输出 / 1M tok(积分)">
            <el-input-number v-model="form.output_price_per_1m" :min="0" :step="0.1" :precision="4" style="width:100%" />
          </el-form-item>
          <el-form-item label="缓存读 / 1M tok(积分)">
            <el-input-number v-model="form.cache_read_price_per_1m" :min="0" :step="0.1" :precision="4" style="width:100%" />
            <div class="hint">可选。0 表示不走缓存计价；这里按积分填写，提交时会自动换算成厘。</div>
          </el-form-item>
        </template>
        <template v-else>
          <el-form-item label="每张图(积分)">
            <el-input-number v-model="form.image_price_per_call" :min="0" :step="0.1" :precision="4" style="width:100%" />
            <div class="hint">例如 10 表示每张图扣 10 积分。</div>
          </el-form-item>
        </template>

        <el-form-item label="描述">
          <el-input v-model="form.description" maxlength="255" show-word-limit />
        </el-form-item>
        <el-form-item label="启用">
          <el-switch v-model="form.enabled" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dlgVisible = false">取消</el-button>
        <el-button type="primary" :loading="dlgLoading" @click="submit">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
code {
  background: #f2f3f5;
  padding: 1px 6px;
  border-radius: 4px;
  font-family: ui-monospace, Menlo, Consolas, monospace;
  font-size: 12px;
}
:global(html.dark) code { background: #1d2026; }
.hint { font-size: 12px; color: var(--el-text-color-secondary); margin-top: 4px; line-height: 1.4; }
</style>
