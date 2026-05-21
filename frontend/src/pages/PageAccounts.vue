<script setup lang="ts">
import { Button } from '@/components/ui/button';
import { ImagePreview } from '@/components/ui/image-preview';
import { Input } from '@/components/ui/input';
import { Pagination } from '@/components/ui/pagination';
import { Sheet, SheetContent, SheetDescription, SheetHeader, SheetTitle } from '@/components/ui/sheet';
import { RefreshCw, Search } from 'lucide-vue-next';
import { computed, inject, nextTick, onMounted, ref, watch } from 'vue';
import { appContextKey } from '../app/context';
import { backend, type Contact, type Instance } from '../wailsbridge';
import ChatBubble, { type ChatMessage } from '@/components/chat/ChatBubble.vue';
import ContactCard from '@/components/chat/ContactCard.vue';
import AISummaryPanel from '@/components/chat/AISummaryPanel.vue';

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

onMounted(() => void loadContacts('init'));

watch(
    () => state.value?.account,
    () => {
        contactOffset.value = 0;
        if (contactLoadTimer) window.clearTimeout(contactLoadTimer);
        void loadContacts('account');
    }
);

watch(contactKeyword, () => {
    contactOffset.value = 0;
    scheduleSearch();
});

watch(contactLimit, () => {
    contactOffset.value = 0;
    void loadContacts('limit');
});

// --- Chat Detail Sheet ---

const summaryOpen = ref(false);
const summaryContact = ref<Contact | null>(null);
const summaryPanelRef = ref<InstanceType<typeof AISummaryPanel> | null>(null);

const chatMessages = ref<ChatMessage[]>([]);
const chatLoading = ref(false);
const chatLoadingMore = ref(false);
const chatHasMore = ref(true);
const chatOffset = ref(0);
const chatPageSize = 10;
const chatScrollRef = ref<HTMLElement | null>(null);
const previewImage = ref('');
const previewOpen = ref(false);

async function openContact(contact: Contact) {
    summaryContact.value = contact;
    summaryOpen.value = true;
    chatMessages.value = [];
    chatOffset.value = 0;
    chatHasMore.value = true;
    summaryPanelRef.value?.reset();

    await loadChatMessages(contact, false);
    summaryPanelRef.value?.fetchProviders();
}

async function loadChatMessages(contact: Contact, prepend: boolean) {
    if (prepend) {
        chatLoadingMore.value = true;
    } else {
        chatLoading.value = true;
        chatOffset.value = 0;
        chatHasMore.value = true;
    }
    try {
        const resp = await backend.GetMessages('all', contact.userName, '', '', chatPageSize, chatOffset.value, 'desc');
        const items = resp.items || [];

        const mapped: ChatMessage[] = items
            .map((m: any) => ({
                time: m.time || '',
                senderName: m.senderName || m.sender || '',
                sender: m.sender || '',
                content: m.content || '',
                type: m.type || 1,
                contents: m.contents,
                isSelf: !!m.isSelf
            }))
            .reverse();

        if (items.length < chatPageSize) chatHasMore.value = false;
        chatOffset.value += items.length;

        if (prepend) {
            const el = chatScrollRef.value;
            const prevHeight = el?.scrollHeight || 0;
            chatMessages.value = [...mapped, ...chatMessages.value];
            nextTick(() => {
                if (el) el.scrollTop = el.scrollHeight - prevHeight;
            });
        } else {
            chatMessages.value = mapped;
            nextTick(() => {
                if (chatScrollRef.value) chatScrollRef.value.scrollTop = chatScrollRef.value.scrollHeight;
                checkNeedMore();
            });
        }
    } catch (e) {
        app.feedback.toast('加载聊天记录失败', String(e));
    } finally {
        chatLoading.value = false;
        chatLoadingMore.value = false;
    }
}

function onChatScroll() {
    const el = chatScrollRef.value;
    if (!el || chatLoadingMore.value || !chatHasMore.value || !summaryContact.value) return;
    if (el.scrollTop < 60) {
        void loadChatMessages(summaryContact.value, true);
    }
}

