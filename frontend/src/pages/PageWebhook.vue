<script setup lang="ts">
import { inject, onMounted, ref } from 'vue'
import { appContextKey } from '../app/context'
import { backend, type WebhookConfig, type WebhookItem } from '../wailsbridge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'

const injected = inject(appContextKey)
if (!injected) throw new Error('chatlog not provided')
const app = injected

const cfg = ref<WebhookConfig>({ host: '', delayMs: 0, items: [] })

async function load() {
  try {
    cfg.value = await backend.GetWebhookConfig()
    if (!cfg.value.items) cfg.value.items = []
  }
  catch (e) {
    app.feedback.toast('读取失败', String(e))
  }
}

function addItem() {
  cfg.value.items.push({
    description: '',
    type: 'message',
    url: '',
    talker: '',
    sender: '',
    keyword: '',
    disabled: false,
  })
}

async function removeItem(index: number) {
  const ok = await app.feedback.confirm({
    title: '删除规则',
    message: '确定删除该规则？此操作将从配置中移除，保存后生效。',
    confirmText: '删除',
    cancelText: '取消',
    danger: true,
  })
  if (!ok) return
  cfg.value.items.splice(index, 1)
}

function normalizeItem(it: WebhookItem) {
  it.type = 'message'
  it.description = (it.description || '').trim()
  it.url = (it.url || '').trim()
  it.talker = (it.talker || '').trim()
  it.sender = (it.sender || '').trim()
  it.keyword = (it.keyword || '').trim()
}

function validateItems(items: WebhookItem[]) {
  for (let i = 0; i < items.length; i++) {
    const item = items[i]
    if (!item.url) {
      app.feedback.toast('校验失败', `第 ${i + 1} 条规则缺少 URL`)
      return false
    }
    if (!item.talker) {
      app.feedback.toast('校验失败', `第 ${i + 1} 条规则缺少 Talker`)
      return false
    }
  }
  return true
}

async function save() {
  const ok = await app.feedback.confirm({
    title: '保存 Webhook 配置',
    message: '确认保存并立即应用当前配置？',
    confirmText: '保存',
    cancelText: '取消',
  })
  if (!ok) return
  const next: WebhookConfig = {
    host: (cfg.value.host || '').trim(),
    delayMs: Number(cfg.value.delayMs || 0),
    items: cfg.value.items.map(x => ({ ...x })),
  }
  for (const it of next.items) normalizeItem(it)
  if (!validateItems(next.items)) return
  await app.run(() => backend.SetWebhookConfig(next), '已保存 Webhook 配置')
}

onMounted(() => {
  void load()
})
</script>

