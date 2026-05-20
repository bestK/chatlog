export type EndpointKey = 'session' | 'chatroom' | 'contact' | 'chatlog';

export type ParamType = 'text' | 'date' | 'select' | 'autocomplete';

export interface AutocompleteOption {
    value: string;
    label: string;
    sub?: string;
}

export interface ParamSpec {
    key: string;
    label: string;
    type: ParamType;
    required?: boolean;
    placeholder?: string;
    hint?: string;
    options?: { value: string; label: string }[];
    source?: (keyword: string) => Promise<AutocompleteOption[]>;
    minFromKey?: string;
    maxFromKey?: string;
    default?: string;
}

interface ContactItem {
    userName: string;
    alias?: string;
    remark?: string;
    nickName?: string;
}

interface ChatRoomItem {
    name: string;
    remark?: string;
    nickName?: string;
}

async function fetchContacts(keyword: string): Promise<AutocompleteOption[]> {
    const params = new URLSearchParams({ format: 'json', limit: '30', offset: '0' });
    if (keyword) params.set('keyword', keyword);
    const resp = await fetch(`/api/v1/contact?${params.toString()}`);
    if (!resp.ok) return [];
    const data = await resp.json();
    const items: ContactItem[] = data?.Items || data?.items || [];
    return items
        .map(c => {
            return {
                value: c.userName,
                label: c.remark || c.nickName || c.alias || c.userName,
                sub: c.userName
            };
        })
        .filter(c => c.value);
}

async function fetchChatRooms(keyword: string): Promise<AutocompleteOption[]> {
    const params = new URLSearchParams({ format: 'json', limit: '30', offset: '0' });
    if (keyword) params.set('keyword', keyword);
    const resp = await fetch(`/api/v1/chatroom?${params.toString()}`);
    if (!resp.ok) return [];
    const data = await resp.json();
    const items: ChatRoomItem[] = data?.Items || data?.items || [];
    return items
        .map(r => {
            return {
                value: r.name,
                label: r.remark || r.nickName || r.name,
                sub: r.name
            };
        })
        .filter(r => r.value);
}

async function searchTalker(keyword: string): Promise<AutocompleteOption[]> {
    const [contacts, rooms] = await Promise.all([fetchContacts(keyword), fetchChatRooms(keyword)]);
    return [...rooms, ...contacts];
}

export interface EndpointSpec {
    key: EndpointKey;
    label: string;
    path: string;
    description: string;
    params: ParamSpec[];
}

const formatOptions = [
    { value: '', label: '默认' },
    { value: 'json', label: 'JSON' },
    { value: 'text', label: '纯文本' }
];

export const endpoints: EndpointSpec[] = [
    {
        key: 'session',
        label: '最近会话',
        path: '/api/v1/session',
        description: '查询最近的会话列表，可按昵称或备注关键词检索。',
        params: [
            { key: 'keyword', label: '关键词', type: 'text', placeholder: '昵称或备注' },
            { key: 'format', label: '输出格式', type: 'select', options: formatOptions }
        ]
    },
    {
        key: 'chatroom',
        label: '群聊',
        path: '/api/v1/chatroom',
        description: '查询群聊列表，可按关键词检索。',
        params: [
            { key: 'keyword', label: '关键词', type: 'text', placeholder: '群名或群 ID' },
            { key: 'format', label: '输出格式', type: 'select', options: formatOptions }
        ]
    },
    {
        key: 'contact',
        label: '联系人',
        path: '/api/v1/contact',
        description: '查询联系人列表，可按关键词检索。',
        params: [
            { key: 'keyword', label: '关键词', type: 'text', placeholder: '昵称、备注或 wxid' },
            { key: 'format', label: '输出格式', type: 'select', options: formatOptions }
        ]
    },
    {
        key: 'chatlog',
        label: '聊天记录',
        path: '/api/v1/chatlog',
        description: '查询指定时间范围内与特定联系人或群聊的聊天记录。',
        params: [
            { key: 'startDate', label: '开始日期', type: 'date', required: true },
            { key: 'endDate', label: '结束日期', type: 'date', hint: '留空则只查开始日期当天' },
            {
                key: 'talker',
                label: '聊天对象',
                type: 'autocomplete',
                required: true,
                placeholder: '搜索群聊或联系人',
                source: searchTalker
            },
            {
                key: 'sender',
                label: '发送者',
                type: 'autocomplete',
                placeholder: '搜索联系人',
                source: fetchContacts
            },
            { key: 'keyword', label: '关键词', type: 'text', placeholder: '消息内容关键词' },
            {
                key: 'format',
                label: '输出格式',
                type: 'select',
                options: [
                    { value: '', label: '默认' },
                    { value: 'text', label: '纯文本' },
                    { value: 'json', label: 'JSON' },
                    { value: 'csv', label: 'CSV' }
                ]
            }
        ]
    }
];

export function getEndpoint(key: EndpointKey): EndpointSpec {
    const ep = endpoints.find(e => e.key === key);
    if (!ep) throw new Error(`unknown endpoint: ${key}`);
    return ep;
}

export interface BuildResult {
    apiUrl: string;
    fullUrl: string;
    error?: string;
}

export function buildUrl(
    ep: EndpointSpec,
    values: Record<string, string>,
    pagination: { offset: number; limit: number }
): BuildResult {
    const params = new URLSearchParams();

    for (const p of ep.params) {
        if (p.required && !values[p.key]) {
            return { apiUrl: '', fullUrl: '', error: `缺少必填项：${p.label}` };
        }
    }

    if (ep.key === 'chatlog') {
        const start = values.startDate || '';
        const end = values.endDate || '';
        let time = start;
        if (end && end !== start) time = `${start}~${end}`;
        if (time) params.set('time', time);
        if (values.talker) params.set('talker', values.talker);
        if (values.sender) params.set('sender', values.sender);
        if (values.keyword) params.set('keyword', values.keyword);
        if (values.format) params.set('format', values.format);
    } else {
        if (values.keyword) params.set('keyword', values.keyword);
        if (values.format) params.set('format', values.format);
    }

    params.set('limit', String(pagination.limit));
    params.set('offset', String(pagination.offset));

    const apiUrl = `${ep.path}?${params.toString()}`;
    const fullUrl = (typeof window !== 'undefined' ? window.location.origin : '') + apiUrl;
    return { apiUrl, fullUrl };
}

// AI 总结相关接口
export interface AISummaryRequest {
    providerId: string;
    messages: string[];
    prompt?: string;
}

export interface AISummaryResponse {
    summary: string;
    model: string;
    tokens?: number;
}

// AI 总结
export async function requestAISummary(req: AISummaryRequest): Promise<AISummaryResponse> {
    const response = await fetch('/api/v1/ai/summary', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json'
        },
        body: JSON.stringify(req)
    });

    if (!response.ok) {
        throw new Error(`AI 总结请求失败: ${response.statusText}`);
    }

    return response.json();
}

// 流式 AI 总结
export async function requestAISummaryStream(req: AISummaryRequest): Promise<ReadableStream<Uint8Array>> {
    const response = await fetch('/api/v1/ai/summary/stream', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json'
        },
        body: JSON.stringify(req)
    });

    if (!response.ok) {
        throw new Error(`AI 流式总结请求失败: ${response.statusText}`);
    }

    return response.body!;
}
