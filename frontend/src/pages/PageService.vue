<script setup lang="ts">
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { computed, inject, onMounted, onUnmounted, reactive, ref, watch } from 'vue';
import { appContextKey } from '../app/context';
import { backend } from '../wailsbridge';

const injected = inject(appContextKey);
if (!injected) throw new Error('chatlog not provided');
const app = injected;

const { httpAddr, state, run } = app;

const listenIPs = ref<string[]>(['127.0.0.1', '0.0.0.0']);
const listenIP = ref('127.0.0.1');
const listenPort = ref('5030');
const ipDropdownOpen = ref(false);
const ipDropdownRef = ref<HTMLElement | null>(null);

function toggleIpDropdown() {
    ipDropdownOpen.value = !ipDropdownOpen.value;
}

function selectIp(ip: string) {
    listenIP.value = ip;
    ipDropdownOpen.value = false;
}

function ipLabel(ip: string): string {
    if (ip === '127.0.0.1') return '仅本机';
    if (ip === '0.0.0.0') return '所有网卡';
    return '局域网';
}

function onClickOutside(e: MouseEvent) {
    if (!ipDropdownOpen.value) return;
    if (ipDropdownRef.value && !ipDropdownRef.value.contains(e.target as Node)) {
        ipDropdownOpen.value = false;
    }
}

onMounted(() => {
    document.addEventListener('click', onClickOutside);
});
onUnmounted(() => {
    document.removeEventListener('click', onClickOutside);
});

function parseAddr(addr: string): { ip: string; port: string } {
    if (!addr) return { ip: '127.0.0.1', port: '5030' };
    const idx = addr.lastIndexOf(':');
    if (idx <= 0) return { ip: addr, port: '5030' };
    return { ip: addr.slice(0, idx) || '127.0.0.1', port: addr.slice(idx + 1) || '5030' };
}

watch(
    httpAddr,
    val => {
        const { ip, port } = parseAddr(val);
        listenIP.value = ip;
        listenPort.value = port;
    },
    { immediate: true }
);

async function loadListenIPs() {
    try {
        const ips = await backend.ListListenIPs();
        if (ips && ips.length) {
            listenIPs.value = ips;
            if (!ips.includes(listenIP.value)) listenIPs.value = [...ips, listenIP.value];
        }
    } catch {
        // keep defaults
    }
}
onMounted(loadListenIPs);

const composedAddr = computed(() => `${listenIP.value}:${listenPort.value}`);

function saveAddr() {
    const port = listenPort.value.trim();
    if (!/^\d+$/.test(port) || Number(port) < 1 || Number(port) > 65535) {
        app.feedback.toast('端口无效', '请输入 1 - 65535 范围内的端口号。');
        return;
    }
    return app.feedback
        .confirm({
            title: '保存 HTTP 地址',
            message: `确认将监听地址保存为 ${composedAddr.value} 并写入配置？`,
            confirmText: '保存',
            cancelText: '取消'
        })
        .then(ok => (ok ? run(() => backend.SetHTTPAddr(composedAddr.value), '已保存') : undefined));
}

async function toggleHTTP() {
    if (state.value?.httpEnabled) {
        const ok = await app.feedback.confirm({
            title: '停止 HTTP 服务',
            message: '确认停止 HTTP 服务？停止后 API 与 MCP 接口将不可访问。',
            confirmText: '停止',
            cancelText: '取消',
            danger: true
        });
        if (!ok) return;
        return run(() => backend.StopHTTP(), '已停止');
    }
    return run(() => backend.StartHTTP(), '已启动');
}

function baseUrl() {
    const addr = state.value?.httpAddr || httpAddr.value || '127.0.0.1:5030';
    return `http://${addr}`;
}

function endpointUrl(path: string) {
    return `${baseUrl()}${path}`;
}

interface Endpoint {
    name: string;
    method: string;
    path: string;
    desc: string;
    params?: Array<{ key: string; placeholder: string; desc: string }>;
}

