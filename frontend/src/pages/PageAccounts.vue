<script setup lang="ts">
import { Button } from '@/components/ui/button';
import { CustomSelect } from '@/components/ui/custom-select';
import { ImagePreview } from '@/components/ui/image-preview';
import { Input } from '@/components/ui/input';
import { Pagination } from '@/components/ui/pagination';
import { Sheet, SheetContent, SheetDescription, SheetHeader, SheetTitle } from '@/components/ui/sheet';
import { Copy, RefreshCw, Search, Sparkles } from 'lucide-vue-next';
import MarkdownRender from 'markstream-vue';
import 'markstream-vue/index.css';
import { computed, inject, nextTick, onMounted, reactive, ref, watch } from 'vue';
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

// --- AI Summary ---

type AIProvider = { id: string; name: string; type: string; baseUrl: string; model: string };
type ChatMessage = {
    time: string;
    senderName: string;
    sender: string;
    content: string;
    type: number;
    contents?: Record<string, any>;
    isSelf: boolean;
};

const summaryOpen = ref(false);
const summaryContact = ref<Contact | null>(null);
const summaryProviders = ref<AIProvider[]>([]);
const summaryProvider = ref(state.value?.selectedAIProvider || '');
const summaryTimeRange = ref('today');

watch(summaryProvider, v => {
    if (v) backend.SetSelectedAIProvider(v);
});

const providerOptions = computed(() => summaryProviders.value.map(p => ({ value: p.id, label: p.name })));
const summaryStream = ref('');
const summaryResult = ref('');
const summaryFinal = ref(false);
const summaryLoading = ref(false);
const summaryError = ref('');
const summaryScrollRef = ref<HTMLElement | null>(null);
const summaryHeight = ref(200);
const summaryResizing = ref(false);

function onResizeStart(e: PointerEvent) {
    summaryResizing.value = true;
    const startY = e.clientY;
    const startH = summaryHeight.value;
    const onMove = (ev: PointerEvent) => {
        const delta = startY - ev.clientY;
        summaryHeight.value = Math.max(80, Math.min(window.innerHeight * 0.7, startH + delta));
    };
    const onUp = () => {
        summaryResizing.value = false;
        document.removeEventListener('pointermove', onMove);
        document.removeEventListener('pointerup', onUp);
    };
    document.addEventListener('pointermove', onMove);
    document.addEventListener('pointerup', onUp);
}

const chatMessages = ref<ChatMessage[]>([]);
const chatLoading = ref(false);
const chatLoadingMore = ref(false);
const chatHasMore = ref(true);
const chatOffset = ref(0);
const chatPageSize = 50;
const chatScrollRef = ref<HTMLElement | null>(null);
const previewImage = ref('');
const previewOpen = ref(false);

const summaryContent = computed(() => summaryStream.value || summaryResult.value);

const timeRangeOptions = [
    { value: 'today', label: '今天' },
    { value: 'yesterday', label: '昨天' },
    { value: 'last-3d', label: '最近 3 天' },
    { value: 'last-7d', label: '最近 7 天' },
    { value: 'last-30d', label: '最近 30 天' },
    { value: 'custom', label: '自定义' }
];

const customStartDate = ref('');
const customEndDate = ref('');

const effectiveTimeRange = computed(() => {
    if (summaryTimeRange.value === 'custom' && customStartDate.value && customEndDate.value) {
        const start = customStartDate.value.replace('T', '/');
        const end = customEndDate.value.replace('T', '/');
        return `${start}~${end}`;
    }
    return summaryTimeRange.value;
});

watch(summaryStream, () => {
    nextTick(() => {
        if (summaryScrollRef.value) {
            summaryScrollRef.value.scrollTop = summaryScrollRef.value.scrollHeight;
        }
    });
});

const mediaCache = reactive<Record<string, string | null>>({});

function mediaKey(type: string, keys: string[]) {
    return `${type}:${keys.filter(Boolean).join(',')}`;
}

function getMediaSrc(type: string, keys: string[]): string | null {
    const k = mediaKey(type, keys);
    if (k in mediaCache) return mediaCache[k];
    mediaCache[k] = null;
    backend.GetMediaData(type, keys.filter(Boolean).join(',')).then(r => {
        mediaCache[k] = r.data || '';
    }).catch(() => {
        mediaCache[k] = '';
    });
    return null;
}

