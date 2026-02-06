export namespace internal {
	
	export class AppearanceConfig {
	    opacity: number;
	
	    static createFrom(source: any = {}) {
	        return new AppearanceConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.opacity = source["opacity"];
	    }
	}
	export class ShortcutsConfig {
	    wakeUp: string[];
	
	    static createFrom(source: any = {}) {
	        return new ShortcutsConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.wakeUp = source["wakeUp"];
	    }
	}
	export class GeneralConfig {
	    launchAtLogin: boolean;
	
	    static createFrom(source: any = {}) {
	        return new GeneralConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.launchAtLogin = source["launchAtLogin"];
	    }
	}
	export class Config {
	    general: GeneralConfig;
	    shortcuts: ShortcutsConfig;
	    appearance: AppearanceConfig;
	
	    static createFrom(source: any = {}) {
	        return new Config(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.general = this.convertValues(source["general"], GeneralConfig);
	        this.shortcuts = this.convertValues(source["shortcuts"], ShortcutsConfig);
	        this.appearance = this.convertValues(source["appearance"], AppearanceConfig);
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

