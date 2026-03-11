<script setup lang="ts">
import type { StatusPill } from '../app/state'
import type { State } from '../wailsbridge'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'

defineProps<{
  page: string
  state: State | null
  statusPill: StatusPill
}>()

defineEmits<{
  (e: 'refresh'): void
}>()
</script>

<template>
  <header class="flex flex-wrap items-start justify-between gap-4 rounded-2xl border border-border/60 bg-card/70 px-5 py-4 shadow-sm backdrop-blur">
    <div class="min-w-0 space-y-3">
      <div class="flex items-center gap-3">
        <Badge variant="outline" class="h-6 rounded-full border-border/60 bg-background/50 px-3 text-[10px] font-bold uppercase tracking-[0.16em] text-muted-foreground">Console</Badge>
        <h2 class="truncate text-lg font-semibold tracking-tight text-foreground">{{ page }}</h2>
      </div>

      <div v-if="state" class="flex flex-wrap items-center gap-2 text-xs text-muted-foreground">
        <Badge variant="secondary" class="h-6 rounded-full bg-muted/50 px-3 font-mono text-[10px] font-medium text-muted-foreground">PID {{ state.pid || '-' }}</Badge>
        <Badge variant="secondary" class="h-6 rounded-full bg-muted/50 px-3 font-mono text-[10px] font-medium text-muted-foreground">{{ state.fullVersion || '-' }}</Badge>
        <Badge variant="secondary" class="max-w-full h-6 rounded-full bg-muted/50 px-3 text-[10px] font-medium text-muted-foreground">
          <span class="truncate">{{ state.lastSession || 'No Session' }}</span>
        </Badge>
        <Badge :variant="state.httpEnabled ? 'default' : 'outline'" class="h-6 rounded-full px-3 text-[10px] font-bold uppercase tracking-[0.14em]" :class="state.httpEnabled ? 'border-emerald-500/20 bg-emerald-500/10 text-emerald-500' : 'border-border/60 bg-background/40 text-muted-foreground'">HTTP</Badge>
        <Badge :variant="state.autoDecrypt ? 'default' : 'outline'" class="h-6 rounded-full px-3 text-[10px] font-bold uppercase tracking-[0.14em]" :class="state.autoDecrypt ? 'border-sky-500/20 bg-sky-500/10 text-sky-500' : 'border-border/60 bg-background/40 text-muted-foreground'">Auto Decrypt</Badge>
      </div>
    </div>

    <Button type="button" variant="outline" class="h-9 shrink-0 rounded-full border-border/60 bg-background/50 px-4 text-[11px] font-semibold tracking-tight shadow-sm hover:bg-background" @click="$emit('refresh')">
      Refresh
    </Button>
  </header>
</template>
