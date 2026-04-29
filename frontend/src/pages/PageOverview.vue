<script setup lang="ts">
import { Button } from '@/components/ui/button';
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
    <div class="space-y-12">
        <section class="space-y-5">
            <header class="space-y-1">
                <h3 class="font-serif text-xl font-medium tracking-tight text-foreground">核心信息</h3>
                <p class="text-sm text-muted-foreground">当前运行时的关键密钥与进程信息。</p>
            </header>

            <div class="grid gap-4 [grid-template-columns:repeat(auto-fit,minmax(220px,1fr))]">
                <div class="space-y-2 rounded-xl bg-muted/30 ring-1 ring-border/40 p-5">
                    <div class="text-xs text-muted-foreground">数据库密钥</div>
                    <div class="font-mono text-sm text-foreground">{{ maskKey(dataKey) || '未设置' }}</div>
                </div>
                <div class="space-y-2 rounded-xl bg-muted/30 ring-1 ring-border/40 p-5">
                    <div class="text-xs text-muted-foreground">图片密钥</div>
                    <div class="font-mono text-sm text-foreground">{{ maskKey(imgKey) || '未设置' }}</div>
                </div>
                <div class="space-y-2 rounded-xl bg-muted/30 ring-1 ring-border/40 p-5">
                    <div class="text-xs text-muted-foreground">进程 PID</div>
                    <div class="flex items-baseline gap-2">
                        <span class="font-serif text-2xl font-medium tracking-tight text-foreground">{{
                            state?.pid || '—'
                        }}</span>
                        <span v-if="state?.pid" class="inline-flex items-center gap-1.5 text-xs text-emerald-500">
                            <span class="size-1.5 rounded-full bg-emerald-500" />运行中
                        </span>
                    </div>
                </div>
            </div>
        </section>

        <section class="space-y-5">
            <header class="space-y-1">
                <h3 class="font-serif text-xl font-medium tracking-tight text-foreground">目录路径</h3>
                <p class="text-sm text-muted-foreground">数据读取与解密输出的本地位置。</p>
            </header>

            <div class="rounded-xl bg-muted/30 ring-1 ring-border/40 p-5 sm:p-6 space-y-4">
                <div class="grid gap-2 md:grid-cols-[minmax(120px,180px)_minmax(0,1fr)] md:items-baseline">
                    <div class="text-sm text-muted-foreground">数据目录</div>
                    <div class="break-all font-mono text-sm text-foreground/90 leading-relaxed">
                        {{ dataDir || '未配置' }}
                    </div>
                </div>
                <div class="h-px bg-border/40" />
                <div class="grid gap-2 md:grid-cols-[minmax(120px,180px)_minmax(0,1fr)] md:items-baseline">
                    <div class="text-sm text-muted-foreground">工作目录</div>
                    <div class="break-all font-mono text-sm text-foreground/90 leading-relaxed">
                        {{ workDir || '未配置' }}
                    </div>
                </div>
            </div>
        </section>

        <section class="space-y-5">
            <header class="flex flex-wrap items-end justify-between gap-3">
                <div class="space-y-1">
                    <h3 class="font-serif text-xl font-medium tracking-tight text-foreground">服务状态</h3>
                    <p class="text-sm text-muted-foreground">HTTP 服务与 MCP 接口的快捷控制。</p>
                </div>
                <div class="flex items-center gap-2">
                    <Button
                        variant="ghost"
                        size="sm"
                        class="h-9 px-3 text-sm font-normal text-muted-foreground hover:text-foreground"
                        @click="run(() => backend.Refresh(), '已刷新')"
                    >
                        刷新
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
            </header>

            <div class="rounded-xl bg-muted/30 ring-1 ring-border/40 p-5 sm:p-6 space-y-4">
                <div class="flex items-center gap-2 text-sm">
                    <span
                        :class="[
                            'size-1.5 rounded-full',
                            state?.httpEnabled ? 'bg-emerald-500' : 'bg-muted-foreground/40'
                        ]"
                    />
                    <span :class="state?.httpEnabled ? 'text-foreground' : 'text-muted-foreground'">
                        {{ state?.httpEnabled ? '服务运行中' : '服务未启动' }}
                    </span>
                </div>

                <div v-if="state?.httpAddr" class="grid gap-4 md:grid-cols-2">
                    <div class="space-y-1.5">
                        <div class="text-xs text-muted-foreground">API 端点</div>
                        <div class="break-all font-mono text-sm text-foreground/90">
                            http://{{ state.httpAddr }}/api/v1/session
                        </div>
                    </div>
                    <div class="space-y-1.5">
                        <div class="text-xs text-muted-foreground">MCP 端点</div>
                        <div class="break-all font-mono text-sm text-foreground/90">
                            http://{{ state.httpAddr }}/mcp
                        </div>
                    </div>
                </div>
                <div v-else class="text-sm text-muted-foreground">启动服务后，这里会呈现可调用的 API 与 MCP 地址。</div>
            </div>
        </section>
    </div>
</template>
