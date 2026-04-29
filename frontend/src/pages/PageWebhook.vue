<script setup lang="ts">
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { inject, onMounted, ref } from 'vue';
import { appContextKey } from '../app/context';
import { backend, type WebhookConfig, type WebhookItem } from '../wailsbridge';

const injected = inject(appContextKey);
if (!injected) throw new Error('chatlog not provided');
const app = injected;

const cfg = ref<WebhookConfig>({ host: '', delayMs: 0, items: [] });

async function load() {
    try {
        cfg.value = await backend.GetWebhookConfig();
        if (!cfg.value.items) cfg.value.items = [];
    } catch (e) {
        app.feedback.toast('读取失败', String(e));
    }
}

function addItem() {
    cfg.value.items.push({
        description: '',
        type: 'message',
        url: '',
        talker: '',
        sender: '',
        keyword: '',
        disabled: false
    });
}

async function removeItem(index: number) {
    const ok = await app.feedback.confirm({
        title: '删除规则',
        message: '确定删除该规则？此操作将从配置中移除，保存后生效。',
        confirmText: '删除',
        cancelText: '取消',
        danger: true
    });
    if (!ok) return;
    cfg.value.items.splice(index, 1);
}

function normalizeItem(it: WebhookItem) {
    it.type = 'message';
    it.description = (it.description || '').trim();
    it.url = (it.url || '').trim();
    it.talker = (it.talker || '').trim();
    it.sender = (it.sender || '').trim();
    it.keyword = (it.keyword || '').trim();
}

function validateItems(items: WebhookItem[]) {
    for (let i = 0; i < items.length; i++) {
        const item = items[i];
        if (!item.url) {
            app.feedback.toast('校验失败', `第 ${i + 1} 条规则缺少 URL`);
            return false;
        }
        if (!item.talker) {
            app.feedback.toast('校验失败', `第 ${i + 1} 条规则缺少 Talker`);
            return false;
        }
    }
    return true;
}

async function save() {
    const ok = await app.feedback.confirm({
        title: '保存 Webhook 配置',
        message: '确认保存并立即应用当前配置？',
        confirmText: '保存',
        cancelText: '取消'
    });
    if (!ok) return;
    const next: WebhookConfig = {
        host: (cfg.value.host || '').trim(),
        delayMs: Number(cfg.value.delayMs || 0),
        items: cfg.value.items.map(x => ({ ...x }))
    };
    for (const it of next.items) normalizeItem(it);
    if (!validateItems(next.items)) return;
    await app.run(() => backend.SetWebhookConfig(next), '已保存 Webhook 配置');
}

onMounted(() => {
    void load();
});
</script>

