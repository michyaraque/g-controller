export namespace app {
	
	export class AppInfo {
	    name: string;
	    version: string;
	    description: string;
	    author: string;
	
	    static createFrom(source: any = {}) {
	        return new AppInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.version = source["version"];
	        this.description = source["description"];
	        this.author = source["author"];
	    }
	}
	export class EngineStatus {
	    connected: boolean;
	    roomId: number;
	    hotel: string;
	    canEdit: boolean;
	    canRead: boolean;
	
	    static createFrom(source: any = {}) {
	        return new EngineStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.connected = source["connected"];
	        this.roomId = source["roomId"];
	        this.hotel = source["hotel"];
	        this.canEdit = source["canEdit"];
	        this.canRead = source["canRead"];
	    }
	}
	export class UserPosition {
	    id: number;
	    x: number;
	    y: number;
	    dir: number;
	
	    static createFrom(source: any = {}) {
	        return new UserPosition(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.x = source["x"];
	        this.y = source["y"];
	        this.dir = source["dir"];
	    }
	}

}

