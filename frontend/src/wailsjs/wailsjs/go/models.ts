export namespace main {
	
	export class Instance {
	    name: string;
	    pid: number;
	    platform: string;
	    fullVersion: string;
	    dataDir: string;
	    exePath: string;
	    status: string;
	
	    static createFrom(source: any = {}) {
	        return new Instance(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.pid = source["pid"];
	        this.platform = source["platform"];
	        this.fullVersion = source["fullVersion"];
	        this.dataDir = source["dataDir"];
	        this.exePath = source["exePath"];
	        this.status = source["status"];
	    }
	}
	export class State {
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
	    pid: number;
	    exePath: string;
	    status: string;
	    nickname: string;
	    smallHeadImgUrl: string;
	
	    static createFrom(source: any = {}) {
	        return new State(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.account = source["account"];
	        this.platform = source["platform"];
	        this.fullVersion = source["fullVersion"];
	        this.dataDir = source["dataDir"];
	        this.dataKey = source["dataKey"];
	        this.imgKey = source["imgKey"];
	        this.workDir = source["workDir"];
	        this.httpEnabled = source["httpEnabled"];
	        this.httpAddr = source["httpAddr"];
	        this.autoDecrypt = source["autoDecrypt"];
	        this.lastSession = source["lastSession"];
	        this.pid = source["pid"];
	        this.exePath = source["exePath"];
	        this.status = source["status"];
	        this.nickname = source["nickname"];
	        this.smallHeadImgUrl = source["smallHeadImgUrl"];
	    }
	}
	export class WebhookItem {
	    description: string;
	    type: string;
	    url: string;
	    talker: string;
	    sender: string;
	    keyword: string;
	    disabled: boolean;
	
	    static createFrom(source: any = {}) {
	        return new WebhookItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.description = source["description"];
	        this.type = source["type"];
	        this.url = source["url"];
	        this.talker = source["talker"];
	        this.sender = source["sender"];
	        this.keyword = source["keyword"];
	        this.disabled = source["disabled"];
	    }
	}
	export class WebhookConfig {
	    host: string;
	    delayMs: number;
	    items: WebhookItem[];
	
	    static createFrom(source: any = {}) {
	        return new WebhookConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.host = source["host"];
	        this.delayMs = source["delayMs"];
	        this.items = this.convertValues(source["items"], WebhookItem);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

