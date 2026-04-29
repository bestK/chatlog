<script setup lang="ts">
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { inject, onBeforeUnmount, onMounted, ref, type Ref } from 'vue';
import { appContextKey } from '../app/context';
import { backend, type KeyProgressEvent } from '../wailsbridge';

const injected = inject(appContextKey);
if (!injected) throw new Error('chatlog not provided');
const app = injected;

const { dataDir, workDir, dataKey, imgKey, run, feedback } = app;

type ActionDialogOptions<T> = {
    loadingRef: Ref<boolean>;
    loadingTitle: string;
    loadingMessage: string;
    successTitle: string;
    successMessage: string;
    failureTitle: string;
    action: () => Promise<T>;
    detail?: (result: T) => string;
};

const loadingDataKey = ref(false);
const loadingImgKey = ref(false);
const loadingDecrypt = ref(false);
let offKeyProgress: (() => void) | undefined;

function isKeyProgressEvent(payload: unknown): payload is KeyProgressEvent {
    if (!payload || typeof payload !== 'object') return false;
    const record = payload as Record<string, unknown>;
    return typeof record.operation === 'string' && typeof record.message === 'string';
}

function maskSecret(secret: string) {
    if (!secret) return '-';
    if (secret.length <= 10) return secret;
    return `${secret.slice(0, 4)}***${secret.slice(-4)}`;
}

function formatError(error: unknown) {
    if (error instanceof Error) return error.message;
    if (typeof error === 'string') return error;
    try {
        return JSON.stringify(error);
    } catch {
        return String(error);
    }
}

async function runWithDialog<T>(options: ActionDialogOptions<T>) {
    if (options.loadingRef.value) return;
    options.loadingRef.value = true;
    feedback.status.openLoading(options.loadingTitle, options.loadingMessage);
    try {
        const result = await options.action();
        await app.refreshAll();
        feedback.status.openResult(
            'success',
            options.successTitle,
            options.successMessage,
            options.detail ? options.detail(result) : ''
        );
    } catch (e) {
        feedback.status.openResult('error', options.failureTitle, formatError(e));
    } finally {
        options.loadingRef.value = false;
    }
}

onMounted(() => {
    offKeyProgress = backend.EventsOn('key:progress', payload => {
        if (!isKeyProgressEvent(payload)) return;
        if (payload.operation !== 'dataKey') return;
        if (!loadingDataKey.value) return;
        feedback.status.update(payload.message);
    });
});

onBeforeUnmount(() => {
    if (offKeyProgress) {
        offKeyProgress();
        offKeyProgress = undefined;
    }
});

function saveDataDir() {
    return app.feedback
        .confirm({
            title: '保存数据目录',
            message: '确认保存并写入配置？',
            confirmText: '保存',
            cancelText: '取消'
        })
        .then(ok => (ok ? run(() => backend.SetDataDir(dataDir.value), '已保存数据目录') : undefined));
}

function saveWorkDir() {
    return app.feedback
        .confirm({
            title: '保存工作目录',
            message: '确认保存并写入配置？',
            confirmText: '保存',
            cancelText: '取消'
        })
        .then(ok => (ok ? run(() => backend.SetWorkDir(workDir.value), '已保存工作目录') : undefined));
}

function saveDataKey() {
    return app.feedback
        .confirm({
            title: '保存数据库密钥',
            message: '确认保存并写入配置？',
            confirmText: '保存',
            cancelText: '取消'
        })
        .then(ok => (ok ? run(() => backend.SetDataKey(dataKey.value), '已保存数据库密钥') : undefined));
}

function autoDataKey() {
    return runWithDialog<string>({
        loadingRef: loadingDataKey,
        loadingTitle: '正在获取数据库密钥',
        loadingMessage: '正在准备获取数据库密钥，请稍候…',
        successTitle: '数据库密钥获取成功',
        successMessage: '已完成读取并同步到当前页面。',
        failureTitle: '数据库密钥获取失败',
        action: () => backend.GetDataKey(),
        detail: key => `密钥预览：${maskSecret(key)}`
    });
}

