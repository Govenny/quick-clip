export namespace internal {
	
	export class Config {
	    hotkey: string;
	    theme: string;
	    autoStart: boolean;
	    pasteDelay: number;
	
	    static createFrom(source: any = {}) {
	        return new Config(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.hotkey = source["hotkey"];
	        this.theme = source["theme"];
	        this.autoStart = source["autoStart"];
	        this.pasteDelay = source["pasteDelay"];
	    }
	}

}

