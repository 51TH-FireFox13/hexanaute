package jsengine

import (
	"fmt"
	"net/url"

	"github.com/dop251/goja"
)

// setupWindow injecte l'objet window global.
func setupWindow(vm *goja.Runtime, baseURL string, result *ExecResult, allowRedirect bool) {
	parsed, _ := url.Parse(baseURL)

	// ── location ──
	loc := vm.NewObject()
	loc.Set("href", vm.ToValue(baseURL))
	loc.Set("protocol", vm.ToValue(hrefProtocol(parsed)))
	loc.Set("host", vm.ToValue(hrefHost(parsed)))
	loc.Set("hostname", vm.ToValue(hrefHostname(parsed)))
	loc.Set("port", vm.ToValue(hrefPort(parsed)))
	loc.Set("pathname", vm.ToValue(hrefPath(parsed)))
	loc.Set("search", vm.ToValue(hrefSearch(parsed)))
	loc.Set("hash", vm.ToValue(hrefHash(parsed)))
	loc.Set("origin", vm.ToValue(hrefOrigin(parsed)))
	loc.Set("assign", vm.ToValue(func(call goja.FunctionCall) goja.Value {
		if allowRedirect {
			result.Redirect = resolveURL(baseURL, call.Argument(0).String())
		}
		return goja.Undefined()
	}))
	loc.Set("replace", vm.ToValue(func(call goja.FunctionCall) goja.Value {
		if allowRedirect {
			result.Redirect = resolveURL(baseURL, call.Argument(0).String())
		}
		return goja.Undefined()
	}))
	loc.Set("reload", vm.ToValue(func(call goja.FunctionCall) goja.Value {
		return goja.Undefined()
	}))
	// Setter sur href → redirect
	loc.DefineAccessorProperty("href",
		vm.ToValue(func(call goja.FunctionCall) goja.Value {
			return vm.ToValue(baseURL)
		}),
		vm.ToValue(func(call goja.FunctionCall) goja.Value {
			if allowRedirect {
				target := call.Argument(0).String()
				if target != "" && target != "#" && target != baseURL {
					result.Redirect = resolveURL(baseURL, target)
				}
			}
			return goja.Undefined()
		}),
		goja.FLAG_TRUE, goja.FLAG_TRUE,
	)

	// ── history ──
	history := vm.NewObject()
	history.Set("pushState", func(call goja.FunctionCall) goja.Value { return goja.Undefined() })
	history.Set("replaceState", func(call goja.FunctionCall) goja.Value { return goja.Undefined() })
	history.Set("back", func(call goja.FunctionCall) goja.Value { return goja.Undefined() })
	history.Set("forward", func(call goja.FunctionCall) goja.Value { return goja.Undefined() })
	history.Set("go", func(call goja.FunctionCall) goja.Value { return goja.Undefined() })
	history.Set("length", vm.ToValue(1))

	// ── screen ──
	screen := vm.NewObject()
	screen.Set("width", vm.ToValue(1920))
	screen.Set("height", vm.ToValue(1080))
	screen.Set("availWidth", vm.ToValue(1920))
	screen.Set("availHeight", vm.ToValue(1040))
	screen.Set("colorDepth", vm.ToValue(24))
	screen.Set("pixelDepth", vm.ToValue(24))
	screen.Set("orientation", vm.NewObject())

	// ── window global ──
	window := vm.NewObject()
	window.Set("location", loc)
	window.Set("history", history)
	window.Set("screen", screen)
	window.Set("innerWidth", vm.ToValue(1280))
	window.Set("innerHeight", vm.ToValue(800))
	window.Set("outerWidth", vm.ToValue(1280))
	window.Set("outerHeight", vm.ToValue(800))
	window.Set("devicePixelRatio", vm.ToValue(1))
	window.Set("scrollX", vm.ToValue(0))
	window.Set("scrollY", vm.ToValue(0))
	window.Set("pageXOffset", vm.ToValue(0))
	window.Set("pageYOffset", vm.ToValue(0))
	window.Set("scrollTo", func(call goja.FunctionCall) goja.Value { return goja.Undefined() })
	window.Set("scrollBy", func(call goja.FunctionCall) goja.Value { return goja.Undefined() })

	// Dialogs — stubs
	window.Set("alert", func(call goja.FunctionCall) goja.Value { return goja.Undefined() })
	window.Set("confirm", func(call goja.FunctionCall) goja.Value { return vm.ToValue(true) })
	window.Set("prompt", func(call goja.FunctionCall) goja.Value { return vm.ToValue("") })
	window.Set("open", func(call goja.FunctionCall) goja.Value { return goja.Null() })
	window.Set("close", func(call goja.FunctionCall) goja.Value { return goja.Undefined() })
	window.Set("print", func(call goja.FunctionCall) goja.Value { return goja.Undefined() })

	// Event listeners (stubs)
	window.Set("addEventListener", func(call goja.FunctionCall) goja.Value { return goja.Undefined() })
	window.Set("removeEventListener", func(call goja.FunctionCall) goja.Value { return goja.Undefined() })
	window.Set("dispatchEvent", func(call goja.FunctionCall) goja.Value { return vm.ToValue(true) })

	// Performance
	perfObj := vm.NewObject()
	perfObj.Set("now", func(call goja.FunctionCall) goja.Value { return vm.ToValue(0) })
	perfObj.Set("mark", func(call goja.FunctionCall) goja.Value { return goja.Undefined() })
	perfObj.Set("measure", func(call goja.FunctionCall) goja.Value { return goja.Undefined() })
	window.Set("performance", perfObj)

	// window.self / window.top / window.parent
	window.Set("self", window)
	window.Set("top", window)
	window.Set("parent", window)
	window.Set("frames", window)
	window.Set("frameElement", goja.Null())
	window.Set("window", window)

	// Fetch/XHR (stubs sécurisés — pas d'accès réseau depuis JS)
	setupFetchStub(vm, window)
	setupXHRStub(vm, window)

	// window.crypto (stub)
	cryptoObj := vm.NewObject()
	cryptoObj.Set("getRandomValues", func(call goja.FunctionCall) goja.Value {
		return call.Argument(0) // retourne le tableau sans le remplir
	})
	cryptoObj.Set("randomUUID", func(call goja.FunctionCall) goja.Value {
		return vm.ToValue("00000000-0000-0000-0000-000000000000")
	})
	window.Set("crypto", cryptoObj)
	vm.Set("crypto", cryptoObj)

	// CustomEvent / Event constructeurs
	vm.RunString(`
		function Event(type, options) {
			this.type = type || '';
			this.bubbles = (options && options.bubbles) || false;
			this.cancelable = (options && options.cancelable) || false;
			this.defaultPrevented = false;
			this.preventDefault = function() { this.defaultPrevented = true; };
			this.stopPropagation = function() {};
			this.stopImmediatePropagation = function() {};
		}
		function CustomEvent(type, options) {
			Event.call(this, type, options);
			this.detail = (options && options.detail) || null;
		}
		CustomEvent.prototype = Object.create(Event.prototype);
	`) // nolint

	vm.Set("window", window)
	vm.Set("self", window)
	vm.Set("top", window)
	vm.Set("parent", window)

	// Globals depuis window
	vm.Set("location", loc)
	vm.Set("history", history)
	vm.Set("screen", screen)
	vm.Set("alert", window.Get("alert"))
	vm.Set("confirm", window.Get("confirm"))
	vm.Set("prompt", window.Get("prompt"))
	vm.Set("open", window.Get("open"))
	vm.Set("close", window.Get("close"))
	vm.Set("scrollTo", window.Get("scrollTo"))
	vm.Set("scrollBy", window.Get("scrollBy"))
	vm.Set("performance", perfObj)
}