function msgDisplay(msg: ChatMessage): {
    kind: 'text' | 'image' | 'video' | 'voice' | 'emoji' | 'other';
    text?: string;
    url?: string;
    mediaType?: string;
    mediaKeys?: string[];
} {
    const c = msg.contents || {};
    switch (msg.type) {
        case 1:
            return { kind: 'text', text: msg.content };
        case 3:
            return { kind: 'image', mediaType: 'image', mediaKeys: [c.md5, c.path, c.thumbpath] };
        case 34:
            return { kind: 'voice', mediaType: 'voice', mediaKeys: [c.voice] };
        case 43:
            return { kind: 'video', mediaType: 'video', mediaKeys: [c.md5, c.rawmd5, c.path] };
        case 47:
            return { kind: 'emoji', url: c.cdnurl || '' };
        case 48:
            return { kind: 'text', text: `[位置] ${c.label || ''}` };
        case 49: {
            const title = c.title || '';
            const url = c.url || '';
            if (title) return { kind: 'text', text: `[${title}]${url ? ' ' + url : ''}` };
            return { kind: 'text', text: msg.content || '[分享]' };
        }
        case 10000:
            return { kind: 'text', text: msg.content || '[系统消息]' };
        default:
            return { kind: 'text', text: msg.content || `[类型:${msg.type}]` };
    }
}

async function openContact(contact: Contact) {
    summaryContact.value = contact;
    summaryStream.value = '';
    summaryResult.value = '';
    summaryFinal.value = false;
    summaryError.value = '';
    summaryTimeRange.value = 'today';
    summaryOpen.value = true;
    chatMessages.value = [];
    chatOffset.value = 0;
    chatHasMore.value = true;

    await loadChatMessages(contact, 'today', false);
    void fetchProviders();
}

