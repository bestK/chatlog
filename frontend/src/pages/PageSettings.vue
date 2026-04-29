<script setup lang="ts">
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
        <div
            class="sticky -top-px z-30 flex flex-wrap items-center justify-between gap-3 rounded-xl bg-muted/30 ring-1 ring-border/40 px-4 py-3 backdrop-blur supports-[backdrop-filter]:bg-muted/40"
        >
            <p class="text-sm text-muted-foreground">配置目录与密钥后，可进行数据库解密。</p>
            <Button
                :disabled="loadingDataKey || loadingImgKey || loadingDecrypt"
                size="sm"
                class="h-9 px-4 text-sm"
                @click="decryptNow"
            >
                {{ loadingDecrypt ? '解密中…' : '开始解密' }}
            </Button>
        </div>

        <section class="space-y-4">
            <header class="space-y-1">
                <h3 class="font-serif text-xl font-medium tracking-tight text-foreground">目录与密钥</h3>
                <p class="text-sm text-muted-foreground">配置数据路径与解密凭据，修改后需保存生效。</p>
            </header>

            <div class="grid gap-x-8 gap-y-6 [grid-template-columns:repeat(auto-fit,minmax(320px,1fr))]">
                <div class="space-y-2">
                    <div class="flex items-center justify-between">
                        <label class="text-sm text-foreground">数据目录</label>
                        <Button
                            variant="ghost"
                            size="sm"
                            class="h-7 px-2 text-xs font-normal text-muted-foreground hover:text-foreground"
                            @click="saveDataDir"
                            >保存</Button
                        >
                    </div>
                    <Input v-model="dataDir" class="h-9 font-mono text-xs" placeholder="微信数据存放路径" />
                </div>

                <div class="space-y-2">
                    <div class="flex items-center justify-between">
                        <label class="text-sm text-foreground">工作目录</label>
                        <Button
                            variant="ghost"
                            size="sm"
                            class="h-7 px-2 text-xs font-normal text-muted-foreground hover:text-foreground"
                            @click="saveWorkDir"
                            >保存</Button
                        >
                    </div>
                    <Input v-model="workDir" class="h-9 font-mono text-xs" placeholder="解密输出路径" />
                </div>

                <div class="space-y-2">
                    <div class="flex items-center justify-between gap-2">
                        <label class="text-sm text-foreground">数据库密钥</label>
                        <div class="flex items-center gap-1">
                            <Button
                                variant="ghost"
                                size="sm"
                                class="h-7 px-2 text-xs font-normal text-muted-foreground hover:text-foreground"
                                :disabled="loadingDataKey || loadingImgKey || loadingDecrypt"
                                @click="autoDataKey"
                            >
                                {{ loadingDataKey ? '获取中…' : '自动获取' }}
                            </Button>
                            <Button
                                variant="ghost"
                                size="sm"
                                class="h-7 px-2 text-xs font-normal text-muted-foreground hover:text-foreground"
                                @click="saveDataKey"
                                >保存</Button
                            >
                        </div>
                    </div>
                    <Input v-model="dataKey" class="h-9 font-mono text-xs" placeholder="请输入 64 位 Hex 密钥" />
                </div>

                <div class="space-y-2">
                    <div class="flex items-center justify-between gap-2">
                        <label class="text-sm text-foreground">图片密钥</label>
                        <div class="flex items-center gap-1">
                            <Button
                                variant="ghost"
                                size="sm"
                                class="h-7 px-2 text-xs font-normal text-muted-foreground hover:text-foreground"
                                :disabled="loadingDataKey || loadingImgKey || loadingDecrypt"
                                @click="autoImgKey"
                            >
                                {{ loadingImgKey ? '获取中…' : '自动获取' }}
                            </Button>
                            <Button
                                variant="ghost"
                                size="sm"
                                class="h-7 px-2 text-xs font-normal text-muted-foreground hover:text-foreground"
                                @click="saveImgKey"
                                >保存</Button
                            >
                        </div>
                    </div>
                    <Input v-model="imgKey" class="h-9 font-mono text-xs" placeholder="请输入图片解密密钥" />
                </div>
            </div>
        </section>

        <section class="space-y-4">
            <header class="space-y-1">
                <h3 class="font-serif text-xl font-medium tracking-tight text-foreground">数据库解密</h3>
                <p class="text-sm text-muted-foreground">开始前请确保目录与密钥已保存。该操作耗资较高，请耐心等待。</p>
            </header>
            <div class="rounded-xl bg-muted/30 ring-1 ring-border/40 px-5 py-4 text-sm text-muted-foreground">
                解密后的数据将存放于 <span class="font-mono text-foreground/90">{{ workDir || '未定义' }}</span
                >。
            </div>
        </section>
    </div>
</template>
