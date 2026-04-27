<script setup lang="ts">
import { ref, computed, reactive, onMounted, onBeforeUnmount, shallowRef } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  Refresh,
  Check,
  Setting,
  Lock,
  User,
  Connection,
  Wallet,
  Message as MailIcon,
} from '@element-plus/icons-vue'
import {
  listSettings,
  updateSettings,
  reloadSettings,
  sendTestEmail,
  type SettingItem,
} from '@/api/settings'
import { useSiteStore } from '@/stores/site'
import AnnouncementContent from '@/components/AnnouncementContent.vue'
import { Editor, Toolbar } from '@wangeditor/editor-for-vue'
import type { IDomEditor, IEditorConfig, IToolbarConfig } from '@wangeditor/editor'
import '@wangeditor/editor/dist/css/style.css'

const SITE_ANNOUNCEMENT_TITLE = 'site.announcement_title'
const SITE_ANNOUNCEMENT_HTML = 'site.announcement_html'
const SITE_ANNOUNCEMENT_POPUP_ENABLED = 'site.announcement_popup_enabled'
const SITE_ANNOUNCEMENT_VERSION = 'site.announcement_version'
const announcementKeys = new Set<string>([
  SITE_ANNOUNCEMENT_TITLE,
  SITE_ANNOUNCEMENT_HTML,
  SITE_ANNOUNCEMENT_POPUP_ENABLED,
  SITE_ANNOUNCEMENT_VERSION,
])

const loading = ref(false)
const saving = ref(false)
const items = ref<SettingItem[]>([])
const draft = reactive<Record<string, string>>({})
const announcementEditorRef = shallowRef<IDomEditor | null>(null)

const toolbarConfig: Partial<IToolbarConfig> = {
  toolbarKeys: [
    'headerSelect',
    'bold',
    'italic',
    'underline',
    'through',
    'color',
    'bgColor',
    'bulletedList',
    'numberedList',
    'blockquote',
    'insertLink',
    'codeBlock',
    'undo',
    'redo',
  ],
}
const editorConfig: Partial<IEditorConfig> = {
  placeholder: '请输入公告内容，支持 HTML 富文本展示',
  scroll: false,
  autoFocus: false,
}

const tabs = [
  { name: 'site', label: '通用设置', icon: Setting },
  { name: 'auth', label: '安全与认证', icon: Lock },
  { name: 'defaults', label: '用户默认值', icon: User },
  { name: 'gateway', label: '网关服务', icon: Connection },
  { name: 'billing', label: '计费与充值', icon: Wallet },
  { name: 'mail', label: '邮件设置', icon: MailIcon },
] as const
const activeTab = ref<(typeof tabs)[number]['name']>('site')

const grouped = computed(() => {
  const map: Record<string, SettingItem[]> = {
    site: [], auth: [], defaults: [], gateway: [], billing: [], mail: [],
  }
  for (const it of items.value) {
    const cat = it.category === 'limit' ? 'defaults' : it.category
    ;(map[cat] ||= []).push(it)
  }
  for (const k of Object.keys(map)) map[k].sort((a, b) => a.key.localeCompare(b.key))
  return map
})

const hasAnnouncementSection = computed(() => items.value.some((it) => announcementKeys.has(it.key)))
const announcementTitle = computed({
  get: () => draft[SITE_ANNOUNCEMENT_TITLE] ?? '',
  set: (value: string) => { draft[SITE_ANNOUNCEMENT_TITLE] = value ?? '' },
})
const announcementHTML = computed({
  get: () => draft[SITE_ANNOUNCEMENT_HTML] ?? '',
  set: (value: string) => { draft[SITE_ANNOUNCEMENT_HTML] = value ?? '' },
})
const announcementPopupEnabled = computed({
  get: () => draft[SITE_ANNOUNCEMENT_POPUP_ENABLED] === 'true',
  set: (value: boolean) => { draft[SITE_ANNOUNCEMENT_POPUP_ENABLED] = value ? 'true' : 'false' },
})
const announcementVersion = computed(() => draft[SITE_ANNOUNCEMENT_VERSION] || '0')

const dirtyCount = computed(() => {
  let n = 0
  for (const it of items.value) {
    if (String(draft[it.key] ?? '') !== String(it.value)) n++
  }
  return n
})

async function load() {
  loading.value = true
  try {
    const d = await listSettings()
    items.value = d.items
    for (const it of d.items) draft[it.key] = it.value
  } finally {
    loading.value = false
  }
}

function reset() {
  for (const it of items.value) draft[it.key] = it.value
  ElMessage.info('已重置为服务端当前值')
}

