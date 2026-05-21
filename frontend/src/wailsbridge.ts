export type State = {
    account: string;
    platform: string;
    fullVersion: string;
    dataDir: string;
    dataKey: string;
    imgKey: string;
    workDir: string;
    httpEnabled: boolean;
    httpAddr: string;
    autoDecrypt: boolean;
    lastSession: string;
    selectedAIProvider: string;
    summaryPrompt: string;
    pid: number;
    exePath: string;
    status: string;
    nickname: string;
    smallHeadImgUrl: string;
};

export type Instance = {
    name: string;
    pid: number;
    platform: string;
    fullVersion: string;
    dataDir: string;
    exePath: string;
    status: string;
};

export type Contact = {
    userName: string;
    alias: string;
    remark: string;
    nickName: string;
    isFriend: boolean;
    localType: number;
    flag: number;
    deleteFlag: number;
    isInChatRoom: number;
    smallHeadImgUrl: string;
    bigHeadImgUrl: string;
};

export type ContactsResp = {
    total: number;
    items: Contact[];
};

export type Message = {
    seq: number;
    time: string;
    talker: string;
    talkerName: string;
    isChatRoom: boolean;
    sender: string;
    senderName: string;
    isSelf: boolean;
    type: number;
    content: string;
    isMentionMe: boolean;
};

export type MessagesResp = {
    total: number;
    items: Message[];
};

export type MediaDataResult = {
    data: string;
    mimeType: string;
};

export type WebhookItem = {
    description: string;
    type: string;
    url: string;
    talker: string;
    sender: string;
    keyword: string;
    disabled: boolean;
};

export type AIProvider = {
    id: string;
    name: string;
    type: string;
    baseUrl: string;
    apiKey: string;
    model: string;
    disabled: boolean;
    createdAt: number;
    updatedAt: number;
};

export type AITestResult = {
    ok: boolean;
    latencyMs: number;
    endpoint: string;
    status: number;
    message: string;
};

export type WebhookConfig = {
    host: string;
    delayMs: number;
    items: WebhookItem[];
};

export type KeyProgressEvent = {
    operation: string;
    message: string;
};

type Backend = {
    GetState(): Promise<State>;
    Refresh(): Promise<State>;
    ListInstances(): Promise<Instance[]>;
    SwitchToPID(pid: number): Promise<State>;
    SwitchToHistory(account: string): Promise<State>;
    GetContacts(keyword: string, isInChatRoom: number, limit: number, offset: number): Promise<ContactsResp>;
    GetMessages(timeStr: string, talker: string, sender: string, keyword: string, limit: number, offset: number, order: string): Promise<MessagesResp>;
    GetMediaData(mediaType: string, key: string): Promise<MediaDataResult>;
    GetDataKey(): Promise<string>;
    GetImgKey(): Promise<string>;
    GetKeys(): Promise<Record<string, string>>;
    Decrypt(): Promise<void>;
    ListListenIPs(): Promise<string[]>;
    SetHTTPAddr(addr: string): Promise<State>;
    SetWorkDir(dir: string): Promise<State>;
    SetDataDir(dir: string): Promise<State>;
    SetDataKey(key: string): Promise<State>;
    SetImgKey(key: string): Promise<State>;
    StartHTTP(): Promise<void>;
    StopHTTP(): Promise<void>;
    SetAutoDecrypt(enabled: boolean): Promise<void>;
    SetSelectedAIProvider(providerID: string): Promise<void>;
    SetSummaryPrompt(prompt: string): Promise<void>;
    GenerateAISummary(providerID: string, messages: string[], prompt: string): Promise<void>;
    GetLogPath(): Promise<string>;
    ReadLogTail(maxLines: number): Promise<string>;
    EnableStateEvents(enabled: boolean): Promise<void>;
    EnableLogEvents(enabled: boolean): Promise<void>;
    GetWebhookConfig(): Promise<WebhookConfig>;
    SetWebhookConfig(cfg: WebhookConfig): Promise<void>;
    ListAIProviders(): Promise<AIProvider[]>;
    SaveAIProvider(p: AIProvider): Promise<AIProvider>;
    DeleteAIProvider(id: string): Promise<void>;
    TestAIProvider(p: AIProvider): Promise<AITestResult>;
    ListAIModels(p: AIProvider): Promise<string[]>;
};

