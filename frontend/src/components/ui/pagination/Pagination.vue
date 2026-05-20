<script setup lang="ts">
import { computed } from 'vue'
import { ChevronLeft, ChevronRight } from 'lucide-vue-next'
import { Button } from '@/components/ui/button'

interface Props {
  total: number
  limit: number
  offset: number
  loading?: boolean
  limitOptions?: number[]
}

const props = withDefaults(defineProps<Props>(), {
  loading: false,
  limitOptions: () => [20, 50, 100, 200],
})

const emit = defineEmits<{
  'update:offset': [value: number]
  'update:limit': [value: number]
}>()

const currentPage = computed(() =>
  props.total <= 0 ? 0 : Math.floor(props.offset / props.limit) + 1
)

const totalPages = computed(() =>
  props.total <= 0 ? 0 : Math.ceil(props.total / props.limit)
)

const rangeText = computed(() => {
  if (props.total <= 0) return '0 条'
  const start = Math.min(props.offset + 1, props.total)
  const end = Math.min(props.offset + props.limit, props.total)
  return `${start}-${end} / ${props.total}`
})

const hasPrev = computed(() => props.offset > 0)
const hasNext = computed(() => props.offset + props.limit < props.total)

function prev() {
  if (!hasPrev.value) return
  emit('update:offset', Math.max(0, props.offset - props.limit))
}

function next() {
  if (!hasNext.value) return
  emit('update:offset', props.offset + props.limit)
}

function onLimitChange(e: Event) {
  const val = Number((e.target as HTMLSelectElement).value)
  if (val > 0) emit('update:limit', val)
}
</script>

<template>
  <div class="flex items-center gap-2 text-xs text-muted-foreground">
    <span>{{ rangeText }}</span>
    <select
      :value="limit"
      class="h-7 rounded border border-input bg-background/40 px-1.5 text-xs focus:outline-none focus:ring-1 focus:ring-ring"
      @change="onLimitChange"
    >
      <option v-for="opt in limitOptions" :key="opt" :value="opt">{{ opt }} / 页</option>
    </select>
    <div class="flex items-center gap-0.5">
      <Button
        variant="ghost"
        size="icon-sm"
        :disabled="loading || !hasPrev"
        @click="prev"
      >
        <ChevronLeft class="size-4" />
      </Button>
      <span class="min-w-[3ch] text-center tabular-nums">{{ currentPage }} / {{ totalPages }}</span>
      <Button
        variant="ghost"
        size="icon-sm"
        :disabled="loading || !hasNext"
        @click="next"
      >
        <ChevronRight class="size-4" />
      </Button>
    </div>
  </div>
</template>