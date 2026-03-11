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
    <section class="space-y-4">
      <div class="border-b border-border/60 pb-3">
        <div class="text-xs font-semibold uppercase tracking-[0.16em] text-muted-foreground">01 · Configuration</div>
      </div>

      <Card class="border-border/60 bg-card/70 shadow-sm">
        <CardHeader class="flex flex-row items-center justify-between space-y-0 pb-6">
          <div class="space-y-1.5">
            <CardTitle class="text-base">基础配置</CardTitle>
            <CardDescription>设置资源地址、推送延迟，并保存当前配置。</CardDescription>
          </div>
          <div class="flex items-center gap-2">
            <Button variant="outline" size="sm" @click="load">Refresh</Button>
            <Button size="sm" @click="save">Save Changes</Button>
          </div>
        </CardHeader>
        <CardContent class="grid gap-6 md:grid-cols-2">
          <div class="space-y-2">
            <div class="text-[11px] font-semibold uppercase tracking-[0.14em] text-muted-foreground">资源地址</div>
            <Input v-model="cfg.host" class="font-mono" placeholder="localhost:5030" />
          </div>
          <div class="space-y-2">
            <div class="text-[11px] font-semibold uppercase tracking-[0.14em] text-muted-foreground">Delay (ms)</div>
            <Input v-model.number="cfg.delayMs" type="number" min="0" step="100" class="font-mono" />
          </div>
        </CardContent>
      </Card>
    </section>

    <section class="space-y-4">
      <div class="border-b border-border/60 pb-3">
        <div class="text-xs font-semibold uppercase tracking-[0.16em] text-muted-foreground">02 · Rules</div>
      </div>

      <Card class="border-border/60 bg-card/70 shadow-sm">
        <CardHeader class="gap-4">
          <div class="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
            <div class="space-y-2">
              <CardTitle class="text-base">规则列表</CardTitle>
              <CardDescription>修改后需保存才会生效。</CardDescription>
            </div>
            <div class="flex items-center gap-2">
              <Badge variant="outline">Rules: {{ cfg.items.length }}</Badge>
              <Button variant="outline" @click="addItem">Add Rule</Button>
            </div>
          </div>
        </CardHeader>

        <CardContent class="space-y-4">
          <div v-if="cfg.items.length === 0" class="rounded-xl border border-dashed border-border/60 bg-background/20 p-8 text-center">
            <div class="text-sm font-medium text-foreground">No Rules Defined</div>
            <div class="mt-2 text-sm text-muted-foreground">Add a rule and save to start pushing messages.</div>
          </div>

          <Card
            v-for="(it, idx) in cfg.items"
            :key="idx"
            class="overflow-hidden border-border/60 bg-background/30 shadow-none transition-all hover:bg-background/40"
          >
            <!-- 规则头部状态栏 -->
            <div class="flex items-center justify-between border-b border-border/40 bg-muted/20 px-4 py-2">
              <div class="flex items-center gap-3">
                <Badge variant="outline" class="h-5 bg-background/50 font-mono text-[10px]">#{{ idx + 1 }}</Badge>
                <span class="text-[11px] font-bold uppercase tracking-widest text-muted-foreground">Rule Details</span>
              </div>
              <div class="flex items-center gap-2">
                <!-- 强化后的状态开关 -->
                <Button
                  variant="ghost"
                  size="sm"
                  class="h-7 gap-2 px-3 text-[10px] font-bold uppercase tracking-tight hover:bg-background"
                  @click="it.disabled = !it.disabled"
                >
                  <div :class="['h-2 w-2 rounded-full transition-all', it.disabled ? 'bg-muted-foreground' : 'bg-emerald-500 shadow-[0_0_8px_rgba(16,185,129,0.5)]']" />
                  <span :class="it.disabled ? 'text-muted-foreground' : 'text-foreground'">
                    {{ it.disabled ? 'Inactive' : 'Active' }}
                  </span>
                </Button>
                <Button
                  variant="ghost"
                  size="sm"
                  class="h-7 px-3 text-[10px] font-bold uppercase tracking-tight text-muted-foreground hover:bg-destructive/10 hover:text-destructive"
                  @click="removeItem(idx)"
                >
                  Delete
                </Button>
              </div>
            </div>
            <CardContent class="grid gap-6 p-6">
              <div class="grid gap-4 md:grid-cols-2">
                <div class="space-y-2">
                  <div class="text-[11px] font-semibold uppercase tracking-[0.14em] text-muted-foreground">Description</div>
                  <Input v-model="it.description" placeholder="e.g. Forward group messages to local service" />
                </div>
                <div class="space-y-2">
                  <div class="text-[11px] font-semibold uppercase tracking-[0.14em] text-muted-foreground">URL</div>
                  <Input v-model="it.url" class="font-mono" placeholder="http://127.0.0.1:3000/api/v1/webhook" />
                </div>
              </div>
              <div class="grid gap-4 md:grid-cols-3">
                <div class="space-y-2">
                  <div class="text-[11px] font-semibold uppercase tracking-[0.14em] text-muted-foreground">Talker</div>
                  <Input v-model="it.talker" class="font-mono" placeholder="Group or User name" />
                </div>
                <div class="space-y-2">
                  <div class="text-[11px] font-semibold uppercase tracking-[0.14em] text-muted-foreground">Sender</div>
                  <Input v-model="it.sender" class="font-mono" placeholder="Sender (Optional)" />
                </div>
                <div class="space-y-2">
                  <div class="text-[11px] font-semibold uppercase tracking-[0.14em] text-muted-foreground">Keyword</div>
                  <Input v-model="it.keyword" class="font-mono" placeholder="Keyword (Optional)" />
                </div>
              </div>
            </CardContent>
          </Card>
        </CardContent>
      </Card>
    </section>
  </div>
</template>