// setupConsole injecte l'objet console.
func setupConsole(vm *goja.Runtime, debug bool) {
	console := vm.NewObject()
	logFn := func(prefix string) func(goja.FunctionCall) goja.Value {
		return func(call goja.FunctionCall) goja.Value {
			if debug {
				args := make([]interface{}, len(call.Arguments))
				for i, a := range call.Arguments {
					args[i] = a
				}
				fmt.Printf("[JS %s]", prefix)
				for _, a := range args {
					fmt.Printf(" %v", a)
				}
				fmt.Println()
			}
			return goja.Undefined()
		}
	}
	console.Set("log", logFn("log"))
	console.Set("warn", logFn("warn"))
	console.Set("error", logFn("error"))
	console.Set("info", logFn("info"))
	console.Set("debug", logFn("debug"))
	console.Set("trace", logFn("trace"))
	console.Set("dir", logFn("dir"))
	console.Set("time", func(call goja.FunctionCall) goja.Value { return goja.Undefined() })
	console.Set("timeEnd", func(call goja.FunctionCall) goja.Value { return goja.Undefined() })
	console.Set("group", func(call goja.FunctionCall) goja.Value { return goja.Undefined() })
	console.Set("groupEnd", func(call goja.FunctionCall) goja.Value { return goja.Undefined() })
	console.Set("groupCollapsed", func(call goja.FunctionCall) goja.Value { return goja.Undefined() })
	console.Set("assert", func(call goja.FunctionCall) goja.Value { return goja.Undefined() })
	console.Set("count", func(call goja.FunctionCall) goja.Value { return goja.Undefined() })
	console.Set("clear", func(call goja.FunctionCall) goja.Value { return goja.Undefined() })

	vm.Set("console", console)
}

