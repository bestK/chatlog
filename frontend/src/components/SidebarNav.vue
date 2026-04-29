<script setup lang="ts">
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
        <div class="mb-6 flex items-center gap-3 px-2">
            <div class="flex size-9 items-center justify-center rounded-lg bg-primary/15">
                <div class="size-3 rounded-sm bg-primary" />
            </div>
            <h1 class="font-serif text-lg font-medium tracking-tight text-foreground">Chatlog</h1>
        </div>

        <nav class="flex-1 space-y-0.5">
            <Button
                v-for="item in nav"
                :key="item.name"
                type="button"
                :variant="page === item.name ? 'secondary' : 'ghost'"
                class="h-auto w-full justify-start rounded-lg px-3 py-2.5 text-left transition-colors"
                :class="page === item.name ? 'bg-secondary/70' : 'text-muted-foreground hover:text-foreground'"
                @click="setPage(item.name)"
            >
                <div class="min-w-0 space-y-0.5">
                    <div class="truncate text-sm font-medium text-foreground">{{ item.name }}</div>
                    <div class="truncate text-xs font-normal text-muted-foreground">{{ item.hint }}</div>
                </div>
            </Button>
        </nav>

        <div v-if="state?.account" class="mt-4 flex items-center gap-3 rounded-lg px-2 py-3">
            <div class="relative shrink-0">
                <img
                    v-if="state.smallHeadImgUrl"
                    :src="state.smallHeadImgUrl"
                    class="size-9 rounded-full object-cover"
                    alt="Avatar"
                    referrerpolicy="no-referrer"
                />
                <div
                    v-else
                    class="flex size-9 items-center justify-center rounded-full bg-muted/30 text-sm font-medium text-foreground"
                >
                    {{ state.nickname?.slice(0, 1)?.toUpperCase() || state.account?.slice(0, 1)?.toUpperCase() || '?' }}
                </div>
                <span
                    class="absolute -right-0.5 -bottom-0.5 size-2.5 rounded-full border-2 border-sidebar"
                    :class="state.status === 'online' ? 'bg-emerald-500' : 'bg-zinc-500'"
                />
            </div>

            <div class="min-w-0 flex-1">
                <div class="truncate text-sm font-medium text-foreground">
                    {{ state.nickname || state.account }}
                </div>
                <div class="truncate text-xs text-muted-foreground">
                    {{ state.status === 'online' ? '已连接' : '未连接' }}
                </div>
            </div>
        </div>
    </aside>
</template>
