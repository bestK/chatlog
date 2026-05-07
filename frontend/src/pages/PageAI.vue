<script setup lang="ts">
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { computed, inject, onMounted, onUnmounted, reactive, ref } from 'vue';
import { appContextKey } from '../app/context';
import { backend, type AIProvider, type AITestResult } from '../wailsbridge';

const injected = inject(appContextKey);
if (!injected) throw new Error('chatlog not provided');
const app = injected;

type ProviderFormState = AIProvider & {
    showKey?: boolean;
};

const providers = ref<AIProvider[]>([]);
const loading = ref(false);
const saving = ref(false);
const editing = ref<ProviderFormState | null>(null);
const testing = ref<Record<string, boolean>>({});
const testResults = reactive<Record<string, AITestResult>>({});
const availableModels = ref<string[]>([]);
const loadingModels = ref(false);
const modelDropdownOpen = ref(false);
const modelDropdownRef = ref<HTMLElement | null>(null);
const modelKeyword = ref('');

const filteredModels = computed(() => {
    const q = modelKeyword.value.trim().toLowerCase();
    if (!q) return availableModels.value;
    return availableModels.value.filter(m => m.toLowerCase().includes(q));
});

function toggleModelDropdown() {
    modelDropdownOpen.value = !modelDropdownOpen.value;
}

function pickModel(name: string) {
    if (!editing.value) return;
    editing.value.model = name;
    modelDropdownOpen.value = false;
}

function onModelClickOutside(e: MouseEvent) {
    if (!modelDropdownOpen.value) return;
    if (modelDropdownRef.value && !modelDropdownRef.value.contains(e.target as Node)) {
        modelDropdownOpen.value = false;
    }
}

async function fetchModels() {
    if (!editing.value) return;
    const form = editing.value;
    if (!form.apiKey.trim()) {
        app.feedback.toast('无法获取', '请先填写 API Key');
        return;
    }
    loadingModels.value = true;
    try {
        const payload: AIProvider = {
            id: form.id,
            name: form.name.trim() || 'untitled',
            type: form.type,
            baseUrl: form.baseUrl.trim(),
            apiKey: form.apiKey.trim(),
            model: '',
            disabled: form.disabled,
            createdAt: form.createdAt,
            updatedAt: form.updatedAt
        };
        availableModels.value = (await backend.ListAIModels(payload)) || [];
        modelDropdownOpen.value = true;
        if (availableModels.value.length === 0) {
            app.feedback.toast('未返回模型', '该端点未返回任何模型');
        }
    } catch (e) {
        app.feedback.toast('获取失败', String(e));
    } finally {
        loadingModels.value = false;
    }
}

const PROVIDER_TYPES = [
    { value: 'openai', label: 'OpenAI', baseUrl: 'https://api.openai.com', sample: 'gpt-4o-mini' },
    { value: 'openai-compatible', label: 'OpenAI 兼容', baseUrl: '', sample: 'gpt-4o-mini' },
    {
        value: 'anthropic',
        label: 'Anthropic',
        baseUrl: 'https://api.anthropic.com',
        sample: 'claude-3-5-sonnet-latest'
    },
    {
        value: 'google',
        label: 'Google Gemini',
        baseUrl: 'https://generativelanguage.googleapis.com',
        sample: 'gemini-1.5-flash'
    }
] as const;

function emptyProvider(): ProviderFormState {
    return {
        id: '',
        name: '',
        type: 'openai',
        baseUrl: '',
        apiKey: '',
        model: '',
        disabled: false,
        createdAt: 0,
        updatedAt: 0
    };
}

async function loadProviders() {
    loading.value = true;
    try {
        providers.value = (await backend.ListAIProviders()) || [];
    } catch (e) {
        app.feedback.toast('加载失败', String(e));
    } finally {
        loading.value = false;
    }
}

onMounted(() => {
    void loadProviders();
    document.addEventListener('click', onModelClickOutside);
});
onUnmounted(() => {
    document.removeEventListener('click', onModelClickOutside);
});