// setupNavigator injecte l'objet navigator.
func setupNavigator(vm *goja.Runtime) {
	nav := vm.NewObject()
	nav.Set("userAgent", vm.ToValue("HexaNaute/0.4.0 (fr; souverain)"))
	nav.Set("appVersion", vm.ToValue("5.0 (compatible)"))
	nav.Set("appName", vm.ToValue("HexaNaute"))
	nav.Set("platform", vm.ToValue("Linux x86_64"))
	nav.Set("language", vm.ToValue("fr-FR"))
	nav.Set("languages", vm.ToValue([]string{"fr-FR", "fr", "en"}))
	nav.Set("onLine", vm.ToValue(true))
	nav.Set("cookieEnabled", vm.ToValue(true))
	nav.Set("doNotTrack", vm.ToValue("1"))
	nav.Set("hardwareConcurrency", vm.ToValue(4))
	nav.Set("maxTouchPoints", vm.ToValue(0))
	nav.Set("vendor", vm.ToValue("HexaNaute"))
	nav.Set("vendorSub", vm.ToValue(""))
	nav.Set("product", vm.ToValue("Gecko")) // pour compatibilité
	nav.Set("productSub", vm.ToValue("20100101"))

	// Plugins vides (bloque le fingerprinting)
	nav.Set("plugins", vm.NewArray(0))
	nav.Set("mimeTypes", vm.NewArray(0))
	nav.Set("javaEnabled", func(call goja.FunctionCall) goja.Value { return vm.ToValue(false) })

	// Geolocation stub
	geo := vm.NewObject()
	geo.Set("getCurrentPosition", func(call goja.FunctionCall) goja.Value { return goja.Undefined() })
	geo.Set("watchPosition", func(call goja.FunctionCall) goja.Value { return vm.ToValue(-1) })
	geo.Set("clearWatch", func(call goja.FunctionCall) goja.Value { return goja.Undefined() })
	nav.Set("geolocation", geo)

	// Permissions stub
	perms := vm.NewObject()
	perms.Set("query", func(call goja.FunctionCall) goja.Value {
		return vm.ToValue(map[string]interface{}{"state": "denied"})
	})
	nav.Set("permissions", perms)

	// Clipboard stub
	clipboard := vm.NewObject()
	clipboard.Set("readText", func(call goja.FunctionCall) goja.Value {
		return vm.ToValue(map[string]interface{}{"then": func(call goja.FunctionCall) goja.Value { return goja.Undefined() }})
	})
	clipboard.Set("writeText", func(call goja.FunctionCall) goja.Value {
		return vm.ToValue(map[string]interface{}{"then": func(call goja.FunctionCall) goja.Value { return goja.Undefined() }})
	})
	nav.Set("clipboard", clipboard)

	// serviceWorker stub
	sw := vm.NewObject()
	sw.Set("register", func(call goja.FunctionCall) goja.Value { return vm.ToValue(rejectPromise(vm)) })
	nav.Set("serviceWorker", sw)

	vm.Set("navigator", nav)
}

