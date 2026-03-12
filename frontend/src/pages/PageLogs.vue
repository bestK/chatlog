<script setup lang="ts">
import { computed, inject, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import { appContextKey } from '../app/context'
import { backend } from '../wailsbridge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'

const injected = inject(appContextKey)
if (!injected) throw new Error('chatlog not provided')
const app = injected

const logPath = ref('')
const content = ref('')
const loading = ref(false)
const keyword = ref('')
const maxLines = 200

const logBox = ref<HTMLElement | null>(null)
let firstScroll = true
let offLogChanged: (() => void) | undefined

async function refresh() {
  loading.value = true
  try {
    logPath.value = await backend.GetLogPath()
    content.value = await backend.ReadLogTail(maxLines)
    if (firstScroll) await ensureInitialScrollToBottom()
  }
  catch (error) {
    app.feedback.toast('读取失败', String(error))
  }
  finally {
    loading.value = false
  }
}

const filtered = computed(() => {
  const kw = keyword.value.trim()
  if (!kw) return content.value
  return content.value
    .split('\n')
    .filter(line => line.toLowerCase().includes(kw.toLowerCase()))
    .join('\n')
})

const filteredHtml = computed(() => {
  return filtered.value
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/^(\d{4}[\/-]\d{2}[\/-]\d{2}\s\d{2}:\d{2}:\d{2})/gm, '<span class="text-cyan-400/80">$1</span>')
    .replace(/\b(INF|INFO|SUCCESS)\b/g, '<span class="font-bold text-emerald-400">$1</span>')
    .replace(/\b(WRN|WARN|WARNING)\b/g, '<span class="font-bold text-amber-400">$1</span>')
    .replace(/\b(ERR|ERROR|FATAL|CRITICAL)\b/g, '<span class="font-bold text-rose-400">$1</span>')
    .replace(/\b(DBG|DEBUG)\b/g, '<span class="font-bold text-sky-400">$1</span>')
})

async function copyText(text: string, label: string) {
  try {
    await navigator.clipboard.writeText(text)
    app.feedback.toast('已复制', `已复制 ${label} 到剪贴板`)
  }
  catch {
    app.feedback.toast('复制失败', '当前环境不支持剪贴板')
  }
}

onMounted(async () => {
  await refresh()
  if (backend.isWails) {
    try {
      await backend.EnableLogEvents(true)
    }
    catch { /* Ignored */ }
  }
  offLogChanged = backend.EventsOn('log:changed', () => {
    void refresh()
  })
})

onUnmounted(() => {
  if (offLogChanged) offLogChanged()
  if (backend.isWails) {
    void backend.EnableLogEvents(false)
  }
})

function isNearBottom(el: HTMLElement) {
  return el.scrollTop + el.clientHeight >= el.scrollHeight - 60
}

function scrollToBottom(el: HTMLElement) {
  requestAnimationFrame(() => {
    el.scrollTop = el.scrollHeight
  })
}

async function ensureInitialScrollToBottom() {
  await nextTick()
  const el = logBox.value
  if (!el) return
  firstScroll = false
  scrollToBottom(el)
}

watch(
  () => filtered.value,
  async () => {
    const el = logBox.value
    if (!el) return
    const shouldScroll = firstScroll || isNearBottom(el)
    await nextTick()
    if (shouldScroll) {
      firstScroll = false
      scrollToBottom(el)
    }
  },
  { flush: 'post' }
)
</script>