const endpoints: Endpoint[] = [
    {
        name: '会话列表',
        method: 'GET',
        path: '/api/v1/session',
        desc: '获取最近会话列表',
        params: [
            { key: 'keyword', placeholder: '关键词', desc: '内容过滤' },
            { key: 'limit', placeholder: '50', desc: '返回数量' },
            { key: 'offset', placeholder: '0', desc: '分页偏移' },
            { key: 'format', placeholder: 'json', desc: '输出格式' }
        ]
    },
    {
        name: '联系人列表',
        method: 'GET',
        path: '/api/v1/contact',
        desc: '获取联系人列表',
        params: [
            { key: 'keyword', placeholder: '关键词', desc: '昵称/备注过滤' },
            { key: 'limit', placeholder: '20', desc: '返回数量' },
            { key: 'offset', placeholder: '0', desc: '分页偏移' },
            { key: 'format', placeholder: 'json', desc: '输出格式' }
        ]
    },
    {
        name: '群聊列表',
        method: 'GET',
        path: '/api/v1/chatroom',
        desc: '获取群聊列表',
        params: [
            { key: 'keyword', placeholder: '关键词', desc: '群名过滤' },
            { key: 'limit', placeholder: '20', desc: '返回数量' },
            { key: 'offset', placeholder: '0', desc: '分页偏移' },
            { key: 'format', placeholder: 'json', desc: '输出格式' }
        ]
    },
    {
        name: '聊天记录',
        method: 'GET',
        path: '/api/v1/chatlog',
        desc: '按条件查询聊天记录',
        params: [
            { key: 'time', placeholder: '2025-01-01', desc: '时间范围' },
            { key: 'talker', placeholder: 'wxid_xxx', desc: '聊天对象' },
            { key: 'sender', placeholder: 'wxid_xxx', desc: '发送者' },
            { key: 'keyword', placeholder: '关键词', desc: '内容搜索' },
            { key: 'limit', placeholder: '20', desc: '返回数量' },
            { key: 'offset', placeholder: '0', desc: '分页偏移' },
            { key: 'format', placeholder: 'json', desc: '输出格式' }
        ]
    },
    {
        name: 'MCP',
        method: 'GET',
        path: '/mcp',
        desc: 'MCP Streamable HTTP 端点'
    }
];

const epParams = reactive<Record<string, Record<string, string>>>({
    '/api/v1/session': { keyword: '', limit: '', offset: '', format: 'json' },
    '/api/v1/contact': { keyword: '', limit: '', offset: '', format: 'json' },
    '/api/v1/chatroom': { keyword: '', limit: '', offset: '', format: 'json' },
    '/api/v1/chatlog': { time: '', talker: '', sender: '', keyword: '', limit: '', offset: '', format: 'json' }
});

function fullUrl(ep: Endpoint): string {
    const base = `${baseUrl()}${ep.path}`;
    const params = epParams[ep.path];
    if (!params) return base;

    const qs = Object.entries(params)
        .map(([key, value]) => (value.trim() ? `${encodeURIComponent(key)}=${encodeURIComponent(value.trim())}` : ''))
        .filter(Boolean)
        .join('&');

    return qs ? `${base}?${qs}` : base;
}

function curlCmd(ep: Endpoint): string {
    return `curl -X ${ep.method} "${fullUrl(ep)}"`;
}

const copiedId = ref('');
const responses = reactive<Record<string, string>>({});
const apiLoading = reactive<Record<string, boolean>>({});

async function copyText(text: string) {
    try {
        await navigator.clipboard.writeText(text);
        app.feedback.toast('已复制', '已复制到剪贴板');
    } catch {
        app.feedback.toast('复制失败', '当前环境不支持剪贴板');
    }
}

async function copyCmd(ep: Endpoint) {
    const cmd = curlCmd(ep);
    try {
        await navigator.clipboard.writeText(cmd);
        app.feedback.toast('已复制', cmd);
        copiedId.value = ep.path;
        setTimeout(() => {
            if (copiedId.value === ep.path) copiedId.value = '';
        }, 1600);
    } catch {
        app.feedback.toast('复制失败', '浏览器不支持剪贴板操作');
    }
}

async function tryApi(ep: Endpoint) {
    const url = fullUrl(ep);
    apiLoading[ep.path] = true;
    responses[ep.path] = '';

    try {
        const res = await fetch(url);
        if (!res.ok) throw new Error(`HTTP Error: ${res.status}`);
        const data = await res.json();
        responses[ep.path] = JSON.stringify(data, null, 2);
    } catch (error) {
        responses[ep.path] = `FAILED: ${String(error)}`;
    } finally {
        apiLoading[ep.path] = false;
    }
}
</script>

