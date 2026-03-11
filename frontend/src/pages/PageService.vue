<script setup lang="ts">
import { inject, reactive, ref } from 'vue'
import { backend } from '../wailsbridge'
import { appContextKey } from '../app/context'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Input } from '@/components/ui/input'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'

const injected = inject(appContextKey)
if (!injected) throw new Error('chatlog not provided')
const app = injected

const { httpAddr, state, run } = app

function saveAddr() {
  return app
    .feedback.confirm({
      title: '保存 HTTP 地址',
      message: '确认保存并写入配置？',
      confirmText: '保存',
      cancelText: '取消',
    })
    .then(ok => (ok ? run(() => backend.SetHTTPAddr(httpAddr.value), '已保存') : undefined))
}

async function toggleHTTP() {
  if (state.value?.httpEnabled) {
    const ok = await app.feedback.confirm({
      title: '停止 HTTP 服务',
      message: '确认停止 HTTP 服务？停止后 API 与 MCP 接口将不可访问。',
      confirmText: '停止',
      cancelText: '取消',
      danger: true,
    })
    if (!ok) return
    return run(() => backend.StopHTTP(), '已停止')
  }
  return run(() => backend.StartHTTP(), '已启动')
}

function baseUrl() {
  const addr = state.value?.httpAddr || httpAddr.value || '127.0.0.1:5030'
  return `http://${addr}`
}

interface Endpoint {
  name: string
  method: string
  path: string
  desc: string
  params?: Array<{ key: string, placeholder: string, desc: string }>
}

const endpoints: Endpoint[] = [
  {
    name: '会话列表',
    method: 'GET',
    path: '/api/v1/session',
    desc: '获取最近会话列表',
    params: [
      { key: 'keyword', placeholder: '关键词', desc: '内容过滤' },
      { key: 'limit', placeholder: '50', desc: '返回数量' },
      { key: 'offset', placeholder: '0', desc: '分页偏移' },
      { key: 'format', placeholder: 'json', desc: '输出格式' },
    ],
  },
  {
    name: '联系人列表',
    method: 'GET',
    path: '/api/v1/contact',
    desc: '获取联系人列表',
    params: [
      { key: 'keyword', placeholder: '关键词', desc: '昵称/备注过滤' },
      { key: 'limit', placeholder: '20', desc: '返回数量' },
      { key: 'offset', placeholder: '0', desc: '分页偏移' },
      { key: 'format', placeholder: 'json', desc: '输出格式' },
    ],
  },
  {
    name: '群聊列表',
    method: 'GET',
    path: '/api/v1/chatroom',
    desc: '获取群聊列表',
    params: [
      { key: 'keyword', placeholder: '关键词', desc: '群名过滤' },
      { key: 'limit', placeholder: '20', desc: '返回数量' },
      { key: 'offset', placeholder: '0', desc: '分页偏移' },
      { key: 'format', placeholder: 'json', desc: '输出格式' },
    ],
  },
  {
    name: '聊天记录',
    method: 'GET',
    path: '/api/v1/chatlog',
    desc: '按条件查询聊天记录',
    params: [
      { key: 'time', placeholder: '2025-01-01', desc: '时间范围' },
      { key: 'talker', placeholder: 'wxid_xxx', desc: '聊天对象' },
      { key: 'sender', placeholder: 'wxid_xxx', desc: '发送者' },
      { key: 'keyword', placeholder: '关键词', desc: '内容搜索' },
      { key: 'limit', placeholder: '20', desc: '返回数量' },
      { key: 'offset', placeholder: '0', desc: '分页偏移' },
      { key: 'format', placeholder: 'json', desc: '输出格式' },
    ],
  },
  {
    name: 'MCP',
    method: 'GET',
    path: '/mcp',
    desc: 'MCP Streamable HTTP 端点',
  },
]

const epParams = reactive<Record<string, Record<string, string>>>({
  '/api/v1/session': { keyword: '', limit: '', offset: '', format: 'json' },
  '/api/v1/contact': { keyword: '', limit: '', offset: '', format: 'json' },
  '/api/v1/chatroom': { keyword: '', limit: '', offset: '', format: 'json' },
  '/api/v1/chatlog': { time: '', talker: '', sender: '', keyword: '', limit: '', offset: '', format: 'json' },
})

function fullUrl(ep: Endpoint): string {
  const base = `${baseUrl()}${ep.path}`
  const params = epParams[ep.path]
  if (!params) return base

  const qs = Object.entries(params)
    .map(([key, value]) => (value.trim() ? `${encodeURIComponent(key)}=${encodeURIComponent(value.trim())}` : ''))
    .filter(Boolean)
    .join('&')

  return qs ? `${base}?${qs}` : base
}

function curlCmd(ep: Endpoint): string {
  return `curl -X ${ep.method} "${fullUrl(ep)}"`
}

const copiedId = ref('')

async function copyCmd(ep: Endpoint) {
  const cmd = curlCmd(ep)
  try {
    await navigator.clipboard.writeText(cmd)
    app.feedback.toast('已复制', cmd)
    copiedId.value = ep.path
    setTimeout(() => {
      if (copiedId.value === ep.path) copiedId.value = ''
    }, 1600)
  }
  catch {
    app.feedback.toast('复制失败', '浏览器不支持剪贴板操作')
  }
}

const responses = reactive<Record<string, any>>({})

