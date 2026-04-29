<script setup lang="ts">
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { computed, inject, nextTick, onMounted, onUnmounted, ref, watch } from 'vue';
import { appContextKey } from '../app/context';
import { backend } from '../wailsbridge';

const injected = inject(appContextKey);
if (!injected) throw new Error('chatlog not provided');
const app = injected;

const logPath = ref('');
const content = ref('');
const loading = ref(false);
const keyword = ref('');
const maxLines = 200;

const logBox = ref<HTMLElement | null>(null);
let firstScroll = true;
let offLogChanged: (() => void) | undefined;

async function refresh() {
    loading.value = true;
    try {
        logPath.value = await backend.GetLogPath();
        content.value = await backend.ReadLogTail(maxLines);
        if (firstScroll) await ensureInitialScrollToBottom();
    } catch (error) {
        app.feedback.toast('读取失败', String(error));
    } finally {
        loading.value = false;
    }
}

const filtered = computed(() => {
    const kw = keyword.value.trim();
    if (!kw) return content.value;
    return content.value
        .split('\n')
        .filter(line => line.toLowerCase().includes(kw.toLowerCase()))
        .join('\n');
});

const filteredHtml = computed(() => {
    return filtered.value
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/^(\d{4}[\/-]\d{2}[\/-]\d{2}\s\d{2}:\d{2}:\d{2})/gm, '<span class="text-cyan-400/80">$1</span>')
        .replace(/\b(INF|INFO|SUCCESS)\b/g, '<span class="font-bold text-emerald-400">$1</span>')
        .replace(/\b(WRN|WARN|WARNING)\b/g, '<span class="font-bold text-amber-400">$1</span>')
        .replace(/\b(ERR|ERROR|FATAL|CRITICAL)\b/g, '<span class="font-bold text-rose-400">$1</span>')
        .replace(/\b(DBG|DEBUG)\b/g, '<span class="font-bold text-sky-400">$1</span>');
});

async function copyText(text: string, label: string) {
    try {
        await navigator.clipboard.writeText(text);
        app.feedback.toast('已复制', `已复制 ${label} 到剪贴板`);
    } catch {
        app.feedback.toast('复制失败', '当前环境不支持剪贴板');
    }
}

onMounted(async () => {
    await refresh();
    if (backend.isWails) {
        try {
            await backend.EnableLogEvents(true);
        } catch {
            /* Ignored */
        }
    }
    offLogChanged = backend.EventsOn('log:changed', () => {
        void refresh();
    });
});

onUnmounted(() => {
    if (offLogChanged) offLogChanged();
    if (backend.isWails) {
        void backend.EnableLogEvents(false);
    }
});

function isNearBottom(el: HTMLElement) {
    return el.scrollTop + el.clientHeight >= el.scrollHeight - 60;
}

function scrollToBottom(el: HTMLElement) {
    requestAnimationFrame(() => {
        el.scrollTop = el.scrollHeight;
    });
}

async function ensureInitialScrollToBottom() {
    await nextTick();
    const el = logBox.value;
    if (!el) return;
    firstScroll = false;
    scrollToBottom(el);
}

watch(
    () => filtered.value,
    async () => {
        const el = logBox.value;
        if (!el) return;
        const shouldScroll = firstScroll || isNearBottom(el);
        await nextTick();
        if (shouldScroll) {
            firstScroll = false;
            scrollToBottom(el);
        }
    },
    { flush: 'post' }
);
</script>

<template>
    <div class="flex h-full flex-col gap-4 overflow-hidden">
        <header class="shrink-0 flex flex-wrap items-end justify-between gap-3">
            <div class="space-y-1 min-w-0">
                <h3 class="font-serif text-xl font-medium tracking-tight text-foreground">运行日志</h3>
                <p class="truncate font-mono text-xs text-muted-foreground" :title="logPath">
                    {{ logPath || '加载中…' }}
                </p>
            </div>
            <div class="flex items-center gap-2">
                <span class="text-xs text-muted-foreground">
                    {{ loading ? '同步中…' : `最近 ${maxLines} 行` }}
                </span>
                <Button
                    variant="ghost"
                    size="sm"
                    class="h-9 px-3 text-sm font-normal text-muted-foreground hover:text-foreground"
                    @click="copyText(filtered, '日志内容')"
                >
                    复制
                </Button>
                <Button
                    variant="ghost"
                    size="sm"
                    class="h-9 px-3 text-sm font-normal text-muted-foreground hover:text-foreground"
                    :disabled="loading"
                    @click="refresh"
                >
                    刷新
                </Button>
            </div>
        </header>

        <div class="shrink-0 relative">
            <svg
                class="absolute left-3 top-1/2 -translate-y-1/2 size-4 text-muted-foreground/50"
                xmlns="http://www.w3.org/2000/svg"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="2"
                stroke-linecap="round"
                stroke-linejoin="round"
            >
                <circle cx="11" cy="11" r="8" />
                <path d="m21 21-4.3-4.3" />
            </svg>
            <Input v-model="keyword" placeholder="搜索日志关键字" class="h-9 pl-9 text-sm" />
        </div>

        <div class="flex flex-1 min-h-0">
            <pre
                ref="logBox"
                class="flex-1 overflow-auto rounded-lg border border-border/40 bg-card/20 p-4 font-mono text-xs leading-relaxed text-foreground/85 selection:bg-primary/30 scroll-smooth"
                v-html="
                    filteredHtml ||
                    '<div class=\'flex h-full items-center justify-center text-sm text-muted-foreground\'>暂无日志</div>'
                "
            />
        </div>
    </div>
</template>

<style scoped>
.scrollbar-thin::-webkit-scrollbar {
    width: 4px;
}
.scrollbar-thin::-webkit-scrollbar-thumb {
    border-radius: 10px;
    background: rgba(255, 255, 255, 0.03);
}
.scrollbar-thin::-webkit-scrollbar-thumb:hover {
    background: rgba(255, 255, 255, 0.08);
}
</style>
