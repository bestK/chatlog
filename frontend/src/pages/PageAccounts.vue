<script setup lang="ts">
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Pagination } from '@/components/ui/pagination';
import { RefreshCw, Search } from 'lucide-vue-next';
import { computed, inject, nextTick, onMounted, ref, watch } from 'vue';
import { appContextKey } from '../app/context';
import { backend, type Contact, type Instance } from '../wailsbridge';

const injected = inject(appContextKey);
if (!injected) throw new Error('chatlog not provided');
const app = injected;

const { instances, run, state } = app;

// --- Account switching ---

const switchingPid = ref<number | null>(null);

function isCurrent(instance: Instance) {
    return state.value?.pid === instance.pid;
}

async function switchTo(pid: number) {
    switchingPid.value = pid;
    try {
        await run(() => backend.SwitchToPID(pid), '已切换账号');
    } finally {
        switchingPid.value = null;
    }
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

// --- Contacts ---

const contactKeyword = ref('');
const contactsLoading = ref(false);
const contactsTotal = ref(0);
const contacts = ref<Contact[]>([]);
const contactLimit = ref(50);
const contactOffset = ref(0);

const hasContacts = computed(() => contacts.value.length > 0);

let contactLoadTimer: number | undefined;

async function loadContacts(source?: string) {
    if (!backend.isWails) {
        contacts.value = [];
        contactsTotal.value = 0;
        return;
    }
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
    }
}

function scheduleSearch(delayMs = 200) {
    if (contactLoadTimer) window.clearTimeout(contactLoadTimer);
    contactLoadTimer = window.setTimeout(() => void loadContacts('search'), delayMs);
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

onMounted(() => void loadContacts('init'));

watch(() => state.value?.account, () => {
    contactOffset.value = 0;
    if (contactLoadTimer) window.clearTimeout(contactLoadTimer);
    void loadContacts('account');
});

watch(contactKeyword, () => {
    contactOffset.value = 0;
    scheduleSearch();
});

watch(contactLimit, () => {
    contactOffset.value = 0;
    void loadContacts('limit');
});
</script>

<template>
    <div class="space-y-12">
        <!-- Instances -->
        <section class="space-y-3">
            <div class="grid gap-4 [grid-template-columns:repeat(auto-fit,minmax(320px,1fr))]">
                <div
                    v-if="instances.length === 0"
                    class="col-span-full rounded-lg border border-dashed border-border/40 py-12 text-center"
                >
                    <div class="text-sm text-foreground/80">未检测到活跃的微信进程</div>
                    <div class="mt-1 text-xs text-muted-foreground">启动并登录微信后，点击顶部"刷新"。</div>
                </div>

                <div
                    v-for="instance in instances"
                    :key="instance.pid"
                    class="rounded-xl p-4 space-y-3"
                    :class="isCurrent(instance)
                        ? 'bg-primary/5 ring-2 ring-primary/30'
                        : 'bg-muted/30 ring-1 ring-border/40'"
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
                            v-if="instances.length > 1"
                            variant="ghost"
                            size="sm"
                            class="h-8 px-2.5 text-xs font-normal"
                            :class="isCurrent(instance)
                                ? 'text-primary cursor-default'
                                : 'text-muted-foreground hover:text-foreground'"
                            :disabled="isCurrent(instance) || instance.status === 'offline' || switchingPid === instance.pid"
                            @click="switchTo(instance.pid)"
                        >
                            {{ isCurrent(instance) ? '当前' : switchingPid === instance.pid ? '切换中…' : instance.status === 'offline' ? '离线' : '切换' }}
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

        <!-- Contacts -->
        <section class="space-y-5">
            <header class="space-y-1">
                <h3 class="font-serif text-xl font-medium tracking-tight text-foreground">联系人</h3>
                <p class="text-sm text-muted-foreground">检索当前账号下的联系人、群聊与公众号。</p>
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
                    <Search class="absolute right-3 top-1/2 -translate-y-1/2 size-[15px] text-muted-foreground/50" />
                </div>
                <Pagination
                    :total="contactsTotal"
                    :limit="contactLimit"
                    :offset="contactOffset"
                    :loading="contactsLoading"
                    @update:offset="contactOffset = $event; loadContacts('page')"
                    @update:limit="contactLimit = $event; contactOffset = 0; loadContacts('limit')"
                />
                <Button
                    variant="ghost"
                    size="icon"
                    :disabled="contactsLoading"
                    class="h-9 w-9 text-muted-foreground hover:text-foreground"
                    @click="loadContacts('refresh')"
                >
                    <RefreshCw class="size-[15px]" :class="contactsLoading ? 'animate-spin' : ''" />
                </Button>
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