// setupStorage injecte localStorage et sessionStorage (en mémoire, pas persisté).
func setupStorage(vm *goja.Runtime) {
	makeStorage := func() *goja.Object {
		store := make(map[string]string)
		s := vm.NewObject()

		s.Set("setItem", func(call goja.FunctionCall) goja.Value {
			key := call.Argument(0).String()
			val := call.Argument(1).String()
			store[key] = val
			return goja.Undefined()
		})
		s.Set("getItem", func(call goja.FunctionCall) goja.Value {
			key := call.Argument(0).String()
			if v, ok := store[key]; ok {
				return vm.ToValue(v)
			}
			return goja.Null()
		})
		s.Set("removeItem", func(call goja.FunctionCall) goja.Value {
			delete(store, call.Argument(0).String())
			return goja.Undefined()
		})
		s.Set("clear", func(call goja.FunctionCall) goja.Value {
			store = make(map[string]string)
			return goja.Undefined()
		})
		s.DefineAccessorProperty("length",
			vm.ToValue(func(call goja.FunctionCall) goja.Value {
				return vm.ToValue(len(store))
			}),
			vm.ToValue(func(call goja.FunctionCall) goja.Value {
				return goja.Undefined()
			}),
			goja.FLAG_TRUE, goja.FLAG_TRUE,
		)
		s.Set("key", func(call goja.FunctionCall) goja.Value {
			idx := int(call.Argument(0).ToInteger())
			i := 0
			for k := range store {
				if i == idx {
					return vm.ToValue(k)
				}
				i++
			}
			return goja.Null()
		})
		return s
	}

	vm.Set("localStorage", makeStorage())
	vm.Set("sessionStorage", makeStorage())
	vm.Set("indexedDB", vm.NewObject()) // stub vide
}

// setupTimers injecte setTimeout/setInterval/requestAnimationFrame.
// Retourne une fonction runTimers à appeler après l'exécution des scripts principaux.
func setupTimers(vm *goja.Runtime, maxCallbacks int) func(vm *goja.Runtime) {
	type pendingTimer struct {
		id        int
		fn        goja.Callable
		delay     int64
		cancelled bool
	}

	var queue []*pendingTimer
	var nextID int
	var cancelled = make(map[int]bool)

	enqueue := func(fn goja.Callable, delay int64) int {
		nextID++
		id := nextID
		queue = append(queue, &pendingTimer{id: id, fn: fn, delay: delay})
		return id
	}

	vm.Set("setTimeout", func(call goja.FunctionCall) goja.Value {
		fn, ok := goja.AssertFunction(call.Argument(0))
		if !ok {
			return vm.ToValue(0)
		}
		delay := int64(0)
		if len(call.Arguments) > 1 {
			delay = call.Argument(1).ToInteger()
		}
		return vm.ToValue(enqueue(fn, delay))
	})

	vm.Set("clearTimeout", func(call goja.FunctionCall) goja.Value {
		id := int(call.Argument(0).ToInteger())
		cancelled[id] = true
		return goja.Undefined()
	})

	vm.Set("setInterval", func(call goja.FunctionCall) goja.Value {
		fn, ok := goja.AssertFunction(call.Argument(0))
		if !ok {
			return vm.ToValue(0)
		}
		// En sandbox, setInterval s'exécute une seule fois
		return vm.ToValue(enqueue(fn, 0))
	})

	vm.Set("clearInterval", func(call goja.FunctionCall) goja.Value {
		id := int(call.Argument(0).ToInteger())
		cancelled[id] = true
		return goja.Undefined()
	})

	vm.Set("requestAnimationFrame", func(call goja.FunctionCall) goja.Value {
		fn, ok := goja.AssertFunction(call.Argument(0))
		if !ok {
			return vm.ToValue(0)
		}
		return vm.ToValue(enqueue(fn, 0))
	})
	vm.Set("cancelAnimationFrame", func(call goja.FunctionCall) goja.Value {
		id := int(call.Argument(0).ToInteger())
		cancelled[id] = true
		return goja.Undefined()
	})
	vm.Set("requestIdleCallback", func(call goja.FunctionCall) goja.Value {
		fn, ok := goja.AssertFunction(call.Argument(0))
		if !ok {
			return vm.ToValue(0)
		}
		return vm.ToValue(enqueue(fn, 0))
	})
	vm.Set("cancelIdleCallback", func(call goja.FunctionCall) goja.Value {
		return goja.Undefined()
	})

	return func(vm *goja.Runtime) {
		count := 0
		for _, t := range queue {
			if count >= maxCallbacks {
				break
			}
			if cancelled[t.id] || t.delay > 200 {
				continue
			}
			_, err := t.fn(goja.Undefined())
			if err != nil && isInterrupt(err) {
				break
			}
			count++
		}
	}
}

