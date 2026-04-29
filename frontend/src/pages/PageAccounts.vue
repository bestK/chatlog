<script setup lang="ts">
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { computed, inject, nextTick, onMounted, ref, watch } from 'vue';
import { appContextKey } from '../app/context';
import { backend, type Contact, type Instance } from '../wailsbridge';

const injected = inject(appContextKey);
if (!injected) throw new Error('chatlog not provided');
const app = injected;

const { instances, run, state } = app;

const contactKeyword = ref('');
const contactsLoading = ref(false);
const contactsTotal = ref(0);
const contacts = ref<Contact[]>([]);
const contactLimit = ref(50);
const contactOffset = ref(0);
const contactLoadingSource = ref<'init' | 'refresh' | 'search' | 'prev' | 'next' | 'limit' | 'account' | null>(null);

const contactRangeText = computed(() => {
    if (contactsTotal.value <= 0) return '0 / 0';
    const start = Math.min(contactOffset.value + 1, contactsTotal.value);
    const end = Math.min(contactOffset.value + contacts.value.length, contactsTotal.value);
    return `${start}-${end} / ${contactsTotal.value}`;
});

const hasContacts = computed(() => contacts.value.length > 0);
const hasKeyword = computed(() => contactKeyword.value.trim().length > 0);
const contactPageText = computed(() => {
    if (contactsTotal.value <= 0) return '第 0 页';
    return `第 ${Math.floor(contactOffset.value / contactLimit.value) + 1} 页`;
});

const prevButtonText = computed(() =>
    contactsLoading.value && contactLoadingSource.value === 'prev' ? '加载中…' : '上一页'
);
const nextButtonText = computed(() =>
    contactsLoading.value && contactLoadingSource.value === 'next' ? '加载中…' : '下一页'
);

let contactLoadTimer: number | undefined;

function getPageScrollContainer() {
    if (typeof document === 'undefined') return null;
    const page = document.querySelector('.page');
    return page instanceof HTMLElement ? page : null;
}

function restorePageScroll(scrollTop: number | null) {
    if (scrollTop === null) return;
    const container = getPageScrollContainer();
    if (!container) return;
    window.requestAnimationFrame(() => {
        container.scrollTop = scrollTop;
    });
}

async function loadContacts(options?: {
    preserveScroll?: boolean;
    source?: 'init' | 'refresh' | 'search' | 'prev' | 'next' | 'limit' | 'account';
}) {
    if (!backend.isWails) {
        contacts.value = [];
        contactsTotal.value = 0;
        return;
    }
    contactLoadingSource.value = options?.source ?? 'refresh';
    const preservedScrollTop = options?.preserveScroll ? getPageScrollContainer()?.scrollTop ?? 0 : null;
    contactsLoading.value = true;
    try {
        const resp = await backend.GetContacts(
            contactKeyword.value.trim(),
            -1,
            contactLimit.value,
            contactOffset.value
        );
        contactsTotal.value = resp.total || 0;
        contacts.value = Array.isArray(resp.items) ? resp.items : [];
    } catch (e) {
        app.feedback.toast('加载联系人失败', String(e));
    } finally {
        contactsLoading.value = false;
        contactLoadingSource.value = null;
        await nextTick();
        restorePageScroll(preservedScrollTop);
    }
}

function scheduleLoadContacts(delayMs = 200) {
    if (contactLoadTimer) window.clearTimeout(contactLoadTimer);
    contactLoadTimer = window.setTimeout(() => {
        void loadContacts({ preserveScroll: true, source: 'search' });
    }, delayMs);
}

function prevContactsPage() {
    contactOffset.value = Math.max(0, contactOffset.value - contactLimit.value);
    void loadContacts({ preserveScroll: true, source: 'prev' });
}

function nextContactsPage() {
    if (contactOffset.value + contactLimit.value >= contactsTotal.value) return;
    contactOffset.value = contactOffset.value + contactLimit.value;
    void loadContacts({ preserveScroll: true, source: 'next' });
}

function getAccountName(instance: Instance) {
    if (state.value?.pid === instance.pid && state.value.nickname) {
        return state.value.nickname;
    }
    return instance.name || '未知账号';
}

function getAccountAvatar(instance: Instance) {
    if (state.value?.pid === instance.pid) {
        return state.value.smallHeadImgUrl || '';
    }
    return '';
}

function getAvatarFallback(instance: Instance) {
    const name = getAccountName(instance).trim();
    return name ? name.slice(0, 1).toUpperCase() : '?';
}

function switchTo(pid: number) {
    return run(() => backend.SwitchToPID(pid), '已切换账号');
}

function getContactName(c: Contact) {
    return c.remark || c.nickName || c.alias || c.userName || '未知联系人';
}

