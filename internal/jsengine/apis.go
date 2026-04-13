package jsengine

import (
	"crypto/rand"
	"fmt"
	"math"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/dop251/goja"
)

// resolvePromise crée une vraie Promise Goja résolue avec val.
func resolvePromise(vm *goja.Runtime, val interface{}) goja.Value {
	p, resolve, _ := vm.NewPromise()
	resolve(val)
	return vm.ToValue(p)
}

// rejectPromise crée une vraie Promise Goja rejetée.
func rejectPromise(vm *goja.Runtime) goja.Value {
	p, _, reject := vm.NewPromise()
	reject(vm.NewGoError(fmt.Errorf("HexaNaute: operation not permitted")))
	return vm.ToValue(p)
}

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
	loc.Set("reload", vm.ToValue(func(call goja.FunctionCall) goja.Value { return goja.Undefined() }))
	loc.DefineAccessorProperty("href",
		vm.ToValue(func(call goja.FunctionCall) goja.Value { return vm.ToValue(baseURL) }),
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
	history.Set("scrollRestoration", vm.ToValue("auto"))
	history.Set("state", goja.Null())

	// ── screen ──
	orientation := vm.NewObject()
	orientation.Set("type", vm.ToValue("landscape-primary"))
	orientation.Set("angle", vm.ToValue(0))
	orientation.Set("onchange", goja.Null())
	orientation.Set("addEventListener", func(c goja.FunctionCall) goja.Value { return goja.Undefined() })
	orientation.Set("removeEventListener", func(c goja.FunctionCall) goja.Value { return goja.Undefined() })
	orientation.Set("lock", func(c goja.FunctionCall) goja.Value { return rejectPromise(vm) })
	orientation.Set("unlock", func(c goja.FunctionCall) goja.Value { return goja.Undefined() })

	screen := vm.NewObject()
	screen.Set("width", vm.ToValue(1920))
	screen.Set("height", vm.ToValue(1080))
	screen.Set("availWidth", vm.ToValue(1920))
	screen.Set("availHeight", vm.ToValue(1040))
	screen.Set("availTop", vm.ToValue(0))
	screen.Set("availLeft", vm.ToValue(0))
	screen.Set("colorDepth", vm.ToValue(24))
	screen.Set("pixelDepth", vm.ToValue(24))
	screen.Set("orientation", orientation)

	// ── performance ──
	startTime := time.Now()
	perfObj := vm.NewObject()
	perfObj.Set("timeOrigin", vm.ToValue(float64(startTime.UnixNano())/1e6))
	perfObj.Set("now", func(call goja.FunctionCall) goja.Value {
		ms := float64(time.Since(startTime).Nanoseconds()) / 1e6
		return vm.ToValue(ms)
	})
	perfObj.Set("mark", func(call goja.FunctionCall) goja.Value { return goja.Undefined() })
	perfObj.Set("measure", func(call goja.FunctionCall) goja.Value { return goja.Undefined() })
	perfObj.Set("clearMarks", func(call goja.FunctionCall) goja.Value { return goja.Undefined() })
	perfObj.Set("clearMeasures", func(call goja.FunctionCall) goja.Value { return goja.Undefined() })
	perfObj.Set("getEntries", func(call goja.FunctionCall) goja.Value { return vm.ToValue([]interface{}{}) })
	perfObj.Set("getEntriesByName", func(call goja.FunctionCall) goja.Value { return vm.ToValue([]interface{}{}) })
	perfObj.Set("getEntriesByType", func(call goja.FunctionCall) goja.Value { return vm.ToValue([]interface{}{}) })
	perfObj.Set("setResourceTimingBufferSize", func(call goja.FunctionCall) goja.Value { return goja.Undefined() })
	perfObj.Set("clearResourceTimings", func(call goja.FunctionCall) goja.Value { return goja.Undefined() })
	perfObj.Set("addEventListener", func(call goja.FunctionCall) goja.Value { return goja.Undefined() })

	// ── crypto (vraie entropie) ──
	cryptoObj := vm.NewObject()
	cryptoObj.Set("getRandomValues", func(call goja.FunctionCall) goja.Value {
		arg := call.Argument(0)
		if goja.IsNull(arg) || goja.IsUndefined(arg) {
			return arg
		}
		obj, ok := arg.Export().(*goja.Object)
		if !ok {
			if obj2, ok2 := arg.(*goja.Object); ok2 {
				obj = obj2
			} else {
				return arg
			}
		}
		_ = obj
		// Accès par l'objet Goja directement
		jsObj := arg.ToObject(vm)
		if jsObj == nil {
			return arg
		}
		lengthVal := jsObj.Get("length")
		if goja.IsNull(lengthVal) || goja.IsUndefined(lengthVal) {
			return arg
		}
		length := int(lengthVal.ToInteger())
		if length <= 0 || length > 65536 {
			return arg
		}
		b := make([]byte, length)
		rand.Read(b) //nolint:errcheck
		for i, bv := range b {
			jsObj.Set(strconv.Itoa(i), vm.ToValue(int(bv)))
		}
		return arg
	})
	cryptoObj.Set("randomUUID", func(call goja.FunctionCall) goja.Value {
		b := make([]byte, 16)
		rand.Read(b) //nolint:errcheck
		b[6] = (b[6] & 0x0f) | 0x40 // version 4
		b[8] = (b[8] & 0x3f) | 0x80 // variant RFC 4122
		uuid := fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
			b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
		return vm.ToValue(uuid)
	})
	// crypto.subtle stub (WebCrypto)
	subtle := vm.NewObject()
	subtle.Set("digest", func(call goja.FunctionCall) goja.Value { return resolvePromise(vm, []byte{}) })
	subtle.Set("generateKey", func(call goja.FunctionCall) goja.Value { return resolvePromise(vm, vm.NewObject()) })
	subtle.Set("exportKey", func(call goja.FunctionCall) goja.Value { return resolvePromise(vm, []byte{}) })
	subtle.Set("importKey", func(call goja.FunctionCall) goja.Value { return resolvePromise(vm, vm.NewObject()) })
	subtle.Set("encrypt", func(call goja.FunctionCall) goja.Value { return resolvePromise(vm, []byte{}) })
	subtle.Set("decrypt", func(call goja.FunctionCall) goja.Value { return resolvePromise(vm, []byte{}) })
	subtle.Set("sign", func(call goja.FunctionCall) goja.Value { return resolvePromise(vm, []byte{}) })
	subtle.Set("verify", func(call goja.FunctionCall) goja.Value { return resolvePromise(vm, false) })
	subtle.Set("deriveKey", func(call goja.FunctionCall) goja.Value { return resolvePromise(vm, vm.NewObject()) })
	subtle.Set("deriveBits", func(call goja.FunctionCall) goja.Value { return resolvePromise(vm, []byte{}) })
	subtle.Set("wrapKey", func(call goja.FunctionCall) goja.Value { return resolvePromise(vm, []byte{}) })
	subtle.Set("unwrapKey", func(call goja.FunctionCall) goja.Value { return resolvePromise(vm, vm.NewObject()) })
	cryptoObj.Set("subtle", subtle)

	// ── speechSynthesis ──
	speech := vm.NewObject()
	speech.Set("pending", vm.ToValue(false))
	speech.Set("speaking", vm.ToValue(false))
	speech.Set("paused", vm.ToValue(false))
	speech.Set("onvoiceschanged", goja.Null())
	speech.Set("getVoices", func(call goja.FunctionCall) goja.Value { return vm.ToValue([]interface{}{}) })
	speech.Set("speak", func(call goja.FunctionCall) goja.Value { return goja.Undefined() })
	speech.Set("cancel", func(call goja.FunctionCall) goja.Value { return goja.Undefined() })
	speech.Set("pause", func(call goja.FunctionCall) goja.Value { return goja.Undefined() })
	speech.Set("resume", func(call goja.FunctionCall) goja.Value { return goja.Undefined() })
	speech.Set("addEventListener", func(call goja.FunctionCall) goja.Value { return goja.Undefined() })
	speech.Set("removeEventListener", func(call goja.FunctionCall) goja.Value { return goja.Undefined() })

	// ── window global ──
	window := vm.NewObject()
	window.Set("location", loc)
	window.Set("history", history)
	window.Set("screen", screen)
	window.Set("innerWidth", vm.ToValue(1280))
	window.Set("innerHeight", vm.ToValue(800))
	window.Set("outerWidth", vm.ToValue(1280))
	window.Set("outerHeight", vm.ToValue(800))
	window.Set("devicePixelRatio", vm.ToValue(1.0))
	window.Set("scrollX", vm.ToValue(0))
	window.Set("scrollY", vm.ToValue(0))
	window.Set("pageXOffset", vm.ToValue(0))
	window.Set("pageYOffset", vm.ToValue(0))
	window.Set("scrollTo", func(call goja.FunctionCall) goja.Value { return goja.Undefined() })
	window.Set("scrollBy", func(call goja.FunctionCall) goja.Value { return goja.Undefined() })
	window.Set("performance", perfObj)
	window.Set("crypto", cryptoObj)
	window.Set("speechSynthesis", speech)

	// Dialogs
	window.Set("alert", func(call goja.FunctionCall) goja.Value { return goja.Undefined() })
	window.Set("confirm", func(call goja.FunctionCall) goja.Value { return vm.ToValue(true) })
	window.Set("prompt", func(call goja.FunctionCall) goja.Value { return vm.ToValue("") })
	window.Set("open", func(call goja.FunctionCall) goja.Value { return goja.Null() })
	window.Set("close", func(call goja.FunctionCall) goja.Value { return goja.Undefined() })
	window.Set("print", func(call goja.FunctionCall) goja.Value { return goja.Undefined() })
	window.Set("stop", func(call goja.FunctionCall) goja.Value { return goja.Undefined() })

	// Event listeners
	window.Set("addEventListener", func(call goja.FunctionCall) goja.Value { return goja.Undefined() })
	window.Set("removeEventListener", func(call goja.FunctionCall) goja.Value { return goja.Undefined() })
	window.Set("dispatchEvent", func(call goja.FunctionCall) goja.Value { return vm.ToValue(true) })

	// matchMedia
	window.Set("matchMedia", func(call goja.FunctionCall) goja.Value {
		query := call.Argument(0).String()
		mql := vm.NewObject()
		matches := evaluateMediaQuery(query)
		mql.Set("matches", vm.ToValue(matches))
		mql.Set("media", vm.ToValue(query))
		mql.Set("onchange", goja.Null())
		mql.Set("addListener", func(c goja.FunctionCall) goja.Value { return goja.Undefined() })
		mql.Set("removeListener", func(c goja.FunctionCall) goja.Value { return goja.Undefined() })
		mql.Set("addEventListener", func(c goja.FunctionCall) goja.Value { return goja.Undefined() })
		mql.Set("removeEventListener", func(c goja.FunctionCall) goja.Value { return goja.Undefined() })
		mql.Set("dispatchEvent", func(c goja.FunctionCall) goja.Value { return vm.ToValue(true) })
		return mql
	})

	// window.self / window.top / window.parent
	window.Set("self", window)
	window.Set("top", window)
	window.Set("parent", window)
	window.Set("frames", window)
	window.Set("frameElement", goja.Null())
	window.Set("window", window)
	window.Set("length", vm.ToValue(0)) // no frames

	// Fetch/XHR
	setupFetchStub(vm, window)
	setupXHRStub(vm, window)

	// Event constructeurs
	vm.RunString(`
		function Event(type, options) {
			this.type = type || '';
			this.bubbles = !!(options && options.bubbles);
			this.cancelable = !!(options && options.cancelable);
			this.composed = !!(options && options.composed);
			this.defaultPrevented = false;
			this.isTrusted = false;
			this.timeStamp = Date.now();
			this.target = null;
			this.currentTarget = null;
			this.preventDefault = function() { this.defaultPrevented = true; };
			this.stopPropagation = function() {};
			this.stopImmediatePropagation = function() {};
		}
		function CustomEvent(type, options) {
			Event.call(this, type, options);
			this.detail = (options && options.detail !== undefined) ? options.detail : null;
		}
		CustomEvent.prototype = Object.create(Event.prototype);
		CustomEvent.prototype.constructor = CustomEvent;
		function ErrorEvent(type, options) {
			Event.call(this, type, options);
			this.message = (options && options.message) || '';
			this.error = (options && options.error) || null;
		}
		ErrorEvent.prototype = Object.create(Event.prototype);
	`) //nolint

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
	vm.Set("crypto", cryptoObj)
	vm.Set("speechSynthesis", speech)
	vm.Set("matchMedia", window.Get("matchMedia"))
}

