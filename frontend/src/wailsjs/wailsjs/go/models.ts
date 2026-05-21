export namespace chatlog {
	
	export class MediaDataResult {
	    data: string;
	    mimeType: string;
	
	    static createFrom(source: any = {}) {
	        return new MediaDataResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.data = source["data"];
	        this.mimeType = source["mimeType"];
	    }
	}

}

export namespace main {
	
	export class AIProvider {
	    id: string;
	    name: string;
	    type: string;
	    baseUrl: string;
	    apiKey: string;
	    model: string;
	    disabled: boolean;
	    createdAt: number;
	    updatedAt: number;
	
	    static createFrom(source: any = {}) {
	        return new AIProvider(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.type = source["type"];
	        this.baseUrl = source["baseUrl"];
	        this.apiKey = source["apiKey"];
	        this.model = source["model"];
	        this.disabled = source["disabled"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	    }
	}
	export class AITestResult {
	    ok: boolean;
	    latencyMs: number;
	    endpoint: string;
	    status: number;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new AITestResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.latencyMs = source["latencyMs"];
	        this.endpoint = source["endpoint"];
	        this.status = source["status"];
	        this.message = source["message"];
	    }
	}
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
	    selectedAIProvider: string;
	    summaryPrompt: string;
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
	        this.selectedAIProvider = source["selectedAIProvider"];
	        this.summaryPrompt = source["summaryPrompt"];
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

export namespace model {
	
	export class WCPayInfo {
	    PaySubType: number;
	    FeeDesc: string;
	    TranscationID: string;
	    TransferID: string;
	    InvalidTime: string;
	    BeginTransferTime: string;
	    EffectiveDate: string;
	    PayMemo: string;
	    ReceiverUsername: string;
	    PayerUsername: string;
	
	    static createFrom(source: any = {}) {
	        return new WCPayInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.PaySubType = source["PaySubType"];
	        this.FeeDesc = source["FeeDesc"];
	        this.TranscationID = source["TranscationID"];
	        this.TransferID = source["TransferID"];
	        this.InvalidTime = source["InvalidTime"];
	        this.BeginTransferTime = source["BeginTransferTime"];
	        this.EffectiveDate = source["EffectiveDate"];
	        this.PayMemo = source["PayMemo"];
	        this.ReceiverUsername = source["ReceiverUsername"];
	        this.PayerUsername = source["PayerUsername"];
	    }
	}
	export class FinderLiveMediaCoverURL {
	    Text: string;
	
	    static createFrom(source: any = {}) {
	        return new FinderLiveMediaCoverURL(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Text = source["Text"];
	    }
	}
	export class FinderLiveMedia {
	    CoverURL: FinderLiveMediaCoverURL;
	    Height: number;
	    Width: number;
	