function getContactAvatar(c: Contact) {
    return c.smallHeadImgUrl || '';
}

function getContactAvatarFallback(c: Contact) {
    const name = getContactName(c).trim();
    return name ? name.slice(0, 1).toUpperCase() : '?';
}

onMounted(() => {
    void loadContacts({ source: 'init' });
});

watch(
    () => state.value?.account,
    () => {
        contactOffset.value = 0;
        if (contactLoadTimer) window.clearTimeout(contactLoadTimer);
        void loadContacts({ preserveScroll: true, source: 'account' });
    }
);

watch(contactKeyword, () => {
    contactOffset.value = 0;
    scheduleLoadContacts();
});

watch(contactLimit, () => {
    contactOffset.value = 0;
    void loadContacts({ preserveScroll: true, source: 'limit' });
});
</script>

<template>
    <div class="space-y-12">
        <section class="space-y-3">
            <div class="grid gap-4 [grid-template-columns:repeat(auto-fit,minmax(320px,1fr))]">
                <div
                    v-if="instances.length === 0"
                    class="col-span-full rounded-lg border border-dashed border-border/40 py-12 text-center"
                >
                    <div class="text-sm text-foreground/80">未检测到活跃的微信进程</div>
                    <div class="mt-1 text-xs text-muted-foreground">启动并登录微信后，点击顶部“刷新”。</div>
                </div>

                <div
                    v-for="instance in instances"
                    :key="instance.pid"
                    class="rounded-xl bg-muted/30 ring-1 ring-border/40 p-4 space-y-3"
                >
                    <div class="flex items-center justify-between gap-3">
                        <div class="flex items-center gap-3 min-w-0">
                            <div class="relative shrink-0">
                                <img
                                    v-if="getAccountAvatar(instance)"
                                    :src="getAccountAvatar(instance)"
                                    class="size-9 rounded-full object-cover"
                                />
                                <div
                                    v-else
                                    class="flex size-9 items-center justify-center rounded-full bg-muted/40 text-sm font-medium"
                                >
                                    {{ getAvatarFallback(instance) }}
                                </div>
                                <span
                                    class="absolute -right-0.5 -bottom-0.5 size-2.5 rounded-full ring-2 ring-background"
                                    :class="instance.status === 'online' ? 'bg-emerald-500' : 'bg-rose-500'"
                                />
                            </div>
                            <div class="min-w-0 space-y-0.5">
                                <div class="truncate text-sm font-medium tracking-tight">
                                    {{ getAccountName(instance) }}
                                </div>
                                <div class="flex flex-wrap items-center gap-x-1.5 text-xs text-muted-foreground">
                                    <span class="font-mono">{{ instance.pid }}</span>
                                    <span class="opacity-50">·</span>
                                    <span>v{{ instance.fullVersion || '-' }}</span>
                                    <span class="opacity-50">·</span>
                                    <span>{{ instance.platform || '-' }}</span>
                                </div>
                            </div>
                        </div>
                        <Button
                            variant="ghost"
                            size="sm"
                            class="h-8 px-2.5 text-xs font-normal text-muted-foreground hover:text-foreground"
                            @click="switchTo(instance.pid)"
                        >
                            切换
                        </Button>
                    </div>

                    <div
                        class="break-all font-mono text-xs leading-relaxed text-foreground/70"
                        :title="instance.dataDir"
                    >
                        {{ instance.dataDir || '未定义路径' }}
                    </div>
                </div>
            </div>
        </section>

        <section class="space-y-5">
            <header class="flex flex-wrap items-end justify-between gap-3">
                <div class="space-y-1">
                    <h3 class="font-serif text-xl font-medium tracking-tight text-foreground">联系人</h3>
                    <p class="text-sm text-muted-foreground">检索当前账号下的联系人、群聊与公众号。</p>
                </div>
                <div class="flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-muted-foreground">
                    <span>{{ contactRangeText }}</span>
                    <span class="opacity-50">·</span>
                    <span>{{ contactPageText }}</span>
                    <template v-if="hasKeyword">
                        <span class="opacity-50">·</span>
                        <span class="text-foreground">“{{ contactKeyword }}”</span>
                    </template>
                </div>
            </header>

            <div
                class="sticky -top-px z-30 flex flex-wrap items-center gap-3 rounded-xl bg-muted/30 ring-1 ring-border/40 px-3 py-2.5 backdrop-blur supports-[backdrop-filter]:bg-muted/40"
            >
                <div class="relative w-full min-w-[200px] flex-1 basis-[260px]">
                    <Input
                        v-model="contactKeyword"
                        class="h-9 bg-background/40 pl-3 pr-9 text-sm"
                        placeholder="搜索昵称、微信号、备注或 ID"
                    />
                    <div class="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground/50">
                        <svg
                            xmlns="http://www.w3.org/2000/svg"
                            width="15"
                            height="15"
                            viewBox="0 0 24 24"
                            fill="none"
                            stroke="currentColor"
                            stroke-width="2"
                            stroke-linecap="round"
                            stroke-linejoin="round"
                        >
                            <circle cx="11" cy="11" r="8" />
                            <path d="m21 21-4.3-4.3" />
                        </svg>
                    </div>
                </div>
                <select
                    v-model.number="contactLimit"
                    class="h-9 rounded-md border border-input bg-background/40 px-2.5 text-sm focus:outline-none focus:ring-1 focus:ring-ring"
                >
                    <option :value="20">20 / 页</option>
                    <option :value="50">50 / 页</option>
                    <option :value="100">100 / 页</option>
                    <option :value="200">200 / 页</option>
                </select>
                <Button
                    variant="ghost"
                    size="icon"
                    :disabled="contactsLoading"
                    class="h-9 w-9 text-muted-foreground hover:text-foreground"
                    @click="loadContacts({ source: 'refresh' })"
                >
                    <svg
                        xmlns="http://www.w3.org/2000/svg"
                        width="15"
                        height="15"
                        viewBox="0 0 24 24"
                        fill="none"
                        stroke="currentColor"
                        stroke-width="2"
                        stroke-linecap="round"
                        stroke-linejoin="round"
                        :class="contactsLoading ? 'animate-spin' : ''"
                    >
                        <path d="M21 12a9 9 0 1 1-9-9c2.52 0 4.93 1 6.74 2.74L21 8" />
                        <path d="M21 3v5h-5" />
                    </svg>
                </Button>
                <div class="flex items-center gap-1">
                    <Button
                        variant="ghost"
                        size="sm"
                        class="h-9 px-3 text-sm font-normal text-muted-foreground hover:text-foreground"
                        :disabled="contactsLoading || contactOffset === 0"
                        @click="prevContactsPage"
                    >
                        {{ prevButtonText }}
                    </Button>
                    <Button
                        variant="ghost"
                        size="sm"
                        class="h-9 px-3 text-sm font-normal text-muted-foreground hover:text-foreground"
                        :disabled="contactsLoading || contactOffset + contactLimit >= contactsTotal"
                        @click="nextContactsPage"
                    >
                        {{ nextButtonText }}
                    </Button>
                </div>
            </div>

            <div class="relative">
                <div
                    v-if="contactsLoading && hasContacts"
                    class="pointer-events-none absolute inset-x-0 -top-1 z-40 flex justify-center"
                >
                    <div class="text-xs text-muted-foreground">同步中…</div>
                </div>

                <div v-if="contactsLoading && !hasContacts" class="py-16 text-center">
                    <div class="text-sm text-foreground/80">正在检索联系人数据库</div>
                    <div class="mt-1 text-xs text-muted-foreground">正在解析当前账号的联系人信息，请稍候。</div>
                </div>

                <div v-else-if="!hasContacts" class="py-16 text-center">
                    <div class="text-sm text-foreground/80">没有匹配的结果</div>
                    <div class="mt-1 text-xs text-muted-foreground">尝试调整关键词，或检查当前登录账号。</div>
                </div>

                <div
                    v-else
                    class="grid gap-3 [grid-template-columns:repeat(auto-fit,minmax(260px,1fr))]"
                    :class="contactsLoading ? 'opacity-50 transition-opacity' : ''"
                >
                    <div
                        v-for="contact in contacts"
                        :key="contact.userName"
                        class="group rounded-lg border border-border/30 bg-card/20 p-4 transition-colors hover:bg-card/40"
                    >
                        <div class="flex items-center gap-3">
                            <img
                                v-if="getContactAvatar(contact)"
                                :src="getContactAvatar(contact)"
                                class="size-10 rounded-full object-cover"
                            />
                            <div
                                v-else
                                class="flex size-10 items-center justify-center rounded-full bg-muted/40 text-sm font-medium text-muted-foreground"
                            >
                                {{ getContactAvatarFallback(contact) }}
                            </div>

                            <div class="min-w-0 flex-1 space-y-0.5">
                                <div class="truncate text-sm font-medium tracking-tight">
                                    {{ getContactName(contact) }}
                                </div>
                                <div class="truncate font-mono text-xs text-muted-foreground">
                                    {{ contact.alias || contact.userName }}
                                </div>
                            </div>
                            <span
                                v-if="contact.isInChatRoom"
                                class="shrink-0 rounded-full bg-primary/10 px-2 py-0.5 text-xs text-primary"
                            >
                                群聊
                            </span>
                        </div>
                    </div>
                </div>
            </div>
        </section>
    </div>
</template>