function saveImgKey() {
    return app.feedback
        .confirm({
            title: '保存图片密钥',
            message: '确认保存并写入配置？',
            confirmText: '保存',
            cancelText: '取消'
        })
        .then(ok => (ok ? run(() => backend.SetImgKey(imgKey.value), '已保存图片密钥') : undefined));
}

function autoImgKey() {
    return runWithDialog<string>({
        loadingRef: loadingImgKey,
        loadingTitle: '正在获取图片密钥',
        loadingMessage: '请保持微信处于运行与登录状态，正在读取当前账号图片密钥。',
        successTitle: '图片密钥获取成功',
        successMessage: '已完成读取并同步到当前页面。',
        failureTitle: '图片密钥获取失败',
        action: () => backend.GetImgKey(),
        detail: key => `密钥预览：${maskSecret(key)}`
    });
}

async function decryptNow() {
    const ok = await app.feedback.confirm({
        title: '开始解密',
        message: '确认开始解密数据库到工作目录？',
        confirmText: '开始',
        cancelText: '取消'
    });
    if (!ok) return;
    return runWithDialog<void>({
        loadingRef: loadingDecrypt,
        loadingTitle: '正在解密数据库',
        loadingMessage: '正在解密并写入工作目录，过程可能持续一段时间，请勿关闭程序。',
        successTitle: '解密完成',
        successMessage: '数据库已成功解密，可前往服务页或日志页继续操作。',
        failureTitle: '解密失败',
        action: () => backend.Decrypt(),
        detail: () => `工作目录：${workDir.value || '-'}`
    });
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
                    全局配置 / GLOBAL SETTINGS
                </div>
            </div>

            <div class="grid gap-6 [grid-template-columns:repeat(auto-fit,minmax(320px,1fr))] xl:gap-8">
                <!-- 目录配置 -->
                <Card class="overflow-hidden border-border/40 bg-card/40 shadow-none">
                    <CardHeader class="border-b border-border/40 bg-muted/5 pb-4">
                        <div class="space-y-1">
                            <CardTitle class="text-[15px] font-bold tracking-tight">目录配置 / Directories</CardTitle>
                            <CardDescription class="text-[11px]"
                                >配置微信数据读取路径与解密后的存储工作区。</CardDescription
                            >
                        </div>
                    </CardHeader>
                    <CardContent class="space-y-6 p-6">
                        <div class="space-y-3">
                            <div class="flex items-center justify-between">
                                <label class="text-[10px] font-bold uppercase tracking-widest text-muted-foreground/60"
                                    >数据目录 / Data Directory</label
                                >
                                <Button
                                    variant="link"
                                    class="h-auto p-0 text-[10px] font-bold uppercase text-primary"
                                    @click="saveDataDir"
                                    >Save</Button
                                >
                            </div>
                            <Input
                                v-model="dataDir"
                                class="h-9 bg-background/30 font-mono text-[11px]"
                                placeholder="微信数据存放路径"
                            />
                        </div>

                        <div class="space-y-3">
                            <div class="flex items-center justify-between">
                                <label class="text-[10px] font-bold uppercase tracking-widest text-muted-foreground/60"
                                    >工作目录 / Work Directory</label
                                >
                                <Button
                                    variant="link"
                                    class="h-auto p-0 text-[10px] font-bold uppercase text-primary"
                                    @click="saveWorkDir"
                                    >Save</Button
                                >
                            </div>
                            <Input
                                v-model="workDir"
                                class="h-9 bg-background/30 font-mono text-[11px]"
                                placeholder="解密输出路径"
                            />
                        </div>
                    </CardContent>
                </Card>

                <!-- 密钥配置 -->
                <Card class="overflow-hidden border-border/40 bg-card/40 shadow-none">
                    <CardHeader class="border-b border-border/40 bg-muted/5 pb-4">
                        <div class="space-y-1">
                            <CardTitle class="text-[15px] font-bold tracking-tight">安全密钥 / Security Keys</CardTitle>
                            <CardDescription class="text-[11px]"
                                >解密所需的关键凭据，可从微信进程中自动探测获取。</CardDescription
                            >
                        </div>
                    </CardHeader>
                    <CardContent class="space-y-6 p-6">
                        <div class="space-y-3">
                            <div class="flex items-center justify-between">
                                <label class="text-[10px] font-bold uppercase tracking-widest text-muted-foreground/60"
                                    >数据库密钥 / Database Key</label
                                >
                                <div class="flex gap-2">
                                    <Button
                                        variant="outline"
                                        class="h-6 px-2 text-[9px] font-bold uppercase tracking-widest"
                                        :disabled="loadingDataKey || loadingImgKey || loadingDecrypt"
                                        @click="autoDataKey"
                                    >
                                        {{ loadingDataKey ? 'Fetching...' : 'Auto Fetch' }}
                                    </Button>
                                    <Button
                                        variant="link"
                                        class="h-auto p-0 text-[10px] font-bold uppercase text-primary"
                                        @click="saveDataKey"
                                        >Save</Button
                                    >
                                </div>
                            </div>
                            <Input
                                v-model="dataKey"
                                class="h-9 bg-background/30 font-mono text-[11px]"
                                placeholder="请输入 64 位 Hex 密钥"
                            />
                        </div>

                        <div class="space-y-3">
                            <div class="flex items-center justify-between">
                                <label class="text-[10px] font-bold uppercase tracking-widest text-muted-foreground/60"
                                    >图片密钥 / Image Key</label
                                >
                                <div class="flex gap-2">
                                    <Button
                                        variant="outline"
                                        class="h-6 px-2 text-[9px] font-bold uppercase tracking-widest"
                                        :disabled="loadingDataKey || loadingImgKey || loadingDecrypt"
                                        @click="autoImgKey"
                                    >
                                        {{ loadingImgKey ? 'Fetching...' : 'Auto Fetch' }}
                                    </Button>
                                    <Button
                                        variant="link"
                                        class="h-auto p-0 text-[10px] font-bold uppercase text-primary"
                                        @click="saveImgKey"
                                        >Save</Button
                                    >
                                </div>
                            </div>
                            <Input
                                v-model="imgKey"
                                class="h-9 bg-background/30 font-mono text-[11px]"
                                placeholder="请输入图片解密密钥"
                            />
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
                    数据维护 / MAINTENANCE
                </div>
            </div>

            <Card class="overflow-hidden border-border/40 bg-card/40 shadow-none">
                <CardHeader class="border-b border-border/40 bg-muted/5 pb-6">
                    <div class="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
                        <div class="space-y-1.5">
                            <CardTitle class="text-[15px] font-bold tracking-tight">数据库解密</CardTitle>
                            <CardDescription class="text-xs max-w-lg"
                                >开始前请确保目录与密钥均已保存。此操作会占用较多 CPU 与 IO
                                资源，请耐心等待。</CardDescription
                            >
                        </div>

                        <div class="flex flex-wrap items-center gap-3">
                            <Badge
                                variant="outline"
                                class="h-6 gap-2 border-amber-500/20 bg-amber-500/10 text-[10px] font-bold uppercase tracking-widest text-amber-500"
                            >
                                <span class="inline-block size-1.5 rounded-full bg-amber-500"></span>
                                High Resource Usage
                            </Badge>

                            <Button
                                :disabled="loadingDataKey || loadingImgKey || loadingDecrypt"
                                class="h-10 px-5 text-sm font-semibold shadow-lg shadow-primary/20 transition-all"
                                @click="decryptNow"
                            >
                                {{ loadingDecrypt ? '正在解密中…' : '开始解密' }}
                            </Button>
                        </div>
                    </div>
                </CardHeader>
                <CardContent class="bg-muted/5 p-4 py-3 flex items-center justify-between">
                    <div class="flex items-center gap-2 text-[10px] text-muted-foreground/60 italic">
                        <svg
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
                            <circle cx="12" cy="12" r="10" />
                            <path d="M12 16v-4" />
                            <path d="M12 8h.01" />
                        </svg>
                        解密后的数据将存放在 "{{ workDir || '未定义' }}" 目录中。
                    </div>
                </CardContent>
            </Card>
        </section>
    </div>
</template>