<template>
    <div class="space-y-10">
        <div
            class="sticky -top-px z-30 flex flex-wrap items-center justify-between gap-3 rounded-xl bg-muted/30 ring-1 ring-border/40 px-4 py-3 backdrop-blur supports-[backdrop-filter]:bg-muted/40"
        >
            <div class="flex items-center gap-2 text-sm">
                <span
                    :class="['size-1.5 rounded-full', state?.httpEnabled ? 'bg-emerald-500' : 'bg-muted-foreground/40']"
                />
                <span :class="state?.httpEnabled ? 'text-foreground' : 'text-muted-foreground'">
                    {{ state?.httpEnabled ? 'HTTP 服务运行中' : 'HTTP 服务未启动' }}
                </span>
            </div>
            <div class="flex items-center gap-2">
                <Button
                    v-if="state?.httpEnabled"
                    variant="ghost"
                    size="sm"
                    class="h-9 px-3 text-sm font-normal text-muted-foreground hover:text-foreground"
                    @click="backend.OpenURL(baseUrl() || endpointUrl('/'))"
                >
                    打开 Web UI
                </Button>
                <Button
                    :variant="state?.httpEnabled ? 'outline' : 'default'"
                    size="sm"
                    class="h-9 px-4 text-sm"
                    @click="toggleHTTP"
                >
                    {{ state?.httpEnabled ? '停止服务' : '启动服务' }}
                </Button>
            </div>
        </div>

        <section class="space-y-4">
            <header class="space-y-1">
                <h3 class="font-serif text-xl font-medium tracking-tight text-foreground">服务控制</h3>
                <p class="text-sm text-muted-foreground">配置并启动 HTTP API 与 MCP 服务。</p>
            </header>

            <div class="rounded-xl bg-muted/30 ring-1 ring-border/40 p-5 sm:p-6 space-y-5">
                <div class="grid gap-3 md:grid-cols-[minmax(140px,200px)_minmax(0,1fr)] md:items-center">
                    <div>
                        <div class="text-sm text-foreground">监听地址</div>
                        <div class="text-xs text-muted-foreground">保存后需重启服务生效。</div>
                    </div>
                    <div class="flex flex-wrap items-center gap-2">
                        <div ref="ipDropdownRef" class="relative min-w-[200px] flex-1">
                            <button
                                type="button"
                                :class="[
                                    'group flex h-10 w-full items-center justify-between gap-2 rounded-md border border-input bg-background/40 px-3 font-mono text-sm transition-colors hover:bg-background/60 focus:outline-none focus:ring-1 focus:ring-ring',
                                    ipDropdownOpen && 'ring-1 ring-ring'
                                ]"
                                @click.stop="toggleIpDropdown"
                            >
                                <span class="flex items-baseline gap-2 truncate">
                                    <span>{{ listenIP }}</span>
                                    <span class="font-sans text-xs text-muted-foreground">{{ ipLabel(listenIP) }}</span>
                                </span>
                                <svg
                                    :class="[
                                        'size-3.5 text-muted-foreground transition-transform',
                                        ipDropdownOpen && 'rotate-180'
                                    ]"
                                    xmlns="http://www.w3.org/2000/svg"
                                    viewBox="0 0 24 24"
                                    fill="none"
                                    stroke="currentColor"
                                    stroke-width="2"
                                    stroke-linecap="round"
                                    stroke-linejoin="round"
                                >
                                    <path d="m6 9 6 6 6-6" />
                                </svg>
                            </button>
                            <Transition
                                enter-from-class="opacity-0 -translate-y-1"
                                enter-active-class="transition duration-150"
                                leave-active-class="transition duration-100"
                                leave-to-class="opacity-0 -translate-y-1"
                            >
                                <div
                                    v-if="ipDropdownOpen"
                                    class="absolute left-0 right-0 top-full z-30 mt-1.5 max-h-60 overflow-auto rounded-md border border-border/60 bg-popover/95 p-1 shadow-lg backdrop-blur"
                                >
                                    <button
                                        v-for="ip in listenIPs"
                                        :key="ip"
                                        type="button"
                                        :class="[
                                            'flex w-full items-center justify-between gap-2 rounded-md px-2.5 py-1.5 text-left text-sm transition-colors',
                                            ip === listenIP
                                                ? 'bg-accent text-foreground'
                                                : 'text-foreground/85 hover:bg-accent/60'
                                        ]"
                                        @click="selectIp(ip)"
                                    >
                                        <span class="flex items-baseline gap-2">
                                            <span class="font-mono">{{ ip }}</span>
                                            <span class="text-xs text-muted-foreground">{{ ipLabel(ip) }}</span>
                                        </span>
                                        <svg
                                            v-if="ip === listenIP"
                                            class="size-3.5 text-foreground/70"
                                            xmlns="http://www.w3.org/2000/svg"
                                            viewBox="0 0 24 24"
                                            fill="none"
                                            stroke="currentColor"
                                            stroke-width="2"
                                            stroke-linecap="round"
                                            stroke-linejoin="round"
                                        >
                                            <path d="M20 6 9 17l-5-5" />
                                        </svg>
                                    </button>
                                </div>
                            </Transition>
                        </div>
                        <span class="font-mono text-sm text-muted-foreground">:</span>
                        <Input
                            v-model="listenPort"
                            inputmode="numeric"
                            maxlength="5"
                            class="h-10 w-24 bg-background/40 font-mono text-sm"
                            placeholder="5030"
                        />
                        <Button
                            variant="ghost"
                            size="sm"
                            class="h-9 px-3 text-sm font-normal text-muted-foreground hover:text-foreground"
                            @click="saveAddr"
                            >保存</Button
                        >
                    </div>
                </div>

                <div class="h-px bg-border/40" />

                <div class="grid gap-x-6 gap-y-4 md:grid-cols-2">
                    <div class="space-y-1.5">
                        <div class="flex items-center justify-between">
                            <div class="text-xs text-muted-foreground">REST API</div>
                            <Button
                                variant="ghost"
                                size="sm"
                                class="h-7 px-2 text-xs font-normal text-muted-foreground hover:text-foreground"
                                @click="copyText(endpointUrl('/api/v1/session'))"
                            >
                                复制
                            </Button>
                        </div>
                        <div class="break-all font-mono text-sm text-foreground/90">
                            {{ endpointUrl('/api/v1/session') }}
                        </div>
                    </div>
                    <div class="space-y-1.5">
                        <div class="flex items-center justify-between">
                            <div class="text-xs text-muted-foreground">MCP 服务</div>
                            <Button
                                variant="ghost"
                                size="sm"
                                class="h-7 px-2 text-xs font-normal text-muted-foreground hover:text-foreground"
                                @click="copyText(endpointUrl('/mcp'))"
                            >
                                复制
                            </Button>
                        </div>
                        <div class="break-all font-mono text-sm text-foreground/90">
                            {{ endpointUrl('/mcp') }}
                        </div>
                    </div>
                </div>
            </div>
        </section>

        <section class="space-y-6">
            <header class="space-y-1">
                <h3 class="font-serif text-xl font-medium tracking-tight text-foreground">API 调试</h3>
                <p class="text-sm text-muted-foreground">填写参数后发送请求预览返回结果。</p>
            </header>

            <div class="space-y-8">
                <article
                    v-for="ep in endpoints"
                    :key="ep.path"
                    class="space-y-4 rounded-xl bg-muted/30 ring-1 ring-border/40 p-5 sm:p-6"
                >
                    <header class="flex flex-wrap items-center justify-between gap-3">
                        <div class="flex flex-wrap items-baseline gap-x-3 gap-y-1">
                            <span
                                :class="[
                                    'rounded px-1.5 py-0.5 font-mono text-xs',
                                    ep.method === 'GET'
                                        ? 'bg-sky-500/10 text-sky-400'
                                        : 'bg-amber-500/10 text-amber-400'
                                ]"
                                >{{ ep.method }}</span
                            >
                            <h4 class="font-serif text-base font-medium tracking-tight text-foreground">
                                {{ ep.name }}
                            </h4>
                            <span class="font-mono text-xs text-muted-foreground">{{ ep.path }}</span>
                        </div>

                        <div class="flex items-center gap-1">
                            <Button
                                size="sm"
                                variant="ghost"
                                class="h-8 px-2.5 text-xs font-normal text-muted-foreground hover:text-foreground"
                                :disabled="copiedId === ep.path"
                                @click="copyCmd(ep)"
                            >
                                {{ copiedId === ep.path ? '已复制' : '复制 cURL' }}
                            </Button>
                            <Button
                                size="sm"
                                class="h-8 px-3 text-xs"
                                :disabled="!state?.httpEnabled || apiLoading[ep.path]"
                                @click="tryApi(ep)"
                            >
                                {{ apiLoading[ep.path] ? '请求中…' : '发送请求' }}
                            </Button>
                        </div>
                    </header>

                    <p class="text-sm text-muted-foreground">{{ ep.desc }}</p>

                    <div class="grid gap-5 xl:grid-cols-2">
                        <div class="space-y-3">
                            <div class="text-xs text-muted-foreground">请求参数</div>
                            <div v-if="ep.params" class="grid grid-cols-1 gap-3 md:grid-cols-2">
                                <div v-for="p in ep.params" :key="p.key" class="space-y-1.5">
                                    <div class="flex items-baseline justify-between gap-2">
                                        <label class="font-mono text-xs text-foreground/85">{{ p.key }}</label>
                                        <span class="text-xs text-muted-foreground">{{ p.desc }}</span>
                                    </div>
                                    <Input
                                        v-model="epParams[ep.path][p.key]"
                                        :placeholder="p.placeholder"
                                        class="h-9 font-mono text-xs"
                                    />
                                </div>
                            </div>
                            <div v-else class="text-sm text-muted-foreground">无需填写参数。</div>
                            <div class="space-y-1 pt-2">
                                <div class="text-xs text-muted-foreground">完整 URL</div>
                                <div class="break-all font-mono text-xs text-foreground/75">{{ fullUrl(ep) }}</div>
                            </div>
                        </div>

                        <div class="space-y-2">
                            <div class="flex items-center justify-between">
                                <div class="text-xs text-muted-foreground">响应结果</div>
                                <div v-if="responses[ep.path]" class="flex items-center gap-1.5 text-xs">
                                    <span
                                        :class="[
                                            'size-1.5 rounded-full',
                                            responses[ep.path].startsWith('FAILED:') ? 'bg-rose-500' : 'bg-emerald-500'
                                        ]"
                                    />
                                    <span class="text-muted-foreground">
                                        {{ responses[ep.path].startsWith('FAILED:') ? '错误' : '200 OK' }}
                                    </span>
                                </div>
                            </div>

                            <div class="relative min-h-[180px] rounded-md bg-background/50 ring-1 ring-border/40">
                                <div
                                    v-if="!responses[ep.path] && !apiLoading[ep.path]"
                                    class="flex h-full min-h-[180px] flex-col items-center justify-center p-6 text-center text-sm text-muted-foreground"
                                >
                                    {{
                                        state?.httpEnabled
                                            ? '填写参数后点击“发送请求”查看返回。'
                                            : '请先启动服务以调试接口。'
                                    }}
                                </div>
                                <div
                                    v-if="apiLoading[ep.path]"
                                    class="flex h-full min-h-[180px] flex-col items-center justify-center gap-3 p-6"
                                >
                                    <div
                                        class="size-5 animate-spin rounded-full border-2 border-primary/60 border-t-transparent"
                                    />
                                    <div class="text-sm text-muted-foreground">请求中…</div>
                                </div>
                                <div
                                    v-if="responses[ep.path]"
                                    class="max-h-[400px] overflow-auto p-4 font-mono text-xs leading-relaxed"
                                >
                                    <pre
                                        :class="[
                                            'whitespace-pre-wrap break-all',
                                            responses[ep.path].startsWith('FAILED:')
                                                ? 'text-rose-400'
                                                : 'text-foreground/90'
                                        ]"
                                        >{{ responses[ep.path] }}</pre
                                    >
                                </div>
                            </div>

                            <div v-if="responses[ep.path]" class="flex justify-end">
                                <Button
                                    variant="ghost"
                                    size="sm"
                                    class="h-8 px-2.5 text-xs font-normal text-muted-foreground hover:text-foreground"
                                    @click="copyText(responses[ep.path])"
                                >
                                    复制 JSON
                                </Button>
                            </div>
                        </div>
                    </div>
                </article>
            </div>
        </section>
    </div>
</template>
