<script setup lang="ts">
import { computed, reactive, ref } from 'vue';
import { marked } from 'marked';
import { buildUrl, endpoints, getEndpoint, requestAISummaryStream, type EndpointKey, type ParamSpec } from './api';
import DatePicker from './components/DatePicker.vue';
import SearchableSelect from './components/SearchableSelect.vue';

const activeKey = ref<EndpointKey>('session');
const activeEndpoint = computed(() => getEndpoint(activeKey.value));

const formValues = reactive<Record<EndpointKey, Record<string, string>>>({
    session: {},
    chatroom: {},
    contact: {},
    chatlog: {}
});

const limit = ref(20);
const offset = ref(0);
const loading = ref(false);
const requestUrl = ref('');
const responseBody = ref('');
const responseStatus = ref('');
const responseTime = ref(0);
const total = ref<number | null>(null);
const errorMsg = ref('');
const copyHint = ref('');

// AI 总结相关状态
const showAISummaryModal = ref(false);
const aiProviders = ref<{ id: string; name: string; type: string; baseUrl: string; model: string }[]>([]);
const selectedProvider = ref('');
const summaryPrompt = ref(
    '请总结以下聊天记录的主要内容，包括：\n1. 聊天主题\n2. 关键讨论点\n3. 重要结论或待办事项\n\n聊天记录：'
);
const aiSummaryResult = ref('');
const aiSummaryLoading = ref(false);
const aiSummaryError = ref('');
const aiSummaryStream = ref('');

const currentValues = computed({
    get: () => formValues[activeKey.value] || {},
    set: (v: Record<string, string>) => {
        formValues[activeKey.value] = v;
    }
});

function selectTab(key: EndpointKey) {
    activeKey.value = key;
    offset.value = 0;
    requestUrl.value = '';
    responseBody.value = '';
    responseStatus.value = '';
    total.value = null;
    errorMsg.value = '';
}

async function send(resetOffset = true) {
    if (resetOffset) offset.value = 0;
    errorMsg.value = '';

    const built = buildUrl(activeEndpoint.value, currentValues.value, {
        offset: offset.value,
        limit: limit.value
    });
    if (built.error) {
        errorMsg.value = built.error;
        return;
    }
    requestUrl.value = built.fullUrl;
    loading.value = true;
    responseBody.value = '';
    responseStatus.value = '';
    total.value = null;

    const start = performance.now();
    try {
        const resp = await fetch(built.apiUrl);
        const totalHeader = parseInt(resp.headers.get('X-Total-Count') || '');
        const ct = resp.headers.get('content-type') || '';
        let body = '';
        let parsedTotal: number | null = null;
        if (ct.includes('application/json')) {
            const data = await resp.json();
            body = JSON.stringify(data, null, 2);
            if (data && typeof data.total === 'number') parsedTotal = data.total;
        } else {
            body = await resp.text();
        }
        responseTime.value = Math.round(performance.now() - start);
        responseStatus.value = resp.ok ? `${resp.status} ${resp.statusText || 'OK'}` : `${resp.status}`;
        responseBody.value = body;
        if (parsedTotal === null && !isNaN(totalHeader)) parsedTotal = totalHeader;
        if (parsedTotal !== null) total.value = parsedTotal;
        if (!resp.ok) errorMsg.value = `HTTP ${resp.status}`;
    } catch (e) {
        responseTime.value = Math.round(performance.now() - start);
        errorMsg.value = `请求失败：${(e as Error).message}`;
    } finally {
        loading.value = false;
    }
}

function setOffset(next: number) {
    offset.value = Math.max(0, next);
    void send(false);
}

const totalPages = computed(() => {
    if (total.value === null || total.value <= 0) return 0;
    return Math.ceil(total.value / limit.value);
});
const currentPage = computed(() => Math.floor(offset.value / limit.value) + 1);

type RangePreset = '7d' | '30d' | '180d';

function formatDate(date: Date): string {
    const year = date.getFullYear();
    const month = String(date.getMonth() + 1).padStart(2, '0');
    const day = String(date.getDate()).padStart(2, '0');
    return `${year}-${month}-${day}`;
}

function applyDateRangePreset(value: RangePreset) {
    const days = value === '7d' ? 7 : value === '30d' ? 30 : 180;
    const end = new Date();
    const start = new Date();
    start.setDate(end.getDate() - days + 1);
    currentValues.value.startDate = formatDate(start);
    currentValues.value.endDate = formatDate(end);
}

