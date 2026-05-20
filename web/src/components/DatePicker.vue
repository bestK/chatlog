<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue';

const props = defineProps<{
    modelValue: string;
    placeholder?: string;
    min?: string;
    max?: string;
}>();

const emit = defineEmits<{
    (e: 'update:modelValue', value: string): void;
    (e: 'rangePreset', value: '7d' | '30d' | '180d'): void;
}>();

const wrapperRef = ref<HTMLElement | null>(null);
const open = ref(false);

const today = new Date();
const viewYear = ref(today.getFullYear());
const viewMonth = ref(today.getMonth()); // 0-11

const WEEKDAYS = ['日', '一', '二', '三', '四', '五', '六'];
const MONTH_NAMES = ['1 月', '2 月', '3 月', '4 月', '5 月', '6 月', '7 月', '8 月', '9 月', '10 月', '11 月', '12 月'];

function pad(n: number): string {
    return n < 10 ? `0${n}` : String(n);
}

function toIso(year: number, month: number, day: number): string {
    return `${year}-${pad(month + 1)}-${pad(day)}`;
}

function parseIso(value: string): { year: number; month: number; day: number } | null {
    if (!value) return null;
    const m = value.match(/^(\d{4})-(\d{2})-(\d{2})$/);
    if (!m) return null;
    const year = Number(m[1]);
    const month = Number(m[2]) - 1;
    const day = Number(m[3]);
    if (isNaN(year) || isNaN(month) || isNaN(day)) return null;
    return { year, month, day };
}

const selected = computed(() => parseIso(props.modelValue));

const displayLabel = computed(() => {
    const s = selected.value;
    if (!s) return '';
    return `${s.year} 年 ${s.month + 1} 月 ${s.day} 日`;
});

const cells = computed(() => {
    const firstDay = new Date(viewYear.value, viewMonth.value, 1);
    const startWeekday = firstDay.getDay();
    const daysInMonth = new Date(viewYear.value, viewMonth.value + 1, 0).getDate();
    const prevDays = new Date(viewYear.value, viewMonth.value, 0).getDate();

    type Cell = { day: number; iso: string; muted: boolean; today: boolean; selected: boolean; disabled: boolean };
    const out: Cell[] = [];

    const minDate = props.min ? parseIso(props.min) : null;
    const maxDate = props.max ? parseIso(props.max) : null;

    function makeCell(year: number, month: number, day: number, muted: boolean): Cell {
        const iso = toIso(year, month, day);
        const isToday = day === today.getDate() && month === today.getMonth() && year === today.getFullYear();
        const isSelected = selected.value
            ? selected.value.year === year && selected.value.month === month && selected.value.day === day
            : false;
        let disabled = false;
        if (minDate) {
            const cmp = `${year}-${pad(month + 1)}-${pad(day)}`;
            if (cmp < `${minDate.year}-${pad(minDate.month + 1)}-${pad(minDate.day)}`) disabled = true;
        }
        if (maxDate) {
            const cmp = `${year}-${pad(month + 1)}-${pad(day)}`;
            if (cmp > `${maxDate.year}-${pad(maxDate.month + 1)}-${pad(maxDate.day)}`) disabled = true;
        }
        return { day, iso, muted, today: isToday, selected: isSelected, disabled };
    }

    // leading days from prev month
    for (let i = startWeekday - 1; i >= 0; i--) {
        const day = prevDays - i;
        const m = viewMonth.value - 1;
        const year = m < 0 ? viewYear.value - 1 : viewYear.value;
        const month = (m + 12) % 12;
        out.push(makeCell(year, month, day, true));
    }
    // current month
    for (let d = 1; d <= daysInMonth; d++) {
        out.push(makeCell(viewYear.value, viewMonth.value, d, false));
    }
    // trailing to fill grid (6 rows × 7 cols = 42)
    let trailing = 1;
    while (out.length < 42) {
        const m = viewMonth.value + 1;
        const year = m > 11 ? viewYear.value + 1 : viewYear.value;
        const month = m % 12;
        out.push(makeCell(year, month, trailing, true));
        trailing++;
    }
    return out;
});

function syncViewToValue() {
    const s = selected.value;
    if (s) {
        viewYear.value = s.year;
        viewMonth.value = s.month;
    } else {
        viewYear.value = today.getFullYear();
        viewMonth.value = today.getMonth();
    }
}

function openPanel() {
    if (open.value) return;
    syncViewToValue();
    open.value = true;
}

function closePanel() {
    open.value = false;
}

function prevMonth() {
    if (viewMonth.value === 0) {
        viewMonth.value = 11;
        viewYear.value -= 1;
    } else {
        viewMonth.value -= 1;
    }
}

function nextMonth() {
    if (viewMonth.value === 11) {
        viewMonth.value = 0;
        viewYear.value += 1;
    } else {
        viewMonth.value += 1;
    }
}

function pick(iso: string, disabled: boolean) {
    if (disabled) return;
    emit('update:modelValue', iso);
    closePanel();
}

function selectToday() {
    const iso = toIso(today.getFullYear(), today.getMonth(), today.getDate());
    emit('update:modelValue', iso);
    closePanel();
}

function clearValue(e?: MouseEvent) {
    e?.stopPropagation();
    emit('update:modelValue', '');
}

function selectRangePreset(value: '7d' | '30d' | '180d') {
    emit('rangePreset', value);
    closePanel();
}

