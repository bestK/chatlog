<script setup lang="ts">
import { Button } from '@/components/ui/button';
import type { StatusPill } from '../app/state';
import type { State } from '../wailsbridge';

defineProps<{
    page: string;
    state: State | null;
    statusPill: StatusPill;
}>();

defineEmits<{
    (e: 'refresh'): void;
}>();
</script>

<template>
    <header class="flex flex-wrap items-start justify-between gap-4 border-b border-border/40 pb-5">
        <div class="min-w-0 space-y-2">
            <h2 class="font-serif text-2xl font-medium tracking-tight text-foreground">{{ page }}</h2>

            <div v-if="state" class="flex flex-wrap items-center gap-x-4 gap-y-1.5 text-sm text-muted-foreground">
                <span class="font-mono text-xs">PID {{ state.pid || '-' }}</span>
                <span class="text-xs opacity-50">·</span>
                <span class="font-mono text-xs">{{ state.fullVersion || '-' }}</span>
                <span class="text-xs opacity-50">·</span>
                <span class="truncate text-xs">{{ state.lastSession || '暂无会话' }}</span>
                <span class="inline-flex items-center gap-1.5 text-xs">
                    <span
                        :class="[
                            'size-1.5 rounded-full',
                            state.httpEnabled ? 'bg-emerald-500' : 'bg-muted-foreground/40'
                        ]"
                    />
                    <span :class="state.httpEnabled ? 'text-foreground' : ''"
                        >HTTP {{ state.httpEnabled ? '运行中' : '未启动' }}</span
                    >
                </span>
                <span class="inline-flex items-center gap-1.5 text-xs">
                    <span
                        :class="['size-1.5 rounded-full', state.autoDecrypt ? 'bg-sky-500' : 'bg-muted-foreground/40']"
                    />
                    <span :class="state.autoDecrypt ? 'text-foreground' : ''"
                        >自动解密 {{ state.autoDecrypt ? '已启用' : '未启用' }}</span
                    >
                </span>
            </div>
        </div>

        <Button
            type="button"
            variant="ghost"
            class="h-9 shrink-0 gap-2 px-3 text-sm font-normal text-muted-foreground hover:text-foreground"
            @click="$emit('refresh')"
        >
            <svg
                xmlns="http://www.w3.org/2000/svg"
                width="14"
                height="14"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="2"
                stroke-linecap="round"
                stroke-linejoin="round"
            >
                <path d="M21 12a9 9 0 1 1-9-9c2.52 0 4.93 1 6.74 2.74L21 8" />
                <path d="M21 3v5h-5" />
            </svg>
            刷新
        </Button>
    </header>
</template>
