<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue';
import type { AutocompleteOption } from '../api';

const props = defineProps<{
    modelValue: string;
    placeholder?: string;
    source: (keyword: string) => Promise<AutocompleteOption[]>;
}>();

const emit = defineEmits<{
    (e: 'update:modelValue', value: string): void;
}>();

const open = ref(false);
const keyword = ref('');
const options = ref<AutocompleteOption[]>([]);
const loading = ref(false);
const wrapperRef = ref<HTMLElement | null>(null);
const inputRef = ref<HTMLInputElement | null>(null);
const activeIndex = ref(-1);

const displayLabel = computed(() => {
    const v = props.modelValue;
    if (!v) return '';
    const hit = options.value.find((o) => o.value === v);
    return hit ? hit.label : v;
});

let searchToken = 0;
let debounceTimer: number | null = null;

async function search(q: string) {
    const token = ++searchToken;
    loading.value = true;
    try {
        const result = await props.source(q);
        if (token !== searchToken) return;
        options.value = result;
        activeIndex.value = result.length > 0 ? 0 : -1;
    } catch {
        if (token !== searchToken) return;
        options.value = [];
    } finally {
        if (token === searchToken) loading.value = false;
    }
}

function scheduleSearch(q: string) {
    if (debounceTimer) window.clearTimeout(debounceTimer);
    debounceTimer = window.setTimeout(() => void search(q), 200);
}

watch(keyword, (q) => {
    if (!open.value) return;
    scheduleSearch(q);
});

function openPanel() {
    if (open.value) return;
    open.value = true;
    keyword.value = '';
    nextTick(() => {
        inputRef.value?.focus();
        void search('');
    });
}

function closePanel() {
    open.value = false;
    activeIndex.value = -1;
}

function pick(opt: AutocompleteOption) {
    emit('update:modelValue', opt.value);
    closePanel();
}

function clear(e: MouseEvent) {
    e.stopPropagation();
    emit('update:modelValue', '');
}

function onClickOutside(e: MouseEvent) {
    if (!open.value) return;
    if (wrapperRef.value && !wrapperRef.value.contains(e.target as Node)) {
        closePanel();
    }
}

function onKeydown(e: KeyboardEvent) {
    if (!open.value) return;
    if (e.key === 'ArrowDown') {
        e.preventDefault();
        activeIndex.value = Math.min(options.value.length - 1, activeIndex.value + 1);
    } else if (e.key === 'ArrowUp') {
        e.preventDefault();
        activeIndex.value = Math.max(0, activeIndex.value - 1);
    } else if (e.key === 'Enter') {
        e.preventDefault();
        const opt = options.value[activeIndex.value];
        if (opt) pick(opt);
    } else if (e.key === 'Escape') {
        e.preventDefault();
        closePanel();
    }
}

onMounted(() => {
    document.addEventListener('click', onClickOutside);
});
onUnmounted(() => {
    document.removeEventListener('click', onClickOutside);
    if (debounceTimer) window.clearTimeout(debounceTimer);
});
</script>

<template>
    <div ref="wrapperRef" class="relative w-full">
        <button
            type="button"
            :class="[
                'flex h-9 w-full items-center justify-between gap-2 rounded-md border border-input bg-background/40 px-3 text-left text-sm transition-colors hover:bg-background/60 focus:outline-none focus:ring-1 focus:ring-ring',
                open && 'ring-1 ring-ring'
            ]"
            @click.stop="openPanel"
        >
            <span v-if="modelValue" class="flex min-w-0 flex-1 items-baseline gap-2">
                <span class="truncate text-foreground">{{ displayLabel }}</span>
                <span v-if="displayLabel !== modelValue" class="truncate font-mono text-xs text-muted-foreground">{{
                    modelValue
                }}</span>
            </span>
            <span v-else class="flex-1 truncate text-muted-foreground">{{ placeholder || '点击搜索' }}</span>
            <button
                v-if="modelValue"
                type="button"
                class="rounded p-0.5 text-muted-foreground hover:text-foreground"
                @click.stop="clear"
            >
                <svg
                    class="size-3.5"
                    xmlns="http://www.w3.org/2000/svg"
                    viewBox="0 0 24 24"
                    fill="none"
                    stroke="currentColor"
                    stroke-width="2"
                    stroke-linecap="round"
                    stroke-linejoin="round"
                >
                    <path d="M18 6 6 18" />
                    <path d="m6 6 12 12" />
                </svg>
            </button>
            <svg
                v-else
                :class="['size-3.5 text-muted-foreground transition-transform', open && 'rotate-180']"
                xmlns="http://www.w3.org/2000/svg"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="2"
                stroke-linecap="round"
                stroke-linejoin="round"
            >
                <path d="m6 9 6 6 6-6" />
            </svg>
        </button>

        <Transition
            enter-from-class="opacity-0 -translate-y-1"
            enter-active-class="transition duration-150"
            leave-active-class="transition duration-100"
            leave-to-class="opacity-0 -translate-y-1"
        >
            <div
                v-if="open"
                class="absolute left-0 right-0 top-full z-30 mt-1.5 overflow-hidden rounded-md border border-border/60 bg-background/95 shadow-lg backdrop-blur"
            >
                <div class="border-b border-border/40 p-1.5">
                    <input
                        ref="inputRef"
                        v-model="keyword"
                        :placeholder="placeholder || '搜索…'"
                        class="h-8 w-full rounded-sm bg-transparent px-2 text-sm focus:outline-none"
                        @keydown="onKeydown"
                    />
                </div>
                <div class="max-h-60 overflow-auto p-1">
                    <div v-if="loading" class="px-2 py-3 text-center text-xs text-muted-foreground">搜索中…</div>
                    <div
                        v-else-if="options.length === 0"
                        class="px-2 py-3 text-center text-xs text-muted-foreground"
                    >
                        无匹配项
                    </div>
                    <button
                        v-for="(opt, idx) in options"
                        :key="opt.value + idx"
                        type="button"
                        :class="[
                            'flex w-full flex-col items-start gap-0.5 rounded-md px-2.5 py-1.5 text-left text-sm transition-colors',
                            idx === activeIndex
                                ? 'bg-accent text-foreground'
                                : 'text-foreground/85 hover:bg-accent/60'
                        ]"
                        @mouseenter="activeIndex = idx"
                        @click="pick(opt)"
                    >
                        <span class="truncate">{{ opt.label }}</span>
                        <span
                            v-if="opt.sub && opt.sub !== opt.label"
                            class="truncate font-mono text-[11px] text-muted-foreground"
                            >{{ opt.sub }}</span
                        >
                    </button>
                </div>
            </div>
        </Transition>
    </div>
</template>
