<script setup lang="ts">
import { computed, reactive, ref } from 'vue';
import { buildUrl, endpoints, getEndpoint, type EndpointKey, type ParamSpec } from './api';
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
                            <input
                                v-else-if="p.type === 'date'"
                                v-model="currentValues[p.key]"
                                type="date"
                                class="h-9 w-full rounded-md border border-input bg-background/40 px-3 text-sm focus:outline-none focus:ring-1 focus:ring-ring"
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

                    <div v-if="responseBody" class="flex justify-end">
                        <button
                            class="h-8 rounded-md px-2.5 text-xs font-normal text-muted-foreground hover:text-foreground"
                            @click="copyText(responseBody, '已复制响应')"
                        >
                            复制响应
                        </button>
                    </div>

                    <div v-if="totalPages > 1" class="flex flex-wrap items-center justify-center gap-1 pt-1">
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
    </div>
</template>
