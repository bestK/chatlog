<script setup lang="ts">
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { inject } from 'vue';
import { appContextKey } from '../app/context';
import { maskKey } from '../app/state';
import { backend } from '../wailsbridge';

const injected = inject(appContextKey);
if (!injected) throw new Error('chatlog not provided');
const app = injected;

const { dataKey, imgKey, dataDir, workDir, state, run } = app;

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
        return run(() => backend.StopHTTP(), '已停止 HTTP 服务');
    }
    return run(() => backend.StartHTTP(), '已启动 HTTP 服务');
}
</script>

<template>
    <div class="space-y-10">
        <section class="space-y-6">
            <div class="flex items-center gap-4 border-b border-border/40 pb-4">
                <div
                    class="flex size-8 items-center justify-center rounded-lg bg-primary/10 text-[10px] font-bold text-primary"
                >
                    01
                </div>
                <div class="text-xs font-bold uppercase tracking-[0.2em] text-foreground/70">
                    核心信息 / KEY INFORMATION
                </div>
            </div>

            <div class="grid gap-4 [grid-template-columns:repeat(auto-fit,minmax(220px,1fr))] md:gap-6">
                <Card class="overflow-hidden border-border/40 bg-card/40 shadow-none">
                    <CardContent class="p-6">
                        <div class="flex flex-col gap-3">
                            <div class="text-[10px] font-bold uppercase tracking-widest text-muted-foreground/60">
                                Data Key
                            </div>
                            <div class="font-mono text-sm font-medium text-foreground">
                                {{ maskKey(dataKey) || '未设置' }}
                            </div>
                        </div>
                    </CardContent>
                </Card>
                <Card class="overflow-hidden border-border/40 bg-card/40 shadow-none">
                    <CardContent class="p-6">
                        <div class="flex flex-col gap-3">
                            <div class="text-[10px] font-bold uppercase tracking-widest text-muted-foreground/60">
                                Img Key
                            </div>
                            <div class="font-mono text-sm font-medium text-foreground">
                                {{ maskKey(imgKey) || '未设置' }}
                            </div>
                        </div>
                    </CardContent>
                </Card>
                <Card class="overflow-hidden border-border/40 bg-card/40 shadow-none">
                    <CardContent class="p-6">
                        <div class="flex flex-col gap-3">
                            <div class="text-[10px] font-bold uppercase tracking-widest text-muted-foreground/60">
                                Process PID
                            </div>
                            <div class="flex items-center gap-2">
                                <span class="text-lg font-bold tracking-tight text-foreground">{{
                                    state?.pid || '-'
                                }}</span>
                                <Badge
                                    v-if="state?.pid"
                                    variant="outline"
                                    class="h-5 bg-emerald-500/10 text-[9px] font-bold text-emerald-500 border-emerald-500/20"
                                    >Running</Badge
                                >
                            </div>
                        </div>
                    </CardContent>
                </Card>
            </div>
        </section>

        <section class="space-y-6">
            <div class="flex items-center gap-4 border-b border-border/40 pb-4">
                <div
                    class="flex size-8 items-center justify-center rounded-lg bg-primary/10 text-[10px] font-bold text-primary"
                >
                    02
                </div>
                <div class="text-xs font-bold uppercase tracking-[0.2em] text-foreground/70">
                    目录路径 / DIRECTORY PATHS
                </div>
            </div>

            <Card class="overflow-hidden border-border/40 bg-card/40 shadow-none">
                <CardContent class="divide-y divide-border/40 p-0">
                    <div class="grid gap-4 p-6 md:grid-cols-[minmax(120px,200px)_minmax(0,1fr)]">
                        <div class="text-[10px] font-bold uppercase tracking-widest text-muted-foreground/60 pt-1">
                            Data Directory
                        </div>
                        <div
                            class="rounded-lg border border-border/40 bg-muted/20 px-3 py-2 font-mono text-[11px] text-foreground/70 break-all leading-relaxed"
                        >
                            {{ dataDir || '未配置' }}
                        </div>
                    </div>
                    <div class="grid gap-4 p-6 md:grid-cols-[minmax(120px,200px)_minmax(0,1fr)]">
                        <div class="text-[10px] font-bold uppercase tracking-widest text-muted-foreground/60 pt-1">
                            Work Directory
                        </div>
                        <div
                            class="rounded-lg border border-border/40 bg-muted/20 px-3 py-2 font-mono text-[11px] text-foreground/70 break-all leading-relaxed"
                        >
                            {{ workDir || '未配置' }}
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
                    03
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
                            <CardDescription class="text-xs"
                                >快速切换 HTTP 服务状态并查看当前可用接口地址。</CardDescription
                            >
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

                            <div class="flex items-center gap-2">
                                <Button
                                    variant="outline"
                                    size="sm"
                                    class="h-9 px-4 text-[10px] font-bold uppercase tracking-widest"
                                    @click="run(() => backend.Refresh(), '已刷新成功')"
                                >
                                    刷新同步
                                </Button>
                                <Button
                                    :variant="state?.httpEnabled ? 'destructive' : 'default'"
                                    class="h-9 px-5 text-[10px] font-bold uppercase tracking-widest shadow-md transition-all"
                                    @click="toggleHTTP"
                                >
                                    {{ state?.httpEnabled ? '停止服务' : '启动服务' }}
                                </Button>
                            </div>
                        </div>
                    </div>
                </CardHeader>

                <CardContent class="p-0">
                    <div v-if="state?.httpAddr" class="grid divide-border/40 md:grid-cols-2 md:divide-x">
                        <div class="group relative flex flex-col gap-3 p-6 transition-colors hover:bg-muted/10">
                            <div class="flex items-center justify-between">
                                <div class="text-[10px] font-bold uppercase tracking-widest text-muted-foreground/60">
                                    API Endpoint
                                </div>
                                <Badge variant="outline" class="h-5 bg-background/50 font-mono text-[9px] uppercase"
                                    >Session</Badge
                                >
                            </div>
                            <div
                                class="flex-1 rounded-lg border border-border/40 bg-muted/20 px-3 py-2 font-mono text-[11px] text-primary/80 break-all"
                            >
                                http://{{ state.httpAddr }}/api/v1/session
                            </div>
                        </div>
                        <div class="group relative flex flex-col gap-3 p-6 transition-colors hover:bg-muted/10">
                            <div class="flex items-center justify-between">
                                <div class="text-[10px] font-bold uppercase tracking-widest text-muted-foreground/60">
                                    MCP Endpoint
                                </div>
                                <Badge variant="outline" class="h-5 bg-background/50 font-mono text-[9px] uppercase"
                                    >MCP</Badge
                                >
                            </div>
                            <div
                                class="flex-1 rounded-lg border border-border/40 bg-muted/20 px-3 py-2 font-mono text-[11px] text-primary/80 break-all"
                            >
                                http://{{ state.httpAddr }}/mcp
                            </div>
                        </div>
                    </div>
                    <div v-else class="flex flex-col items-center justify-center py-12 px-6 text-center opacity-40">
                        <div class="text-[11px] font-bold uppercase tracking-widest">HTTP 服务未运行</div>
                        <div class="mt-1 text-[10px]">启动服务后可在此处查看 API 端点地址</div>
                    </div>
                </CardContent>
            </Card>
        </section>
    </div>
</template>