<template>
    <div class="space-y-10">
        <div
            class="sticky -top-px z-30 flex flex-wrap items-center justify-between gap-3 rounded-xl bg-muted/30 ring-1 ring-border/40 px-4 py-3 backdrop-blur supports-[backdrop-filter]:bg-muted/40"
        >
            <p class="text-sm text-muted-foreground">基础配置与转发规则共用同一份配置，修改后需统一保存生效。</p>
            <div class="flex items-center gap-2">
                <Button
                    variant="ghost"
                    size="sm"
                    class="h-9 px-3 text-sm font-normal text-muted-foreground hover:text-foreground"
                    @click="load"
                >
                    刷新
                </Button>
                <Button size="sm" class="h-9 px-4 text-sm" @click="save">保存</Button>
            </div>
        </div>

        <section class="space-y-4">
            <header class="space-y-1">
                <h3 class="font-serif text-xl font-medium tracking-tight text-foreground">基础配置</h3>
                <p class="text-sm text-muted-foreground">推送消息中静态资源的访问地址与队列延迟。</p>
            </header>

            <div class="rounded-xl bg-muted/30 ring-1 ring-border/40 p-5 sm:p-6">
                <div class="grid gap-x-6 gap-y-5 md:grid-cols-2">
                    <div class="space-y-2">
                        <label class="text-sm text-foreground">资源地址</label>
                        <Input
                            v-model="cfg.host"
                            class="h-10 bg-background/40 font-mono text-sm"
                            placeholder="localhost:5030"
                        />
                        <p class="text-xs text-muted-foreground">
                            推送消息中图片、文件等资源的访问 Host，通常指向 Chatlog 服务。
                        </p>
                    </div>
                    <div class="space-y-2">
                        <label class="text-sm text-foreground"
                            >队列延迟 <span class="text-muted-foreground">(毫秒)</span></label
                        >
                        <Input
                            v-model.number="cfg.delayMs"
                            type="number"
                            min="0"
                            step="100"
                            class="h-10 bg-background/40 font-mono text-sm"
                        />
                        <p class="text-xs text-muted-foreground">
                            数据库变更后触发推送的等待时间，避免锁定或资源未同步。
                        </p>
                    </div>
                </div>
            </div>
        </section>

        <section class="space-y-5">
            <header class="flex flex-wrap items-end justify-between gap-3">
                <div class="space-y-1">
                    <h3 class="font-serif text-xl font-medium tracking-tight text-foreground">转发规则</h3>
                    <p class="text-sm text-muted-foreground">匹配条件与目标 URL，保存后生效。</p>
                </div>
                <div class="flex items-center gap-3 text-xs text-muted-foreground">
                    <span>共 {{ cfg.items.length }} 条</span>
                    <Button
                        variant="ghost"
                        size="sm"
                        class="h-9 px-3 text-sm font-normal text-muted-foreground hover:text-foreground"
                        @click="addItem"
                        >+ 新增规则</Button
                    >
                </div>
            </header>

            <div v-if="cfg.items.length === 0" class="rounded-xl bg-muted/30 ring-1 ring-border/40 py-14 text-center">
                <div class="text-sm text-foreground/80">暂无推送规则</div>
                <p class="mt-1 text-xs text-muted-foreground">添加转发规则并保存，即可开始推送。</p>
                <Button variant="link" size="sm" class="mt-3 text-sm" @click="addItem">立即添加</Button>
            </div>

            <div v-else class="space-y-4">
                <article
                    v-for="(it, idx) in cfg.items"
                    :key="idx"
                    class="space-y-4 rounded-xl bg-muted/30 ring-1 ring-border/40 p-5 sm:p-6"
                >
                    <div class="flex items-center justify-between gap-2">
                        <div class="flex items-baseline gap-2">
                            <span class="font-serif text-base font-medium tracking-tight text-foreground"
                                >规则 {{ idx + 1 }}</span
                            >
                            <Button
                                variant="ghost"
                                size="sm"
                                class="h-7 gap-1.5 px-2 text-xs font-normal text-muted-foreground hover:text-foreground"
                                @click="it.disabled = !it.disabled"
                            >
                                <span
                                    :class="[
                                        'size-1.5 rounded-full',
                                        it.disabled ? 'bg-muted-foreground/40' : 'bg-emerald-500'
                                    ]"
                                />
                                {{ it.disabled ? '已禁用' : '已启用' }}
                            </Button>
                        </div>
                        <Button
                            variant="ghost"
                            size="sm"
                            class="h-8 px-2.5 text-xs font-normal text-muted-foreground hover:bg-destructive/10 hover:text-destructive"
                            @click="removeItem(idx)"
                        >
                            删除
                        </Button>
                    </div>

                    <div class="space-y-2">
                        <label class="text-xs text-muted-foreground">目标 URL</label>
                        <Input
                            v-model="it.url"
                            class="h-9 font-mono text-xs"
                            placeholder="http://127.0.0.1:3000/api/v1/webhook"
                        />
                    </div>
                    <div class="grid gap-3 md:grid-cols-3">
                        <div class="space-y-2">
                            <label class="text-xs text-muted-foreground">Talker / 对象 WXID</label>
                            <Input v-model="it.talker" class="h-9 font-mono text-xs" placeholder="群组或用户 ID" />
                        </div>
                        <div class="space-y-2">
                            <label class="text-xs text-muted-foreground">Sender / 发送者 (可选)</label>
                            <Input v-model="it.sender" class="h-9 font-mono text-xs" placeholder="具体发言人 ID" />
                        </div>
                        <div class="space-y-2">
                            <label class="text-xs text-muted-foreground">Keyword / 关键词 (可选)</label>
                            <Input v-model="it.keyword" class="h-9 font-mono text-xs" placeholder="包含特定词汇触发" />
                        </div>
                    </div>
                    <div class="space-y-2">
                        <label class="text-xs text-muted-foreground">备注</label>
                        <Input
                            v-model="it.description"
                            class="h-9 text-sm"
                            placeholder="例如：转发群组消息到本地日志服务"
                        />
                    </div>
                </article>
            </div>
        </section>
    </div>
</template>