function startCreate() {
    editing.value = { ...emptyProvider(), showKey: true };
}

function startEdit(p: AIProvider) {
    editing.value = { ...p, showKey: false };
}

function cancelEdit() {
    editing.value = null;
}

function maskKey(value: string): string {
    if (!value) return '未设置';
    if (value.length <= 8) return '***';
    return `${value.slice(0, 4)}***${value.slice(-4)}`;
}

const typeLabel = computed(() => (type: string) => {
    const hit = PROVIDER_TYPES.find(t => t.value === type);
    return hit ? hit.label : type;
});

function applyTypeDefaults(form: ProviderFormState) {
    const hit = PROVIDER_TYPES.find(t => t.value === form.type);
    if (!hit) return;
    if (!form.baseUrl && hit.baseUrl) form.baseUrl = hit.baseUrl;
    if (!form.model && hit.sample) form.model = hit.sample;
}

async function saveProvider() {
    if (!editing.value) return;
    const form = editing.value;
    if (!form.name.trim()) {
        app.feedback.toast('保存失败', '请填写名称');
        return;
    }
    if (!form.apiKey.trim()) {
        app.feedback.toast('保存失败', '请填写 API Key');
        return;
    }
    saving.value = true;
    try {
        const payload: AIProvider = {
            id: form.id,
            name: form.name.trim(),
            type: form.type,
            baseUrl: form.baseUrl.trim(),
            apiKey: form.apiKey.trim(),
            model: form.model.trim(),
            disabled: form.disabled,
            createdAt: form.createdAt,
            updatedAt: form.updatedAt
        };
        await backend.SaveAIProvider(payload);
        app.feedback.toast('保存成功', form.name);
        editing.value = null;
        await loadProviders();
    } catch (e) {
        app.feedback.toast('保存失败', String(e));
    } finally {
        saving.value = false;
    }
}

async function removeProvider(p: AIProvider) {
    const ok = await app.feedback.confirm({
        title: '删除提供商',
        message: `确认删除「${p.name}」？此操作不可撤销。`,
        confirmText: '删除',
        cancelText: '取消',
        danger: true
    });
    if (!ok) return;
    try {
        await backend.DeleteAIProvider(p.id);
        app.feedback.toast('已删除', p.name);
        await loadProviders();
    } catch (e) {
        app.feedback.toast('删除失败', String(e));
    }
}

async function testProvider(p: AIProvider) {
    testing.value[p.id] = true;
    try {
        const result = await backend.TestAIProvider(p);
        testResults[p.id] = result;
        if (result.ok) {
            app.feedback.toast('连通正常', `${p.name} · ${result.latencyMs} ms`);
        } else {
            app.feedback.toast('测试失败', result.message || `HTTP ${result.status}`);
        }
    } catch (e) {
        testResults[p.id] = {
            ok: false,
            latencyMs: 0,
            endpoint: '',
            status: 0,
            message: String(e)
        };
        app.feedback.toast('测试失败', String(e));
    } finally {
        testing.value[p.id] = false;
    }
}

async function testFormProvider() {
    if (!editing.value) return;
    const form = editing.value;
    testing.value['__form'] = true;
    try {
        const payload: AIProvider = {
            id: form.id,
            name: form.name.trim() || 'untitled',
            type: form.type,
            baseUrl: form.baseUrl.trim(),
            apiKey: form.apiKey.trim(),
            model: form.model.trim(),
            disabled: form.disabled,
            createdAt: form.createdAt,
            updatedAt: form.updatedAt
        };
        const result = await backend.TestAIProvider(payload);
        testResults['__form'] = result;
        if (result.ok) {
            app.feedback.toast('连通正常', `${result.latencyMs} ms · ${result.endpoint}`);
        } else {
            app.feedback.toast('测试失败', result.message || `HTTP ${result.status}`);
        }
    } catch (e) {
        app.feedback.toast('测试失败', String(e));
    } finally {
        testing.value['__form'] = false;
    }
}
</script>

