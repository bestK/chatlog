<script setup lang="ts">
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { inject, reactive, ref } from 'vue';
import { appContextKey } from '../app/context';
import { backend } from '../wailsbridge';

const injected = inject(appContextKey);
if (!injected) throw new Error('chatlog not provided');
const app = injected;

const { httpAddr, state, run } = app;

function saveAddr() {
    return app.feedback
        .confirm({
            title: '保存 HTTP 地址',
            message: '确认保存并写入配置？',
            confirmText: '保存',
            cancelText: '取消'
        })
        .then(ok => (ok ? run(() => backend.SetHTTPAddr(httpAddr.value), '已保存') : undefined));
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
    <div class="space-y-8">
        <section class="space-y-6">
            <div class="flex items-center gap-4 border-b border-border/40 pb-4">
                <div
                    class="flex size-8 items-center justify-center rounded-lg bg-primary/10 text-[10px] font-bold text-primary"
                >
                    01
                </div>
                <div class="text-xs font-bold uppercase tracking-[0.2em] text-foreground/70">
                    服务控制台 / SERVICE CONSOLE
                </div>
            </div>

            <Card class="overflow-hidden border-border/40 bg-card/40 shadow-none">
                <CardHeader class="border-b border-border/40 bg-muted/5 pb-6">
                    <div class="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
                        <div class="space-y-1.5">
                            <CardTitle class="text-lg font-bold tracking-tight">服务管理</CardTitle>
                            <CardDescription class="text-xs">配置并启动 HTTP API 与 MCP 服务器进程。</CardDescription>
                        </div>

                        <div class="flex items-center gap-3">
                            <div
                                class="flex items-center gap-2 rounded-full border border-border/40 bg-background/50 px-3 py-1.5"
                            >
                                <div
                                    :class="[
                                        'size-2 rounded-full transition-all',
                                        state?.httpEnabled
                                            ? 'bg-emerald-500 shadow-[0_0_8px_rgba(16,185,129,0.5)] animate-pulse'
                                            : 'bg-muted-foreground'
                                    ]"
                                />
                                <span class="font-mono text-[10px] font-bold uppercase tracking-widest">
                                    {{ state?.httpEnabled ? 'Online' : 'Offline' }}
                                </span>
                            </div>

                            <Button
                                :variant="state?.httpEnabled ? 'destructive' : 'default'"
                                class="h-9 px-5 text-xs font-bold uppercase tracking-tight shadow-md transition-all"
                                @click="toggleHTTP"
                            >
                                {{ state?.httpEnabled ? '停止服务' : '启动服务' }}
                            </Button>
                        </div>
                    </div>
                </CardHeader>

                <CardContent class="divide-y divide-border/40 p-0">
                    <div class="grid gap-4 p-6 md:gap-6 lg:grid-cols-[minmax(0,1fr)_minmax(0,2fr)]">
                        <div class="space-y-2">
                            <div class="text-[10px] font-bold uppercase tracking-widest text-muted-foreground/60">
                                监听地址 / Listen Address
                            </div>
                            <div class="text-[11px] leading-relaxed text-muted-foreground">
                                指定服务运行的本地端口与地址，保存后需重启服务才能生效。
                            </div>
                        </div>

                        <div class="flex flex-wrap items-center gap-2">
                            <Input
                                v-model="httpAddr"
                                class="h-10 min-w-[180px] flex-1 font-mono text-sm"
                                placeholder="127.0.0.1:5030"
                            />
                            <Button
                                variant="outline"
                                class="h-10 px-4 font-bold uppercase tracking-tight"
                                @click="saveAddr"
                                >Save</Button
                            >
                        </div>
                    </div>

                    <div class="grid gap-0 md:grid-cols-2">
                        <div class="group relative flex flex-col gap-3 p-6 transition-colors hover:bg-muted/10">
                            <div class="flex items-center justify-between">
                                <div class="text-[10px] font-bold uppercase tracking-widest text-muted-foreground/60">
                                    REST API
                                </div>
                                <Badge variant="outline" class="h-5 bg-background/50 font-mono text-[9px] uppercase"
                                    >v1 (Session)</Badge
                                >
                            </div>

                            <div class="flex items-center gap-3">
                                <div
                                    class="flex-1 overflow-hidden truncate rounded-lg border border-border/40 bg-muted/20 px-3 py-2 font-mono text-[11px] text-foreground/70"
                                >
                                    {{ endpointUrl('/api/v1/session') }}
                                </div>
                                <Button
                                    variant="ghost"
                                    size="icon"
                                    class="size-8 opacity-0 transition-opacity group-hover:opacity-100"
                                    @click="copyText(endpointUrl('/api/v1/session'))"
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
                                        <rect width="14" height="14" x="8" y="8" rx="2" ry="2" />
                                        <path d="M4 16c-1.1 0-2-.9-2-2V4c0-1.1.9-2 2-2h10c1.1 0 2 .9 2 2" />
                                    </svg>
                                </Button>
                            </div>
                        </div>

                        <div
                            class="group relative flex flex-col gap-3 border-l border-border/40 p-6 transition-colors hover:bg-muted/10"
                        >
                            <div class="flex items-center justify-between">
                                <div class="text-[10px] font-bold uppercase tracking-widest text-muted-foreground/60">
                                    MCP Service
                                </div>
                                <Badge variant="outline" class="h-5 bg-background/50 font-mono text-[9px] uppercase"
                                    >Streamable</Badge
                                >
                            </div>

                            <div class="flex items-center gap-3">
                                <div
                                    class="flex-1 overflow-hidden truncate rounded-lg border border-border/40 bg-muted/20 px-3 py-2 font-mono text-[11px] text-foreground/70"
                                >
                                    {{ endpointUrl('/mcp') }}
                                </div>
                                <Button
                                    variant="ghost"
                                    size="icon"
                                    class="size-8 opacity-0 transition-opacity group-hover:opacity-100"
                                    @click="copyText(endpointUrl('/mcp'))"
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
                                        <rect width="14" height="14" x="8" y="8" rx="2" ry="2" />
                                        <path d="M4 16c-1.1 0-2-.9-2-2V4c0-1.1.9-2 2-2h10c1.1 0 2 .9 2 2" />
                                    </svg>
                                </Button>
                            </div>
                        </div>
                    </div>
                </CardContent>
            </Card>
        </section>

        <section class="space-y-6">
            <div class="flex items-center gap-4 border-b border-border/40 pb-4">
                <div
                    class="flex size-8 items-center justify-center rounded-lg bg-primary/10 text-[10px] font-bold text-primary"
                >
                    02
                </div>
                <div class="text-xs font-bold uppercase tracking-[0.2em] text-foreground/70">
                    API 调试区 / PLAYGROUND
                </div>
            </div>

            <div class="space-y-6">
                <Card
                    v-for="ep in endpoints"
                    :key="ep.path"
                    class="overflow-hidden border-border/40 bg-card/40 shadow-none"
                >
                    <CardHeader class="border-b border-border/40 bg-muted/5 p-5">
                        <div class="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
                            <div class="flex items-center gap-3">
                                <Badge
                                    :class="[
                                        'h-6 px-2 font-mono text-[10px] font-bold uppercase tracking-wider',
                                        ep.method === 'GET'
                                            ? 'border-sky-500/20 bg-sky-500/10 text-sky-500'
                                            : 'border-amber-500/20 bg-amber-500/10 text-amber-500'
                                    ]"
                                >
                                    {{ ep.method }}
                                </Badge>
                                <CardTitle class="text-[15px] font-bold tracking-tight">{{ ep.name }}</CardTitle>
                                <span class="text-xs text-muted-foreground opacity-60">{{ ep.path }}</span>
                            </div>

                            <div class="flex items-center gap-2">
                                <Button
                                    size="sm"
                                    variant="outline"
                                    class="h-8 gap-2 px-3 text-[10px] font-bold uppercase tracking-widest"
                                    :disabled="copiedId === ep.path"
                                    @click="copyCmd(ep)"
                                >
                                    <svg
                                        v-if="copiedId !== ep.path"
                                        xmlns="http://www.w3.org/2000/svg"
                                        width="12"
                                        height="12"
                                        viewBox="0 0 24 24"
                                        fill="none"
                                        stroke="currentColor"
                                        stroke-width="2"
                                        stroke-linecap="round"
                                        stroke-linejoin="round"
                                    >
                                        <rect width="14" height="14" x="8" y="8" rx="2" ry="2" />
                                        <path d="M4 16c-1.1 0-2-.9-2-2V4c0-1.1.9-2 2-2h10c1.1 0 2 .9 2 2" />
                                    </svg>
                                    <svg
                                        v-else
                                        xmlns="http://www.w3.org/2000/svg"
                                        width="12"
                                        height="12"
                                        viewBox="0 0 24 24"
                                        fill="none"
                                        stroke="currentColor"
                                        stroke-width="2"
                                        stroke-linecap="round"
                                        stroke-linejoin="round"
                                    >
                                        <path d="M20 6 9 17l-5-5" />
                                    </svg>
                                    {{ copiedId === ep.path ? '已复制' : 'CURL' }}
                                </Button>

                                <Button
                                    size="sm"
                                    class="h-8 gap-2 px-4 text-[10px] font-bold uppercase tracking-widest shadow-lg shadow-primary/10"
                                    :disabled="!state?.httpEnabled || apiLoading[ep.path]"
                                    @click="tryApi(ep)"
                                >
                                    <svg
                                        v-if="!apiLoading[ep.path]"
                                        xmlns="http://www.w3.org/2000/svg"
                                        width="12"
                                        height="12"
                                        viewBox="0 0 24 24"
                                        fill="none"
                                        stroke="currentColor"
                                        stroke-width="2"
                                        stroke-linecap="round"
                                        stroke-linejoin="round"
                                    >
                                        <path d="m5 3 14 9-14 9V3z" />
                                    </svg>
                                    <svg
                                        v-else
                                        class="animate-spin"
                                        xmlns="http://www.w3.org/2000/svg"
                                        width="12"
                                        height="12"
                                        viewBox="0 0 24 24"
                                        fill="none"
                                        stroke="currentColor"
                                        stroke-width="2"
                                        stroke-linecap="round"
                                        stroke-linejoin="round"
                                    >
                                        <path d="M21 12a9 9 0 1 1-6.219-8.56" />
                                    </svg>
                                    {{ apiLoading[ep.path] ? '请求中...' : '发送请求' }}
                                </Button>
                            </div>
                        </div>

                        <div class="mt-3 text-[11px] text-muted-foreground">{{ ep.desc }}</div>
                    </CardHeader>

                    <CardContent class="grid gap-0 p-0 xl:grid-cols-2">
                        <div class="flex flex-col border-b border-border/40 xl:border-r xl:border-b-0">
                            <div class="border-b border-border/40 bg-muted/10 px-5 py-2">
                                <div class="text-[9px] font-bold uppercase tracking-[0.2em] text-muted-foreground/80">
                                    请求参数 / Parameters
                                </div>
                            </div>

                            <div class="flex-1 space-y-5 p-5">
                                <div v-if="ep.params" class="grid grid-cols-1 gap-x-4 gap-y-5 md:grid-cols-2">
                                    <div v-for="p in ep.params" :key="p.key" class="space-y-2">
                                        <div class="flex items-center justify-between gap-3">
                                            <label
                                                class="text-[10px] font-bold uppercase tracking-widest text-foreground/70"
                                                >{{ p.key }}</label
                                            >
                                            <span class="text-[9px] text-muted-foreground opacity-60">{{
                                                p.desc
                                            }}</span>
                                        </div>
                                        <Input
                                            v-model="epParams[ep.path][p.key]"
                                            :placeholder="p.placeholder"
                                            class="h-9 bg-background/30 font-mono text-[11px]"
                                        />
                                    </div>
                                </div>

                                <div
                                    v-else
                                    class="flex min-h-[100px] items-center justify-center rounded-xl border border-dashed border-border/40 bg-muted/5"
                                >
                                    <span class="text-[11px] uppercase tracking-widest text-muted-foreground/60"
                                        >无必填参数 (No Parameters)</span
                                    >
                                </div>
                            </div>

                            <div class="mt-auto border-t border-border/40 bg-muted/5 px-5 py-3">
                                <div class="flex flex-col gap-1.5">
                                    <div
                                        class="text-[9px] font-bold uppercase tracking-widest text-muted-foreground/60"
                                    >
                                        Full URL
                                    </div>
                                    <div class="break-all font-mono text-[10px] text-muted-foreground/80">
                                        {{ fullUrl(ep) }}
                                    </div>
                                </div>
                            </div>
                        </div>

                        <div class="flex flex-col bg-muted/5">
                            <div
                                class="flex items-center justify-between border-b border-border/40 bg-muted/10 px-5 py-2"
                            >
                                <div class="text-[9px] font-bold uppercase tracking-[0.2em] text-muted-foreground/80">
                                    响应结果 / Response
                                </div>
                                <div v-if="responses[ep.path]" class="flex items-center gap-2">
                                    <div
                                        :class="[
                                            'size-1.5 rounded-full',
                                            responses[ep.path].startsWith('FAILED:') ? 'bg-rose-500' : 'bg-emerald-500'
                                        ]"
                                    />
                                    <span class="text-[9px] font-bold uppercase tracking-widest opacity-60">
                                        Status: {{ responses[ep.path].startsWith('FAILED:') ? 'Error' : '200 OK' }}
                                    </span>
                                </div>
                            </div>

                            <div class="relative min-h-[180px] flex-1">
                                <div
                                    v-if="!responses[ep.path] && !apiLoading[ep.path]"
                                    class="absolute inset-0 flex flex-col items-center justify-center p-6 text-center"
                                >
                                    <div
                                        class="mb-3 flex size-10 items-center justify-center rounded-full border border-border/40 bg-background/50"
                                    >
                                        <svg
                                            xmlns="http://www.w3.org/2000/svg"
                                            width="16"
                                            height="16"
                                            viewBox="0 0 24 24"
                                            fill="none"
                                            stroke="currentColor"
                                            stroke-width="2"
                                            stroke-linecap="round"
                                            stroke-linejoin="round"
                                            class="opacity-20"
                                        >
                                            <path d="M12 2v10" />
                                            <path d="m16 8-4 4-4-4" />
                                            <path d="M22 12A10 10 0 1 1 12 2" />
                                            <path
                                                d="M22 12c-1.1 0-2 .9-2 2v4c0 1.1-.9 2-2 2H6c-1.1 0-2-.9-2-2v-4c0-1.1.9-2-2-2"
                                            />
                                        </svg>
                                    </div>
                                    <div
                                        class="mb-1 text-[11px] font-bold uppercase tracking-widest text-muted-foreground/60"
                                    >
                                        等待请求 (Awaiting Request)
                                    </div>
                                    <div class="text-[10px] text-muted-foreground/40">
                                        {{
                                            state?.httpEnabled
                                                ? '填写参数并点击“发送请求”来预览 API 结果'
                                                : '请先启动服务以进行交互调试'
                                        }}
                                    </div>
                                </div>

                                <div
                                    v-if="apiLoading[ep.path]"
                                    class="absolute inset-0 flex flex-col items-center justify-center bg-background/20 p-6 backdrop-blur-[1px]"
                                >
                                    <div
                                        class="mb-3 size-6 animate-spin rounded-full border-2 border-primary border-t-transparent opacity-60"
                                    />
                                    <div class="text-[11px] font-bold uppercase tracking-widest text-primary/60">
                                        请求发送中 (Syncing...)
                                    </div>
                                </div>

                                <div
                                    v-if="responses[ep.path]"
                                    class="h-full max-h-[400px] overflow-auto p-5 font-mono text-[11px] leading-relaxed"
                                >
                                    <pre
                                        :class="[
                                            'whitespace-pre-wrap break-all transition-colors',
                                            responses[ep.path].startsWith('FAILED:')
                                                ? 'text-rose-400'
                                                : 'text-foreground'
                                        ]"
                                        >{{ responses[ep.path] }}</pre
                                    >
                                </div>
                            </div>

                            <div v-if="responses[ep.path]" class="flex justify-end border-t border-border/40 px-5 py-2">
                                <Button
                                    variant="ghost"
                                    class="h-6 gap-2 text-[9px] font-bold uppercase text-muted-foreground hover:text-foreground"
                                    @click="copyText(responses[ep.path])"
                                >
                                    Copy JSON
                                </Button>
                            </div>
                        </div>
                    </CardContent>
                </Card>
            </div>
        </section>
    </div>
</template>