function checkNeedMore() {
    nextTick(() => {
        const el = chatScrollRef.value;
        if (!el || !chatHasMore.value || chatLoadingMore.value || !summaryContact.value) return;
        if (el.scrollHeight <= el.clientHeight && chatMessages.value.length > 0) {
            void loadChatMessages(summaryContact.value, true);
        }
    });
}
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
                    :class="isCurrent(instance) ? 'bg-primary/5 ring-primary/30' : 'bg-muted/30 ring-1 ring-border/40'"
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
                            :class="
                                isCurrent(instance)
                                    ? 'text-primary cursor-default'
                                    : 'text-muted-foreground hover:text-foreground'
                            "
                            :disabled="
                                isCurrent(instance) || instance.status === 'offline' || switchingPid === instance.pid
                            "
                            @click="switchTo(instance.pid)"
                        >
                            {{
                                isCurrent(instance)
                                    ? '当前'
                                    : switchingPid === instance.pid
                                    ? '切换中…'
                                    : instance.status === 'offline'
                                    ? '离线'
                                    : '切换'
                            }}
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
                    @update:offset="
                        contactOffset = $event;
                        loadContacts('page');
                    "
                    @update:limit="
                        contactLimit = $event;
                        contactOffset = 0;
                        loadContacts('limit');
                    "
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
                    <ContactCard
                        v-for="contact in contacts"
                        :key="contact.userName"
                        :contact="contact"
                        @click="openContact(contact)"
                    />
                </div>
            </div>
        </section>

        <!-- Contact Detail Sheet -->
        <Sheet v-model:open="summaryOpen">
            <SheetContent side="right" class="w-full sm:max-w-xl flex flex-col overflow-hidden">
                <SheetHeader class="shrink-0">
                    <SheetTitle>{{ summaryContact ? getContactName(summaryContact) : '' }}</SheetTitle>
                    <SheetDescription>
                        {{ summaryContact?.alias || summaryContact?.userName || '' }}
                    </SheetDescription>
                </SheetHeader>

                <div class="flex-1 flex flex-col gap-4 overflow-hidden pt-4">
                    <!-- Chat messages -->
                    <div
                        ref="chatScrollRef"
                        class="flex-1 min-h-0 overflow-auto rounded-md bg-muted/20 ring-1 ring-border/30 p-3 space-y-1.5"
                        @scroll="onChatScroll"
                    >
                        <div v-if="chatLoadingMore" class="flex items-center justify-center py-2">
                            <span class="size-3 animate-spin rounded-full border-2 border-muted-foreground/40 border-t-foreground"></span>
                            <span class="ml-1.5 text-xs text-muted-foreground">加载更多…</span>
                        </div>
                        <div v-else-if="!chatHasMore && chatMessages.length > 0" class="text-center py-1">
                            <span class="text-xs text-muted-foreground/60">已加载全部</span>
                        </div>
                        <div v-if="chatLoading" class="flex items-center justify-center py-8">
                            <span class="size-4 animate-spin rounded-full border-2 border-muted-foreground/40 border-t-foreground"></span>
                            <span class="ml-2 text-xs text-muted-foreground">加载中…</span>
                        </div>
                        <div v-else-if="chatMessages.length === 0" class="py-8 text-center text-xs text-muted-foreground">
                            今天暂无聊天记录
                        </div>
                        <ChatBubble
                            v-for="(msg, i) in chatMessages"
                            :key="i"
                            :msg="msg"
                            @preview="previewImage = $event; previewOpen = true;"
                        />
                    </div>

                    <!-- AI Summary -->
                    <AISummaryPanel
                        ref="summaryPanelRef"
                        :contact-user-name="summaryContact?.userName || ''"
                    />
                </div>
            </SheetContent>
        </Sheet>

        <ImagePreview v-model:open="previewOpen" :src="previewImage" />
    </div>
</template>