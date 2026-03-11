<script setup lang="ts">
import { inject, onBeforeUnmount, onMounted, ref, type Ref } from 'vue'
import { backend, type KeyProgressEvent } from '../wailsbridge'
import { appContextKey } from '../app/context'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'

const injected = inject(appContextKey)
if (!injected) throw new Error('chatlog not provided')
const app = injected

const { dataDir, workDir, dataKey, imgKey, run, feedback } = app

type ActionDialogOptions<T> = {
  loadingRef: Ref<boolean>
  loadingTitle: string
  loadingMessage: string
  successTitle: string
  successMessage: string
  failureTitle: string
  action: () => Promise<T>
  detail?: (result: T) => string
}

const loadingDataKey = ref(false)
const loadingImgKey = ref(false)
const loadingDecrypt = ref(false)
let offKeyProgress: (() => void) | undefined

function isKeyProgressEvent(payload: unknown): payload is KeyProgressEvent {
  if (!payload || typeof payload !== 'object') return false
  const record = payload as Record<string, unknown>
  return typeof record.operation === 'string' && typeof record.message === 'string'
}

function maskSecret(secret: string) {
  if (!secret) return '-'
  if (secret.length <= 10) return secret
  return `${secret.slice(0, 4)}***${secret.slice(-4)}`
}

function formatError(error: unknown) {
  if (error instanceof Error) return error.message
  if (typeof error === 'string') return error
  try {
    return JSON.stringify(error)
  }
  catch {
    return String(error)
  }
}

async function runWithDialog<T>(options: ActionDialogOptions<T>) {
  if (options.loadingRef.value) return
  options.loadingRef.value = true
  feedback.status.openLoading(options.loadingTitle, options.loadingMessage)
  try {
    const result = await options.action()
    await app.refreshAll()
    feedback.status.openResult(
      'success',
      options.successTitle,
      options.successMessage,
      options.detail ? options.detail(result) : '',
    )
  }
  catch (e) {
    feedback.status.openResult('error', options.failureTitle, formatError(e))
  }
  finally {
    options.loadingRef.value = false
  }
}

onMounted(() => {
  offKeyProgress = backend.EventsOn('key:progress', (payload) => {
    if (!isKeyProgressEvent(payload)) return
    if (payload.operation !== 'dataKey') return
    if (!loadingDataKey.value) return
    feedback.status.update(payload.message)
  })
})

onBeforeUnmount(() => {
  if (offKeyProgress) {
    offKeyProgress()
    offKeyProgress = undefined
  }
})

function saveDataDir() {
  return app
    .feedback.confirm({
      title: '保存数据目录',
      message: '确认保存并写入配置？',
      confirmText: '保存',
      cancelText: '取消',
    })
    .then(ok => (ok ? run(() => backend.SetDataDir(dataDir.value), '已保存数据目录') : undefined))
}

function saveWorkDir() {
  return app
    .feedback.confirm({
      title: '保存工作目录',
      message: '确认保存并写入配置？',
      confirmText: '保存',
      cancelText: '取消',
    })
    .then(ok => (ok ? run(() => backend.SetWorkDir(workDir.value), '已保存工作目录') : undefined))
}

function saveDataKey() {
  return app
    .feedback.confirm({
      title: '保存数据库密钥',
      message: '确认保存并写入配置？',
      confirmText: '保存',
      cancelText: '取消',
    })
    .then(ok => (ok ? run(() => backend.SetDataKey(dataKey.value), '已保存数据库密钥') : undefined))
}

function autoDataKey() {
  return runWithDialog<string>({
    loadingRef: loadingDataKey,
    loadingTitle: '正在获取数据库密钥',
    loadingMessage: '正在准备获取数据库密钥，请稍候…',
    successTitle: '数据库密钥获取成功',
    successMessage: '已完成读取并同步到当前页面。',
    failureTitle: '数据库密钥获取失败',
    action: () => backend.GetDataKey(),
    detail: key => `密钥预览：${maskSecret(key)}`,
  })
}

function saveImgKey() {
  return app
    .feedback.confirm({
      title: '保存图片密钥',
      message: '确认保存并写入配置？',
      confirmText: '保存',
      cancelText: '取消',
    })
    .then(ok => (ok ? run(() => backend.SetImgKey(imgKey.value), '已保存图片密钥') : undefined))
}

function autoImgKey() {
  return runWithDialog<string>({
    loadingRef: loadingImgKey,
    loadingTitle: '正在获取图片密钥',
    loadingMessage: '请保持微信处于运行与登录状态，正在读取当前账号图片密钥。',
    successTitle: '图片密钥获取成功',
    successMessage: '已完成读取并同步到当前页面。',
    failureTitle: '图片密钥获取失败',
    action: () => backend.GetImgKey(),
    detail: key => `密钥预览：${maskSecret(key)}`,
  })
}