type Runtime = {
    EventsOn(name: string, callback: (data: unknown) => void): () => void;
    BrowserOpenURL(url: string): void;
};

declare global {
    interface Window {
        go: {
            main: {
                App: Backend;
            };
        };
        runtime: Runtime;
    }
}

export const backend = {
    isWails: typeof window !== 'undefined' && !!window.go && !!window.runtime,
    GetState: () => window.go.main.App.GetState(),
    Refresh: () => window.go.main.App.Refresh(),
    ListInstances: () => window.go.main.App.ListInstances(),
    SwitchToPID: (pid: number) => window.go.main.App.SwitchToPID(pid),
    SwitchToHistory: (account: string) => window.go.main.App.SwitchToHistory(account),
    GetContacts: (keyword: string, isInChatRoom: number, limit: number, offset: number) =>
        window.go.main.App.GetContacts(keyword, isInChatRoom, limit, offset),
    GetDataKey: () => window.go.main.App.GetDataKey(),
    GetImgKey: () => window.go.main.App.GetImgKey(),
    GetKeys: () => window.go.main.App.GetKeys(),
    Decrypt: () => window.go.main.App.Decrypt(),
    ListListenIPs: () => window.go.main.App.ListListenIPs(),
    SetHTTPAddr: (addr: string) => window.go.main.App.SetHTTPAddr(addr),
    SetWorkDir: (dir: string) => window.go.main.App.SetWorkDir(dir),
    SetDataDir: (dir: string) => window.go.main.App.SetDataDir(dir),
    SetDataKey: (key: string) => window.go.main.App.SetDataKey(key),
    SetImgKey: (key: string) => window.go.main.App.SetImgKey(key),
    StartHTTP: () => window.go.main.App.StartHTTP(),
    StopHTTP: () => window.go.main.App.StopHTTP(),
    SetAutoDecrypt: (enabled: boolean) => window.go.main.App.SetAutoDecrypt(enabled),
    SetSelectedAIProvider: (providerID: string) => window.go.main.App.SetSelectedAIProvider(providerID),
    SetSummaryPrompt: (prompt: string) => window.go.main.App.SetSummaryPrompt(prompt),
    GetMessages: (timeStr: string, talker: string, sender: string, keyword: string, limit: number, offset: number, order: string) =>
        window.go.main.App.GetMessages(timeStr, talker, sender, keyword, limit, offset, order),
    GetMediaData: (mediaType: string, key: string) =>
        window.go.main.App.GetMediaData(mediaType, key),
    GenerateAISummary: (providerID: string, messages: string[], prompt: string) =>
        window.go.main.App.GenerateAISummary(providerID, messages, prompt),
    GetLogPath: () => window.go.main.App.GetLogPath(),
    ReadLogTail: (maxLines: number) => window.go.main.App.ReadLogTail(maxLines),
    EnableStateEvents: (enabled: boolean) => window.go.main.App.EnableStateEvents(enabled),
    EnableLogEvents: (enabled: boolean) => window.go.main.App.EnableLogEvents(enabled),
    GetWebhookConfig: () => window.go.main.App.GetWebhookConfig(),
    SetWebhookConfig: (cfg: WebhookConfig) => window.go.main.App.SetWebhookConfig(cfg),
    ListAIProviders: () => window.go.main.App.ListAIProviders(),
    SaveAIProvider: (p: AIProvider) => window.go.main.App.SaveAIProvider(p),
    DeleteAIProvider: (id: string) => window.go.main.App.DeleteAIProvider(id),
    TestAIProvider: (p: AIProvider) => window.go.main.App.TestAIProvider(p),
    ListAIModels: (p: AIProvider) => window.go.main.App.ListAIModels(p),
    EventsOn: (name: string, callback: (data: unknown) => void) => {
        if (!window.runtime || !window.runtime.EventsOn) return;
        return window.runtime.EventsOn(name, callback);
    },
    OpenURL: (url: string) => {
        if (window.runtime && window.runtime.BrowserOpenURL) {
            window.runtime.BrowserOpenURL(url);
            return;
        }
        if (typeof window !== 'undefined') window.open(url, '_blank', 'noopener');
    }
};