function isBool(it: SettingItem) { return it.type === 'bool' }
function isInt(it: SettingItem) { return it.type === 'int' }
function isFloat(it: SettingItem) { return it.type === 'float' }
function inputType(it: SettingItem) {
  if (it.type === 'email') return 'email'
  if (it.type === 'url') return 'url'
  return 'text'
}
function isAnnouncementField(it: SettingItem) { return announcementKeys.has(it.key) }
function visibleItems(tabName: (typeof tabs)[number]['name']) {
  return (grouped.value[tabName] || []).filter((it) => !(tabName === 'site' && isAnnouncementField(it)))
}
function onEditorCreated(editor: IDomEditor) {
  announcementEditorRef.value = editor
}

async function save() {
  const diff: Record<string, string> = {}
  for (const it of items.value) {
    const v = draft[it.key] ?? ''
    if (String(v) !== String(it.value)) diff[it.key] = String(v)
  }
  if (Object.keys(diff).length === 0) {
    ElMessage.info('没有需要保存的修改')
    return
  }
  saving.value = true
  try {
    await updateSettings(diff)
    ElMessage.success(`已保存 ${Object.keys(diff).length} 项`)
    await load()
    await useSiteStore().refresh()
  } finally {
    saving.value = false
  }
}

async function doReload() {
  await ElMessageBox.confirm('从数据库强制重载最新值到内存缓存?', '确认', {
    type: 'warning',
  }).catch(() => 'cancel')
  try {
    await reloadSettings()
    ElMessage.success('已重载')
    await load()
  } catch { /* 拦截器已处理 */ }
}

const mailDlg = ref(false)
const mailTo = ref('')
const mailSending = ref(false)
async function submitTestMail() {
  if (!mailTo.value) {
    ElMessage.warning('请输入收件邮箱')
    return
  }
  mailSending.value = true
  try {
    await sendTestEmail(mailTo.value)
    ElMessage.success('测试邮件已发出')
    mailDlg.value = false
  } catch { /* 拦截器已处理 */ } finally {
    mailSending.value = false
  }
}

onMounted(load)
onBeforeUnmount(() => announcementEditorRef.value?.destroy())
</script>

<template>
  <div class="page-container">
    <div class="card-block" v-loading="loading">
      <div class="flex-between settings-head">
        <div>
          <div class="page-title" style="margin:0">系统设置</div>
          <div class="settings-subtitle">
            所有修改在点击"保存修改"后立即生效,无需重启服务
          </div>
        </div>
        <div class="flex-wrap-gap">
          <el-button :icon="Refresh" @click="doReload">强制重载</el-button>
          <el-button :icon="MailIcon" @click="mailDlg = true">发测试邮件</el-button>
          <el-button :disabled="dirtyCount === 0" @click="reset">重置</el-button>
          <el-button
            type="primary"
            :icon="Check"
            :loading="saving"
            @click="save"
          >
            保存修改<span v-if="dirtyCount > 0"> ({{ dirtyCount }})</span>
          </el-button>
        </div>
      </div>

      <el-tabs v-model="activeTab" class="settings-tabs">
        <el-tab-pane v-for="t in tabs" :key="t.name" :name="t.name">
          <template #label>
            <span class="tab-label">
              <el-icon><component :is="t.icon" /></el-icon>
              <span>{{ t.label }}</span>
            </span>
          </template>

          <div class="tab-body">
            <el-empty
              v-if="!grouped[t.name] || grouped[t.name].length === 0"
              description="暂无可配置项"
            />
            <el-form
              v-else
              label-width="170px"
              label-position="right"
              class="setting-form"
            >
              <el-form-item v-if="t.name === 'site' && hasAnnouncementSection" label="公告管理">
                <div class="announcement-panel">
                  <div class="announcement-panel-head">
                    <div>
                      <div class="announcement-panel-title">用户总览公告</div>
                      <div class="hint">保存后会同步到用户总览页的公告按钮与弹窗。</div>
                    </div>
                    <el-tag effect="plain" size="small">版本 {{ announcementVersion }}</el-tag>
                  </div>

                  <el-input
                    v-model="announcementTitle"
                    placeholder="请输入公告标题"
                    clearable
                    maxlength="80"
                    show-word-limit
                    class="announcement-title-input"
                  />

                  <div class="announcement-toolbar-card">
                    <Toolbar
                      v-if="announcementEditorRef"
                      :editor="announcementEditorRef"
                      :default-config="toolbarConfig"
                      mode="default"
                    />
                    <Editor
                      v-model="announcementHTML"
                      :default-config="editorConfig"
                      mode="default"
                      class="announcement-editor"
                      @onCreated="onEditorCreated"
                    />
                  </div>

                  <div class="announcement-options">
                    <el-switch v-model="announcementPopupEnabled" />
                    <div>
                      <div class="announcement-option-title">自动弹窗</div>
                      <div class="hint">开启后，用户进入“个人总览”时会自动看到公告；关闭后仅保留手动查看按钮。</div>
                    </div>
                  </div>

                  <div class="announcement-preview-card">
                    <div class="announcement-preview-head">
                      <span>前台预览</span>
                      <el-tag size="small" effect="plain">{{ announcementPopupEnabled ? '自动弹窗开启' : '仅按钮查看' }}</el-tag>
                    </div>
                    <div v-if="announcementTitle" class="announcement-preview-title">{{ announcementTitle }}</div>
                    <AnnouncementContent :html="announcementHTML" empty-text="暂无公告内容" />
                  </div>
                </div>
              </el-form-item>

              <el-form-item
                v-for="it in visibleItems(t.name)"
                :key="it.key"
                :label="it.label || it.key"
              >
                <div class="field-wrap">
                  <el-switch
                    v-if="isBool(it)"
                    :model-value="draft[it.key] === 'true'"
                    @update:model-value="(v) => (draft[it.key] = v ? 'true' : 'false')"
                  />
                  <el-input-number
                    v-else-if="isInt(it)"
                    :model-value="Number(draft[it.key] || 0)"
                    :min="0"
                    :controls-position="'right'"
                    style="width: 240px"
                    @update:model-value="(v) => (draft[it.key] = String(v ?? 0))"
                  />
                  <el-input-number
                    v-else-if="isFloat(it)"
                    :model-value="Number(draft[it.key] || 0)"
                    :min="0"
                    :max="1"
                    :step="0.05"
                    :precision="2"
                    :controls-position="'right'"
                    style="width: 240px"
                    @update:model-value="(v) => (draft[it.key] = String(v ?? 0))"
                  />
                  <el-input
                    v-else
                    v-model="draft[it.key]"
                    :placeholder="it.desc || it.label"
                    :type="inputType(it)"
                    clearable
                    style="max-width: 520px"
                  />
                  <div v-if="it.desc" class="hint">{{ it.desc }}</div>
                </div>
              </el-form-item>
            </el-form>
          </div>
        </el-tab-pane>
      </el-tabs>
    </div>

    <el-dialog v-model="mailDlg" title="发送 SMTP 测试邮件" width="420px">
      <el-form label-width="80px">
        <el-form-item label="收件人">
          <el-input v-model="mailTo" placeholder="your@mail.com" type="email" clearable />
        </el-form-item>
        <div style="font-size:12px;color:var(--el-text-color-secondary)">
          使用 <code>configs/config.yaml</code> 的 SMTP 配置发送;未配置时会直接失败。
        </div>
      </el-form>
      <template #footer>
        <el-button @click="mailDlg = false">取消</el-button>
        <el-button type="primary" :loading="mailSending" @click="submitTestMail">发送</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.settings-head {
  margin-bottom: 4px;
}
.settings-subtitle {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  margin-top: 4px;
}