	    static createFrom(source: any = {}) {
	        return new FinderLiveMedia(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.CoverURL = this.convertValues(source["CoverURL"], FinderLiveMediaCoverURL);
	        this.Height = source["Height"];
	        this.Width = source["Width"];
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
	export class FinderLive {
	    FinderLiveID: string;
	    FinderUsername: string;
	    FinderObjectID: string;
	    FinderNonceID: string;
	    Nickname: string;
	    HeadURL: string;
	    Desc: string;
	    LiveStatus: number;
	    LiveSourceTypeStr: string;
	    ExtFlag: number;
	    LiveSecondaryDeviceFlagStr: string;
	    LiveFlag: number;
	    AuthIconURL: string;
	    AuthIconTypeStr: string;
	    BindType: number;
	    BizUsername: string;
	    BizNickname: string;
	    ChargeFlag: number;
	    ReplayStatus: number;
	    SpamLiveExtFlagString: string;
	    EnterSessionID: string;
	    LiveMode: number;
	    LiveSubMode: number;
	    Media: FinderLiveMedia;
	    ShareScene: number;
	
	    static createFrom(source: any = {}) {
	        return new FinderLive(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.FinderLiveID = source["FinderLiveID"];
	        this.FinderUsername = source["FinderUsername"];
	        this.FinderObjectID = source["FinderObjectID"];
	        this.FinderNonceID = source["FinderNonceID"];
	        this.Nickname = source["Nickname"];
	        this.HeadURL = source["HeadURL"];
	        this.Desc = source["Desc"];
	        this.LiveStatus = source["LiveStatus"];
	        this.LiveSourceTypeStr = source["LiveSourceTypeStr"];
	        this.ExtFlag = source["ExtFlag"];
	        this.LiveSecondaryDeviceFlagStr = source["LiveSecondaryDeviceFlagStr"];
	        this.LiveFlag = source["LiveFlag"];
	        this.AuthIconURL = source["AuthIconURL"];
	        this.AuthIconTypeStr = source["AuthIconTypeStr"];
	        this.BindType = source["BindType"];
	        this.BizUsername = source["BizUsername"];
	        this.BizNickname = source["BizNickname"];
	        this.ChargeFlag = source["ChargeFlag"];
	        this.ReplayStatus = source["ReplayStatus"];
	        this.SpamLiveExtFlagString = source["SpamLiveExtFlagString"];
	        this.EnterSessionID = source["EnterSessionID"];
	        this.LiveMode = source["LiveMode"];
	        this.LiveSubMode = source["LiveSubMode"];
	        this.Media = this.convertValues(source["Media"], FinderLiveMedia);
	        this.ShareScene = source["ShareScene"];
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
	export class PatInfo {
	    FromUsername: string;
	    ChatUsername: string;
	    PattedUsername: string;
	    PatSuffix: string;
	    PatSuffixVersion: number;
	    Template: string;
	
	    static createFrom(source: any = {}) {
	        return new PatInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.FromUsername = source["FromUsername"];
	        this.ChatUsername = source["ChatUsername"];
	        this.PattedUsername = source["PattedUsername"];
	        this.PatSuffix = source["PatSuffix"];
	        this.PatSuffixVersion = source["PatSuffixVersion"];
	        this.Template = source["Template"];
	    }
	}
	export class PatRecord {
	    FromUser: string;
	    PattedUser: string;
	    Templete: string;
	    CreateTime: number;
	    SvrId: string;
	    ReadStatus: number;
	
	    static createFrom(source: any = {}) {
	        return new PatRecord(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.FromUser = source["FromUser"];
	        this.PattedUser = source["PattedUser"];
	        this.Templete = source["Templete"];
	        this.CreateTime = source["CreateTime"];
	        this.SvrId = source["SvrId"];
	        this.ReadStatus = source["ReadStatus"];
	    }
	}
	export class Records {
	    Record: PatRecord[];
	
	    static createFrom(source: any = {}) {
	        return new Records(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Record = this.convertValues(source["Record"], PatRecord);
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
	export class PatMsg {
	    ChatUser: string;
	    RecordNum: number;
	    Records: Records;
	
	    static createFrom(source: any = {}) {
	        return new PatMsg(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ChatUser = source["ChatUser"];
	        this.RecordNum = source["RecordNum"];
	        this.Records = this.convertValues(source["Records"], Records);
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
	export class ReferMsg {
	    Type: number;
	    SvrID: string;
	    FromUsr: string;
	    ChatUsr: string;
	    DisplayName: string;
	    MsgSource: string;
	    Content: string;
	    StrID: string;
	    CreateTime: number;
	
	    static createFrom(source: any = {}) {
	        return new ReferMsg(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Type = source["Type"];
	        this.SvrID = source["SvrID"];
	        this.FromUsr = source["FromUsr"];
	        this.ChatUsr = source["ChatUsr"];
	        this.DisplayName = source["DisplayName"];
	        this.MsgSource = source["MsgSource"];
	        this.Content = source["Content"];
	        this.StrID = source["StrID"];
	        this.CreateTime = source["CreateTime"];
	    }
	}
	export class FinderMegaVideo {
	    ObjectID: string;
	    ObjectNonceID: string;
	
	    static createFrom(source: any = {}) {
	        return new FinderMegaVideo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ObjectID = source["ObjectID"];
	        this.ObjectNonceID = source["ObjectNonceID"];
	    }
	}
	export class FinderMedia {
	    ThumbURL: string;
	    FullCoverURL: string;
	    VideoPlayDuration: string;
	    URL: string;
	    CoverURL: string;
	    Height: string;
	    MediaType: string;
	    FullClipInset: string;
	    Width: string;
	
	    static createFrom(source: any = {}) {
	        return new FinderMedia(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ThumbURL = source["ThumbURL"];
	        this.FullCoverURL = source["FullCoverURL"];
	        this.VideoPlayDuration = source["VideoPlayDuration"];
	        this.URL = source["URL"];
	        this.CoverURL = source["CoverURL"];
	        this.Height = source["Height"];
	        this.MediaType = source["MediaType"];
	        this.FullClipInset = source["FullClipInset"];
	        this.Width = source["Width"];
	    }
	}
	export class FinderMediaList {
	    Media: FinderMedia[];
	
	    static createFrom(source: any = {}) {
	        return new FinderMediaList(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Media = this.convertValues(source["Media"], FinderMedia);
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
	export class FinderFeed {
	    ObjectID: string;
	    FeedType: string;
	    Nickname: string;
	    Avatar: string;
	    Desc: string;
	    MediaCount: string;
	    ObjectNonceID: string;
	    LiveID: string;
	    Username: string;
	    AuthIconURL: string;
	    AuthIconType: number;
	    ContactJumpInfoStr: string;
	    SourceCommentScene: number;
	    MediaList: FinderMediaList;
	    MegaVideo: FinderMegaVideo;
	    BizUsername: string;
	    BizNickname: string;
	    BizAvatar: string;
	    BizUsernameV2: string;
	    BizAuthIconURL: string;
	    BizAuthIconType: number;
	    EcSource: string;
	    LastGMsgID: string;
	    ShareBypData: string;
	    IsDebug: number;
	    ContentType: number;
	    FinderForwardSource: string;
	
	    static createFrom(source: any = {}) {
	        return new FinderFeed(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ObjectID = source["ObjectID"];
	        this.FeedType = source["FeedType"];
	        this.Nickname = source["Nickname"];
	        this.Avatar = source["Avatar"];
	        this.Desc = source["Desc"];
	        this.MediaCount = source["MediaCount"];
	        this.ObjectNonceID = source["ObjectNonceID"];
	        this.LiveID = source["LiveID"];
	        this.Username = source["Username"];
	        this.AuthIconURL = source["AuthIconURL"];
	        this.AuthIconType = source["AuthIconType"];
	        this.ContactJumpInfoStr = source["ContactJumpInfoStr"];
	        this.SourceCommentScene = source["SourceCommentScene"];
	        this.MediaList = this.convertValues(source["MediaList"], FinderMediaList);
	        this.MegaVideo = this.convertValues(source["MegaVideo"], FinderMegaVideo);
	        this.BizUsername = source["BizUsername"];
	        this.BizNickname = source["BizNickname"];
	        this.BizAvatar = source["BizAvatar"];
	        this.BizUsernameV2 = source["BizUsernameV2"];
	        this.BizAuthIconURL = source["BizAuthIconURL"];
	        this.BizAuthIconType = source["BizAuthIconType"];
	        this.EcSource = source["EcSource"];
	        this.LastGMsgID = source["LastGMsgID"];
	        this.ShareBypData = source["ShareBypData"];
	        this.IsDebug = source["IsDebug"];
	        this.ContentType = source["ContentType"];
	        this.FinderForwardSource = source["FinderForwardSource"];
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
	export class RecordXML {
	    RecordInfo: RecordInfo;
	
	    static createFrom(source: any = {}) {
	        return new RecordXML(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.RecordInfo = this.convertValues(source["RecordInfo"], RecordInfo);
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
	export class DataItemLocation {
	    Lat: string;
	    Lng: string;
	    Scale: string;
	    Label: string;
	    PoiName: string;
	
	    static createFrom(source: any = {}) {
	        return new DataItemLocation(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Lat = source["Lat"];
	        this.Lng = source["Lng"];
	        this.Scale = source["Scale"];
	        this.Label = source["Label"];
	        this.PoiName = source["PoiName"];
	    }
	}
	export class DataItem {
	    DataType: string;
	    DataID: string;
	    HTMLID: string;
	    DataFmt: string;
	    SourceName: string;
	    SourceTime: string;
	    SourceHeadURL: string;
	    DataDesc: string;
	    ThumbSourcePath: string;
	    ThumbSize: string;
	    CDNDataURL: string;
	    CDNDataKey: string;
	    CDNThumbURL: string;
	    CDNThumbKey: string;
	    DataSourcePath: string;
	    FullMD5: string;
	    ThumbFullMD5: string;
	    ThumbHead256MD5: string;
	    DataSize: string;
	    CDNEncryVer: string;
	    SrcChatname: string;
	    SrcMsgLocalID: string;
	    SrcMsgCreateTime: string;
	    MessageUUID: string;
	    FromNewMsgID: string;
	    Link: string;
	    StreamWebURL: string;
	    Location: DataItemLocation;
	    DataTitle: string;
	    RecordXML?: RecordXML;
	
	    static createFrom(source: any = {}) {
	        return new DataItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.DataType = source["DataType"];
	        this.DataID = source["DataID"];
	        this.HTMLID = source["HTMLID"];
	        this.DataFmt = source["DataFmt"];
	        this.SourceName = source["SourceName"];
	        this.SourceTime = source["SourceTime"];
	        this.SourceHeadURL = source["SourceHeadURL"];
	        this.DataDesc = source["DataDesc"];
	        this.ThumbSourcePath = source["ThumbSourcePath"];
	        this.ThumbSize = source["ThumbSize"];
	        this.CDNDataURL = source["CDNDataURL"];
	        this.CDNDataKey = source["CDNDataKey"];
	        this.CDNThumbURL = source["CDNThumbURL"];
	        this.CDNThumbKey = source["CDNThumbKey"];
	        this.DataSourcePath = source["DataSourcePath"];
	        this.FullMD5 = source["FullMD5"];
	        this.ThumbFullMD5 = source["ThumbFullMD5"];
	        this.ThumbHead256MD5 = source["ThumbHead256MD5"];
	        this.DataSize = source["DataSize"];
	        this.CDNEncryVer = source["CDNEncryVer"];
	        this.SrcChatname = source["SrcChatname"];
	        this.SrcMsgLocalID = source["SrcMsgLocalID"];
	        this.SrcMsgCreateTime = source["SrcMsgCreateTime"];
	        this.MessageUUID = source["MessageUUID"];
	        this.FromNewMsgID = source["FromNewMsgID"];
	        this.Link = source["Link"];
	        this.StreamWebURL = source["StreamWebURL"];
	        this.Location = this.convertValues(source["Location"], DataItemLocation);
	        this.DataTitle = source["DataTitle"];
	        this.RecordXML = this.convertValues(source["RecordXML"], RecordXML);
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
	export class DataList {
	    Count: string;
	    DataItems: DataItem[];
	
	    static createFrom(source: any = {}) {
	        return new DataList(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Count = source["Count"];
	        this.DataItems = this.convertValues(source["DataItems"], DataItem);
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
	export class RecordInfo {
	    XMLName: xml.Name;
	    FromScene: string;
	    FavUsername: string;
	    FavCreateTime: string;
	    IsChatRoom: string;
	    Title: string;
	    Desc: string;
	    Info: string;
	    DataList: DataList;
	
	    static createFrom(source: any = {}) {
	        return new RecordInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.XMLName = this.convertValues(source["XMLName"], xml.Name);
	        this.FromScene = source["FromScene"];
	        this.FavUsername = source["FavUsername"];
	        this.FavCreateTime = source["FavCreateTime"];
	        this.IsChatRoom = source["IsChatRoom"];
	        this.Title = source["Title"];
	        this.Desc = source["Desc"];
	        this.Info = source["Info"];
	        this.DataList = this.convertValues(source["DataList"], DataList);
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
	export class RecordItem {
	    CDATA: string;
	    RecordInfo?: RecordInfo;
	
	    static createFrom(source: any = {}) {
	        return new RecordItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.CDATA = source["CDATA"];
	        this.RecordInfo = this.convertValues(source["RecordInfo"], RecordInfo);
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
	export class AppAttach {
	    TotalLen: string;
	    AttachID: string;
	    CDNAttachURL: string;
	    EmoticonMD5: string;
	    AESKey: string;
	    FileExt: string;
	    IsLargeFileMsg: string;
	
	    static createFrom(source: any = {}) {
	        return new AppAttach(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.TotalLen = source["TotalLen"];
	        this.AttachID = source["AttachID"];
	        this.CDNAttachURL = source["CDNAttachURL"];
	        this.EmoticonMD5 = source["EmoticonMD5"];
	        this.AESKey = source["AESKey"];
	        this.FileExt = source["FileExt"];
	        this.IsLargeFileMsg = source["IsLargeFileMsg"];
	    }
	}
	export class App {
	    Type: number;
	    Title: string;
	    Des: string;
	    URL: string;
	    AppAttach?: AppAttach;
	    MD5: string;
	    RecordItem?: RecordItem;
	    SourceDisplayName: string;
	    FinderFeed?: FinderFeed;
	    ReferMsg?: ReferMsg;
	    PatMsg?: PatMsg;
	    PatInfo?: PatInfo;
	    FinderLive?: FinderLive;
	    WCPayInfo?: WCPayInfo;
	
	    static createFrom(source: any = {}) {
	        return new App(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Type = source["Type"];
	        this.Title = source["Title"];
	        this.Des = source["Des"];
	        this.URL = source["URL"];
	        this.AppAttach = this.convertValues(source["AppAttach"], AppAttach);
	        this.MD5 = source["MD5"];
	        this.RecordItem = this.convertValues(source["RecordItem"], RecordItem);
	        this.SourceDisplayName = source["SourceDisplayName"];
	        this.FinderFeed = this.convertValues(source["FinderFeed"], FinderFeed);
	        this.ReferMsg = this.convertValues(source["ReferMsg"], ReferMsg);
	        this.PatMsg = this.convertValues(source["PatMsg"], PatMsg);
	        this.PatInfo = this.convertValues(source["PatInfo"], PatInfo);
	        this.FinderLive = this.convertValues(source["FinderLive"], FinderLive);
	        this.WCPayInfo = this.convertValues(source["WCPayInfo"], WCPayInfo);
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
	
	export class Contact {
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
	
	    static createFrom(source: any = {}) {
	        return new Contact(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.userName = source["userName"];
	        this.alias = source["alias"];
	        this.remark = source["remark"];
	        this.nickName = source["nickName"];
	        this.isFriend = source["isFriend"];
	        this.localType = source["localType"];
	        this.flag = source["flag"];
	        this.deleteFlag = source["deleteFlag"];
	        this.isInChatRoom = source["isInChatRoom"];
	        this.smallHeadImgUrl = source["smallHeadImgUrl"];
	        this.bigHeadImgUrl = source["bigHeadImgUrl"];
	    }
	}
	export class Member {
	    Username: string;
	    Nickname: string;
	
	    static createFrom(source: any = {}) {
	        return new Member(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Username = source["Username"];
	        this.Nickname = source["Nickname"];
	    }
	}
	export class MemberList {
	    Members: Member[];
	
	    static createFrom(source: any = {}) {
	        return new MemberList(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Members = this.convertValues(source["Members"], Member);
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
	export class Link {
	    Name: string;
	    Type: string;
	    MemberList: MemberList;
	    Separator: string;
	    Title: string;
	
	    static createFrom(source: any = {}) {
	        return new Link(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Name = source["Name"];
	        this.Type = source["Type"];
	        this.MemberList = this.convertValues(source["MemberList"], MemberList);
	        this.Separator = source["Separator"];
	        this.Title = source["Title"];
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
	export class LinkList {
	    Links: Link[];
	
	    static createFrom(source: any = {}) {
	        return new LinkList(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Links = this.convertValues(source["Links"], Link);
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
	export class ContentTemplate {
	    Type: string;
	    Plain: string;
	    Template: string;
	    LinkList: LinkList;
	
	    static createFrom(source: any = {}) {
	        return new ContentTemplate(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Type = source["Type"];
	        this.Plain = source["Plain"];
	        this.Template = source["Template"];
	        this.LinkList = this.convertValues(source["LinkList"], LinkList);
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
	
	
	
	export class UsernameItem {
	    Value: string;
	
	    static createFrom(source: any = {}) {
	        return new UsernameItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Value = source["Value"];
	    }
	}
	export class QRMemberList {
	    Usernames: UsernameItem[];
	
	    static createFrom(source: any = {}) {
	        return new QRMemberList(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Usernames = this.convertValues(source["Usernames"], UsernameItem);
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
	export class QRLink {
	    Scene: string;
	    Text: string;
	    MemberList: QRMemberList;
	    QRCode: string;
	
	    static createFrom(source: any = {}) {
	        return new QRLink(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Scene = source["Scene"];
	        this.Text = source["Text"];
	        this.MemberList = this.convertValues(source["MemberList"], QRMemberList);
	        this.QRCode = source["QRCode"];
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
	export class DelChatRoomMember {
	    Plain: string;
	    Text: string;
	    Link: QRLink;
	
	    static createFrom(source: any = {}) {
	        return new DelChatRoomMember(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Plain = source["Plain"];
	        this.Text = source["Text"];
	        this.Link = this.convertValues(source["Link"], QRLink);
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
	export class Emoji {
	    FromUsername: string;
	    ToUsername: string;
	    Type: string;
	    IdBuffer: string;
	    Md5: string;
	    Len: string;
	    CdnURL: string;
	    AesKey: string;
	    Width: string;
	    Height: string;
	
	    static createFrom(source: any = {}) {
	        return new Emoji(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.FromUsername = source["FromUsername"];
	        this.ToUsername = source["ToUsername"];
	        this.Type = source["Type"];
	        this.IdBuffer = source["IdBuffer"];
	        this.Md5 = source["Md5"];
	        this.Len = source["Len"];
	        this.CdnURL = source["CdnURL"];
	        this.AesKey = source["AesKey"];
	        this.Width = source["Width"];
	        this.Height = source["Height"];
	    }
	}
	
	
	
	
	
	
	
	export class Image {
	    MD5: string;
	
	    static createFrom(source: any = {}) {
	        return new Image(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.MD5 = source["MD5"];
	    }
	}
	
	
	export class Location {
	    X: string;
	    Y: string;
	    Scale: string;
	    Label: string;
	    MapType: string;
	    Adcode: string;
	    CityName: string;
	
	    static createFrom(source: any = {}) {
	        return new Location(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.X = source["X"];
	        this.Y = source["Y"];
	        this.Scale = source["Scale"];
	        this.Label = source["Label"];
	        this.MapType = source["MapType"];
	        this.Adcode = source["Adcode"];
	        this.CityName = source["CityName"];
	    }
	}
	export class Video {
	    Md5: string;
	    RawMd5: string;
	
	    static createFrom(source: any = {}) {
	        return new Video(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Md5 = source["Md5"];
	        this.RawMd5 = source["RawMd5"];
	    }
	}
	export class MediaMsg {
	    XMLName: xml.Name;
	    Image: Image;
	    Video: Video;
	    App: App;
	    Emoji: Emoji;
	    Location: Location;
	
	    static createFrom(source: any = {}) {
	        return new MediaMsg(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.XMLName = this.convertValues(source["XMLName"], xml.Name);
	        this.Image = this.convertValues(source["Image"], Image);
	        this.Video = this.convertValues(source["Video"], Video);
	        this.App = this.convertValues(source["App"], App);
	        this.Emoji = this.convertValues(source["Emoji"], Emoji);
	        this.Location = this.convertValues(source["Location"], Location);
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
	
	
	export class RevokeMsg {
	    Content: string;
	    RevokeTime: number;
	
	    static createFrom(source: any = {}) {
	        return new RevokeMsg(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Content = source["Content"];
	        this.RevokeTime = source["RevokeTime"];
	    }
	}
	export class SysMsgTemplate {
	    ContentTemplate: ContentTemplate;
	
	    static createFrom(source: any = {}) {
	        return new SysMsgTemplate(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ContentTemplate = this.convertValues(source["ContentTemplate"], ContentTemplate);
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
	export class SysMsg {
	    Type: string;
	    DelChatRoomMember?: DelChatRoomMember;
	    SysMsgTemplate?: SysMsgTemplate;
	    RevokeMsg?: RevokeMsg;
	
	    static createFrom(source: any = {}) {
	        return new SysMsg(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Type = source["Type"];
	        this.DelChatRoomMember = this.convertValues(source["DelChatRoomMember"], DelChatRoomMember);
	        this.SysMsgTemplate = this.convertValues(source["SysMsgTemplate"], SysMsgTemplate);
	        this.RevokeMsg = this.convertValues(source["RevokeMsg"], RevokeMsg);
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
	export class JSONTime {
	
	
	    static createFrom(source: any = {}) {
	        return new JSONTime(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	
	    }
	}
	export class Message {
	    seq: number;
	    // Go type: JSONTime
	    time: any;
	    talker: string;
	    talkerName: string;
	    isChatRoom: boolean;
	    sender: string;
	    senderName: string;
	    isSelf: boolean;
	    type: number;
	    subType: number;
	    content: string;
	    isMentionMe: boolean;
	    contents?: Record<string, any>;
	    mediaMsg?: MediaMsg;
	    sysMsg?: SysMsg;
	
	    static createFrom(source: any = {}) {
	        return new Message(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.seq = source["seq"];
	        this.time = this.convertValues(source["time"], null);
	        this.talker = source["talker"];
	        this.talkerName = source["talkerName"];
	        this.isChatRoom = source["isChatRoom"];
	        this.sender = source["sender"];
	        this.senderName = source["senderName"];
	        this.isSelf = source["isSelf"];
	        this.type = source["type"];
	        this.subType = source["subType"];
	        this.content = source["content"];
	        this.isMentionMe = source["isMentionMe"];
	        this.contents = source["contents"];
	        this.mediaMsg = this.convertValues(source["mediaMsg"], MediaMsg);
	        this.sysMsg = this.convertValues(source["sysMsg"], SysMsg);
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

export namespace wechatdb {
	
	export class GetContactsResp {
	    total: number;
	    items: model.Contact[];
	
	    static createFrom(source: any = {}) {
	        return new GetContactsResp(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.total = source["total"];
	        this.items = this.convertValues(source["items"], model.Contact);
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
	export class GetMessagesResp {
	    total: number;
	    items: model.Message[];
	
	    static createFrom(source: any = {}) {
	        return new GetMessagesResp(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.total = source["total"];
	        this.items = this.convertValues(source["items"], model.Message);
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

export namespace xml {
	
	export class Name {
	    Space: string;
	    Local: string;
	
	    static createFrom(source: any = {}) {
	        return new Name(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Space = source["Space"];
	        this.Local = source["Local"];
	    }
	}

}