// setupFetchStub injecte un fetch() qui ne fait aucune vraie requête.
func setupFetchStub(vm *goja.Runtime, window *goja.Object) {
	// fetch() retourne une Promise rejetée (pas d'accès réseau depuis JS en v0.2.0)
	fetchFn := vm.ToValue(func(call goja.FunctionCall) goja.Value {
		return rejectPromise(vm)
	})
	window.Set("fetch", fetchFn)
	vm.Set("fetch", fetchFn)
}

// setupXHRStub injecte un XMLHttpRequest stub qui ne fait rien.
func setupXHRStub(vm *goja.Runtime, window *goja.Object) {
	vm.RunString(`
		function XMLHttpRequest() {
			this.readyState = 0;
			this.status = 0;
			this.statusText = '';
			this.responseText = '';
			this.response = '';
			this.responseURL = '';
			this.onreadystatechange = null;
			this.onload = null;
			this.onerror = null;
			this.open = function(method, url, async) {};
			this.send = function(data) {};
			this.setRequestHeader = function(header, value) {};
			this.getResponseHeader = function(header) { return null; };
			this.getAllResponseHeaders = function() { return ''; };
			this.abort = function() {};
			this.addEventListener = function(event, handler) {};
			this.removeEventListener = function(event, handler) {};
			this.upload = { addEventListener: function() {}, onprogress: null };
			this.withCredentials = false;
		}
		XMLHttpRequest.UNSENT = 0;
		XMLHttpRequest.OPENED = 1;
		XMLHttpRequest.HEADERS_RECEIVED = 2;
		XMLHttpRequest.LOADING = 3;
		XMLHttpRequest.DONE = 4;
	`) // nolint
}

// rejectPromise crée une Promise immédiatement rejetée.
func rejectPromise(vm *goja.Runtime) goja.Value {
	p, err := vm.RunString(`(function() {
		return {
			then: function(res, rej) { if(rej) try { rej(new Error('HexaNaute: network access from JS is sandboxed')); } catch(e) {} return this; },
			catch: function(fn) { try { fn(new Error('HexaNaute: network access from JS is sandboxed')); } catch(e) {} return this; },
			finally: function(fn) { if(fn) try { fn(); } catch(e) {} return this; }
		};
	})()`)
	if err != nil {
		return goja.Undefined()
	}
	return p
}

// ── helpers URL ──

func hrefProtocol(u *url.URL) string {
	if u == nil {
		return "https:"
	}
	return u.Scheme + ":"
}

func hrefHost(u *url.URL) string {
	if u == nil {
		return ""
	}
	return u.Host
}

func hrefHostname(u *url.URL) string {
	if u == nil {
		return ""
	}
	return u.Hostname()
}

func hrefPort(u *url.URL) string {
	if u == nil {
		return ""
	}
	return u.Port()
}

func hrefPath(u *url.URL) string {
	if u == nil {
		return "/"
	}
	if u.Path == "" {
		return "/"
	}
	return u.Path
}

func hrefSearch(u *url.URL) string {
	if u == nil || u.RawQuery == "" {
		return ""
	}
	return "?" + u.RawQuery
}

func hrefHash(u *url.URL) string {
	if u == nil || u.Fragment == "" {
		return ""
	}
	return "#" + u.Fragment
}

func hrefOrigin(u *url.URL) string {
	if u == nil {
		return ""
	}
	return u.Scheme + "://" + u.Host
}

func resolveURL(base, ref string) string {
	if ref == "" {
		return base
	}
	baseURL, err := url.Parse(base)
	if err != nil {
		return ref
	}
	refURL, err := url.Parse(ref)
	if err != nil {
		return ref
	}
	return baseURL.ResolveReference(refURL).String()
}