.settings-tabs {
  margin-top: 8px;
}
.settings-tabs :deep(.el-tabs__header) {
  margin-bottom: 16px;
}
.tab-label {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.tab-body {
  padding-top: 4px;
}
.setting-form .el-form-item {
  margin-bottom: 18px;
}
.field-wrap {
  width: 100%;
}
.hint {
  margin-top: 4px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
  line-height: 1.5;
}

.announcement-panel {
  width: 100%;
  display: flex;
  flex-direction: column;
  gap: 14px;
}
.announcement-panel-head,
.announcement-options,
.announcement-preview-head {
  display: flex;
  justify-content: space-between;
  gap: 12px;
}
.announcement-panel-head {
  align-items: center;
}
.announcement-panel-title,
.announcement-option-title {
  font-weight: 600;
  color: var(--el-text-color-primary);
}
.announcement-toolbar-card,
.announcement-preview-card {
  border: 1px solid var(--el-border-color);
  border-radius: 16px;
  overflow: hidden;
  background: var(--el-bg-color);
}
.announcement-toolbar-card :deep(.w-e-toolbar) {
  border: 0;
  border-bottom: 1px solid var(--el-border-color);
  background: var(--el-fill-color-lighter);
}
.announcement-toolbar-card :deep(.w-e-bar-item button) {
  color: var(--el-text-color-primary);
}
.announcement-toolbar-card :deep(.w-e-text-container) {
  background: transparent;
}
.announcement-editor {
  min-height: 280px;
}
.announcement-editor :deep(.w-e-text-container [data-slate-editor]) {
  min-height: 240px;
  padding: 14px;
}
.announcement-options {
  align-items: flex-start;
}
.announcement-preview-card {
  padding: 16px;
}
.announcement-preview-head {
  align-items: center;
  margin-bottom: 12px;
}
.announcement-preview-title {
  margin-bottom: 12px;
  font-size: 18px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}

@media (max-width: 640px) {
  .setting-form :deep(.el-form-item__label) {
    width: auto !important;
    padding-right: 8px !important;
    line-height: 1.5;
  }
  .announcement-panel-head,
  .announcement-options,
  .announcement-preview-head {
    flex-direction: column;
    align-items: flex-start;
  }
}
</style>