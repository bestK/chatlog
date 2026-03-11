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
    .filter(line => line.includes(kw))
    .join('\n')
})

const filteredHtml = computed(() => {
  return filtered.value
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/^(\d{4}[\/-]\d{2}[\/-]\d{2}\s\d{2}:\d{2}:\d{2})/gm, '<span class="text-cyan-300">$1</span>')
    .replace(/\b(INF|INFO|SUCCESS)\b/g, '<span class="font-semibold text-emerald-300">$1</span>')
    .replace(/\b(WRN|WARN|WARNING)\b/g, '<span class="font-semibold text-amber-300">$1</span>')
    .replace(/\b(ERR|ERROR|FATAL|CRITICAL)\b/g, '<span class="font-semibold text-rose-300">$1</span>')
    .replace(/\b(DBG|DEBUG)\b/g, '<span class="font-semibold text-sky-300">$1</span>')
})

async function copyText(text: string) {
  try {
    await navigator.clipboard.writeText(text)
    app.feedback.toast('已复制', '已复制到剪贴板')
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
    catch {
    }
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
  return el.scrollTop + el.clientHeight >= el.scrollHeight - 40
}

function scrollToBottom(el: HTMLElement) {
  el.scrollTop = el.scrollHeight
}

watch(
  () => filtered.value,
  async () => {
    await nextTick()
    const el = logBox.value
    if (!el) return
    if (firstScroll) {
      firstScroll = false
      scrollToBottom(el)
      return
    }
    if (isNearBottom(el)) {
      scrollToBottom(el)
    }
  },
)
</script>

<template>
  <div class="space-y-6">
    <div class="border-b border-border/60 pb-3">
      <div class="text-xs font-semibold uppercase tracking-[0.16em] text-muted-foreground">01 · System Logs</div>
    </div>

    <Card class="flex min-h-0 flex-col border-border/60 bg-card/70 shadow-sm">
      <CardHeader class="gap-3">
        <CardTitle class="text-base">日志查看</CardTitle>
        <CardDescription>支持刷新、复制与关键字过滤，默认读取最近 {{ maxLines }} 行。</CardDescription>
      </CardHeader>

      <CardContent class="flex min-h-0 flex-1 flex-col gap-5">
        <div class="grid gap-4">
          <div class="grid gap-2">
            <div class="text-[11px] font-semibold uppercase tracking-[0.14em] text-muted-foreground">Log File Path</div>
            <div class="flex flex-col gap-2 lg:flex-row">
              <Input :model-value="logPath" readonly class="font-mono text-xs" />
              <div class="flex gap-2">
                <Button variant="outline" @click="copyText(logPath)">Copy Path</Button>
                <Button variant="outline" @click="refresh">Refresh</Button>
              </div>
            </div>
          </div>

          <div class="grid gap-2">
            <div class="text-[11px] font-semibold uppercase tracking-[0.14em] text-muted-foreground">Filter Content</div>
            <div class="flex flex-col gap-2 lg:flex-row">
              <Input v-model="keyword" placeholder="e.g. ERROR / decrypt / webhook" />
              <Button variant="outline" @click="copyText(filtered)">Copy Log</Button>
            </div>
          </div>
        </div>

        <div class="flex flex-wrap gap-2">
          <Badge :variant="loading ? 'default' : 'secondary'">{{ loading ? 'Reading logs...' : `Lines: ${maxLines}` }}</Badge>
          <Badge v-if="keyword.trim()" variant="outline">Filtering: {{ keyword.trim() }}</Badge>
        </div>

        <div class="min-h-0 flex-1 overflow-hidden rounded-xl border border-border/60 bg-black/80 shadow-inner">
          <pre
            ref="logBox"
            class="h-full overflow-auto p-5 font-mono text-xs leading-6 whitespace-pre-wrap break-words text-zinc-100"
            v-html="filteredHtml"
          />
        </div>
      </CardContent>
    </Card>
  </div>
</template>
