<script setup lang="ts">
import { computed, inject, nextTick, onMounted, ref, watch } from 'vue'
import { backend, type Contact, type Instance } from '../wailsbridge'
import { appContextKey } from '../app/context'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'

const injected = inject(appContextKey)
if (!injected) throw new Error('chatlog not provided')
const app = injected

const { instances, run, state } = app

const contactKeyword = ref('')
const contactsLoading = ref(false)
const contactsTotal = ref(0)
const contacts = ref<Contact[]>([])
const contactLimit = ref(50)
const contactOffset = ref(0)
const contactLoadingSource = ref<'init' | 'refresh' | 'search' | 'prev' | 'next' | 'limit' | 'account' | null>(null)

const contactRangeText = computed(() => {
  if (contactsTotal.value <= 0) return '0 / 0'
  const start = Math.min(contactOffset.value + 1, contactsTotal.value)
  const end = Math.min(contactOffset.value + contacts.value.length, contactsTotal.value)
  return `${start}-${end} / ${contactsTotal.value}`
})

const hasContacts = computed(() => contacts.value.length > 0)
const hasKeyword = computed(() => contactKeyword.value.trim().length > 0)
const contactPageText = computed(() => {
  if (contactsTotal.value <= 0) return '第 0 页'
  return `第 ${Math.floor(contactOffset.value / contactLimit.value) + 1} 页`
})

const prevButtonText = computed(() => (contactsLoading.value && contactLoadingSource.value === 'prev' ? '加载中…' : '上一页'))
const nextButtonText = computed(() => (contactsLoading.value && contactLoadingSource.value === 'next' ? '加载中…' : '下一页'))

let contactLoadTimer: number | undefined

function getPageScrollContainer() {
  if (typeof document === 'undefined') return null
  const page = document.querySelector('.page')
  return page instanceof HTMLElement ? page : null
}

function restorePageScroll(scrollTop: number | null) {
  if (scrollTop === null) return
  const container = getPageScrollContainer()
  if (!container) return
  window.requestAnimationFrame(() => {
    container.scrollTop = scrollTop
  })
}

async function loadContacts(options?: {
  preserveScroll?: boolean
  source?: 'init' | 'refresh' | 'search' | 'prev' | 'next' | 'limit' | 'account'
}) {
  if (!backend.isWails) {
    contacts.value = []
    contactsTotal.value = 0
    return
  }
  contactLoadingSource.value = options?.source ?? 'refresh'
  const preservedScrollTop = options?.preserveScroll ? getPageScrollContainer()?.scrollTop ?? 0 : null
  contactsLoading.value = true
  try {
    const resp = await backend.GetContacts(
      contactKeyword.value.trim(),
      -1,
      contactLimit.value,
      contactOffset.value,
    )
    contactsTotal.value = resp.total || 0
    contacts.value = Array.isArray(resp.items) ? resp.items : []
  }
  catch (e) {
    app.feedback.toast('加载联系人失败', String(e))
  }
  finally {
    contactsLoading.value = false
    contactLoadingSource.value = null
    await nextTick()
    restorePageScroll(preservedScrollTop)
  }
}

function scheduleLoadContacts(delayMs = 200) {
  if (contactLoadTimer) window.clearTimeout(contactLoadTimer)
  contactLoadTimer = window.setTimeout(() => {
    void loadContacts({ preserveScroll: true, source: 'search' })
  }, delayMs)
}

function prevContactsPage() {
  contactOffset.value = Math.max(0, contactOffset.value - contactLimit.value)
  void loadContacts({ preserveScroll: true, source: 'prev' })
}

function nextContactsPage() {
  if (contactOffset.value + contactLimit.value >= contactsTotal.value) return
  contactOffset.value = contactOffset.value + contactLimit.value
  void loadContacts({ preserveScroll: true, source: 'next' })
}

function getAccountName(instance: Instance) {
  if (state.value?.pid === instance.pid && state.value.nickname) {
    return state.value.nickname
  }
  return instance.name || '未知账号'
}