function onClickOutside(e: MouseEvent) {
    if (!open.value) return;
    if (wrapperRef.value && !wrapperRef.value.contains(e.target as Node)) {
        closePanel();
    }
}

watch(
    () => props.modelValue,
    () => {
        if (open.value) syncViewToValue();
    }
);

onMounted(() => document.addEventListener('click', onClickOutside));
onUnmounted(() => document.removeEventListener('click', onClickOutside));
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
            <span v-if="modelValue" class="truncate text-foreground">{{ displayLabel }}</span>
            <span v-else class="flex-1 truncate text-muted-foreground">{{ placeholder || '选择日期' }}</span>
            <span class="flex shrink-0 items-center gap-1">
                <button
                    v-if="modelValue"
                    type="button"
                    class="rounded p-0.5 text-muted-foreground hover:text-foreground"
                    @click.stop="clearValue"
                    title="清除"
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
                    class="size-3.5 text-muted-foreground"
                    xmlns="http://www.w3.org/2000/svg"
                    viewBox="0 0 24 24"
                    fill="none"
                    stroke="currentColor"
                    stroke-width="2"
                    stroke-linecap="round"
                    stroke-linejoin="round"
                >
                    <rect x="3" y="4" width="18" height="18" rx="2" ry="2" />
                    <line x1="16" y1="2" x2="16" y2="6" />
                    <line x1="8" y1="2" x2="8" y2="6" />
                    <line x1="3" y1="10" x2="21" y2="10" />
                </svg>
            </span>
        </button>

        <Transition
            enter-from-class="opacity-0 -translate-y-1"
            enter-active-class="transition duration-150"
            leave-active-class="transition duration-100"
            leave-to-class="opacity-0 -translate-y-1"
        >
            <div
                v-if="open"
                class="absolute left-0 z-30 mt-1.5 w-72 overflow-hidden rounded-md border border-border/60 bg-background/95 shadow-lg backdrop-blur"
            >
                <header class="flex items-center justify-between border-b border-border/40 px-2 py-1.5">
                    <button
                        type="button"
                        class="flex size-7 items-center justify-center rounded-md text-muted-foreground hover:bg-accent/60 hover:text-foreground"
                        @click="prevMonth"
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
                            <path d="m15 18-6-6 6-6" />
                        </svg>
                    </button>
                    <span class="text-sm font-medium text-foreground">
                        {{ viewYear }} 年 {{ MONTH_NAMES[viewMonth] }}
                    </span>
                    <button
                        type="button"
                        class="flex size-7 items-center justify-center rounded-md text-muted-foreground hover:bg-accent/60 hover:text-foreground"
                        @click="nextMonth"
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
                            <path d="m9 18 6-6-6-6" />
                        </svg>
                    </button>
                </header>
                <div class="grid grid-cols-7 gap-0.5 px-2 pt-2 pb-1">
                    <div
                        v-for="w in WEEKDAYS"
                        :key="w"
                        class="flex h-6 items-center justify-center text-[11px] text-muted-foreground/70"
                    >
                        {{ w }}
                    </div>
                </div>
                <div class="grid grid-cols-7 gap-0.5 px-2 pb-2">
                    <button
                        v-for="(c, idx) in cells"
                        :key="idx"
                        type="button"
                        :disabled="c.disabled"
                        :class="[
                            'flex h-7 items-center justify-center rounded-md text-xs transition-colors',
                            c.disabled && 'cursor-not-allowed text-muted-foreground/30',
                            !c.disabled && c.selected && 'bg-foreground text-background hover:bg-foreground/90',
                            !c.disabled && !c.selected && c.today && 'bg-accent/60 text-foreground',
                            !c.disabled &&
                                !c.selected &&
                                !c.today &&
                                c.muted &&
                                'text-muted-foreground/50 hover:bg-accent/40',
                            !c.disabled &&
                                !c.selected &&
                                !c.today &&
                                !c.muted &&
                                'text-foreground/85 hover:bg-accent/60'
                        ]"
                        @click="pick(c.iso, c.disabled)"
                    >
                        {{ c.day }}
                    </button>
                </div>
                <footer class="flex items-center justify-between border-t border-border/40 px-2 py-1.5">
                    <button
                        type="button"
                        class="rounded-md px-2 py-1 text-xs text-muted-foreground hover:text-foreground"
                        @click="clearValue()"
                    >
                        清除
                    </button>
                    <div class="flex items-center gap-1">
                        <button
                            type="button"
                            class="rounded-md px-2 py-1 text-xs text-muted-foreground hover:bg-accent/60 hover:text-foreground"
                            @click="selectRangePreset('7d')"
                        >
                            近 7 天
                        </button>
                        <button
                            type="button"
                            class="rounded-md px-2 py-1 text-xs text-muted-foreground hover:bg-accent/60 hover:text-foreground"
                            @click="selectRangePreset('30d')"
                        >
                            近 30 天
                        </button>
                        <button
                            type="button"
                            class="rounded-md px-2 py-1 text-xs text-muted-foreground hover:bg-accent/60 hover:text-foreground"
                            @click="selectRangePreset('180d')"
                        >
                            近半年
                        </button>
                        <button
                            type="button"
                            class="rounded-md px-2 py-1 text-xs text-muted-foreground hover:text-foreground"
                            @click="selectToday"
                        >
                            今天
                        </button>
                    </div>
                </footer>
            </div>
        </Transition>
    </div>
</template>
