export namespace main {
	
	export class ElabftwInfo {
	    raw: Record<string, any>;
	
	    static createFrom(source: any = {}) {
	        return new ElabftwInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.raw = source["raw"];
	    }
	}
	export class ElabftwInstance {
	    id: number;
	    siteUrl: string;
	    apiKey?: string;
	    verifyTls: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ElabftwInstance(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.siteUrl = source["siteUrl"];
	        this.apiKey = source["apiKey"];
	        this.verifyTls = source["verifyTls"];
	    }
	}
	export class Entry {
	    id: number;
	    title: string;
	    body: string;
	    createdAt: string;
	    modifiedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new Entry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.body = source["body"];
	        this.createdAt = source["createdAt"];
	        this.modifiedAt = source["modifiedAt"];
	    }
	}
	export class EntryRemoteLink {
	    localId: number;
	    instanceId: number;
	    siteUrl: string;
	    remoteId: number;
	    type: string;
	    url: string;
	
	    static createFrom(source: any = {}) {
	        return new EntryRemoteLink(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.localId = source["localId"];
	        this.instanceId = source["instanceId"];
	        this.siteUrl = source["siteUrl"];
	        this.remoteId = source["remoteId"];
	        this.type = source["type"];
	        this.url = source["url"];
	    }
	}
	export class EntrySummary {
	    id: number;
	    title: string;
	    createdAt: string;
	    modifiedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new EntrySummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.createdAt = source["createdAt"];
	        this.modifiedAt = source["modifiedAt"];
	    }
	}
	export class ProfileEntry {
	    uuid: string;
	    display_name?: string;
	    created_at: string;
	    salt?: string;
	    encrypted_verifier?: string;
	
	    static createFrom(source: any = {}) {
	        return new ProfileEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.uuid = source["uuid"];
	        this.display_name = source["display_name"];
	        this.created_at = source["created_at"];
	        this.salt = source["salt"];
	        this.encrypted_verifier = source["encrypted_verifier"];
	    }
	}
	export class ProfileIndex {
	    version: number;
	    profiles: ProfileEntry[];
	
	    static createFrom(source: any = {}) {
	        return new ProfileIndex(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.version = source["version"];
	        this.profiles = this.convertValues(source["profiles"], ProfileEntry);
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
	export class PushEntryResult {
	    localId: number;
	    remoteId: number;
	    action: string;
	    type: string;
	
	    static createFrom(source: any = {}) {
	        return new PushEntryResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.localId = source["localId"];
	        this.remoteId = source["remoteId"];
	        this.action = source["action"];
	        this.type = source["type"];
	    }
	}
	export class StoredUpload {
	    id: number;
	    realName: string;
	    longName: string;
	    storageName: string;
	    hash: string;
	    hashAlgorithm: string;
	    filesize: number;
	    state: string;
	    createdAt: string;
	    modifiedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new StoredUpload(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.realName = source["realName"];
	        this.longName = source["longName"];
	        this.storageName = source["storageName"];
	        this.hash = source["hash"];
	        this.hashAlgorithm = source["hashAlgorithm"];
	        this.filesize = source["filesize"];
	        this.state = source["state"];
	        this.createdAt = source["createdAt"];
	        this.modifiedAt = source["modifiedAt"];
	    }
	}

}

