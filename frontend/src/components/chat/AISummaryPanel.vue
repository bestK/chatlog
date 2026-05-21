<script setup lang="ts">
import { Button } from '@/components/ui/button';
import { CustomSelect } from '@/components/ui/custom-select';
import { Copy, Sparkles } from 'lucide-vue-next';
import MarkdownRender from 'markstream-vue';
import 'markstream-vue/index.css';
import { computed, inject, nextTick, ref, watch } from 'vue';
import { appContextKey } from '@/app/context';
import { backend } from '@/wailsbridge';

const props = defineProps<{
    contactUserName: string;
}>();

const injected = inject(appContextKey);
if (!injected) throw new Error('chatlog not provided');
const app = injected;
const { state } = app;

type AIProvider = { id: string; name: string; type: string; baseUrl: string; model: string };

const summaryProviders = ref<AIProvider[]>([]);
const summaryProvider = ref(state.value?.selectedAIProvider || '');
const summaryTimeRange = ref('today');

watch(summaryProvider, v => {
    if (v) backend.SetSelectedAIProvider(v);
});

const providerOptions = computed(() => summaryProviders.value.map(p => ({ value: p.id, label: p.name })));
const summaryStream = ref('');
const summaryResult = ref('');
const summaryFinal = ref(false);
const summaryLoading = ref(false);
const summaryError = ref('');
const summaryScrollRef = ref<HTMLElement | null>(null);
const summaryHeight = ref(200);
const summaryResizing = ref(false);

const summaryContent = computed(() => summaryStream.value || summaryResult.value);

const timeRangeOptions = [
    { value: 'today', label: '今天' },
    { value: 'yesterday', label: '昨天' },
    { value: 'last-3d', label: '最近 3 天' },
    { value: 'last-7d', label: '最近 7 天' },
    { value: 'last-30d', label: '最近 30 天' },
    { value: 'custom', label: '自定义' }
];

const customStartDate = ref('');
const customEndDate = ref('');

const effectiveTimeRange = computed(() => {
    if (summaryTimeRange.value === 'custom' && customStartDate.value && customEndDate.value) {
        const start = customStartDate.value.replace('T', '/');
        const end = customEndDate.value.replace('T', '/');
        return `${start}~${end}`;
    }
    return summaryTimeRange.value;
});

watch(summaryStream, () => {
    nextTick(() => {
        if (summaryScrollRef.value) {
            summaryScrollRef.value.scrollTop = summaryScrollRef.value.scrollHeight;
        }
    });
});

function onResizeStart(e: PointerEvent) {
    summaryResizing.value = true;
    const startY = e.clientY;
    const startH = summaryHeight.value;
    const onMove = (ev: PointerEvent) => {
        const delta = startY - ev.clientY;
        summaryHeight.value = Math.max(80, Math.min(window.innerHeight * 0.7, startH + delta));
    };
    const onUp = () => {
        summaryResizing.value = false;
        document.removeEventListener('pointermove', onMove);
        document.removeEventListener('pointerup', onUp);
    };
    document.addEventListener('pointermove', onMove);
    document.addEventListener('pointerup', onUp);
}

async function generateSummary() {
    if (!props.contactUserName || !summaryProvider.value) return;

    summaryLoading.value = true;
    summaryError.value = '';
    summaryStream.value = '';
    summaryResult.value = '';
    summaryFinal.value = false;

    try {
        const resp = await backend.GetMessages(
            effectiveTimeRange.value,
            props.contactUserName,
            '', '', 500, 0, 'asc'
        );
        const items = resp.items || [];
        if (items.length === 0) {
            summaryError.value = '该时间范围内没有聊天记录';
            summaryLoading.value = false;
            return;
        }

        const messages: string[] = items.map((m: any) => {
            const name = m.senderName || m.sender || '';
            const content = m.content || '';
            const time = m.time || '';
            return `${time} ${name}: ${content}`;
        });

        const offChunk = backend.EventsOn('ai:summary:chunk', (chunk: unknown) => {
            if (typeof chunk === 'string') {
                summaryStream.value += chunk;
            }
        });
        const offDone = backend.EventsOn('ai:summary:done', () => {
            summaryResult.value = summaryStream.value;
            summaryFinal.value = true;
            summaryLoading.value = false;
            offChunk?.();
            offDone?.();
        });

        summaryHeight.value = Math.round(window.innerHeight * 0.7);
        await backend.GenerateAISummary(summaryProvider.value, messages, '');
    } catch (e) {
        summaryError.value = String((e as Error).message || e);
        summaryLoading.value = false;
    }
}