async function tryApi(ep: Endpoint) {
  const url = fullUrl(ep)
  try {
    const res = await fetch(url)
    const data = await res.json()
    responses[ep.path] = JSON.stringify(data, null, 2)
  }
  catch (error) {
    responses[ep.path] = String(error)
  }
}
</script>

<template>
  <div class="space-y-8">
    <section class="space-y-4">
      <div class="border-b border-border/60 pb-3">
        <div class="text-xs font-semibold uppercase tracking-[0.16em] text-muted-foreground">01 · HTTP & MCP Service</div>
      </div>

      <Card class="border-border/60 bg-card/70 shadow-sm">
        <CardHeader class="gap-3">
          <CardTitle class="text-base">服务状态</CardTitle>
          <CardDescription>控制 HTTP / MCP 服务并管理监听地址。</CardDescription>
        </CardHeader>
        <CardContent class="space-y-5">
          <div class="flex flex-wrap items-center gap-3">
            <Badge
              :variant="state?.httpEnabled ? 'secondary' : 'outline'"
              :class="state?.httpEnabled
                ? 'rounded-lg px-3 py-1 text-[12px] font-semibold tracking-[0.08em] border-emerald-500/35 bg-emerald-500/14 text-emerald-100 shadow-[inset_0_1px_0_rgba(255,255,255,0.05)]'
                : 'rounded-lg px-3 py-1 text-[12px] font-semibold tracking-[0.08em] border-border/70 bg-background/40 text-muted-foreground'"
            >
              {{ state?.httpEnabled ? 'RUNNING' : 'STOPPED' }}
            </Badge>
            <Button :variant="state?.httpEnabled ? 'destructive' : 'default'" @click="toggleHTTP">
              {{ state?.httpEnabled ? 'Stop' : 'Start' }}
            </Button>
          </div>

          <div class="grid gap-2">
            <div class="text-[11px] font-semibold uppercase tracking-[0.14em] text-muted-foreground">Listen Address</div>
            <div class="flex flex-col gap-2 lg:flex-row">
              <Input v-model="httpAddr" class="font-mono" placeholder="127.0.0.1:5030" />
              <Button variant="outline" @click="saveAddr">Save</Button>
            </div>
          </div>

          <div v-if="state?.httpAddr" class="grid gap-4 md:grid-cols-2">
            <div class="rounded-xl border border-border/60 bg-background/30 p-4">
              <div class="mb-2 text-[11px] font-semibold uppercase tracking-[0.14em] text-muted-foreground">API</div>
              <div class="font-mono text-xs text-primary break-all">http://{{ state.httpAddr }}/api/v1/session</div>
            </div>
            <div class="rounded-xl border border-border/60 bg-background/30 p-4">
              <div class="mb-2 text-[11px] font-semibold uppercase tracking-[0.14em] text-muted-foreground">MCP</div>
              <div class="font-mono text-xs text-primary break-all">http://{{ state.httpAddr }}/mcp</div>
            </div>
          </div>
        </CardContent>
      </Card>
    </section>

    <section class="space-y-4">
      <div class="border-b border-border/60 pb-3">
        <div class="text-xs font-semibold uppercase tracking-[0.16em] text-muted-foreground">02 · API Playground</div>
      </div>

      <div class="space-y-4">
        <Card v-for="ep in endpoints" :key="ep.path" class="border-border/60 bg-card/70 shadow-sm">
          <CardHeader class="gap-4">
            <div class="flex flex-col gap-3 lg:flex-row lg:items-start lg:justify-between">
              <div class="space-y-2">
                <div class="flex flex-wrap items-center gap-2">
                  <Badge variant="outline" class="font-mono">{{ ep.method }}</Badge>
                  <CardTitle class="text-base">{{ ep.name }}</CardTitle>
                </div>
                <CardDescription>{{ ep.desc }}</CardDescription>
              </div>

              <div class="flex flex-wrap gap-2">
                <Button variant="outline" :disabled="!state?.httpEnabled" @click="tryApi(ep)">Try it</Button>
                <Button :variant="copiedId === ep.path ? 'default' : 'secondary'" @click="copyCmd(ep)">
                  {{ copiedId === ep.path ? 'Copied ✓' : 'Copy curl' }}
                </Button>
              </div>
            </div>
            <div class="rounded-xl border border-border/60 bg-background/30 px-4 py-3 font-mono text-xs text-foreground break-all">
              {{ fullUrl(ep) }}
            </div>
          </CardHeader>

          <CardContent class="space-y-4">
            <div v-if="ep.params" class="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
              <div v-for="p in ep.params" :key="p.key" class="space-y-2">
                <div class="text-[11px] font-semibold uppercase tracking-[0.14em] text-muted-foreground">
                  {{ p.key }}
                  <span class="ml-2 normal-case tracking-normal text-muted-foreground/80">{{ p.desc }}</span>
                </div>
                <Input v-model="epParams[ep.path][p.key]" :placeholder="p.placeholder" class="font-mono" />
              </div>
            </div>

            <div v-if="responses[ep.path]" class="rounded-xl border border-border/60 bg-black/80 p-4 font-mono text-xs leading-6 text-zinc-100 whitespace-pre-wrap break-all">
              {{ responses[ep.path] }}
            </div>
          </CardContent>
        </Card>
      </div>
    </section>
  </div>
</template>