const visiblePages = computed(() => {
    const pages: (number | '...')[] = [];
    const total = totalPages.value;
    const cur = currentPage.value;
    if (total <= 7) {
        for (let i = 1; i <= total; i++) pages.push(i);
        return pages;
    }
    pages.push(1);
    if (cur > 3) pages.push('...');
    for (let i = Math.max(2, cur - 1); i <= Math.min(total - 1, cur + 1); i++) pages.push(i);
    if (cur < total - 2) pages.push('...');
    pages.push(total);
    return pages;
});

function goPage(p: number) {
    setOffset((p - 1) * limit.value);
}

function changeLimit(v: string) {
    limit.value = Number(v) || 20;
    setOffset(0);
}

async function copyText(text: string, label = '已复制') {
    try {
        await navigator.clipboard.writeText(text);
    } catch {
        const el = document.createElement('textarea');
        el.value = text;
        document.body.appendChild(el);
        el.select();
        document.execCommand('copy');
        document.body.removeChild(el);
    }
    copyHint.value = label;
    setTimeout(() => (copyHint.value = ''), 1500);
}

function buildCurl(): string {
    if (!requestUrl.value) {
        const built = buildUrl(activeEndpoint.value, currentValues.value, {
            offset: offset.value,
            limit: limit.value
        });
        if (built.error) return '';
        return `curl '${built.fullUrl}'`;
    }
    return `curl '${requestUrl.value}'`;
}

function isParamFilled(p: ParamSpec): boolean {
    const v = currentValues.value[p.key];
    return typeof v === 'string' && v.length > 0;
}

// AI 总结相关方法
async function fetchAIProviders() {
    try {
        const resp = await fetch('/api/v1/ai/providers');
        const data = await resp.json();
        aiProviders.value = data.providers || [];
        if (aiProviders.value.length > 0 && !selectedProvider.value) {
            selectedProvider.value = aiProviders.value[0].id;
        }
    } catch (e) {
        console.error('获取 AI 提供商失败:', e);
    }
}

function openAISummaryModal() {
    aiSummaryResult.value = '';
    aiSummaryError.value = '';
    void fetchAIProviders();
    showAISummaryModal.value = true;
}

function closeAISummaryModal() {
    showAISummaryModal.value = false;
}

function extractMessagesFromResponse(): string[] {
    try {
        const data = JSON.parse(responseBody.value);
        // 如果是数组，直接返回
        if (Array.isArray(data)) {
            return data
                .map((item: any) => {
                    if (typeof item === 'string') return item;
                    if (item.content) return String(item.content);
                    if (item.message) return String(item.message);
                    return JSON.stringify(item);
                })
                .filter((m: string) => m.trim().length > 0);
        }
        // 如果是对象且有 data 字段
        if (data.data && Array.isArray(data.data)) {
            return data.data
                .map((item: any) => {
                    if (typeof item === 'string') return item;
                    if (item.content) return String(item.content);
                    if (item.message) return String(item.message);
                    return JSON.stringify(item);
                })
                .filter((m: string) => m.trim().length > 0);
        }
        return [];
    } catch {
        // 如果不是 JSON，尝试按行分割
        return responseBody.value.split('\n').filter(l => l.trim().length > 0);
    }
}

async function generateAISummary() {
    if (!selectedProvider.value) {
        aiSummaryError.value = '请先选择一个 AI 提供商';
        return;
    }
    const messages = extractMessagesFromResponse();
    if (messages.length === 0) {
        aiSummaryError.value = '没有可总结的聊天内容';
        return;
    }
    aiSummaryLoading.value = true;
    aiSummaryError.value = '';
    aiSummaryResult.value = '';
    aiSummaryStream.value = '';
    try {
        const stream = await requestAISummaryStream({
            providerId: selectedProvider.value,
            messages: messages.slice(0, 50), // 限制消息数量，避免超出 token 限制
            prompt: summaryPrompt.value
        });

        const reader = stream.getReader();
        const decoder = new TextDecoder();

        while (true) {
            const { done, value } = await reader.read();
            if (done) break;

            const chunk = decoder.decode(value);
            const lines = chunk.split('\n');

            for (const line of lines) {
                if (line.startsWith('data: ')) {
                    const data = line.slice(6);
                    if (data === '[DONE]') {
                        aiSummaryResult.value = aiSummaryStream.value;
                        return;
                    }
                    if (data) {
                        aiSummaryStream.value += data;
                    }
                }
            }
        }
    } catch (e) {
        aiSummaryError.value = `生成失败：${(e as Error).message}`;
    } finally {
        aiSummaryLoading.value = false;
    }
}