// evaluateMediaQuery évalue basiquement une media query CSS.
func evaluateMediaQuery(query string) bool {
	q := strings.ToLower(strings.TrimSpace(query))
	switch {
	case strings.Contains(q, "prefers-color-scheme: dark"):
		return true
	case strings.Contains(q, "prefers-color-scheme: light"):
		return false
	case strings.Contains(q, "prefers-color-scheme"):
		return false
	case strings.Contains(q, "prefers-reduced-motion: reduce"):
		return false
	case strings.Contains(q, "max-width: 0"), strings.Contains(q, "max-width:0"):
		return false
	case strings.Contains(q, "min-width"):
		return true // écran 1920px
	case strings.Contains(q, "min-height"):
		return true
	case strings.Contains(q, "print"):
		return false
	case strings.Contains(q, "screen"):
		return true
	case q == "all", q == "(all)":
		return true
	}
	return false
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
	console.Set("dirxml", logFn("dirxml"))
	console.Set("time", func(call goja.FunctionCall) goja.Value { return goja.Undefined() })
	console.Set("timeEnd", func(call goja.FunctionCall) goja.Value { return goja.Undefined() })
	console.Set("timeLog", func(call goja.FunctionCall) goja.Value { return goja.Undefined() })
	console.Set("group", func(call goja.FunctionCall) goja.Value { return goja.Undefined() })
	console.Set("groupEnd", func(call goja.FunctionCall) goja.Value { return goja.Undefined() })
	console.Set("groupCollapsed", func(call goja.FunctionCall) goja.Value { return goja.Undefined() })
	console.Set("assert", func(call goja.FunctionCall) goja.Value { return goja.Undefined() })
	console.Set("count", func(call goja.FunctionCall) goja.Value { return goja.Undefined() })
	console.Set("countReset", func(call goja.FunctionCall) goja.Value { return goja.Undefined() })
	console.Set("clear", func(call goja.FunctionCall) goja.Value { return goja.Undefined() })
	console.Set("table", func(call goja.FunctionCall) goja.Value { return goja.Undefined() })
	console.Set("profile", func(call goja.FunctionCall) goja.Value { return goja.Undefined() })
	console.Set("profileEnd", func(call goja.FunctionCall) goja.Value { return goja.Undefined() })
	vm.Set("console", console)
}