async function loadChatMessages(contact: Contact, _time: string, prepend: boolean) {
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

        if (items.length < chatPageSize) {
            chatHasMore.value = false;
        }
        chatOffset.value += items.length;

        if (prepend) {
            const el = chatScrollRef.value;
            const prevHeight = el?.scrollHeight || 0;
            chatMessages.value = [...mapped, ...chatMessages.value];
            nextTick(() => {
                if (el) {
                    el.scrollTop = el.scrollHeight - prevHeight;
                }
            });
        } else {
            chatMessages.value = mapped;
            nextTick(() => {
                if (chatScrollRef.value) {
                    chatScrollRef.value.scrollTop = chatScrollRef.value.scrollHeight;
                }
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
        void loadChatMessages(summaryContact.value, 'all', true);
    }
}

async function generateSummary() {
    if (!summaryContact.value || !summaryProvider.value) return;

    summaryLoading.value = true;
    summaryError.value = '';
    summaryStream.value = '';
    summaryResult.value = '';
    summaryFinal.value = false;

    try {
        const resp = await backend.GetMessages(
            effectiveTimeRange.value,
            summaryContact.value.userName,
            '',
            '',
            500,
            0,
            'asc'
        );
        const items = resp.items || [];
        if (items.length === 0) {
            summaryError.value = '该时间范围内没有聊天记录';
            summaryLoading.value = false;
            return;
        }

        const messages: string[] = items.map((m: any) => {
            const name = m.senderName || m.sender || '';
            const content = m.content || '';
            const time = m.time || '';
            return `${time} ${name}: ${content}`;
        });

        const offChunk = backend.EventsOn('ai:summary:chunk', (chunk: unknown) => {
            if (typeof chunk === 'string') {
                summaryStream.value += chunk;
            }
        });
        const offDone = backend.EventsOn('ai:summary:done', () => {
            summaryResult.value = summaryStream.value;
            summaryFinal.value = true;
            summaryLoading.value = false;
            offChunk?.();
            offDone?.();
        });

        summaryHeight.value = Math.round(window.innerHeight * 0.7);
        await backend.GenerateAISummary(summaryProvider.value, messages, '');
    } catch (e) {
        summaryError.value = String((e as Error).message || e);
        summaryLoading.value = false;
    }
}

function copySummary() {
    const text = summaryStream.value || summaryResult.value;
    if (text) {
        navigator.clipboard.writeText(text);
        app.feedback.toast('已复制', '总结内容已复制到剪贴板');
    }
}

async function fetchProviders() {
    try {
        const list = await backend.ListAIProviders();
        summaryProviders.value = (list || []).map(p => ({
            id: p.id,
            name: p.name,
            type: p.type,
            baseUrl: p.baseUrl,
            model: p.model
        }));
        if (summaryProviders.value.length > 0) {
            const exists = summaryProviders.value.some(p => p.id === summaryProvider.value);
            if (!exists) {
                summaryProvider.value = summaryProviders.value[0].id;
            }
        }
    } catch {
        /* ignore */
    }
}

function selectTimeRange(val: string) {
    summaryTimeRange.value = val;
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
                    <div
                        v-for="contact in contacts"
                        :key="contact.userName"
                        class="group cursor-pointer rounded-lg border border-border/30 bg-card/20 p-4 transition-colors hover:bg-card/40"
                        @click="openContact(contact)"
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
                            <span
                                class="size-3 animate-spin rounded-full border-2 border-muted-foreground/40 border-t-foreground"
                            ></span>
                            <span class="ml-1.5 text-xs text-muted-foreground">加载更多…</span>
                        </div>
                        <div v-else-if="!chatHasMore && chatMessages.length > 0" class="text-center py-1">
                            <span class="text-xs text-muted-foreground/60">已加载全部</span>
                        </div>
                        <div v-if="chatLoading" class="flex items-center justify-center py-8">
                            <span
                                class="size-4 animate-spin rounded-full border-2 border-muted-foreground/40 border-t-foreground"
                            ></span>
                            <span class="ml-2 text-xs text-muted-foreground">加载中…</span>
                        </div>
                        <div
                            v-else-if="chatMessages.length === 0"
                            class="py-8 text-center text-xs text-muted-foreground"
                        >
                            今天暂无聊天记录
                        </div>
                        <div
                            v-for="(msg, i) in chatMessages"
                            :key="i"
                            class="space-y-0.5"
                            :class="msg.isSelf ? 'text-right' : ''"
                        >
                            <div class="text-[10px] text-muted-foreground/60">
                                <span>{{ msg.senderName }}</span>
                                <span class="ml-1.5">{{ msg.time }}</span>
                            </div>
                            <div
                                class="inline-block max-w-[85%] whitespace-pre-wrap break-words rounded-lg px-2.5 py-1.5 text-left text-xs"
                                :class="msg.isSelf ? 'bg-primary/10 text-foreground' : 'bg-muted/40 text-foreground/90'"
                            >
                                <template v-if="msgDisplay(msg).kind === 'image'">
                                    <template v-if="getMediaSrc(msgDisplay(msg).mediaType!, msgDisplay(msg).mediaKeys!) === null">
                                        <span class="text-muted-foreground/50">[图片加载中…]</span>
                                    </template>
                                    <template v-else-if="getMediaSrc(msgDisplay(msg).mediaType!, msgDisplay(msg).mediaKeys!)">
                                        <img
                                            :src="getMediaSrc(msgDisplay(msg).mediaType!, msgDisplay(msg).mediaKeys!)!"
                                            class="max-w-[200px] max-h-[150px] rounded object-cover cursor-pointer hover:opacity-80 transition-opacity"
                                            @click.stop="
                                                previewImage = getMediaSrc(msgDisplay(msg).mediaType!, msgDisplay(msg).mediaKeys!) || '';
                                                previewOpen = true;
                                            "
                                        />
                                    </template>
                                    <span v-else class="text-muted-foreground/50">[图片]</span>
                                </template>
                                <template v-else-if="msgDisplay(msg).kind === 'video'">
                                    <template v-if="getMediaSrc(msgDisplay(msg).mediaType!, msgDisplay(msg).mediaKeys!) === null">
                                        <span class="text-muted-foreground/50">[视频加载中…]</span>
                                    </template>
                                    <template v-else-if="getMediaSrc(msgDisplay(msg).mediaType!, msgDisplay(msg).mediaKeys!)">
                                        <video
                                            :src="getMediaSrc(msgDisplay(msg).mediaType!, msgDisplay(msg).mediaKeys!)!"
                                            class="max-w-[200px] max-h-[150px] rounded"
                                            controls
                                            preload="none"
                                        />
                                    </template>
                                    <span v-else class="text-muted-foreground/50">[视频]</span>
                                </template>
                                <template v-else-if="msgDisplay(msg).kind === 'voice'">
                                    <template v-if="getMediaSrc(msgDisplay(msg).mediaType!, msgDisplay(msg).mediaKeys!) === null">
                                        <span class="text-muted-foreground/50">[语音加载中…]</span>
                                    </template>
                                    <template v-else-if="getMediaSrc(msgDisplay(msg).mediaType!, msgDisplay(msg).mediaKeys!)">
                                        <audio
                                            :src="getMediaSrc(msgDisplay(msg).mediaType!, msgDisplay(msg).mediaKeys!)!"
                                            controls
                                            preload="none"
                                            class="h-8 max-w-[180px]"
                                        />
                                    </template>
                                    <span v-else class="text-muted-foreground/50">[语音]</span>
                                </template>
                                <template v-else-if="msgDisplay(msg).kind === 'emoji'">
                                    <img v-if="msgDisplay(msg).url" :src="msgDisplay(msg).url" class="size-16 object-contain" loading="lazy" />
                                    <span v-else class="text-muted-foreground">[表情]</span>
                                </template>
                                <template v-else>
                                    {{ msgDisplay(msg).text }}
                                </template>
                            </div>
                        </div>
                    </div>

                    <!-- Summary section -->
                    <div class="shrink-0 space-y-3 border-t border-border/40 pt-3 pb-2 px-3">
                        <div class="flex flex-wrap items-center gap-2">
                            <CustomSelect
                                :model-value="summaryTimeRange"
                                :options="timeRangeOptions"
                                direction="up"
                                @update:model-value="selectTimeRange"
                            />

                            <!-- Custom date range -->
                            <template v-if="summaryTimeRange === 'custom'">
                                <input
                                    v-model="customStartDate"
                                    type="datetime-local"
                                    class="h-8 flex-1 min-w-[130px] rounded-md border border-input bg-background/40 px-2 text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-ring [color-scheme:dark]"
                                />
                                <span class="text-xs text-muted-foreground">~</span>
                                <input
                                    v-model="customEndDate"
                                    type="datetime-local"
                                    class="h-8 flex-1 min-w-[130px] rounded-md border border-input bg-background/40 px-2 text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-ring [color-scheme:dark]"
                                />
                            </template>

                            <div class="flex-1 min-w-[120px]">
                                <CustomSelect
                                    :model-value="summaryProvider"
                                    :options="providerOptions"
                                    placeholder="选择提供商"
                                    direction="up"
                                    @update:model-value="(v: string) => { summaryProvider = v; }"
                                />
                            </div>

                            <Button
                                size="sm"
                                class="h-8 gap-1 px-3 text-xs"
                                :disabled="summaryLoading || !summaryProvider"
                                @click="generateSummary"
                            >
                                <Sparkles class="size-3" />
                                {{ summaryLoading ? '生成中…' : '总结' }}
                            </Button>
                        </div>

                        <!-- Custom date range (separate row) -->
                        <div v-if="summaryTimeRange === 'custom'" class="flex items-center gap-2">
                            <input
                                v-model="customStartDate"
                                type="datetime-local"
                                class="h-8 flex-1 rounded-md border border-input bg-background/40 px-2 text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-ring [color-scheme:dark]"
                            />
                            <span class="text-xs text-muted-foreground">~</span>
                            <input
                                v-model="customEndDate"
                                type="datetime-local"
                                class="h-8 flex-1 rounded-md border border-input bg-background/40 px-2 text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-ring [color-scheme:dark]"
                            />
                        </div>

                        <div v-if="summaryError" class="text-xs text-destructive">{{ summaryError }}</div>

                        <div
                            v-if="summaryContent"
                            class="relative pt-2 flex flex-col"
                            :class="!summaryResizing && 'transition-[height] duration-300 ease-out'"
                            :style="{ height: summaryHeight + 'px' }"
                        >
                            <div
                                class="flex items-center justify-between mb-2 shrink-0 cursor-row-resize select-none"
                                @pointerdown="onResizeStart"
                            >
                                <span class="text-xs font-medium text-foreground/80">AI 总结</span>
                                <div class="flex flex-col gap-[3px]">
                                    <div class="h-[1.5px] w-5 rounded-full bg-muted-foreground/30"></div>
                                    <div class="h-[1.5px] w-5 rounded-full bg-muted-foreground/30"></div>
                                    <div class="h-[1.5px] w-5 rounded-full bg-muted-foreground/30"></div>
                                </div>
                                <button
                                    class="text-xs text-muted-foreground hover:text-foreground px-1.5 py-0.5 rounded hover:bg-muted/40 transition-colors"
                                    @click.stop="copySummary"
                                    @pointerdown.stop
                                >
                                    <Copy class="size-3 inline mr-0.5" />复制
                                </button>
                            </div>
                            <div
                                ref="summaryScrollRef"
                                class="flex-1 min-h-0 overflow-auto rounded-md bg-muted/30 ring-1 ring-border/40 p-3"
                            >
                                <MarkdownRender :content="summaryContent" :final="summaryFinal" :max-live-nodes="0" />
                            </div>
                        </div>
                    </div>
                </div>
            </SheetContent>
        </Sheet>

        <ImagePreview v-model:open="previewOpen" :src="previewImage" />
    </div>
</template>