function getAccountAvatar(instance: Instance) {
  if (state.value?.pid === instance.pid) {
    return state.value.smallHeadImgUrl || ''
  }
  return ''
}

function getAvatarFallback(instance: Instance) {
  const name = getAccountName(instance).trim()
  return name ? name.slice(0, 1).toUpperCase() : '?'
}

function switchTo(pid: number) {
  return run(() => backend.SwitchToPID(pid), '已切换账号')
}

function getContactName(c: Contact) {
  return c.remark || c.nickName || c.alias || c.userName || '未知联系人'
}

function getContactAvatar(c: Contact) {
  return c.smallHeadImgUrl || ''
}

function getContactAvatarFallback(c: Contact) {
  const name = getContactName(c).trim()
  return name ? name.slice(0, 1).toUpperCase() : '?'
}

onMounted(() => {
  void loadContacts({ source: 'init' })
})

watch(
  () => state.value?.account,
  () => {
    contactOffset.value = 0
    if (contactLoadTimer) window.clearTimeout(contactLoadTimer)
    void loadContacts({ preserveScroll: true, source: 'account' })
  },
)

watch(contactKeyword, () => {
  contactOffset.value = 0
  scheduleLoadContacts()
})

watch(contactLimit, () => {
  contactOffset.value = 0
  void loadContacts({ preserveScroll: true, source: 'limit' })
})
</script>