<template>
  <div class="flex h-full flex-col space-y-2 overflow-hidden">
    <div class="flex items-center gap-3 border-b border-border/40 pb-2 shrink-0">
      <div class="flex size-6 items-center justify-center rounded bg-primary/10 text-[8px] font-bold text-primary">01</div>
      <div class="text-[9px] font-bold uppercase tracking-[0.2em] text-foreground/60">运行日志 / LOGS</div>
    </div>

    <Card class="flex flex-1 flex-col min-h-0 overflow-hidden border-border/40 bg-card/40 shadow-none">
      <div class="shrink-0 border-b border-border/40 bg-muted/10 px-3 py-1.5 flex flex-col gap-2">
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-3">
            <span class="text-[11px] font-bold tracking-tight text-foreground/80">日志监控</span>
            <Badge :variant="loading ? 'default' : 'secondary'" class="h-4 px-1.5 font-mono text-[8px] font-bold uppercase tracking-wider">
              {{ loading ? 'Syncing...' : `L: ${maxLines}` }}
            </Badge>
          </div>
          
          <div class="flex items-center gap-1.5">
            <Button variant="ghost" size="sm" class="h-6 px-2 text-[9px] font-bold uppercase tracking-widest text-muted-foreground hover:text-foreground" @click="copyText(filtered, '日志内容')">Copy Logs</Button>
            <Button
              variant="outline"
              size="sm"
              class="h-6 gap-1.5 px-2 text-[9px] font-bold uppercase tracking-widest"
              :disabled="loading"
              @click="refresh"
            >
              <svg :class="['size-2.5', loading ? 'animate-spin' : '']" xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 12a9 9 0 1 1-9-9c2.52 0 4.93 1 6.74 2.74L21 8" /><path d="M21 3v5h-5" /></svg>
              Refresh
            </Button>
          </div>
        </div>

        <div class="flex items-center gap-4">
          <div class="flex flex-1 items-center gap-2 min-w-0">
            <span class="text-[8px] font-bold uppercase tracking-widest text-muted-foreground/50 shrink-0">PATH</span>
            <div class="group relative flex flex-1 items-center min-w-0">
              <div class="flex-1 overflow-hidden truncate rounded border border-border/20 bg-muted/30 px-2 py-0.5 font-mono text-[9px] text-foreground/50">
                {{ logPath || 'Loading...' }}
              </div>
              <button class="ml-1 opacity-0 group-hover:opacity-100 transition-opacity" @click="copyText(logPath, '路径')">
                <svg xmlns="http://www.w3.org/2000/svg" width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="text-muted-foreground hover:text-primary"><rect width="14" height="14" x="8" y="8" rx="2" ry="2" /><path d="M4 16c-1.1 0-2-.9-2-2V4c0-1.1.9-2 2-2h10c1.1 0 2 .9 2 2" /></svg>
              </button>
            </div>
          </div>

          <div class="flex flex-[1.5] items-center gap-2">
            <span class="text-[8px] font-bold uppercase tracking-widest text-muted-foreground/50 shrink-0">FILTER</span>
            <div class="relative flex-1">
              <svg class="absolute left-2 top-1/2 -translate-y-1/2 size-2.5 text-muted-foreground/40" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="11" cy="11" r="8"/><path d="m21 21-4.3-4.3"/></svg>
              <Input
                v-model="keyword"
                placeholder="Search keywords..."
                class="h-6 pl-6 bg-background/20 font-mono text-[10px] py-0 border-border/20 focus-visible:ring-offset-0 shadow-none"
              />
            </div>
          </div>
        </div>
      </div>

      <div class="flex flex-1 min-h-0 p-2">
        <div class="relative h-full overflow-hidden rounded-lg border border-border/40 bg-[#080808] shadow-2xl flex flex-col">
          <div class="shrink-0 flex items-center justify-between bg-black/40 px-3 py-1 backdrop-blur-md border-b border-white/5">
            <div class="flex items-center gap-1.5">
              <div class="size-1.5 rounded-full bg-rose-500/60" />
              <div class="size-1.5 rounded-full bg-amber-500/60" />
              <div class="size-1.5 rounded-full bg-emerald-500/60" />
            </div>
            <div class="text-[7px] font-bold uppercase tracking-[0.4em] text-zinc-700">Live Console Stream</div>
          </div>
          
          <pre
            ref="logBox"
            class="flex-1 overflow-auto p-4 pt-6 font-mono text-[10px] leading-5 text-zinc-300 selection:bg-primary/30 scroll-smooth scrollbar-thin scrollbar-track-transparent scrollbar-thumb-zinc-900"
            v-html="filteredHtml || '<div class=\'flex h-full items-center justify-center opacity-10 uppercase tracking-[0.3em] text-[10px]\'>No Data</div>'"
          />
        </div>
      </div>
    </Card>
  </div>
</template>

<style scoped>
.scrollbar-thin::-webkit-scrollbar { width: 4px; }
.scrollbar-thin::-webkit-scrollbar-thumb { border-radius: 10px; background: rgba(255,255,255,0.03); }
.scrollbar-thin::-webkit-scrollbar-thumb:hover { background: rgba(255,255,255,0.08); }
</style>