function copySummary() {
    const text = summaryStream.value || summaryResult.value;
    if (text) {
        navigator.clipboard.writeText(text);
        app.feedback.toast('已复制', '总结内容已复制到剪贴板');
    }
}

async function fetchProviders() {
    try {
        const list = await backend.ListAIProviders();
        summaryProviders.value = (list || []).map(p => ({
            id: p.id, name: p.name, type: p.type, baseUrl: p.baseUrl, model: p.model
        }));
        if (summaryProviders.value.length > 0) {
            const exists = summaryProviders.value.some(p => p.id === summaryProvider.value);
            if (!exists) {
                summaryProvider.value = summaryProviders.value[0].id;
            }
        }
    } catch { /* ignore */ }
}

function reset() {
    summaryStream.value = '';
    summaryResult.value = '';
    summaryFinal.value = false;
    summaryError.value = '';
    summaryTimeRange.value = 'today';
}

defineExpose({ fetchProviders, reset });
</script>

<template>
    <div class="shrink-0 space-y-3 border-t border-border/40 pt-3 pb-2 px-3">
        <div class="flex flex-wrap items-center gap-2">
            <CustomSelect
                :model-value="summaryTimeRange"
                :options="timeRangeOptions"
                direction="up"
                @update:model-value="(v: string) => { summaryTimeRange = v; }"
            />

            <template v-if="summaryTimeRange === 'custom'">
                <input
                    v-model="customStartDate"
                    type="datetime-local"
                    class="h-8 flex-1 min-w-[130px] rounded-md border border-input bg-background/40 px-2 text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-ring [color-scheme:dark]"
                />
                <span class="text-xs text-muted-foreground">~</span>
                <input
                    v-model="customEndDate"
                    type="datetime-local"
                    class="h-8 flex-1 min-w-[130px] rounded-md border border-input bg-background/40 px-2 text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-ring [color-scheme:dark]"
                />
            </template>

            <div class="flex-1 min-w-[120px]">
                <CustomSelect
                    :model-value="summaryProvider"
                    :options="providerOptions"
                    placeholder="选择提供商"
                    direction="up"
                    @update:model-value="(v: string) => { summaryProvider = v; }"
                />
            </div>

            <Button
                size="sm"
                class="h-8 gap-1 px-3 text-xs"
                :disabled="summaryLoading || !summaryProvider"
                @click="generateSummary"
            >
                <Sparkles class="size-3" />
                {{ summaryLoading ? '生成中…' : '总结' }}
            </Button>
        </div>

        <div v-if="summaryTimeRange === 'custom'" class="flex items-center gap-2">
            <input
                v-model="customStartDate"
                type="datetime-local"
                class="h-8 flex-1 rounded-md border border-input bg-background/40 px-2 text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-ring [color-scheme:dark]"
            />
            <span class="text-xs text-muted-foreground">~</span>
            <input
                v-model="customEndDate"
                type="datetime-local"
                class="h-8 flex-1 rounded-md border border-input bg-background/40 px-2 text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-ring [color-scheme:dark]"
            />
        </div>

        <div v-if="summaryError" class="text-xs text-destructive">{{ summaryError }}</div>

        <div
            v-if="summaryContent"
            class="relative pt-2 flex flex-col"
            :class="!summaryResizing && 'transition-[height] duration-300 ease-out'"
            :style="{ height: summaryHeight + 'px' }"
        >
            <div
                class="flex items-center justify-between mb-2 shrink-0 cursor-row-resize select-none"
                @pointerdown="onResizeStart"
            >
                <span class="text-xs font-medium text-foreground/80">AI 总结</span>
                <div class="flex flex-col gap-[3px]">
                    <div class="h-[1.5px] w-5 rounded-full bg-muted-foreground/30"></div>
                    <div class="h-[1.5px] w-5 rounded-full bg-muted-foreground/30"></div>
                    <div class="h-[1.5px] w-5 rounded-full bg-muted-foreground/30"></div>
                </div>
                <button
                    class="text-xs text-muted-foreground hover:text-foreground px-1.5 py-0.5 rounded hover:bg-muted/40 transition-colors"
                    @click.stop="copySummary"
                    @pointerdown.stop
                >
                    <Copy class="size-3 inline mr-0.5" />复制
                </button>
            </div>
            <div
                ref="summaryScrollRef"
                class="flex-1 min-h-0 overflow-auto rounded-md bg-muted/30 ring-1 ring-border/40 p-3"
            >
                <MarkdownRender :content="summaryContent" :final="summaryFinal" :max-live-nodes="0" />
            </div>
        </div>
    </div>
</template>