async function decryptNow() {
  const ok = await app.feedback.confirm({
    title: '开始解密',
    message: '确认开始解密数据库到工作目录？',
    confirmText: '开始',
    cancelText: '取消',
  })
  if (!ok) return
  return runWithDialog<void>({
    loadingRef: loadingDecrypt,
    loadingTitle: '正在解密数据库',
    loadingMessage: '正在解密并写入工作目录，过程可能持续一段时间，请勿关闭程序。',
    successTitle: '解密完成',
    successMessage: '数据库已成功解密，可前往服务页或日志页继续操作。',
    failureTitle: '解密失败',
    action: () => backend.Decrypt(),
    detail: () => `工作目录：${workDir.value || '-'}`,
  })
}
</script>

<template>
  <div class="space-y-8">
    <section class="space-y-4">
      <div class="border-b border-border/60 pb-3">
        <div class="text-xs font-semibold uppercase tracking-[0.16em] text-muted-foreground">01 · Global Settings</div>
      </div>

      <Card class="border-border/60 bg-card/70 shadow-sm">
        <CardContent class="flex flex-wrap items-center gap-3 pt-6">
          <Badge variant="outline">1. Directories</Badge>
          <Badge variant="outline">2. Secret Keys</Badge>
          <Badge variant="outline">3. Execution</Badge>
          <span class="text-sm text-muted-foreground">Save each step before proceeding.</span>
        </CardContent>
      </Card>

      <div class="grid gap-6 xl:grid-cols-2">
        <Card class="border-border/60 bg-card/70 shadow-sm">
          <CardHeader>
            <CardTitle class="text-base">Directory Config</CardTitle>
            <CardDescription>Paths for reading data and writing decrypted files.</CardDescription>
          </CardHeader>
          <CardContent class="space-y-5">
            <div class="space-y-2">
              <div class="text-[11px] font-semibold uppercase tracking-[0.14em] text-muted-foreground">Data Directory</div>
              <div class="flex flex-col gap-2 lg:flex-row">
                <Input v-model="dataDir" class="font-mono" />
                <Button variant="outline" @click="saveDataDir">Save</Button>
              </div>
            </div>

            <div class="space-y-2">
              <div class="text-[11px] font-semibold uppercase tracking-[0.14em] text-muted-foreground">Work Directory</div>
              <div class="flex flex-col gap-2 lg:flex-row">
                <Input v-model="workDir" class="font-mono" />
                <Button variant="outline" @click="saveWorkDir">Save</Button>
              </div>
            </div>
          </CardContent>
        </Card>

        <Card class="border-border/60 bg-card/70 shadow-sm">
          <CardHeader>
            <CardTitle class="text-base">Security Keys</CardTitle>
            <CardDescription>Manual entry or auto-fetch from WeChat.</CardDescription>
          </CardHeader>
          <CardContent class="space-y-5">
            <div class="space-y-2">
              <div class="text-[11px] font-semibold uppercase tracking-[0.14em] text-muted-foreground">Database Key</div>
              <div class="flex flex-col gap-2">
                <Input v-model="dataKey" class="font-mono" placeholder="Hex string" />
                <div class="flex flex-wrap gap-2">
                  <Button variant="outline" @click="saveDataKey">Save</Button>
                  <Button :disabled="loadingDataKey || loadingImgKey || loadingDecrypt" @click="autoDataKey">
                    {{ loadingDataKey ? 'Fetching...' : 'Auto' }}
                  </Button>
                </div>
              </div>
            </div>

            <div class="space-y-2">
              <div class="text-[11px] font-semibold uppercase tracking-[0.14em] text-muted-foreground">Image Key</div>
              <div class="flex flex-col gap-2">
                <Input v-model="imgKey" class="font-mono" placeholder="Hex string" />
                <div class="flex flex-wrap gap-2">
                  <Button variant="outline" @click="saveImgKey">Save</Button>
                  <Button :disabled="loadingDataKey || loadingImgKey || loadingDecrypt" @click="autoImgKey">
                    {{ loadingImgKey ? 'Fetching...' : 'Auto' }}
                  </Button>
                </div>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>
    </section>

    <section class="space-y-4">
      <div class="border-b border-border/60 pb-3">
        <div class="text-xs font-semibold uppercase tracking-[0.16em] text-muted-foreground">02 · Maintenance</div>
      </div>

      <Card class="border-border/60 bg-card/70 shadow-sm">
        <CardHeader>
          <div class="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
            <div class="space-y-2">
              <CardTitle class="text-base">Database Decryption</CardTitle>
              <CardDescription>Ensure directories and keys are saved before starting. This may take time.</CardDescription>
            </div>
            <div class="flex flex-wrap items-center gap-2">
              <Badge variant="outline" class="border-amber-500/30 bg-amber-500/10 text-amber-200">
                High Resource Usage
              </Badge>
              <span class="text-sm text-muted-foreground">Check disk space before proceeding.</span>
            </div>
          </div>
        </CardHeader>
        <CardContent>
          <Button :disabled="loadingDataKey || loadingImgKey || loadingDecrypt" @click="decryptNow">
            {{ loadingDecrypt ? 'Decrypting...' : 'Run Decryption Now' }}
          </Button>
        </CardContent>
      </Card>
    </section>
  </div>
</template>
