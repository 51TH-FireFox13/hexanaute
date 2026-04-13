package jsengine

import "github.com/dop251/goja"

// setupCreepJSDefenses injecte les stubs anti-fingerprinting pour les APIs
// analysées par CreepJS : Canvas, WebGL, Audio, Workers, Observers, WebRTC, etc.
// Doit être appelée APRÈS setupDocument (accès à `document`).
func setupCreepJSDefenses(vm *goja.Runtime) {
	vm.RunString(`(function(global) {
		'use strict';

		// ── Canvas stub ──────────────────────────────────────────────────────
		// PNG 1x1 transparent déterministe — empreinte stable et non-unique
		var FAKE_PNG = 'data:image/png;base64,' +
			'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAAC0lEQVQI12NgAAIABQ' +
			'AABjE+ibYAAAAASUVORK5CYII=';

		function makeGradient() {
			return { addColorStop: function() {} };
		}

		function make2DContext(canvas) {
			var state = {
				fillStyle: '#000000', strokeStyle: '#000000',
				font: '10px sans-serif', globalAlpha: 1, lineWidth: 1,
				textBaseline: 'alphabetic', textAlign: 'start',
				shadowColor: 'rgba(0,0,0,0)', shadowBlur: 0,
				lineCap: 'butt', lineJoin: 'miter', miterLimit: 10
			};
			var ctx = {
				canvas: canvas,
				get fillStyle() { return state.fillStyle; },
				set fillStyle(v) { state.fillStyle = v; },
				get strokeStyle() { return state.strokeStyle; },
				set strokeStyle(v) { state.strokeStyle = v; },
				get font() { return state.font; },
				set font(v) { state.font = v; },
				get globalAlpha() { return state.globalAlpha; },
				set globalAlpha(v) { state.globalAlpha = v; },
				get lineWidth() { return state.lineWidth; },
				set lineWidth(v) { state.lineWidth = v; },
				get textBaseline() { return state.textBaseline; },
				set textBaseline(v) { state.textBaseline = v; },
				get textAlign() { return state.textAlign; },
				set textAlign(v) { state.textAlign = v; },

				fillRect: function() {},
				clearRect: function() {},
				strokeRect: function() {},
				fillText: function() {},
				strokeText: function() {},
				measureText: function(text) {
					var len = text ? text.length : 0;
					return {
						width: len * 6.5,
						actualBoundingBoxAscent: 10,
						actualBoundingBoxDescent: 2,
						actualBoundingBoxLeft: 0,
						actualBoundingBoxRight: len * 6.5,
						fontBoundingBoxAscent: 12,
						fontBoundingBoxDescent: 3
					};
				},
				beginPath: function() {},
				closePath: function() {},
				moveTo: function() {},
				lineTo: function() {},
				arc: function() {},
				arcTo: function() {},
				ellipse: function() {},
				bezierCurveTo: function() {},
				quadraticCurveTo: function() {},
				rect: function() {},
				fill: function() {},
				stroke: function() {},
				clip: function() {},
				isPointInPath: function() { return false; },
				isPointInStroke: function() { return false; },
				save: function() {},
				restore: function() {},
				scale: function() {},
				rotate: function() {},
				translate: function() {},
				transform: function() {},
				setTransform: function() {},
				resetTransform: function() {},
				createLinearGradient: function() { return makeGradient(); },
				createRadialGradient: function() { return makeGradient(); },
				createConicGradient: function() { return makeGradient(); },
				createPattern: function() { return null; },
				drawImage: function() {},
				getImageData: function(x, y, w, h) {
					var len = Math.max(1, w) * Math.max(1, h) * 4;
					return {
						data: new Uint8ClampedArray(len),
						width: Math.max(1, w),
						height: Math.max(1, h),
						colorSpace: 'srgb'
					};
				},
				putImageData: function() {},
				createImageData: function(w, h) {
					var len = Math.max(1, w) * Math.max(1, h) * 4;
					return {
						data: new Uint8ClampedArray(len),
						width: Math.max(1, w),
						height: Math.max(1, h)
					};
				},
				setLineDash: function() {},
				getLineDash: function() { return []; },
				setLineDashOffset: function() {},
				drawFocusIfNeeded: function() {},
				scrollPathIntoView: function() {},
				addEventListener: function() {},
				removeEventListener: function() {}
			};
			return ctx;
		}

		function makeWebGL1Stub(canvas) {
			var EXT_DEBUG = { UNMASKED_VENDOR_WEBGL: 37445, UNMASKED_RENDERER_WEBGL: 37446 };
			return {
				canvas: canvas,
				drawingBufferWidth: 300,
				drawingBufferHeight: 150,
				getParameter: function(p) {
					switch(p) {
						case 37445: return 'HexaRelay';
						case 37446: return 'HexaNaute Renderer 1.0';
						case 7936:  return 'HexaRelay';
						case 7937:  return 'HexaNaute Renderer 1.0';
						case 7938:  return 'WebGL 1.0 (HexaNaute/0.4.0)';
						case 35724: return 'WebGL GLSL ES 1.0 (HexaNaute)';
						case 3379:  return 16384;
						case 34921: return 16;
						case 36347: return 1024;
						case 36348: return 16;
						case 34076: return 16384;
						case 3386:  return new Int32Array([16384, 16384]);
						default:    return null;
					}
				},
				getExtension: function(name) {
					if (name === 'WEBGL_debug_renderer_info') return EXT_DEBUG;
					if (name === 'OES_standard_derivatives') return {};
					if (name === 'OES_texture_float') return {};
					return null;
				},
				getSupportedExtensions: function() {
					return ['WEBGL_debug_renderer_info','OES_standard_derivatives','OES_texture_float'];
				},
				getShaderPrecisionFormat: function() {
					return { rangeMin: 127, rangeMax: 127, precision: 23 };
				},
				createBuffer: function() { return {}; },
				createTexture: function() { return {}; },
				createProgram: function() { return {}; },
				createShader: function() { return {}; },
				createRenderbuffer: function() { return {}; },
				createFramebuffer: function() { return {}; },
				bindBuffer: function() {}, bindTexture: function() {},
				bindRenderbuffer: function() {}, bindFramebuffer: function() {},
				bufferData: function() {},
				enable: function() {}, disable: function() {},
				viewport: function() {},
				clear: function() {}, clearColor: function() {}, clearDepth: function() {},
				flush: function() {}, finish: function() {},
				isContextLost: function() { return false; },
				readPixels: function() {},
				deleteBuffer: function() {}, deleteTexture: function() {},
				deleteProgram: function() {}, deleteShader: function() {},
				attachShader: function() {}, compileShader: function() {},
				linkProgram: function() {}, useProgram: function() {},
				shaderSource: function() {}, getShaderInfoLog: function() { return ''; },
				getProgramInfoLog: function() { return ''; },
				getShaderParameter: function() { return true; },
				getProgramParameter: function() { return true; },
				getAttribLocation: function() { return -1; },
				getUniformLocation: function() { return null; },
				vertexAttribPointer: function() {}, enableVertexAttribArray: function() {},
				drawArrays: function() {}, drawElements: function() {},
				texImage2D: function() {}, texParameteri: function() {},
				generateMipmap: function() {}, pixelStorei: function() {}
			};
		}

		function makeWebGL2Stub(canvas) {
			var ctx = makeWebGL1Stub(canvas);
			ctx.getParameter = function(p) {
				if (p === 7938) return 'WebGL 2.0 (HexaNaute/0.4.0)';
				if (p === 35724) return 'WebGL GLSL ES 3.00 (HexaNaute)';
				return makeWebGL1Stub(canvas).getParameter(p);
			};
			ctx.createVertexArray = function() { return {}; };
			ctx.bindVertexArray = function() {};
			ctx.deleteVertexArray = function() {};
			return ctx;
		}

		function makeCanvasStub() {
			var self = {
				nodeName: 'CANVAS',
				tagName: 'CANVAS',
				nodeType: 1,
				width: 300,
				height: 150,
				style: { width: '', height: '', display: '', visibility: '' },
				getContext: function(type, opts) {
					if (type === '2d') return make2DContext(self);
					if (type === 'webgl' || type === 'experimental-webgl')
						return makeWebGL1Stub(self);
					if (type === 'webgl2') return makeWebGL2Stub(self);
					return null;
				},
				toDataURL: function(type, quality) { return FAKE_PNG; },
				toBlob: function(cb, type, quality) {
					if (typeof cb === 'function') setTimeout(function() { cb(null); }, 0);
				},
				getBoundingClientRect: function() {
					return { top:0, left:0, bottom:150, right:300,
					         width:300, height:150, x:0, y:0,
					         toJSON: function() { return this; } };
				},
				getBoundingRect: function() { return this.getBoundingClientRect(); },
				setAttribute: function(name, val) {
					if (name === 'width') self.width = parseInt(val) || 300;
					if (name === 'height') self.height = parseInt(val) || 150;
				},
				getAttribute: function(name) {
					if (name === 'width') return String(self.width);
					if (name === 'height') return String(self.height);
					return null;
				},
				addEventListener: function() {},
				removeEventListener: function() {},
				dispatchEvent: function() { return true; },
				parentNode: null,
				ownerDocument: (typeof document !== 'undefined') ? document : null
			};
			return self;
		}

		// Patch document.createElement pour retourner le stub canvas
		if (typeof document !== 'undefined' && document && typeof document.createElement === 'function') {
			var _origCreate = document.createElement.bind(document);
			document.createElement = function(tag, opts) {
				if (typeof tag === 'string' && tag.toLowerCase() === 'canvas') {
					return makeCanvasStub();
				}
				return _origCreate(tag, opts);
			};
		}

		// HTMLCanvasElement stub global
		global.HTMLCanvasElement = {
			prototype: {
				toDataURL: function() { return FAKE_PNG; },
				getContext: function(type) {
					if (type === '2d') return make2DContext(this);
					return null;
				}
			}
		};

		// ── AudioContext / OfflineAudioContext ────────────────────────────────
		var AUDIO_FP = 0.000122480392456055; // valeur déterministe stable

		function AudioBuffer(opts) {
			opts = opts || {};
			this.sampleRate = opts.sampleRate || 44100;
			this.length = opts.length || 4096;
			this.numberOfChannels = opts.numberOfChannels || 1;
			this.duration = this.length / this.sampleRate;
			this._data = null;
		}
		AudioBuffer.prototype.getChannelData = function(channel) {
			if (!this._data) {
				this._data = new Float32Array(this.length);
				this._data[0] = AUDIO_FP; // fingerprint déterministe
			}
			return this._data;
		};
		AudioBuffer.prototype.copyFromChannel = function() {};
		AudioBuffer.prototype.copyToChannel = function() {};
		global.AudioBuffer = AudioBuffer;

		function makeAudioParam(defaultValue) {
			return {
				value: defaultValue || 0,
				defaultValue: defaultValue || 0,
				minValue: -3.4028234663852886e+38,
				maxValue:  3.4028234663852886e+38,
				setValueAtTime: function() { return this; },
				linearRampToValueAtTime: function() { return this; },
				exponentialRampToValueAtTime: function() { return this; },
				setTargetAtTime: function() { return this; },
				setValueCurveAtTime: function() { return this; },
				cancelScheduledValues: function() { return this; },
				cancelAndHoldAtCurrentValue: function() { return this; }
			};
		}

		function makeAudioNodeBase(ctx) {
			return {
				context: ctx,
				numberOfInputs: 0,
				numberOfOutputs: 1,
				channelCount: 2,
				channelCountMode: 'max',
				channelInterpretation: 'speakers',
				connect: function(dest) { return dest; },
				disconnect: function() {},
				addEventListener: function() {},
				removeEventListener: function() {},
				dispatchEvent: function() { return true; }
			};
		}

		function makeOscillator(ctx) {
			var node = makeAudioNodeBase(ctx);
			node.type = 'sine';
			node.frequency = makeAudioParam(440);
			node.detune = makeAudioParam(0);
			node.start = function() {};
			node.stop = function() {};
			node.onended = null;
			return node;
		}

		function makeDynamicsCompressor(ctx) {
			var node = makeAudioNodeBase(ctx);
			node.threshold = makeAudioParam(-24);
			node.knee = makeAudioParam(30);
			node.ratio = makeAudioParam(12);
			node.attack = makeAudioParam(0.003);
			node.release = makeAudioParam(0.25);
			node.reduction = 0;
			return node;
		}

		function makeGainNode(ctx) {
			var node = makeAudioNodeBase(ctx);
			node.gain = makeAudioParam(1);
			return node;
		}

		function makeAnalyser(ctx) {
			var node = makeAudioNodeBase(ctx);
			node.fftSize = 2048;
			node.frequencyBinCount = 1024;
			node.minDecibels = -100;
			node.maxDecibels = -30;
			node.smoothingTimeConstant = 0.8;
			node.getFloatFrequencyData = function(arr) {};
			node.getByteFrequencyData = function(arr) {};
			node.getFloatTimeDomainData = function(arr) {
				if (arr && arr.length > 0) arr[0] = AUDIO_FP;
			};
			node.getByteTimeDomainData = function(arr) {};
			return node;
		}

		function makeBufferSource(ctx) {
			var node = makeAudioNodeBase(ctx);
			node.buffer = null;
			node.loop = false;
			node.loopStart = 0;
			node.loopEnd = 0;
			node.playbackRate = makeAudioParam(1);
			node.detune = makeAudioParam(0);
			node.start = function() {};
			node.stop = function() {};
			node.onended = null;
			return node;
		}

		function makeOfflineCtx(numChannels, length, sampleRate) {
			var ctx = {
				sampleRate: sampleRate || 44100,
				length: length || 4096,
				numberOfChannels: numChannels || 1,
				currentTime: 0,
				state: 'suspended',
				destination: { channelCount: numChannels || 1,
				               connect: function() {}, disconnect: function() {},
				               addEventListener: function() {} },
				oncomplete: null,
				onstatechange: null,
				createOscillator: function() { return makeOscillator(ctx); },
				createDynamicsCompressor: function() { return makeDynamicsCompressor(ctx); },
				createGain: function() { return makeGainNode(ctx); },
				createAnalyser: function() { return makeAnalyser(ctx); },
				createBufferSource: function() { return makeBufferSource(ctx); },
				createBuffer: function(ch, len, sr) {
					return new AudioBuffer({ numberOfChannels:ch, length:len, sampleRate:sr });
				},
				createScriptProcessor: function() { return makeAudioNodeBase(ctx); },
				createBiquadFilter: function() {
					var n = makeAudioNodeBase(ctx);
					n.type = 'lowpass';
					n.frequency = makeAudioParam(350);
					n.gain = makeAudioParam(0);
					n.Q = makeAudioParam(1);
					return n;
				},
				createWaveShaper: function() {
					var n = makeAudioNodeBase(ctx);
					n.curve = null; n.oversample = 'none';
					return n;
				},
				createConvolver: function() {
					var n = makeAudioNodeBase(ctx);
					n.buffer = null; n.normalize = true;
					return n;
				},
				createPanner: function() { return makeAudioNodeBase(ctx); },
				createStereoPanner: function() {
					var n = makeAudioNodeBase(ctx);
					n.pan = makeAudioParam(0);
					return n;
				},
				createChannelSplitter: function() { return makeAudioNodeBase(ctx); },
				createChannelMerger: function() { return makeAudioNodeBase(ctx); },
				decodeAudioData: function(buffer, success, error) {
					var buf = new AudioBuffer({ numberOfChannels:2, length:4096, sampleRate:44100 });
					if (typeof success === 'function') success(buf);
					return Promise.resolve(buf);
				},
				startRendering: function() {
					var buf = new AudioBuffer({
						numberOfChannels: ctx.numberOfChannels,
						length: ctx.length,
						sampleRate: ctx.sampleRate
					});
					var ev = { renderedBuffer: buf };
					if (typeof ctx.oncomplete === 'function') {
						try { ctx.oncomplete(ev); } catch(e) {}
					}
					return Promise.resolve(buf);
				},
				resume: function() { ctx.state='running'; return Promise.resolve(); },
				suspend: function() { ctx.state='suspended'; return Promise.resolve(); },
				close: function() { ctx.state='closed'; return Promise.resolve(); },
				addEventListener: function() {},
				removeEventListener: function() {},
				dispatchEvent: function() { return true; }
			};
			return ctx;
		}

		function AudioContext(opts) {
			var ctx = makeOfflineCtx(2, 44100 * 10, (opts && opts.sampleRate) || 44100);
			ctx.state = 'running';
			ctx.baseLatency = 0.01;
			ctx.outputLatency = 0.01;
			ctx.createMediaStreamDestination = function() {
				return { stream: null, connect: function(){}, disconnect: function(){} };
			};
			ctx.createMediaElementSource = function() { return makeAudioNodeBase(ctx); };
			ctx.createMediaStreamSource = function() { return makeAudioNodeBase(ctx); };
			return ctx;
		}

		function OfflineAudioContext(numChannels, length, sampleRate) {
			return makeOfflineCtx(numChannels, length, sampleRate);
		}
		OfflineAudioContext.prototype = {};

		global.AudioContext = AudioContext;
		global.webkitAudioContext = AudioContext;
		global.OfflineAudioContext = OfflineAudioContext;
		global.webkitOfflineAudioContext = OfflineAudioContext;

		// SpeechSynthesisUtterance
		function SpeechSynthesisUtterance(text) {
			this.text = text || '';
			this.lang = 'fr-FR';
			this.voice = null;
			this.volume = 1; this.rate = 1; this.pitch = 1;
			this.onstart = null; this.onend = null; this.onerror = null;
			this.onpause = null; this.onresume = null; this.onmark = null;
			this.onboundary = null;
			this.addEventListener = function() {};
			this.removeEventListener = function() {};
		}
		global.SpeechSynthesisUtterance = SpeechSynthesisUtterance;

		// ── RTCPeerConnection (bloque les fuites IP WebRTC) ──────────────────
		function RTCPeerConnection(config) {
			this.localDescription = null;
			this.remoteDescription = null;
			this.currentLocalDescription = null;
			this.currentRemoteDescription = null;
			this.pendingLocalDescription = null;
			this.pendingRemoteDescription = null;
			this.iceConnectionState = 'new';
			this.iceGatheringState = 'new';
			this.signalingState = 'stable';
			this.connectionState = 'new';
			this.canTrickleIceCandidates = null;
			this.sctp = null;
			this.onicecandidate = null;
			this.onicecandidateerror = null;
			this.oniceconnectionstatechange = null;
			this.onicegatheringstatechange = null;
			this.onsignalingstatechange = null;
			this.onconnectionstatechange = null;
			this.ondatachannel = null;
			this.ontrack = null;
			this.onnegotiationneeded = null;

			this.createOffer = function() { return Promise.reject(new Error('WebRTC: sandboxed')); };
			this.createAnswer = function() { return Promise.reject(new Error('WebRTC: sandboxed')); };
			this.setLocalDescription = function() { return Promise.reject(new Error('WebRTC: sandboxed')); };
			this.setRemoteDescription = function() { return Promise.reject(new Error('WebRTC: sandboxed')); };
			this.addIceCandidate = function() { return Promise.resolve(); };
			this.createDataChannel = function(label, opts) {
				return {
					label: label || '', readyState: 'closed',
					bufferedAmount: 0, id: null,
					send: function() {}, close: function() {},
					addEventListener: function() {}, removeEventListener: function() {}
				};
			};
			this.addTrack = function() { return {}; };
			this.removeTrack = function() {};
			this.getSenders = function() { return []; };
			this.getReceivers = function() { return []; };
			this.getTransceivers = function() { return []; };
			this.getStats = function() { return Promise.resolve(new Map()); };
			this.getConfiguration = function() { return config || {}; };
			this.setConfiguration = function() {};
			this.restartIce = function() {};
			this.close = function() { this.signalingState = 'closed'; };
			this.addEventListener = function() {};
			this.removeEventListener = function() {};
			this.dispatchEvent = function() { return true; };
		}
		RTCPeerConnection.generateCertificate = function() {
			return Promise.reject(new Error('WebRTC: sandboxed'));
		};
		global.RTCPeerConnection = RTCPeerConnection;
		global.webkitRTCPeerConnection = RTCPeerConnection;
		global.mozRTCPeerConnection = RTCPeerConnection;
		global.RTCSessionDescription = function(init) { this.type = (init && init.type) || ''; this.sdp = (init && init.sdp) || ''; };
		global.RTCIceCandidate = function(init) { this.candidate = (init && init.candidate) || ''; this.sdpMid = null; this.sdpMLineIndex = null; };

		// ── Workers ──────────────────────────────────────────────────────────
		function Worker(url, opts) {
			this.onmessage = null;
			this.onerror = null;
			this.onmessageerror = null;
			this.postMessage = function() {};
			this.terminate = function() {};
			this.addEventListener = function() {};
			this.removeEventListener = function() {};
			this.dispatchEvent = function() { return true; };
		}
		function SharedWorker(url, opts) {
			this.port = {
				postMessage: function() {}, start: function() {},
				close: function() {}, onmessage: null, onmessageerror: null,
				addEventListener: function() {}, removeEventListener: function() {}
			};
			this.onerror = null;
			this.addEventListener = function() {};
			this.removeEventListener = function() {};
		}
		global.Worker = Worker;
		global.SharedWorker = SharedWorker;

		// ── Observers ────────────────────────────────────────────────────────
		function MutationObserver(callback) {
			this._callback = callback;
			this.observe = function(target, options) {};
			this.disconnect = function() {};
			this.takeRecords = function() { return []; };
		}
		function IntersectionObserver(callback, options) {
			this._callback = callback;
			this.root = (options && options.root) || null;
			this.rootMargin = (options && options.rootMargin) || '0px';
			this.thresholds = (options && options.threshold)
				? (Array.isArray(options.threshold) ? options.threshold : [options.threshold])
				: [0];
			this.observe = function(target) {};
			this.unobserve = function(target) {};
			this.disconnect = function() {};
			this.takeRecords = function() { return []; };
		}
		function ResizeObserver(callback) {
			this._callback = callback;
			this.observe = function(target, opts) {};
			this.unobserve = function(target) {};
			this.disconnect = function() {};
		}
		global.MutationObserver = MutationObserver;
		global.IntersectionObserver = IntersectionObserver;
		global.ResizeObserver = ResizeObserver;

		// ── AbortController / AbortSignal ────────────────────────────────────
		function AbortSignal() {
			this.aborted = false;
			this.reason = undefined;
			this.onabort = null;
			this.addEventListener = function() {};
			this.removeEventListener = function() {};
			this.dispatchEvent = function() { return true; };
			this.throwIfAborted = function() {
				if (this.aborted) throw this.reason;
			};
		}
		AbortSignal.abort = function(reason) {
			var s = new AbortSignal();
			s.aborted = true;
			s.reason = reason !== undefined ? reason : new DOMException('The operation was aborted.','AbortError');
			return s;
		};
		AbortSignal.timeout = function(ms) { return new AbortSignal(); };
		AbortSignal.any = function(signals) { return new AbortSignal(); };

		function AbortController() {
			this.signal = new AbortSignal();
			this.abort = function(reason) {
				if (!this.signal.aborted) {
					this.signal.aborted = true;
					this.signal.reason = reason !== undefined ? reason
						: new DOMException('The operation was aborted.','AbortError');
				}
			};
		}
		global.AbortController = AbortController;
		global.AbortSignal = AbortSignal;

		// ── Notification ─────────────────────────────────────────────────────
		function Notification(title, options) {
			this.title = title || '';
			this.body = (options && options.body) || '';
			this.icon = (options && options.icon) || '';
			this.tag = (options && options.tag) || '';
			this.data = (options && options.data) || null;
			this.dir = 'auto'; this.lang = 'fr-FR';
			this.silent = false; this.requireInteraction = false;
			this.onclick = null; this.onerror = null;
			this.onshow = null; this.onclose = null;
			this.close = function() {};
			this.addEventListener = function() {};
			this.removeEventListener = function() {};
		}
		Notification.permission = 'denied';
		Notification.maxActions = 2;
		Notification.requestPermission = function(cb) {
			if (typeof cb === 'function') cb('denied');
			return Promise.resolve('denied');
		};
		global.Notification = Notification;

		// ── DOMException ─────────────────────────────────────────────────────
		if (typeof DOMException === 'undefined') {
			function DOMException(message, name) {
				this.message = message || '';
				this.name = name || 'Error';
				this.code = 0;
			}
			DOMException.prototype = Object.create(Error.prototype);
			global.DOMException = DOMException;
		}

		// ── URL / URLSearchParams ─────────────────────────────────────────────
		if (typeof URLSearchParams === 'undefined') {
			function URLSearchParams(init) {
				var params = [];
				if (typeof init === 'string') {
					init.replace(/^\?/, '').split('&').forEach(function(pair) {
						var kv = pair.split('=');
						if (kv[0]) params.push([decodeURIComponent(kv[0]), decodeURIComponent(kv[1] || '')]);
					});
				}
				this.append = function(k, v) { params.push([k, v]); };
				this.get = function(k) { for (var i=0;i<params.length;i++) if (params[i][0]===k) return params[i][1]; return null; };
				this.getAll = function(k) { return params.filter(function(p){return p[0]===k;}).map(function(p){return p[1];}); };
				this.has = function(k) { return params.some(function(p){return p[0]===k;}); };
				this.set = function(k,v) { this.delete(k); params.push([k,v]); };
				this.delete = function(k) { params = params.filter(function(p){return p[0]!==k;}); };
				this.toString = function() { return params.map(function(p){return encodeURIComponent(p[0])+'='+encodeURIComponent(p[1]);}).join('&'); };
				this.forEach = function(cb) { params.forEach(function(p){cb(p[1],p[0]);}); };
				this.keys = function() { return { values: params.map(function(p){return p[0];}), next: function(){} }; };
				this.entries = function() { return { values: params, next: function(){} }; };
			}
			global.URLSearchParams = URLSearchParams;
		}

		// ── Blob / File ───────────────────────────────────────────────────────
		if (typeof Blob === 'undefined') {
			function Blob(parts, opts) {
				this.size = 0;
				this.type = (opts && opts.type) || '';
				if (parts) { for (var i=0;i<parts.length;i++) { this.size += (typeof parts[i]==='string') ? parts[i].length : (parts[i].byteLength || parts[i].length || 0); } }
				this.slice = function(start, end, type) { return new Blob([], {type: type || this.type}); };
				this.arrayBuffer = function() { return Promise.resolve(new ArrayBuffer(0)); };
				this.text = function() { return Promise.resolve(''); };
				this.stream = function() { return null; };
			}
			global.Blob = Blob;
			function File(parts, name, opts) {
				Blob.call(this, parts, opts);
				this.name = name || '';
				this.lastModified = (opts && opts.lastModified) || Date.now();
			}
			File.prototype = Object.create(Blob.prototype);
			global.File = File;
		}

		// ── FileReader ────────────────────────────────────────────────────────
		if (typeof FileReader === 'undefined') {
			function FileReader() {
				this.result = null; this.error = null;
				this.readyState = 0;
				this.onload = null; this.onerror = null; this.onabort = null;
				this.onloadstart = null; this.onloadend = null; this.onprogress = null;
				this.readAsText = function(blob, enc) {};
				this.readAsDataURL = function(blob) {};
				this.readAsArrayBuffer = function(blob) {};
				this.readAsBinaryString = function(blob) {};
				this.abort = function() {};
				this.addEventListener = function() {};
				this.removeEventListener = function() {};
			}
			FileReader.EMPTY = 0; FileReader.LOADING = 1; FileReader.DONE = 2;
			global.FileReader = FileReader;
		}

		// ── FormData ──────────────────────────────────────────────────────────
		if (typeof FormData === 'undefined') {
			function FormData() {
				var data = [];
				this.append = function(k,v) { data.push([k,v]); };
				this.get = function(k) { for(var i=0;i<data.length;i++) if(data[i][0]===k) return data[i][1]; return null; };
				this.has = function(k) { return data.some(function(d){return d[0]===k;}); };
				this.delete = function(k) { data = data.filter(function(d){return d[0]!==k;}); };
				this.set = function(k,v) { this.delete(k); data.push([k,v]); };
				this.getAll = function(k) { return data.filter(function(d){return d[0]===k;}).map(function(d){return d[1];}); };
				this.forEach = function(cb) { data.forEach(function(d){cb(d[1],d[0]);}); };
				this.entries = function() { return data[Symbol.iterator](); };
				this.keys = function() { return data.map(function(d){return d[0];})[Symbol.iterator](); };
				this.values = function() { return data.map(function(d){return d[1];})[Symbol.iterator](); };
			}
			global.FormData = FormData;
		}

		// ── PointerEvent / TouchEvent (stubs) ─────────────────────────────────
		function PointerEvent(type, opts) {
			Event.call(this, type, opts);
			this.pointerId = (opts && opts.pointerId) || 0;
			this.pointerType = (opts && opts.pointerType) || 'mouse';
			this.isPrimary = true;
			this.clientX = 0; this.clientY = 0;
			this.screenX = 0; this.screenY = 0;
			this.pressure = 0; this.tiltX = 0; this.tiltY = 0;
		}
		PointerEvent.prototype = Object.create(Event.prototype);
		global.PointerEvent = PointerEvent;

		// ── WeakRef / FinalizationRegistry (ES2021) ──────────────────────────
		if (typeof WeakRef === 'undefined') {
			function WeakRef(target) { this._target = target; }
			WeakRef.prototype.deref = function() { return this._target; };
			global.WeakRef = WeakRef;
		}
		if (typeof FinalizationRegistry === 'undefined') {
			function FinalizationRegistry(cb) {
				this.register = function() {};
				this.unregister = function() {};
			}
			global.FinalizationRegistry = FinalizationRegistry;
		}

		// ── structuredClone ───────────────────────────────────────────────────
		if (typeof structuredClone === 'undefined') {
			global.structuredClone = function(val) {
				try { return JSON.parse(JSON.stringify(val)); } catch(e) { return val; }
			};
		}

		// ── Expose constructeurs sur window ───────────────────────────────────
		if (typeof window !== 'undefined') {
			window.AudioContext = AudioContext;
			window.webkitAudioContext = AudioContext;
			window.OfflineAudioContext = OfflineAudioContext;
			window.webkitOfflineAudioContext = OfflineAudioContext;
			window.AudioBuffer = AudioBuffer;
			window.RTCPeerConnection = RTCPeerConnection;
			window.webkitRTCPeerConnection = RTCPeerConnection;
			window.mozRTCPeerConnection = RTCPeerConnection;
			window.Worker = Worker;
			window.SharedWorker = SharedWorker;
			window.MutationObserver = MutationObserver;
			window.IntersectionObserver = IntersectionObserver;
			window.ResizeObserver = ResizeObserver;
			window.AbortController = AbortController;
			window.AbortSignal = AbortSignal;
			window.Notification = Notification;
			window.SpeechSynthesisUtterance = SpeechSynthesisUtterance;
			window.HTMLCanvasElement = global.HTMLCanvasElement;
		}

	})(typeof window !== 'undefined' ? window : this);`) //nolint
}