function copyAISummary() {
    const text = aiSummaryStream.value || aiSummaryResult.value;
    if (text) {
        void copyText(text, '总结已复制');
    }
}

const renderedMarkdown = computed(() => {
    const raw = aiSummaryStream.value || aiSummaryResult.value;
    if (!raw) return '';
    return marked.parse(raw) as string;
});
</script>

<template>
    <div class="min-h-screen pb-20">
        <header class="border-b border-border/60 bg-background/80 backdrop-blur sticky top-0 z-40">
            <div class="mx-auto flex w-full max-w-5xl items-center justify-between gap-4 px-4 py-4 sm:px-6">
                <div class="flex items-baseline gap-3 min-w-0">
                    <span class="font-serif text-2xl font-medium tracking-tight">Chatlog</span>
                    <span class="hidden text-xs text-muted-foreground sm:inline"
                        >通过 HTTP API 访问聊天记录、联系人与会话</span
                    >
                </div>
                <div class="flex items-center gap-3 text-xs text-muted-foreground">
                    <a
                        href="https://github.com/sjzar/chatlog"
                        target="_blank"
                        rel="noopener"
                        class="hover:text-foreground"
                        >GitHub</a
                    >
                    <a
                        href="https://github.com/sjzar/chatlog/blob/main/docs/mcp.md"
                        target="_blank"
                        rel="noopener"
                        class="hover:text-foreground"
                        >MCP 文档</a
                    >
                </div>
            </div>
        </header>

        <main class="mx-auto w-full max-w-5xl space-y-10 px-4 py-8 sm:px-6">
            <section class="space-y-4">
                <header class="space-y-1">
                    <h2 class="font-serif text-xl font-medium tracking-tight">API 调试</h2>
                    <p class="text-sm text-muted-foreground">选择接口并填写参数，发起请求查看返回结果。</p>
                </header>

                <div class="flex flex-wrap items-center gap-1 border-b border-border/60">
                    <button
                        v-for="ep in endpoints"
                        :key="ep.key"
                        :class="[
                            '-mb-px px-3 py-2 text-sm transition-colors',
                            activeKey === ep.key
                                ? 'border-b-2 border-foreground font-medium text-foreground'
                                : 'border-b-2 border-transparent text-muted-foreground hover:text-foreground'
                        ]"
                        @click="selectTab(ep.key)"
                    >
                        {{ ep.label }}
                    </button>
                </div>

                <div class="rounded-xl bg-muted/40 ring-1 ring-border/60 p-5 sm:p-6 space-y-4">
                    <div class="flex flex-wrap items-baseline gap-x-3 gap-y-1">
                        <span class="rounded bg-foreground/10 px-1.5 py-0.5 font-mono text-xs">GET</span>
                        <span class="font-mono text-sm text-foreground/85">{{ activeEndpoint.path }}</span>
                    </div>
                    <p class="text-sm text-muted-foreground">{{ activeEndpoint.description }}</p>

                    <div class="grid gap-4 sm:grid-cols-2">
                        <div v-for="p in activeEndpoint.params" :key="p.key" class="space-y-1.5">
                            <label class="flex items-baseline gap-1.5 text-xs text-muted-foreground">
                                <span class="text-foreground/85">{{ p.label }}</span>
                                <span v-if="p.required" class="text-destructive">*</span>
                                <span v-else class="text-muted-foreground/60">可选</span>
                            </label>
                            <input
                                v-if="p.type === 'text'"
                                v-model="currentValues[p.key]"
                                :placeholder="p.placeholder"
                                class="h-9 w-full rounded-md border border-input bg-background/40 px-3 text-sm focus:outline-none focus:ring-1 focus:ring-ring"
                            />
                            <DatePicker
                                v-else-if="p.type === 'date'"
                                :model-value="currentValues[p.key] ?? ''"
                                :placeholder="p.placeholder || '选择日期'"
                                @update:model-value="(v: string) => (currentValues[p.key] = v)"
                                @range-preset="applyDateRangePreset"
                            />
                            <select
                                v-else-if="p.type === 'select'"
                                :value="currentValues[p.key] ?? ''"
                                class="h-9 w-full rounded-md border border-input bg-background/40 px-2.5 text-sm focus:outline-none focus:ring-1 focus:ring-ring"
                                @change="
                                    (e: Event) => (currentValues[p.key] = (e.target as HTMLSelectElement).value)
                                "
                            >
                                <option v-for="opt in p.options" :key="opt.value" :value="opt.value">
                                    {{ opt.label }}
                                </option>
                            </select>
                            <SearchableSelect
                                v-else-if="p.type === 'autocomplete' && p.source"
                                :model-value="currentValues[p.key] ?? ''"
                                :placeholder="p.placeholder"
                                :source="p.source"
                                @update:model-value="(v: string) => (currentValues[p.key] = v)"
                            />
                            <p v-if="p.hint" class="text-xs text-muted-foreground/70">{{ p.hint }}</p>
                        </div>

                        <div class="space-y-1.5">
                            <label class="text-xs text-muted-foreground">每页条数</label>
                            <select
                                :value="limit"
                                class="h-9 w-full rounded-md border border-input bg-background/40 px-2.5 text-sm focus:outline-none focus:ring-1 focus:ring-ring"
                                @change="(e: Event) => changeLimit((e.target as HTMLSelectElement).value)"
                            >
                                <option :value="10">10</option>
                                <option :value="20">20</option>
                                <option :value="50">50</option>
                                <option :value="100">100</option>
                                <option :value="200">200</option>
                            </select>
                        </div>
                    </div>

                    <div class="flex flex-wrap items-center justify-between gap-2 pt-1">
                        <div class="flex flex-wrap gap-1.5 text-[11px] text-muted-foreground/70">
                            <span v-for="p in activeEndpoint.params.filter(isParamFilled)" :key="p.key"
                                >· {{ p.label }}</span
                            >
                        </div>
                        <div class="flex items-center gap-2">
                            <button
                                class="h-8 rounded-md px-2.5 text-xs font-normal text-muted-foreground hover:text-foreground"
                                @click="copyText(buildCurl(), '已复制 cURL')"
                            >
                                复制 cURL
                            </button>
                            <button
                                :disabled="loading"
                                class="h-9 rounded-md bg-primary px-4 text-sm text-primary-foreground hover:opacity-90 disabled:opacity-60"
                                @click="send(true)"
                            >
                                {{ loading ? '请求中…' : '发送请求' }}
                            </button>
                        </div>
                    </div>

                    <p v-if="errorMsg" class="text-sm text-destructive">{{ errorMsg }}</p>
                </div>
            </section>

            <section v-if="requestUrl || loading" class="space-y-4">
                <header class="flex flex-wrap items-baseline justify-between gap-2">
                    <h2 class="font-serif text-xl font-medium tracking-tight">响应结果</h2>
                    <div class="flex items-center gap-3 text-xs text-muted-foreground">
                        <span v-if="responseStatus" class="inline-flex items-center gap-1.5">
                            <span
                                :class="[
                                    'size-1.5 rounded-full',
                                    responseStatus.startsWith('2') ? 'bg-emerald-500' : 'bg-rose-500'
                                ]"
                            />
                            <span>{{ responseStatus }}</span>
                        </span>
                        <span v-if="responseTime > 0">· {{ responseTime }} ms</span>
                        <span v-if="total !== null">· 共 {{ total }} 条</span>
                    </div>
                </header>

                <div class="rounded-xl bg-muted/40 ring-1 ring-border/60 p-5 sm:p-6 space-y-4">
                    <div class="flex items-center gap-2">
                        <code
                            class="flex-1 truncate rounded-md bg-background/50 px-3 py-1.5 font-mono text-xs text-foreground/85"
                            :title="requestUrl"
                            >{{ requestUrl || '—' }}</code
                        >
                        <button
                            class="h-8 rounded-md px-2.5 text-xs font-normal text-muted-foreground hover:text-foreground"
                            :disabled="!requestUrl"
                            @click="copyText(requestUrl, '已复制 URL')"
                        >
                            复制 URL
                        </button>
                    </div>

                    <div v-if="totalPages > 1" class="flex flex-wrap items-center justify-center gap-1">
                        <button
                            class="h-8 rounded-md px-2.5 text-xs text-muted-foreground hover:text-foreground disabled:opacity-40"
                            :disabled="currentPage <= 1 || loading"
                            @click="goPage(currentPage - 1)"
                        >
                            上一页
                        </button>
                        <template v-for="(p, i) in visiblePages" :key="i">
                            <span
                                v-if="p === '...'"
                                class="h-8 inline-flex items-center px-1.5 text-xs text-muted-foreground/60"
                                >…</span
                            >
                            <button
                                v-else
                                :class="[
                                    'h-8 min-w-8 rounded-md px-2 text-xs',
                                    p === currentPage
                                        ? 'bg-foreground text-background'
                                        : 'text-muted-foreground hover:text-foreground'
                                ]"
                                :disabled="loading"
                                @click="goPage(p)"
                            >
                                {{ p }}
                            </button>
                        </template>
                        <button
                            class="h-8 rounded-md px-2.5 text-xs text-muted-foreground hover:text-foreground disabled:opacity-40"
                            :disabled="currentPage >= totalPages || loading"
                            @click="goPage(currentPage + 1)"
                        >
                            下一页
                        </button>
                    </div>

                    <div
                        class="relative max-h-[480px] overflow-auto rounded-md bg-background/50 ring-1 ring-border/40 p-4"
                    >
                        <div
                            v-if="loading"
                            class="flex min-h-[120px] items-center justify-center text-sm text-muted-foreground"
                        >
                            <span
                                class="size-3.5 animate-spin rounded-full border-2 border-muted-foreground/40 border-t-foreground"
                            />
                            <span class="ml-2">请求中…</span>
                        </div>
                        <pre
                            v-else-if="responseBody"
                            class="whitespace-pre-wrap break-all font-mono text-xs leading-relaxed text-foreground/90"
                            >{{ responseBody }}</pre
                        >
                        <div v-else class="text-sm text-muted-foreground">暂无响应数据。</div>
                    </div>

                    <div v-if="responseBody" class="flex justify-end gap-2">
                        <button
                            class="h-8 rounded-md px-2.5 text-xs font-normal text-muted-foreground hover:text-foreground"
                            @click="copyText(responseBody, '已复制响应')"
                        >
                            复制响应
                        </button>
                        <button
                            class="h-8 rounded-md bg-primary px-2.5 text-xs font-normal text-primary-foreground hover:opacity-90"
                            @click="openAISummaryModal"
                        >
                            AI 总结
                        </button>
                    </div>
                </div>
            </section>

            <section class="space-y-4">
                <header class="space-y-1">
                    <h2 class="font-serif text-xl font-medium tracking-tight">MCP 集成</h2>
                    <p class="text-sm text-muted-foreground">支持 Model Context Protocol，可与兼容 MCP 的助手集成。</p>
                </header>
                <div class="rounded-xl bg-muted/40 ring-1 ring-border/60 p-5 sm:p-6">
                    <div class="grid gap-4 sm:grid-cols-2">
                        <div class="space-y-1.5">
                            <div class="text-xs text-muted-foreground">SSE 端点</div>
                            <code class="block break-all rounded-md bg-background/50 px-3 py-1.5 font-mono text-xs"
                                >/sse</code
                            >
                        </div>
                        <div class="space-y-1.5">
                            <div class="text-xs text-muted-foreground">Streamable HTTP</div>
                            <code class="block break-all rounded-md bg-background/50 px-3 py-1.5 font-mono text-xs"
                                >/mcp</code
                            >
                        </div>
                    </div>
                    <p class="mt-4 text-sm text-muted-foreground">
                        集成指南：<a
                            class="text-foreground underline-offset-4 hover:underline"
                            target="_blank"
                            rel="noopener"
                            href="https://github.com/sjzar/chatlog/blob/main/docs/mcp.md"
                            >docs/mcp.md</a
                        >
                    </p>
                </div>
            </section>
        </main>

        <Transition
            enter-from-class="translate-y-2 opacity-0"
            enter-active-class="transition duration-200"
            leave-active-class="transition duration-150"
            leave-to-class="translate-y-2 opacity-0"
        >
            <div
                v-if="copyHint"
                class="fixed bottom-6 left-1/2 -translate-x-1/2 rounded-full bg-foreground px-4 py-1.5 text-xs text-background shadow-lg"
            >
                {{ copyHint }}
            </div>
        </Transition>

        <!-- AI 总结对话框 -->
        <Transition
            enter-from-class="opacity-0"
            enter-active-class="transition duration-200"
            leave-active-class="transition duration-150"
            leave-to-class="opacity-0"
        >
            <div
                v-if="showAISummaryModal"
                class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4 backdrop-blur-sm"
                @click.self="closeAISummaryModal"
            >
                <div
                    class="w-full max-w-2xl max-h-[90vh] overflow-y-auto rounded-xl bg-background shadow-2xl ring-1 ring-border/60"
                >
                    <div
                        class="sticky top-0 z-10 flex items-center justify-between gap-4 border-b border-border/60 bg-background/95 px-5 py-4 backdrop-blur"
                    >
                        <h3 class="font-serif text-lg font-medium tracking-tight">AI 总结</h3>
                        <button
                            class="h-8 w-8 rounded-md text-muted-foreground hover:bg-muted hover:text-foreground"
                            @click="closeAISummaryModal"
                        >
                            <svg
                                class="size-4"
                                xmlns="http://www.w3.org/2000/svg"
                                viewBox="0 0 24 24"
                                fill="none"
                                stroke="currentColor"
                                stroke-width="2"
                                stroke-linecap="round"
                                stroke-linejoin="round"
                            >
                                <path d="M18 6 6 18" />
                                <path d="m6 6 12 12" />
                            </svg>
                        </button>
                    </div>

                    <div class="space-y-5 p-5">
                        <!-- 提供商选择 -->
                        <div class="space-y-2">
                            <label class="text-sm font-medium text-foreground">选择 AI 提供商</label>
                            <select
                                v-model="selectedProvider"
                                class="h-9 w-full rounded-md border border-input bg-background/50 px-3 text-sm focus:outline-none focus:ring-1 focus:ring-ring"
                            >
                                <option value="" disabled>请选择提供商</option>
                                <option v-for="p in aiProviders" :key="p.id" :value="p.id">
                                    {{ p.name }} ({{ p.model || p.type }})
                                </option>
                            </select>
                            <p v-if="aiProviders.length === 0" class="text-xs text-muted-foreground">
                                暂无可用提供商，请先在配置中添加 AI 提供商。
                            </p>
                        </div>

                        <!-- 提示词编辑 -->
                        <div class="space-y-2">
                            <label class="text-sm font-medium text-foreground">提示词</label>
                            <textarea
                                v-model="summaryPrompt"
                                rows="4"
                                class="w-full rounded-md border border-input bg-background/50 px-3 py-2 text-sm focus:outline-none focus:ring-1 focus:ring-ring resize-none"
                                placeholder="请输入提示词..."
                            ></textarea>
                            <p class="text-xs text-muted-foreground">可修改提示词以自定义总结风格和内容。</p>
                        </div>

                        <!-- 生成按钮 -->
                        <div class="flex items-center justify-between gap-3">
                            <div v-if="aiSummaryError" class="text-xs text-destructive">{{ aiSummaryError }}</div>
                            <div v-else class="text-xs text-muted-foreground">
                                将分析当前响应中的聊天记录（最多 50 条）
                            </div>
                            <button
                                class="h-9 rounded-md bg-primary px-4 text-sm font-medium text-primary-foreground hover:opacity-90 disabled:opacity-50"
                                :disabled="aiSummaryLoading || !selectedProvider"
                                @click="generateAISummary"
                            >
                                {{ aiSummaryLoading ? '生成中…' : '生成总结' }}
                            </button>
                        </div>

                        <!-- 加载状态 -->
                        <div v-if="aiSummaryLoading" class="flex items-center justify-center gap-2 py-8">
                            <span
                                class="size-4 animate-spin rounded-full border-2 border-muted-foreground/40 border-t-foreground"
                            ></span>
                            <span class="text-sm text-muted-foreground">AI 正在分析聊天记录…</span>
                        </div>

                        <!-- 总结结果 -->
                        <div v-if="aiSummaryResult || aiSummaryStream" class="space-y-3">
                            <div class="flex items-center justify-between">
                                <label class="text-sm font-medium text-foreground">总结结果</label>
                                <button
                                    class="h-7 rounded-md px-2.5 text-xs font-normal text-muted-foreground hover:text-foreground"
                                    @click="copyAISummary"
                                >
                                    复制
                                </button>
                            </div>
                            <div
                                class="relative max-h-[300px] overflow-auto rounded-md bg-muted/30 ring-1 ring-border/40 p-4"
                            >
                                <div
                                    class="markdown-body text-sm leading-relaxed text-foreground/90"
                                    v-html="renderedMarkdown"
                                />
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        </Transition>
    </div>
</template>