<template>
  <div class="space-y-8">
    <section class="space-y-6">
      <div class="flex items-center gap-4 border-b border-border/40 pb-4">
        <div class="flex size-8 items-center justify-center rounded-lg bg-primary/10 text-[10px] font-bold text-primary">01</div>
        <div class="text-xs font-bold uppercase tracking-[0.2em] text-foreground/70">账号进程 / PROCESSES</div>
      </div>

      <div class="grid gap-6 lg:grid-cols-2">
        <div v-if="instances.length === 0" class="col-span-full rounded-2xl border border-dashed border-border/40 bg-muted/5 py-12 text-center">
          <div class="text-sm font-semibold text-foreground/80">未探测到活跃微信进程</div>
          <div class="mt-1 text-xs text-muted-foreground">请启动并登录微信后点击页面顶部刷新</div>
        </div>

        <Card
          v-for="instance in instances"
          :key="instance.pid"
          class="group relative overflow-hidden border-border/40 bg-card/40 transition-all hover:border-primary/30 hover:bg-card/60"
        >
          <CardContent class="p-6">
            <div class="flex items-start justify-between">
              <div class="flex items-center gap-4">
                <div class="relative">
                  <img
                    v-if="getAccountAvatar(instance)"
                    :src="getAccountAvatar(instance)"
                    class="size-14 rounded-2xl border border-border/20 object-cover shadow-sm"
                  >
                  <div v-else class="flex size-14 items-center justify-center rounded-2xl border border-border/20 bg-muted/30 text-lg font-bold">
                    {{ getAvatarFallback(instance) }}
                  </div>
                  <div
                    class="absolute -right-1 -top-1 size-4 rounded-full border-2 border-background"
                    :class="instance.status === 'online' ? 'bg-emerald-500' : 'bg-rose-500'"
                  ></div>
                </div>
                <div class="space-y-1">
                  <div class="text-base font-bold tracking-tight">{{ getAccountName(instance) }}</div>
                  <div class="flex flex-wrap items-center gap-x-2 gap-y-1 font-mono text-[10px] text-muted-foreground">
                    <span class="rounded bg-muted/50 px-1.5 py-0.5">PID {{ instance.pid }}</span>
                    <span>v{{ instance.fullVersion || '-' }}</span>
                    <span class="opacity-40">/</span>
                    <span>{{ instance.platform || '-' }}</span>
                  </div>
                </div>
              </div>
              <Button variant="secondary" size="sm" class="h-8 px-3 text-xs" @click="switchTo(instance.pid)">
                切换账号
              </Button>
            </div>

            <div class="mt-6 space-y-2">
              <div class="text-[10px] font-bold uppercase tracking-widest text-muted-foreground/60">数据存储路径</div>
              <div class="rounded-xl border border-border/40 bg-muted/20 p-3">
                <div class="break-all font-mono text-[11px] leading-relaxed text-foreground/70">
                  {{ instance.dataDir || '未定义路径' }}
                </div>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>
    </section>

    <section class="space-y-6">
      <div class="flex items-center gap-4 border-b border-border/40 pb-4">
        <div class="flex size-8 items-center justify-center rounded-lg bg-primary/10 text-[10px] font-bold text-primary">02</div>
        <div class="text-xs font-bold uppercase tracking-[0.2em] text-foreground/70">联系人目录 / DIRECTORY</div>
      </div>

      <Card class="border-border/40 bg-card/40 shadow-none">
        <CardHeader class="sticky -top-px z-30 border-b border-border/40 bg-card/80 p-0 backdrop-blur-md rounded-t-xl">
          <div class="divide-y divide-border/40 ">
            <!-- 标题与摘要状态 -->
            <div class="flex flex-col gap-4 p-5 lg:flex-row lg:items-center lg:justify-between rounded-t-xl">
              <div class="space-y-1">
                <CardTitle class="text-lg font-bold tracking-tight">联系人筛选</CardTitle>
                <CardDescription class="text-xs">检索当前账号下的联系人、群聊及公众号数据</CardDescription>
              </div>
              <div class="flex flex-wrap items-center gap-2">
                <Badge variant="secondary" class="h-6 rounded-md bg-muted/50 font-mono text-[10px] font-medium text-muted-foreground uppercase">
                  RANGE: {{ contactRangeText }}
                </Badge>
                <Badge variant="secondary" class="h-6 rounded-md bg-muted/50 font-mono text-[10px] font-medium text-muted-foreground uppercase">
                  {{ contactPageText }}
                </Badge>
                <Badge v-if="hasKeyword" variant="secondary" class="h-6 gap-2 rounded-md bg-primary/12 font-mono text-[10px] uppercase text-primary">
                  <span class="opacity-60">FILTER:</span> {{ contactKeyword }}
                </Badge>
              </div>
            </div>

            <!-- 工具栏 -->
            <div class="grid grid-cols-1 gap-4 p-5 md:grid-cols-[1fr_auto_auto]">
              <div class="relative w-full">
                <Input v-model="contactKeyword" class="h-10 pl-4 pr-10 font-mono text-sm" placeholder="搜索昵称、微信号、备注或 ID..." />
                <div class="absolute right-3 top-1/2 -translate-y-1/2 opacity-30">
                  <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="11" cy="11" r="8"/><path d="m21 21-4.3-4.3"/></svg>
                </div>
              </div>
              <div class="flex items-center gap-3">
                <select
                  v-model.number="contactLimit"
                  class="h-10 w-[120px] rounded-md border border-input bg-background/50 px-3 text-xs font-medium focus:ring-1 focus:ring-primary outline-none"
                >
                  <option :value="20">20 每页</option>
                  <option :value="50">50 每页</option>
                  <option :value="100">100 每页</option>
                  <option :value="200">200 每页</option>
                </select>
                <Button variant="ghost" size="icon" :disabled="contactsLoading" class="h-10 w-10 shrink-0 border border-border/40" @click="loadContacts({ source: 'refresh' })">
                  <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" :class="contactsLoading ? 'animate-spin' : ''"><path d="M21 12a9 9 0 1 1-9-9c2.52 0 4.93 1 6.74 2.74L21 8"/><path d="M21 3v5h-5"/></svg>
                </Button>
              </div>
              <div class="flex items-center gap-2">
                <Button variant="outline" size="sm" class="h-10 px-4 text-xs font-bold uppercase transition-all" :disabled="contactsLoading || contactOffset === 0" @click="prevContactsPage">
                  {{ contactLoadingSource === 'prev' ? '...' : 'Prev' }}
                </Button>
                <Button size="sm" class="h-10 px-4 text-xs font-bold uppercase transition-all shadow-md shadow-primary/10" :disabled="contactsLoading || contactOffset + contactLimit >= contactsTotal" @click="nextContactsPage">
                  {{ contactLoadingSource === 'next' ? '...' : 'Next' }}
                </Button>
              </div>
            </div>
          </div>
        </CardHeader>

        <CardContent class="relative p-6">
          <div v-if="contactsLoading && hasContacts" class="pointer-events-none absolute inset-x-0 top-0 z-50 flex justify-center">
            <div class="mt-4 animate-pulse rounded-full bg-primary/90 px-4 py-1.5 text-[10px] font-bold uppercase tracking-widest text-primary-foreground shadow-lg">同步中 (Syncing...)</div>
          </div>

          <div v-if="contactsLoading && !hasContacts" class="py-20 text-center">
            <div class="mx-auto flex size-12 animate-spin items-center justify-center rounded-full border-2 border-primary border-t-transparent opacity-40"></div>
            <div class="mt-6 text-sm font-bold tracking-tight text-foreground/80">正在检索联系人数据库</div>
            <div class="mt-1 text-xs text-muted-foreground">请稍候，我们正在解析当前绑定的微信进程数据</div>
          </div>

          <div v-else-if="!hasContacts" class="py-20 text-center">
            <div class="text-sm font-bold tracking-tight text-foreground/80">未找到匹配的结果</div>
            <div class="mt-1 text-xs text-muted-foreground">尝试调整关键词或是重置筛选器，也可以检查当前登录账号</div>
          </div>

          <div v-else class="grid gap-4 md:grid-cols-2 xl:grid-cols-3" :class="contactsLoading ? 'opacity-40 animate-pulse transition-opacity' : ''">
            <Card
              v-for="contact in contacts"
              :key="contact.userName"
              class="group overflow-hidden border-border/30 bg-background/40 transition-shadow hover:shadow-md"
            >
              <CardContent class="p-4">
                <div class="flex items-center gap-4">
                  <img
                    v-if="getContactAvatar(contact)"
                    :src="getContactAvatar(contact)"
                    class="size-10 rounded-xl border border-border/20 object-cover shadow-xs"
                  >
                  <div v-else class="flex size-10 items-center justify-center rounded-xl border border-border/20 bg-muted/40 text-sm font-bold opacity-60">
                    {{ getContactAvatarFallback(contact) }}
                  </div>

                  <div class="min-w-0 flex-1 space-y-0.5">
                    <div class="truncate text-sm font-bold tracking-tight text-foreground/90">{{ getContactName(contact) }}</div>
                    <div class="truncate font-mono text-[10px] text-muted-foreground/70">
                      {{ contact.alias || contact.userName }}
                    </div>
                  </div>
                </div>

                <div class="mt-4 flex flex-wrap items-center gap-2">
                  <Badge variant="outline" class="h-5 border-border/40 bg-muted/20 px-1.5 font-mono text-[9px] font-normal text-muted-foreground uppercase">
                    ID: {{ contact.localType }}
                  </Badge>
                  <Badge variant="outline" class="h-5 border-border/40 bg-muted/20 px-1.5 font-mono text-[9px] font-normal text-muted-foreground uppercase">
                    FLAG: {{ contact.flag }}
                  </Badge>
                  <Badge v-if="contact.isInChatRoom" variant="outline" class="h-5 border-primary/20 bg-primary/5 px-1.5 font-mono text-[9px] font-bold text-primary uppercase">
                    ROOM
                  </Badge>
                </div>
              </CardContent>
            </Card>
          </div>
        </CardContent>
      </Card>
    </section>
  </div>
</template>
