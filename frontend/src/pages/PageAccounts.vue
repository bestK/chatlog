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
  if (contactsTotal.value <= 0) return '0'
  const start = Math.min(contactOffset.value + 1, contactsTotal.value)
  const end = Math.min(contactOffset.value + contacts.value.length, contactsTotal.value)
  return `${start}-${end}/${contactsTotal.value}`
})

const hasContacts = computed(() => contacts.value.length > 0)

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
    <section class="space-y-4">
      <div class="border-b border-border/60 pb-3">
        <div class="text-xs font-semibold uppercase tracking-[0.16em] text-muted-foreground">01 · WeChat Processes</div>
      </div>

      <Card class="border-border/60 bg-card/70 shadow-sm">
        <CardHeader>
          <div class="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
            <div class="space-y-2">
              <CardTitle class="text-base">账号进程</CardTitle>
              <CardDescription>Check status before switching accounts.</CardDescription>
            </div>
            <Badge variant="outline">Detected: {{ instances.length }}</Badge>
          </div>
        </CardHeader>
        <CardContent>
          <div v-if="instances.length === 0" class="rounded-xl border border-dashed border-border/60 bg-background/20 p-8 text-center">
            <div class="text-sm font-medium text-foreground">No WeChat Process Detected</div>
            <div class="mt-2 text-sm text-muted-foreground">Please start and login to WeChat first.</div>
          </div>

          <div v-else class="grid gap-4 xl:grid-cols-2">
            <Card
              v-for="instance in instances"
              :key="instance.pid"
              class="border-border/60 bg-background/20 shadow-none"
            >
              <CardContent class="space-y-4 pt-6">
                <div class="flex items-start justify-between gap-4">
                  <div class="flex min-w-0 items-center gap-3">
                    <img
                      v-if="getAccountAvatar(instance)"
                      :src="getAccountAvatar(instance)"
                      :alt="`${getAccountName(instance)} 头像`"
                      class="size-11 rounded-xl border border-border/60 object-cover"
                    >
                    <div
                      v-else
                      class="flex size-11 items-center justify-center rounded-xl border border-border/60 bg-muted/20 text-sm font-semibold text-foreground"
                    >
                      {{ getAvatarFallback(instance) }}
                    </div>

                    <div class="min-w-0 space-y-1">
                      <div class="truncate text-sm font-medium text-foreground">{{ getAccountName(instance) }}</div>
                      <div class="truncate font-mono text-xs text-muted-foreground">
                        PID {{ instance.pid }} · v{{ instance.fullVersion || '-' }} · {{ instance.platform || '-' }}
                      </div>
                    </div>
                  </div>

                  <Badge
                    variant="outline"
                    :class="instance.status === 'online'
                      ? 'border-emerald-500/30 bg-emerald-500/10 text-emerald-300'
                      : 'border-rose-500/30 bg-rose-500/10 text-rose-300'"
                  >
                    {{ instance.status === 'online' ? 'Online' : instance.status === 'offline' ? 'Offline' : instance.status || 'Unknown' }}
                  </Badge>
                </div>

                <div class="rounded-xl border border-border/60 bg-background/30 p-4">
                  <div class="mb-2 text-[11px] font-semibold uppercase tracking-[0.14em] text-muted-foreground">Data Directory</div>
                  <div class="font-mono text-xs text-foreground break-all">{{ instance.dataDir || '-' }}</div>
                </div>

                <Button @click="switchTo(instance.pid)">Switch</Button>
              </CardContent>
            </Card>
          </div>
        </CardContent>
      </Card>
    </section>

    <section class="space-y-4">
      <div class="border-b border-border/60 pb-3">
        <div class="text-xs font-semibold uppercase tracking-[0.16em] text-muted-foreground">02 · Contacts</div>
      </div>

      <Card class="border-border/60 bg-card/70 shadow-sm overflow-hidden">
        <CardHeader class="sticky top-0 z-20 gap-4 border-b border-border/60 bg-[color:color-mix(in_oklab,var(--card)_88%,black_12%)]/95 pb-4 backdrop-blur supports-[backdrop-filter]:bg-[color:color-mix(in_oklab,var(--card)_82%,black_18%)]/85">
          <div class="flex flex-col gap-4 xl:flex-row xl:items-end xl:justify-between">
            <div class="space-y-3">
              <div>
                <CardTitle class="text-base">联系人筛选</CardTitle>
                <CardDescription>搜索昵称、备注或微信 ID，并快速调整分页大小。</CardDescription>
              </div>
              <div class="flex flex-wrap gap-2">
                <Badge variant="outline">范围: {{ contactRangeText }}</Badge>
                <Badge variant="outline">关键字: {{ contactKeyword.trim() ? '已过滤' : '全部' }}</Badge>
              </div>
            </div>

            <div class="flex w-full flex-col gap-2 xl:max-w-2xl xl:flex-row xl:items-center">
              <Input v-model="contactKeyword" class="font-mono xl:flex-1" placeholder="搜索昵称 / 备注 / ID" />
              <select
                v-model.number="contactLimit"
                class="flex h-9 rounded-md border border-input bg-transparent px-3 py-2 text-sm shadow-xs outline-none transition-[color,box-shadow] focus-visible:border-ring focus-visible:ring-ring/50 focus-visible:ring-[3px]"
                title="每页数量"
              >
                <option :value="20">20/页</option>
                <option :value="50">50/页</option>
                <option :value="100">100/页</option>
                <option :value="200">200/页</option>
              </select>
              <div class="flex flex-wrap gap-2">
                <Button variant="outline" :disabled="contactsLoading" @click="loadContacts({ source: 'refresh' })">刷新</Button>
                <Button variant="outline" :disabled="contactsLoading || contactOffset === 0" @click="prevContactsPage">
                  {{ prevButtonText }}
                </Button>
                <Button :disabled="contactsLoading || contactOffset + contactLimit >= contactsTotal" @click="nextContactsPage">
                  {{ nextButtonText }}
                </Button>
              </div>
            </div>
          </div>
        </CardHeader>

        <CardContent class="relative pt-6">
          <div v-if="contactsLoading && hasContacts" class="pointer-events-none absolute inset-x-6 top-6 z-10 flex justify-center">
            <Badge class="shadow-sm">正在更新联系人…</Badge>
          </div>

          <div v-if="contactsLoading && !hasContacts" class="rounded-xl border border-dashed border-border/60 bg-background/20 p-8 text-center">
            <div class="text-sm font-medium text-foreground">正在加载联系人…</div>
            <div class="mt-2 text-sm text-muted-foreground">请稍候</div>
          </div>

          <div v-else-if="!hasContacts" class="rounded-xl border border-dashed border-border/60 bg-background/20 p-8 text-center">
            <div class="text-sm font-medium text-foreground">暂无联系人</div>
            <div class="mt-2 text-sm text-muted-foreground">可尝试先解密数据或切换账号后刷新</div>
          </div>

          <div v-else class="grid gap-4 lg:grid-cols-2" :class="contactsLoading ? 'opacity-70' : ''">
            <Card
              v-for="contact in contacts"
              :key="contact.userName"
              class="border-border/60 bg-background/20 shadow-none"
            >
              <CardContent class="space-y-4 pt-6">
                <div class="flex items-start gap-3">
                  <img
                    v-if="getContactAvatar(contact)"
                    :src="getContactAvatar(contact)"
                    :alt="`${getContactName(contact)} 头像`"
                    class="size-10 rounded-xl border border-border/60 object-cover"
                  >
                  <div
                    v-else
                    class="flex size-10 items-center justify-center rounded-xl border border-border/60 bg-muted/20 text-sm font-semibold text-foreground"
                  >
                    {{ getContactAvatarFallback(contact) }}
                  </div>

                  <div class="min-w-0 flex-1 space-y-1">
                    <div class="truncate text-sm font-medium text-foreground">{{ getContactName(contact) }}</div>
                    <div class="truncate font-mono text-xs text-muted-foreground">
                      {{ contact.userName }}<span v-if="contact.alias"> · {{ contact.alias }}</span>
                    </div>
                  </div>
                </div>

                <div class="flex flex-wrap gap-2 text-xs text-muted-foreground">
                  <Badge variant="outline">localType: {{ contact.localType }}</Badge>
                  <Badge variant="outline">flag: {{ contact.flag }}</Badge>
                  <Badge variant="outline">is_in_chat_room: {{ contact.isInChatRoom }}</Badge>
                </div>
              </CardContent>
            </Card>
          </div>
        </CardContent>
      </Card>
    </section>
  </div>
</template>