// setupNavigator injecte navigator avec toutes les propriétés CreepJS.
func setupNavigator(vm *goja.Runtime) {
	nav := vm.NewObject()

	// Identité — cohérente avec le User-Agent
	nav.Set("userAgent", vm.ToValue("HexaNaute/0.4.0 (fr; souverain)"))
	nav.Set("appVersion", vm.ToValue("5.0 (compatible; HexaNaute/0.4.0)"))
	nav.Set("appName", vm.ToValue("Netscape")) // valeur standard pour compatibilité
	nav.Set("appCodeName", vm.ToValue("Mozilla"))
	nav.Set("platform", vm.ToValue("Linux x86_64"))
	nav.Set("product", vm.ToValue("Gecko"))
	nav.Set("productSub", vm.ToValue("20100101"))
	nav.Set("vendor", vm.ToValue("HexaRelay"))
	nav.Set("vendorSub", vm.ToValue(""))

	// Langue
	nav.Set("language", vm.ToValue("fr-FR"))
	nav.Set("languages", vm.ToValue([]string{"fr-FR", "fr", "en-US", "en"}))

	// Capacités matérielles
	nav.Set("hardwareConcurrency", vm.ToValue(4))
	nav.Set("deviceMemory", vm.ToValue(4)) // 4 GB
	nav.Set("maxTouchPoints", vm.ToValue(0))

	// Flags
	nav.Set("cookieEnabled", vm.ToValue(true))
	nav.Set("onLine", vm.ToValue(true))
	nav.Set("doNotTrack", vm.ToValue("1"))
	nav.Set("webdriver", vm.ToValue(false)) // CRITIQUE : doit être false, pas undefined

	// Plugins/MIME (vides pour un navigateur custom)
	nav.Set("plugins", vm.NewArray(0))
	nav.Set("mimeTypes", vm.NewArray(0))
	nav.Set("javaEnabled", func(call goja.FunctionCall) goja.Value { return vm.ToValue(false) })

	// Network Information API
	conn := vm.NewObject()
	conn.Set("effectiveType", vm.ToValue("4g"))
	conn.Set("rtt", vm.ToValue(50))
	conn.Set("downlink", vm.ToValue(10.0))
	conn.Set("downlinkMax", vm.ToValue(math.Inf(1)))
	conn.Set("saveData", vm.ToValue(false))
	conn.Set("type", vm.ToValue("wifi"))
	conn.Set("onchange", goja.Null())
	conn.Set("addEventListener", func(c goja.FunctionCall) goja.Value { return goja.Undefined() })
	conn.Set("removeEventListener", func(c goja.FunctionCall) goja.Value { return goja.Undefined() })
	nav.Set("connection", conn)
	nav.Set("mozConnection", goja.Undefined())
	nav.Set("webkitConnection", goja.Undefined())

	// User-Agent Client Hints (navigator.userAgentData)
	uaData := vm.NewObject()
	brands := []map[string]string{
		{"brand": "HexaNaute", "version": "4"},
		{"brand": "Not=A?Brand", "version": "99"},
	}
	uaData.Set("brands", vm.ToValue(brands))
	uaData.Set("mobile", vm.ToValue(false))
	uaData.Set("platform", vm.ToValue("Linux"))
	uaData.Set("getHighEntropyValues", func(call goja.FunctionCall) goja.Value {
		hints := map[string]interface{}{
			"architecture":    "x86",
			"bitness":         "64",
			"brands":          brands,
			"fullVersionList": []map[string]string{{"brand": "HexaNaute", "version": "0.4.0"}},
			"mobile":          false,
			"model":           "",
			"platform":        "Linux",
			"platformVersion": "6.1.0",
			"uaFullVersion":   "0.4.0",
			"wow64":           false,
		}
		return resolvePromise(vm, hints)
	})
	uaData.Set("toJSON", func(call goja.FunctionCall) goja.Value {
		return vm.ToValue(map[string]interface{}{
			"brands":  brands,
			"mobile":  false,
			"platform": "Linux",
		})
	})
	nav.Set("userAgentData", uaData)

	// Battery API
	nav.Set("getBattery", func(call goja.FunctionCall) goja.Value {
		battery := vm.NewObject()
		battery.Set("charging", true)
		battery.Set("chargingTime", 0)
		battery.Set("dischargingTime", math.Inf(1))
		battery.Set("level", 1.0)
		battery.Set("onchargingchange", goja.Null())
		battery.Set("onchargingtimechange", goja.Null())
		battery.Set("ondischargingtimechange", goja.Null())
		battery.Set("onlevelchange", goja.Null())
		battery.Set("addEventListener", func(c goja.FunctionCall) goja.Value { return goja.Undefined() })
		battery.Set("removeEventListener", func(c goja.FunctionCall) goja.Value { return goja.Undefined() })
		return resolvePromise(vm, battery)
	})

	// MediaDevices (caméra/micro refusés — vie privée)
	mediaDevices := vm.NewObject()
	mediaDevices.Set("enumerateDevices", func(call goja.FunctionCall) goja.Value {
		return resolvePromise(vm, []interface{}{})
	})
	mediaDevices.Set("getUserMedia", func(call goja.FunctionCall) goja.Value { return rejectPromise(vm) })
	mediaDevices.Set("getDisplayMedia", func(call goja.FunctionCall) goja.Value { return rejectPromise(vm) })
	mediaDevices.Set("getSupportedConstraints", func(call goja.FunctionCall) goja.Value {
		return vm.NewObject()
	})
	mediaDevices.Set("addEventListener", func(call goja.FunctionCall) goja.Value { return goja.Undefined() })
	mediaDevices.Set("removeEventListener", func(call goja.FunctionCall) goja.Value { return goja.Undefined() })
	nav.Set("mediaDevices", mediaDevices)

	// Geolocation (refusé)
	geo := vm.NewObject()
	geo.Set("getCurrentPosition", func(call goja.FunctionCall) goja.Value { return goja.Undefined() })
	geo.Set("watchPosition", func(call goja.FunctionCall) goja.Value { return vm.ToValue(-1) })
	geo.Set("clearWatch", func(call goja.FunctionCall) goja.Value { return goja.Undefined() })
	nav.Set("geolocation", geo)

	// Permissions (retourne une vraie Promise)
	perms := vm.NewObject()
	perms.Set("query", func(call goja.FunctionCall) goja.Value {
		status := vm.NewObject()
		status.Set("state", vm.ToValue("denied"))
		status.Set("onchange", goja.Null())
		status.Set("addEventListener", func(c goja.FunctionCall) goja.Value { return goja.Undefined() })
		return resolvePromise(vm, status)
	})
	nav.Set("permissions", perms)

	// Clipboard
	clipboard := vm.NewObject()
	clipboard.Set("readText", func(call goja.FunctionCall) goja.Value { return rejectPromise(vm) })
	clipboard.Set("writeText", func(call goja.FunctionCall) goja.Value { return rejectPromise(vm) })
	clipboard.Set("read", func(call goja.FunctionCall) goja.Value { return rejectPromise(vm) })
	clipboard.Set("write", func(call goja.FunctionCall) goja.Value { return rejectPromise(vm) })
	nav.Set("clipboard", clipboard)

	// serviceWorker (refusé)
	sw := vm.NewObject()
	sw.Set("register", func(call goja.FunctionCall) goja.Value { return rejectPromise(vm) })
	sw.Set("getRegistrations", func(call goja.FunctionCall) goja.Value {
		return resolvePromise(vm, []interface{}{})
	})
	sw.Set("addEventListener", func(call goja.FunctionCall) goja.Value { return goja.Undefined() })
	nav.Set("serviceWorker", sw)

	// Share API (refusé)
	nav.Set("share", func(call goja.FunctionCall) goja.Value { return rejectPromise(vm) })
	nav.Set("canShare", func(call goja.FunctionCall) goja.Value { return vm.ToValue(false) })

	// Divers
	nav.Set("sendBeacon", func(call goja.FunctionCall) goja.Value { return vm.ToValue(false) })
	nav.Set("vibrate", func(call goja.FunctionCall) goja.Value { return vm.ToValue(false) })
	nav.Set("registerProtocolHandler", func(call goja.FunctionCall) goja.Value { return goja.Undefined() })

	vm.Set("navigator", nav)
}

