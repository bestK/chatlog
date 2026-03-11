<script setup lang="ts">
import { inject } from 'vue'
import { appContextKey } from '../app/context'
import { maskKey } from '../app/state'
import { backend } from '../wailsbridge'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'

const injected = inject(appContextKey)
if (!injected) throw new Error('chatlog not provided')
const app = injected

const { dataKey, imgKey, dataDir, workDir, state, run } = app

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
    return run(() => backend.StopHTTP(), '已停止 HTTP 服务')
  }
  return run(() => backend.StartHTTP(), '已启动 HTTP 服务')
}
</script>

<template>
  <div class="space-y-8">
    <section class="space-y-4">
      <div class="border-b border-border/60 pb-3">
        <div class="text-xs font-semibold uppercase tracking-[0.16em] text-muted-foreground">01 · Key Information</div>
      </div>

      <Card class="border-border/60 bg-card/70 shadow-sm">
        <CardContent class="grid gap-4 pt-6 md:grid-cols-3">
          <div class="space-y-2 rounded-xl border border-border/60 bg-background/30 p-4">
            <div class="text-[11px] font-semibold uppercase tracking-[0.14em] text-muted-foreground">Data Key</div>
            <div class="font-mono text-sm text-foreground">{{ maskKey(dataKey) || '-' }}</div>
          </div>
          <div class="space-y-2 rounded-xl border border-border/60 bg-background/30 p-4">
            <div class="text-[11px] font-semibold uppercase tracking-[0.14em] text-muted-foreground">Img Key</div>
            <div class="font-mono text-sm text-foreground">{{ maskKey(imgKey) || '-' }}</div>
          </div>
          <div class="space-y-2 rounded-xl border border-border/60 bg-background/30 p-4">
            <div class="text-[11px] font-semibold uppercase tracking-[0.14em] text-muted-foreground">Process PID</div>
            <div class="text-sm font-medium text-foreground">{{ state?.pid || '-' }}</div>
          </div>
        </CardContent>
      </Card>
    </section>

    <section class="space-y-4">
      <div class="border-b border-border/60 pb-3">
        <div class="text-xs font-semibold uppercase tracking-[0.16em] text-muted-foreground">02 · Directory Paths</div>
      </div>

      <Card class="border-border/60 bg-card/70 shadow-sm">
        <CardContent class="space-y-4 pt-6">
          <div class="space-y-2">
            <div class="text-[11px] font-semibold uppercase tracking-[0.14em] text-muted-foreground">Data Directory</div>
            <div class="rounded-xl border border-border/60 bg-background/30 px-4 py-3 font-mono text-xs text-foreground break-all">
              {{ dataDir || '-' }}
            </div>
          </div>
          <div class="space-y-2">
            <div class="text-[11px] font-semibold uppercase tracking-[0.14em] text-muted-foreground">Work Directory</div>
            <div class="rounded-xl border border-border/60 bg-background/30 px-4 py-3 font-mono text-xs text-foreground break-all">
              {{ workDir || '-' }}
            </div>
          </div>
        </CardContent>
      </Card>
    </section>

    <section class="space-y-4">
      <div class="border-b border-border/60 pb-3">
        <div class="text-xs font-semibold uppercase tracking-[0.16em] text-muted-foreground">03 · Quick Actions & Services</div>
      </div>

      <Card class="border-border/60 bg-card/70 shadow-sm">
        <CardHeader class="gap-3">
          <CardTitle class="text-base">服务控制</CardTitle>
          <CardDescription>快速切换 HTTP 服务并查看当前接口地址。</CardDescription>
        </CardHeader>
        <CardContent class="space-y-5">
          <div class="flex flex-wrap gap-3">
            <Button :variant="state?.httpEnabled ? 'destructive' : 'default'" @click="toggleHTTP">
              {{ state?.httpEnabled ? 'Stop HTTP Service' : 'Start HTTP Service' }}
            </Button>
            <Button variant="outline" @click="run(() => backend.Refresh(), 'Refreshed')">
              Refresh Status
            </Button>
          </div>

          <div v-if="state?.httpAddr" class="grid gap-4 md:grid-cols-2">
            <div class="space-y-2 rounded-xl border border-border/60 bg-background/30 p-4">
              <div class="flex items-center justify-between gap-2">
                <div class="text-[11px] font-semibold uppercase tracking-[0.14em] text-muted-foreground">API Endpoint</div>
                <Badge variant="outline" class="rounded-md">Session</Badge>
              </div>
              <div class="font-mono text-xs text-primary break-all">http://{{ state.httpAddr }}/api/v1/session</div>
            </div>
            <div class="space-y-2 rounded-xl border border-border/60 bg-background/30 p-4">
              <div class="flex items-center justify-between gap-2">
                <div class="text-[11px] font-semibold uppercase tracking-[0.14em] text-muted-foreground">MCP Endpoint</div>
                <Badge variant="outline" class="rounded-md">MCP</Badge>
              </div>
              <div class="font-mono text-xs text-primary break-all">http://{{ state.httpAddr }}/mcp</div>
            </div>
          </div>
        </CardContent>
      </Card>
    </section>
  </div>
</template>
