import { computed, onMounted, ref } from 'vue'
import { backend, type Instance, type State } from '../wailsbridge'
import type { createFeedbackService } from './feedback'

export type Page = '概览' | '账号' | '服务' | 'Webhook' | '设置' | '日志'
export type StatusPill = { cls: string; text: string }

function isState(value: unknown): value is State {
  if (!value || typeof value !== 'object') return false
  const record = value as Record<string, unknown>
  return (
    typeof record.account === 'string'
    && typeof record.platform === 'string'
    && typeof record.fullVersion === 'string'
    && typeof record.dataDir === 'string'
    && typeof record.dataKey === 'string'
    && typeof record.imgKey === 'string'
    && typeof record.workDir === 'string'
    && typeof record.httpEnabled === 'boolean'
    && typeof record.httpAddr === 'string'
    && typeof record.autoDecrypt === 'boolean'
    && typeof record.lastSession === 'string'
    && typeof record.pid === 'number'
    && typeof record.exePath === 'string'
    && typeof record.status === 'string'
    && typeof record.smallHeadImgUrl === 'string'
  )
}

export function maskKey(value: string) {
  if (!value) return ''
  if (value.length <= 10) return value
  return `${value.slice(0, 4)}***${value.slice(-4)}`
}

type AppFeedback = Pick<ReturnType<typeof createFeedbackService>, 'toast'>

export function createAppState(feedback: AppFeedback) {
  const page = ref<Page>('概览')
  const state = ref<State | null>(null)
  const onboardingStep = ref(0) // 0: closed, 1: get key, 2: decrypt, 3: start http

  const instances = ref<Instance[]>([])
  const httpAddr = ref('')
  const workDir = ref('')
  const dataDir = ref('')
  const dataKey = ref('')
  const imgKey = ref('')

  const nav: Array<{ name: Page; hint: string }> = [
    { name: '概览', hint: '快捷操作' },
    { name: '账号', hint: '进程/历史' },
    { name: '服务', hint: 'HTTP/MCP' },
    { name: 'Webhook', hint: '回调' },
    { name: '设置', hint: '路径/参数' },
    { name: '日志', hint: '诊断' },
  ]

  const statusPill = computed<StatusPill>(() => {
    const current = state.value
    if (!current) return { cls: 'pill', text: '未连接' }
    if (current.status === 'online') return { cls: 'pill pillOk', text: '在线' }
    if (current.status === 'offline') return { cls: 'pill pillBad', text: '离线' }
    return { cls: 'pill', text: current.status || '未知' }
  })

  const previewBanner = computed(() => (backend.isWails ? '' : '浏览器预览模式：后端能力不可用'))

  async function refreshAll() {
    try {
      const nextState = await backend.GetState()
      state.value = nextState
      httpAddr.value = nextState.httpAddr || ''
      workDir.value = nextState.workDir || ''
      dataDir.value = nextState.dataDir || ''
      dataKey.value = nextState.dataKey || ''
      imgKey.value = nextState.imgKey || ''
      instances.value = await backend.ListInstances()

      // Auto-start onboarding if no data key and not already onboarding
      if (!nextState.dataKey && onboardingStep.value === 0) {
        onboardingStep.value = 1
      }
    }
    catch (error) {
      feedback.toast('刷新失败', String(error))
    }
  }

  async function run(action: () => Promise<unknown>, okMessage: string) {
    try {
      await action()
      feedback.toast('完成', okMessage)
      await refreshAll()
    }
    catch (error) {
      feedback.toast('操作失败', String(error))
    }
  }

  onMounted(async () => {
    if (backend.isWails) {
      try {
        await backend.EnableStateEvents(true)
      }
      catch {
      }
    }

    await refreshAll()

    const off = backend.EventsOn('state', (payload) => {
      if (!isState(payload)) return
      state.value = payload
      httpAddr.value = payload.httpAddr || ''
      workDir.value = payload.workDir || ''
      dataDir.value = payload.dataDir || ''
      dataKey.value = payload.dataKey || ''
      imgKey.value = payload.imgKey || ''
    })

    if (off) {
      window.addEventListener('beforeunload', () => {
        off()
        if (backend.isWails) {
          void backend.EnableStateEvents(false)
        }
      }, { once: true })
    }
  })

  return {
    page,
    nav,
    state,
    instances,
    httpAddr,
    workDir,
    dataDir,
    dataKey,
    imgKey,
    statusPill,
    previewBanner,
    onboardingStep,
    run,
    refreshAll,
  }
}