<template>
    <div class="space-y-10">
        <div
            class="sticky -top-px z-30 flex flex-wrap items-center justify-between gap-3 rounded-xl bg-muted/30 ring-1 ring-border/40 px-4 py-3 backdrop-blur supports-[backdrop-filter]:bg-muted/40"
        >
            <p class="text-sm text-muted-foreground">配置 AI 提供商，用于聊天分析与摘要等场景。</p>
            <div class="flex items-center gap-2">
                <Button
                    variant="ghost"
                    size="sm"
                    class="h-9 px-3 text-sm font-normal text-muted-foreground hover:text-foreground"
                    :disabled="loading"
                    @click="loadProviders"
                >
                    刷新
                </Button>
                <Button size="sm" class="h-9 px-4 text-sm" @click="startCreate">+ 新增提供商</Button>
            </div>
        </div>

        <section class="space-y-4">
            <header class="space-y-1">
                <h3 class="font-serif text-xl font-medium tracking-tight text-foreground">提供商列表</h3>
                <p class="text-sm text-muted-foreground">已配置的 AI 服务，可测试连通性或编辑配置。</p>
            </header>

            <div
                v-if="!loading && providers.length === 0"
                class="rounded-xl bg-muted/30 ring-1 ring-border/40 py-14 text-center"
            >
                <div class="text-sm text-foreground/80">暂无提供商</div>
                <p class="mt-1 text-xs text-muted-foreground">添加 AI 提供商后可在此管理。</p>
                <Button variant="link" size="sm" class="mt-3 text-sm" @click="startCreate">立即添加</Button>
            </div>

            <div v-else class="space-y-4">
                <article
                    v-for="p in providers"
                    :key="p.id"
                    class="space-y-3 rounded-xl bg-muted/30 ring-1 ring-border/40 p-5 sm:p-6"
                >
                    <div class="flex flex-wrap items-baseline justify-between gap-3">
                        <div class="flex items-baseline gap-3">
                            <span class="font-serif text-base font-medium tracking-tight text-foreground">{{
                                p.name
                            }}</span>
                            <span class="text-xs text-muted-foreground">{{ typeLabel(p.type) }}</span>
                            <span v-if="p.disabled" class="text-xs text-muted-foreground/70">· 已禁用</span>
                        </div>
                        <div class="flex items-center gap-1">
                            <Button
                                variant="ghost"
                                size="sm"
                                class="h-8 px-2.5 text-xs font-normal text-muted-foreground hover:text-foreground"
                                :disabled="testing[p.id]"
                                @click="testProvider(p)"
                            >
                                {{ testing[p.id] ? '测试中…' : '测试' }}
                            </Button>
                            <Button
                                variant="ghost"
                                size="sm"
                                class="h-8 px-2.5 text-xs font-normal text-muted-foreground hover:text-foreground"
                                @click="startEdit(p)"
                            >
                                编辑
                            </Button>
                            <Button
                                variant="ghost"
                                size="sm"
                                class="h-8 px-2.5 text-xs font-normal text-muted-foreground hover:bg-destructive/10 hover:text-destructive"
                                @click="removeProvider(p)"
                            >
                                删除
                            </Button>
                        </div>
                    </div>

                    <div class="grid gap-x-6 gap-y-2 sm:grid-cols-2 text-xs">
                        <div class="flex items-baseline gap-2">
                            <span class="w-16 text-muted-foreground">BaseURL</span>
                            <span class="break-all font-mono text-foreground/85">{{ p.baseUrl || '默认' }}</span>
                        </div>
                        <div class="flex items-baseline gap-2">
                            <span class="w-16 text-muted-foreground">模型</span>
                            <span class="break-all font-mono text-foreground/85">{{ p.model || '未指定' }}</span>
                        </div>
                        <div class="flex items-baseline gap-2">
                            <span class="w-16 text-muted-foreground">API Key</span>
                            <span class="font-mono text-foreground/85">{{ maskKey(p.apiKey) }}</span>
                        </div>
                        <div v-if="testResults[p.id]" class="flex items-baseline gap-2">
                            <span class="w-16 text-muted-foreground">最近测试</span>
                            <span
                                :class="testResults[p.id].ok ? 'text-emerald-500' : 'text-rose-500'"
                                class="break-all font-mono"
                            >
                                {{
                                    testResults[p.id].ok
                                        ? `OK · ${testResults[p.id].latencyMs} ms`
                                        : testResults[p.id].message || `HTTP ${testResults[p.id].status}`
                                }}
                            </span>
                        </div>
                    </div>
                </article>
            </div>
        </section>

        <section v-if="editing" class="space-y-4">
            <header class="space-y-1">
                <h3 class="font-serif text-xl font-medium tracking-tight text-foreground">
                    {{ editing.id ? '编辑提供商' : '新增提供商' }}
                </h3>
                <p class="text-sm text-muted-foreground">填写凭据后可保存或测试连通性。</p>
            </header>

            <div class="rounded-xl bg-muted/30 ring-1 ring-border/40 p-5 sm:p-6 space-y-5">
                <div class="grid gap-x-6 gap-y-5 md:grid-cols-2">
                    <div class="space-y-2">
                        <label class="text-sm text-foreground">名称</label>
                        <Input
                            v-model="editing.name"
                            class="h-9 bg-background/40 text-sm"
                            placeholder="例如：OpenAI · 主账号"
                        />
                    </div>
                    <div class="space-y-2">
                        <label class="text-sm text-foreground">类型</label>
                        <select
                            v-model="editing.type"
                            class="h-9 w-full rounded-md border border-input bg-background/40 px-3 text-sm focus:outline-none focus:ring-1 focus:ring-ring"
                            @change="editing && applyTypeDefaults(editing)"
                        >
                            <option v-for="t in PROVIDER_TYPES" :key="t.value" :value="t.value">{{ t.label }}</option>
                        </select>
                    </div>
                    <div class="space-y-2 md:col-span-2">
                        <label class="text-sm text-foreground">Base URL</label>
                        <Input
                            v-model="editing.baseUrl"
                            class="h-9 bg-background/40 font-mono text-xs"
                            placeholder="留空使用默认地址"
                        />
                    </div>
                    <div class="space-y-2 md:col-span-2">
                        <label class="text-sm text-foreground">API Key</label>
                        <div class="flex items-center gap-2">
                            <Input
                                v-model="editing.apiKey"
                                :type="editing.showKey ? 'text' : 'password'"
                                class="h-9 flex-1 bg-background/40 font-mono text-xs"
                                placeholder="sk-..."
                            />
                            <Button
                                variant="ghost"
                                size="sm"
                                class="h-9 px-3 text-sm font-normal text-muted-foreground hover:text-foreground"
                                @click="editing && (editing.showKey = !editing.showKey)"
                            >
                                {{ editing.showKey ? '隐藏' : '显示' }}
                            </Button>
                        </div>
                    </div>
                    <div class="space-y-2">
                        <div class="flex items-center justify-between gap-2">
                            <label class="text-sm text-foreground">默认模型</label>
                            <Button
                                variant="ghost"
                                size="sm"
                                class="h-7 px-2 text-xs font-normal text-muted-foreground hover:text-foreground"
                                :disabled="loadingModels"
                                @click="fetchModels"
                            >
                                {{ loadingModels ? '获取中…' : '获取列表' }}
                            </Button>
                        </div>
                        <div ref="modelDropdownRef" class="relative">
                            <div class="flex items-center gap-2">
                                <Input
                                    v-model="editing.model"
                                    class="h-9 flex-1 bg-background/40 font-mono text-xs"
                                    placeholder="例如：gpt-4o-mini"
                                />
                                <Button
                                    v-if="availableModels.length > 0"
                                    variant="ghost"
                                    size="sm"
                                    class="h-9 px-2 text-xs font-normal text-muted-foreground hover:text-foreground"
                                    @click.stop="toggleModelDropdown"
                                >
                                    <svg
                                        :class="['size-3.5 transition-transform', modelDropdownOpen && 'rotate-180']"
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
                                </Button>
                            </div>
                            <Transition
                                enter-from-class="opacity-0 -translate-y-1"
                                enter-active-class="transition duration-150"
                                leave-active-class="transition duration-100"
                                leave-to-class="opacity-0 -translate-y-1"
                            >
                                <div
                                    v-if="modelDropdownOpen && availableModels.length > 0"
                                    class="absolute left-0 right-0 top-full z-30 mt-1.5 overflow-hidden rounded-md border border-border/60 bg-popover/95 shadow-lg backdrop-blur"
                                >
                                    <div class="border-b border-border/40 p-1.5">
                                        <input
                                            v-model="modelKeyword"
                                            placeholder="搜索模型…"
                                            class="h-8 w-full rounded-sm bg-transparent px-2 text-sm focus:outline-none"
                                        />
                                    </div>
                                    <div class="max-h-60 overflow-auto p-1">
                                        <div
                                            v-if="filteredModels.length === 0"
                                            class="px-2 py-3 text-center text-xs text-muted-foreground"
                                        >
                                            无匹配项
                                        </div>
                                        <button
                                            v-for="m in filteredModels"
                                            :key="m"
                                            type="button"
                                            :class="[
                                                'flex w-full items-center justify-between gap-2 rounded-md px-2.5 py-1.5 text-left text-sm transition-colors',
                                                m === editing.model
                                                    ? 'bg-accent text-foreground'
                                                    : 'text-foreground/85 hover:bg-accent/60'
                                            ]"
                                            @click="pickModel(m)"
                                        >
                                            <span class="truncate font-mono text-xs">{{ m }}</span>
                                            <svg
                                                v-if="m === editing.model"
                                                class="size-3.5 text-foreground/70"
                                                xmlns="http://www.w3.org/2000/svg"
                                                viewBox="0 0 24 24"
                                                fill="none"
                                                stroke="currentColor"
                                                stroke-width="2"
                                                stroke-linecap="round"
                                                stroke-linejoin="round"
                                            >
                                                <path d="M20 6 9 17l-5-5" />
                                            </svg>
                                        </button>
                                    </div>
                                </div>
                            </Transition>
                        </div>
                    </div>
                    <div class="space-y-2">
                        <label class="text-sm text-foreground">状态</label>
                        <label class="flex h-9 items-center gap-2 text-sm text-muted-foreground">
                            <input v-model="editing.disabled" type="checkbox" class="size-4 accent-foreground" />
                            禁用此提供商
                        </label>
                    </div>
                </div>

                <div class="flex flex-wrap items-center justify-end gap-2 pt-2">
                    <Button
                        variant="ghost"
                        size="sm"
                        class="h-9 px-3 text-sm font-normal text-muted-foreground hover:text-foreground"
                        @click="cancelEdit"
                    >
                        取消
                    </Button>
                    <Button
                        variant="ghost"
                        size="sm"
                        class="h-9 px-3 text-sm font-normal text-muted-foreground hover:text-foreground"
                        :disabled="testing['__form']"
                        @click="testFormProvider"
                    >
                        {{ testing['__form'] ? '测试中…' : '测试连通' }}
                    </Button>
                    <Button size="sm" class="h-9 px-4 text-sm" :disabled="saving" @click="saveProvider">
                        {{ saving ? '保存中…' : '保存' }}
                    </Button>
                </div>

                <div v-if="testResults['__form']" class="text-xs">
                    <span
                        :class="testResults['__form'].ok ? 'text-emerald-500' : 'text-rose-500'"
                        class="break-all font-mono"
                    >
                        {{
                            testResults['__form'].ok
                                ? `OK · ${testResults['__form'].latencyMs} ms · ${testResults['__form'].endpoint}`
                                : testResults['__form'].message || `HTTP ${testResults['__form'].status}`
                        }}
                    </span>
                </div>
            </div>
        </section>
    </div>
</template>