// setupStorage injecte localStorage et sessionStorage (en mémoire, pas persisté).
func setupStorage(vm *goja.Runtime) {
	makeStorage := func() *goja.Object {
		store := make(map[string]string)
		s := vm.NewObject()
		s.Set("setItem", func(call goja.FunctionCall) goja.Value {
			store[call.Argument(0).String()] = call.Argument(1).String()
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
		s.DefineAccessorProperty("length",
			vm.ToValue(func(call goja.FunctionCall) goja.Value { return vm.ToValue(len(store)) }),
			vm.ToValue(func(call goja.FunctionCall) goja.Value { return goja.Undefined() }),
			goja.FLAG_TRUE, goja.FLAG_TRUE,
		)
		return s
	}

	vm.Set("localStorage", makeStorage())
	vm.Set("sessionStorage", makeStorage())
	vm.Set("indexedDB", vm.NewObject())
	vm.Set("caches", vm.NewObject())
}

// setupTimers injecte setTimeout/setInterval/requestAnimationFrame.
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
		cancelled[int(call.Argument(0).ToInteger())] = true
		return goja.Undefined()
	})
	vm.Set("setInterval", func(call goja.FunctionCall) goja.Value {
		fn, ok := goja.AssertFunction(call.Argument(0))
		if !ok {
			return vm.ToValue(0)
		}
		return vm.ToValue(enqueue(fn, 0))
	})
	vm.Set("clearInterval", func(call goja.FunctionCall) goja.Value {
		cancelled[int(call.Argument(0).ToInteger())] = true
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
		cancelled[int(call.Argument(0).ToInteger())] = true
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
		cancelled[int(call.Argument(0).ToInteger())] = true
		return goja.Undefined()
	})
	vm.Set("queueMicrotask", func(call goja.FunctionCall) goja.Value {
		fn, ok := goja.AssertFunction(call.Argument(0))
		if !ok {
			return goja.Undefined()
		}
		enqueue(fn, 0)
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

// setupFetchStub injecte un fetch() sandbox (aucune vraie requête).
func setupFetchStub(vm *goja.Runtime, window *goja.Object) {
	fetchFn := vm.ToValue(func(call goja.FunctionCall) goja.Value {
		return rejectPromise(vm)
	})
	window.Set("fetch", fetchFn)
	vm.Set("fetch", fetchFn)
}

// setupXHRStub injecte XMLHttpRequest stub.
func setupXHRStub(vm *goja.Runtime, window *goja.Object) {
	vm.RunString(`
		function XMLHttpRequest() {
			this.readyState = 0;
			this.status = 0;
			this.statusText = '';
			this.responseText = '';
			this.response = null;
			this.responseType = '';
			this.responseURL = '';
			this.responseXML = null;
			this.timeout = 0;
			this.withCredentials = false;
			this.upload = {
				addEventListener: function() {},
				removeEventListener: function() {},
				onprogress: null, onerror: null, onabort: null, onload: null
			};
			this.onreadystatechange = null;
			this.onload = null;
			this.onerror = null;
			this.onabort = null;
			this.ontimeout = null;
			this.onprogress = null;
			this.onloadstart = null;
			this.onloadend = null;
			this.open = function(method, url, async) {};
			this.send = function(data) {
				if (typeof this.onerror === 'function') {
					try { this.onerror(new Error('HexaNaute: XHR sandboxed')); } catch(e) {}
				}
			};
			this.abort = function() {};
			this.setRequestHeader = function(header, value) {};
			this.getResponseHeader = function(header) { return null; };
			this.getAllResponseHeaders = function() { return ''; };
			this.overrideMimeType = function(mime) {};
			this.addEventListener = function(event, handler) {};
			this.removeEventListener = function(event, handler) {};
			this.dispatchEvent = function() { return true; };
		}
		XMLHttpRequest.UNSENT = 0;
		XMLHttpRequest.OPENED = 1;
		XMLHttpRequest.HEADERS_RECEIVED = 2;
		XMLHttpRequest.LOADING = 3;
		XMLHttpRequest.DONE = 4;
	`) //nolint
}

// ── helpers URL ──────────────────────────────────────────────────────────────

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
