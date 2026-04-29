<script setup lang="ts">
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import type { Page } from '../app/state';
import type { State } from '../wailsbridge';

defineProps<{
    nav: Array<{ name: Page; hint: string }>;
    page: Page;
    state: State | null;
}>();

const emit = defineEmits<{
    (e: 'update:page', value: Page): void;
}>();

function setPage(page: Page) {
    emit('update:page', page);
}
</script>

<template>
    <aside
        class="flex h-full flex-col border-r border-border/60 bg-sidebar/70 px-3 py-4 backdrop-blur md:px-4 md:py-5 xl:px-5 xl:py-6"
    >
        <div class="mb-8 flex items-center gap-3 px-2">
            <div
                class="flex size-10 items-center justify-center rounded-xl border border-border/60 bg-card/70 shadow-sm"
            >
                <div class="size-4 rounded-md bg-primary shadow-[0_0_24px_rgba(53,215,255,0.35)]" />
            </div>
            <div class="min-w-0">
                <h1 class="truncate text-base font-semibold tracking-tight text-foreground">Chatlog</h1>
                <p class="text-[11px] font-medium uppercase tracking-[0.16em] text-muted-foreground">Wails Desktop</p>
            </div>
        </div>

        <nav class="flex-1 space-y-1">
            <Button
                v-for="item in nav"
                :key="item.name"
                type="button"
                :variant="page === item.name ? 'secondary' : 'ghost'"
                class="h-auto w-full justify-start rounded-xl px-3 py-3 text-left"
                :class="
                    page === item.name ? 'border border-border/60 bg-secondary/80 shadow-sm' : 'text-muted-foreground'
                "
                @click="setPage(item.name)"
            >
                <div class="flex w-full items-center justify-between gap-3">
                    <div class="min-w-0 space-y-1">
                        <div class="truncate text-sm font-medium text-foreground">{{ item.name }}</div>
                        <div class="truncate text-xs text-muted-foreground">{{ item.hint }}</div>
                    </div>
                    <div
                        v-if="page === item.name"
                        class="h-6 w-1 rounded-full bg-primary shadow-[0_0_16px_rgba(53,215,255,0.35)]"
                    />
                </div>
            </Button>
        </nav>

        <div v-if="state?.account" class="mt-6 rounded-2xl border border-border/60 bg-card/60 p-3 shadow-sm">
            <div class="flex items-center gap-3">
                <div class="relative shrink-0">
                    <img
                        v-if="state.smallHeadImgUrl"
                        :src="state.smallHeadImgUrl"
                        class="size-10 rounded-xl border border-border/60 object-cover"
                        alt="Avatar"
                        referrerpolicy="no-referrer"
                    />
                    <div
                        v-else
                        class="flex size-10 items-center justify-center rounded-xl border border-border/60 bg-muted/20 text-sm font-semibold text-foreground"
                    >
                        {{
                            state.nickname?.slice(0, 1)?.toUpperCase() ||
                            state.account?.slice(0, 1)?.toUpperCase() ||
                            '?'
                        }}
                    </div>
                    <span
                        class="absolute -right-1 -bottom-1 size-3 rounded-full border-2 border-card"
                        :class="state.status === 'online' ? 'bg-emerald-400' : 'bg-zinc-500'"
                    />
                </div>

                <div class="min-w-0 flex-1">
                    <div class="truncate text-sm font-medium text-foreground">
                        {{ state.nickname || state.account }}
                    </div>
                    <div class="mt-1">
                        <Badge variant="outline" class="rounded-md text-[10px] uppercase tracking-[0.14em]">
                            {{ state.status === 'online' ? 'Connected' : 'Disconnected' }}
                        </Badge>
                    </div>
                </div>
            </div>
        </div>
    </aside>
</template>