<template>
  <div class="space-y-8">
    <!-- 页面头部操作栏 -->
    <header class="sticky top-4 z-20 flex flex-col gap-4 rounded-xl border border-border/40 bg-card/75 px-5 py-4 shadow-sm backdrop-blur-md md:flex-row md:items-center md:justify-between">
      <div class="space-y-1">
        <div class="text-[10px] font-bold uppercase tracking-[0.2em] text-muted-foreground/70">Page Actions</div>
        <p class="text-sm text-muted-foreground">基础配置和转发规则共用同一份配置，修改后需统一保存生效。</p>
      </div>
      <div class="flex items-center gap-3">
        <Button variant="outline" class="h-10 gap-2 px-4 text-xs font-bold uppercase tracking-widest" @click="load">
          <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M21 12a9 9 0 1 1-9-9c2.52 0 4.93 1 6.74 2.74L21 8" />
            <path d="M21 3v5h-5" />
          </svg>
          Refresh
        </Button>
        <Button class="h-10 gap-2 px-6 text-xs font-bold uppercase tracking-widest shadow-lg shadow-primary/20" @click="save">
          <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M19 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11l5 5v11a2 2 0 0 1-2 2z" />
            <polyline points="17 21 17 13 7 13 7 21" />
            <polyline points="7 3 7 8 15 8" />
          </svg>
          Save
        </Button>
      </div>
    </header>

    <div class="grid gap-8">
      <section class="space-y-4">
        <div class="flex items-center gap-4 border-b border-border/40 pb-4">
          <div class="flex size-8 items-center justify-center rounded-lg bg-primary/10 text-[10px] font-bold text-primary">01</div>
          <div class="text-xs font-bold uppercase tracking-[0.2em] text-foreground/70">环境配置 / CONFIGURATION</div>
        </div>

        <Card class="border-border/60 bg-card/50 shadow-sm overflow-hidden">
          <CardHeader class="bg-muted/5 border-b border-border/40 pb-5">
            <div class="space-y-1">
              <CardTitle class="text-[15px] font-bold tracking-tight">基础配置</CardTitle>
              <CardDescription class="text-xs text-muted-foreground/70">定义消息推送中资源文件的基础地址与队列延迟。</CardDescription>
            </div>
          </CardHeader>
          <CardContent class="grid gap-8 p-6 md:grid-cols-2">
            <div class="space-y-3">
              <div class="flex items-center justify-between">
                <div class="text-[10px] font-bold uppercase tracking-widest text-muted-foreground/80">资源地址 / Host Address</div>
                <Badge variant="outline" class="h-5 bg-background/50 font-mono text-[9px] uppercase">Required</Badge>
              </div>
              <Input v-model="cfg.host" class="h-10 font-mono text-sm bg-background/50" placeholder="localhost:5030" />
              <p class="text-[10px] text-muted-foreground/60 italic leading-relaxed">推送消息中图片、文件等静态资源的访问 Host，通常指向 Chatlog 服务地址。</p>
            </div>
            <div class="space-y-3">
              <div class="flex items-center justify-between">
                <div class="text-[10px] font-bold uppercase tracking-widest text-muted-foreground/80">队列延迟 / Delay (ms)</div>
                <Badge variant="outline" class="h-5 bg-background/50 font-mono text-[9px] uppercase">ms</Badge>
              </div>
              <Input v-model.number="cfg.delayMs" type="number" min="0" step="100" class="h-10 font-mono text-sm bg-background/50" />
              <p class="text-[10px] text-muted-foreground/60 italic leading-relaxed">捕获数据库变更后触发推送的等待时间，防止数据库锁定或资源未完全同步。</p>
            </div>
          </CardContent>
        </Card>
      </section>

      <section class="space-y-4">
        <div class="flex items-center gap-4 border-b border-border/40 pb-4">
          <div class="flex size-8 items-center justify-center rounded-lg bg-primary/10 text-[10px] font-bold text-primary">02</div>
          <div class="text-xs font-bold uppercase tracking-[0.2em] text-foreground/70">转发规则 / RULES ENGINE</div>
        </div>

        <Card class="border-border/60 bg-card/50 shadow-sm overflow-hidden">
          <CardHeader class="bg-muted/5 border-b border-border/40 p-5">
            <div class="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
              <div class="space-y-1">
                <CardTitle class="text-[15px] font-bold tracking-tight">规则列表</CardTitle>
                <CardDescription class="text-xs text-muted-foreground/70 flex items-center gap-2">
                  <span class="inline-block size-1.5 rounded-full bg-amber-500 animate-pulse"></span>
                  定义匹配条件并指定目标 URL。修改后需点击网页顶部按钮保存生效。
                </CardDescription>
              </div>
              <div class="flex items-center gap-2">
                <Badge variant="outline" class="h-7 px-3 bg-background/50 text-[10px] font-bold uppercase tracking-widest border-border/60">Total: {{ cfg.items.length }}</Badge>
                <Button variant="outline" size="sm" class="h-8 gap-2 px-3 text-[10px] font-bold uppercase tracking-widest border-border/60 shadow-sm hover:bg-background" @click="addItem">
                  <svg xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                    <line x1="12" y1="5" x2="12" y2="19" />
                    <line x1="5" y1="12" x2="19" y2="12" />
                  </svg>
                  Add Rule
                </Button>
              </div>
            </div>
          </CardHeader>

          <CardContent class="space-y-6 p-6">
            <div v-if="cfg.items.length === 0" class="flex flex-col items-center justify-center rounded-2xl border border-dashed border-border/60 bg-background/20 py-16 text-center">
              <div class="mb-4 flex size-12 items-center justify-center rounded-full bg-muted/30">
                <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="opacity-20">
                  <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z" />
                </svg>
              </div>
              <div class="text-sm font-bold tracking-tight text-foreground/80">暂无推送规则 (No Rules)</div>
              <p class="mt-1 text-xs text-muted-foreground max-w-[240px]">添加新的转发规则并保存，即可开始监听并推送聊天动态。</p>
              <Button variant="link" size="sm" class="mt-4 text-xs font-bold" @click="addItem">立即添加规则</Button>
            </div>

            <div v-else class="space-y-6">
              <Card
                v-for="(it, idx) in cfg.items"
                :key="idx"
                class="group relative overflow-hidden border-border/40 bg-background/40 transition-all hover:bg-background/60 hover:shadow-md"
              >
                <div class="flex items-center justify-between bg-muted/20 px-4 pt-4 pb-2">
                  <div class="flex items-center gap-3">
                    <Badge variant="outline" class="h-5 border-border/60 bg-background/50 font-mono text-[9px] font-bold">#{{ (idx + 1).toString().padStart(2, '0') }}</Badge>
                    <span class="text-[10px] font-bold uppercase tracking-[0.2em] text-muted-foreground/70">规则 {{ idx + 1 }}</span>
                  </div>
                  <div class="flex items-center gap-2">
                    <Button
                      variant="ghost"
                      size="sm"
                      class="h-7 gap-2 px-3 text-[10px] font-bold uppercase tracking-tight hover:bg-background"
                      @click="it.disabled = !it.disabled"
                    >
                      <div :class="['size-1.5 rounded-full transition-all', it.disabled ? 'bg-muted-foreground/40' : 'bg-emerald-500 shadow-[0_0_8px_rgba(16,185,129,0.5)]']" />
                      <span :class="it.disabled ? 'text-muted-foreground' : 'text-foreground'">{{ it.disabled ? 'Inactive' : 'Active' }}</span>
                    </Button>
                    <div class="h-3 w-px bg-border/40 mx-1"></div>
                    <Button
                      variant="ghost"
                      size="sm"
                      class="h-7 px-3 text-[10px] font-bold uppercase tracking-tight text-muted-foreground/60 hover:bg-destructive/10 hover:text-destructive transition-colors"
                      @click="removeItem(idx)"
                    >
                      Delete
                    </Button>
                  </div>
                </div>

                <CardContent class="grid gap-6 px-4 pb-4 pt-1">
                  <div class="grid gap-4 md:grid-cols-2">
                    <div class="space-y-2 md:col-span-2">
                      <label class="text-[10px] font-bold uppercase tracking-widest text-muted-foreground/60">地址 / Target URL</label>
                      <Input v-model="it.url" class="h-9 font-mono text-xs" placeholder="http://127.0.0.1:3000/api/v1/webhook" />
                    </div>
                  </div>
                  <div class="grid gap-4 md:grid-cols-3">
                    <div class="space-y-2">
                      <label class="text-[10px] font-bold uppercase tracking-widest text-muted-foreground/60">对象 / Talker (WXID)</label>
                      <Input v-model="it.talker" class="h-9 font-mono text-xs" placeholder="群组 ID 或用户 ID" />
                    </div>
                    <div class="space-y-2">
                      <label class="text-[10px] font-bold uppercase tracking-widest text-muted-foreground/60">发送者 / Sender (Optional)</label>
                      <Input v-model="it.sender" class="h-9 font-mono text-xs" placeholder="具体发言人 ID (可选)" />
                    </div>
                    <div class="space-y-2">
                      <label class="text-[10px] font-bold uppercase tracking-widest text-muted-foreground/60">关键词 / Keyword (Optional)</label>
                      <Input v-model="it.keyword" class="h-9 font-mono text-xs" placeholder="包含特定词汇触发 (可选)" />
                    </div>
                  </div>
                  <div class="space-y-2">
                    <label class="text-[10px] font-bold uppercase tracking-widest text-muted-foreground/60">描述 / Description</label>
                    <Input v-model="it.description" class="h-9 text-xs" placeholder="例如：转发群组消息到本地日志服务" />
                  </div>
                </CardContent>
              </Card>
            </div>
          </CardContent>
        </Card>
      </section>
    </div>
  </div>
</template>
