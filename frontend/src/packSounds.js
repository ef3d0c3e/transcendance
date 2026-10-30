// Sound effects of the pack opening, synthesised with the Web Audio API so no
// file has to be downloaded: a tick as the cards roll past the payline, a
// power-up when a reel lands (by rarity), and a cue when another opening is
// started.

// Not the selector's former key, its "off" would mute the sound for good
const STORAGE_KEY = "pack-sound";
const MASTER_GAIN = 0.9;
// Ticks closer together than this are merged, five reels can cross at once
const TICK_MIN_GAP = 0.03;

let context = null;
let master = null;
let noiseBuffer = null;
let lastTick = 0;

// Browsers keep a context suspended until the page got a click or a key
// press, a suspended one resumes on the next. It is created on load, or by
// the start cue with create set.
function audio(create = false) {
	if (!context) {
		if (!create) {
			return null;
		}
		const AudioContext = window.AudioContext || window.webkitAudioContext;
		if (!AudioContext) {
			return null;
		}
		context = new AudioContext();
		// Loud overall, a compressor keeping the peaks from clipping
		const limiter = context.createDynamicsCompressor();
		limiter.threshold.value = -12;
		limiter.ratio.value = 8;
		limiter.connect(context.destination);
		master = context.createGain();
		master.gain.value = MASTER_GAIN;
		master.connect(limiter);
	}
	if (context.state === "suspended") {
		context.resume();
	}
	return context;
}

// How long a resume may take before the sounds count as blocked, in ms
const ALLOWED_WAIT_MS = 300;

// Whether the sounds play now, without waiting for a click on this page
export async function soundsAllowed() {
	const ctx = audio(true);
	if (!ctx || ctx.state === "running") {
		return true;
	}
	if (navigator.getAutoplayPolicy?.("audiocontext") === "disallowed") {
		return false;
	}
	// A blocked resume neither resolves nor fails until the click
	await Promise.race([
		ctx.resume().catch(() => {}),
		new Promise((resolve) => {
			setTimeout(resolve, ALLOWED_WAIT_MS);
		}),
	]);
	return ctx.state === "running";
}

const UNLOCK_EVENTS = ["pointerdown", "click", "keydown", "touchend"];

// Opening straight from the inventory starts without a click on this page.
// Chromium browsers let it play right away, the click that led here counting;
// others wait for one here, every interaction tries until the sounds play.
export function unlockSounds() {
	audio(true);
	const unlock = async () => {
		try {
			await audio(true)?.resume();
		} catch {
			// Still blocked, the next interaction tries again
		}
		if (context?.state === "running") {
			for (const type of UNLOCK_EVENTS) {
				window.removeEventListener(type, unlock);
			}
		}
	};
	for (const type of UNLOCK_EVENTS) {
		window.addEventListener(type, unlock);
	}
}

function noise(ctx) {
	if (!noiseBuffer) {
		noiseBuffer = ctx.createBuffer(1, ctx.sampleRate, ctx.sampleRate);
		const data = noiseBuffer.getChannelData(0);
		for (let i = 0; i < data.length; i++) {
			data[i] = Math.random() * 2 - 1;
		}
	}
	return noiseBuffer;
}

// Gain that rises in attack seconds and decays exponentially to silence
function envelope(ctx, at, { gain, attack = 0.002, duration }) {
	const node = ctx.createGain();
	node.gain.setValueAtTime(0.0001, at);
	node.gain.exponentialRampToValueAtTime(gain, at + attack);
	node.gain.exponentialRampToValueAtTime(0.0001, at + duration);
	node.connect(master);
	return node;
}

// One oscillator note, optionally gliding to `to` Hz and through a lowpass
function tone(ctx, at, { type = "sine", freq, to, glide, filter, filterTo, detune = 0, ...shape }) {
	const out = envelope(ctx, at, shape);
	const osc = ctx.createOscillator();
	osc.type = type;
	osc.detune.value = detune;
	osc.frequency.setValueAtTime(freq, at);
	if (to) {
		osc.frequency.exponentialRampToValueAtTime(to, at + (glide ?? shape.duration));
	}
	let target = out;
	if (filter) {
		const lowpass = ctx.createBiquadFilter();
		lowpass.type = "lowpass";
		lowpass.frequency.setValueAtTime(filter, at);
		if (filterTo) {
			lowpass.frequency.exponentialRampToValueAtTime(filterTo, at + shape.duration);
		}
		lowpass.connect(out);
		target = lowpass;
	}
	osc.connect(target);
	osc.start(at);
	osc.stop(at + shape.duration + 0.05);
}

// Filtered white noise, the band optionally sweeping to `to` Hz
function hiss(ctx, at, { type = "bandpass", freq, to, q = 1, ...shape }) {
	const out = envelope(ctx, at, shape);
	const source = ctx.createBufferSource();
	source.buffer = noise(ctx);
	source.loop = true;
	const band = ctx.createBiquadFilter();
	band.type = type;
	band.Q.value = q;
	band.frequency.setValueAtTime(freq, at);
	if (to) {
		band.frequency.exponentialRampToValueAtTime(to, at + shape.duration);
	}
	source.connect(band);
	band.connect(out);
	source.start(at, Math.random() * 0.5);
	source.stop(at + shape.duration + 0.05);
}

// Notes one after the other, every `step` seconds
function arpeggio(ctx, at, notes, step, note) {
	notes.forEach((freq, index) => {
		tone(ctx, at + index * step, { ...note, freq });
	});
}

const C5 = 523.25;
const E5 = 659.25;
const G5 = 783.99;
const C6 = 1046.5;
const E6 = 1318.5;
const G6 = 1568;
const C7 = 2093;
const B5 = 987.77;
// The chiptune is played an octave down, for a chunkier sound
const OCTAVE = 0.5;

// Square notes one after the other, an octave down
function chiptune(ctx, at, notes, step, note) {
	arpeggio(ctx, at, notes.map((freq) => freq * OCTAVE), step, { type: "square", ...note });
}

// The classic coin: a short B, then a held E
function coin(ctx, at, gain, octave = 1) {
	tone(ctx, at, { type: "square", freq: B5 * octave, gain, duration: 0.08 });
	tone(ctx, at + 0.08, { type: "square", freq: E6 * octave, gain, duration: 0.45 });
}

const SOUNDS = {
	// A low knock with a little noise
	tick(ctx, at, reel) {
		tone(ctx, at, { type: "square", freq: 700 + reel * 40, to: 300, gain: 0.3, duration: 0.05, filter: 2500 });
		hiss(ctx, at, { freq: 1500, q: 2, gain: 0.6, duration: 0.02 });
	},
	// Arpeggios climbing higher and longer with the rarity
	land: {
		common(ctx, at) {
			chiptune(ctx, at, [C5, E5, G5], 0.06, { gain: 0.1, duration: 0.09 });
		},
		rare(ctx, at) {
			chiptune(ctx, at, [C5, E5, G5, C6, E6], 0.05, { gain: 0.11, duration: 0.1 });
			tone(ctx, at + 0.25, { type: "triangle", freq: (C5 / 2) * OCTAVE, gain: 0.25, duration: 0.35 });
		},
		legendary(ctx, at) {
			chiptune(ctx, at, [C5, E5, G5, C6, E6, G6, C7], 0.045, { gain: 0.11, duration: 0.1 });
			// Every note of the chord at once
			chiptune(ctx, at + 0.32, [C6, E6, G6], 0, { gain: 0.07, attack: 0.02, duration: 0.8, filter: 6000, filterTo: 1500 });
			tone(ctx, at + 0.32, { type: "triangle", freq: (C5 / 2) * OCTAVE, gain: 0.3, duration: 0.8 });
		},
	},
	// Two coins dropped in, a low one then a high one
	start(ctx, at) {
		coin(ctx, at, 0.15, 0.5);
		coin(ctx, at + 0.15, 0.15);
	},
};

export function loadMuted() {
	try {
		return localStorage.getItem(STORAGE_KEY) === "off";
	} catch {
		// Storage blocked: sound on
		return false;
	}
}

export function saveMuted(muted) {
	try {
		localStorage.setItem(STORAGE_KEY, muted ? "off" : "on");
	} catch {
		// Not remembered, still used for this visit
	}
}

// Plays the opening's sounds, doing nothing when muted or without audio
export function packSounds(muted) {
	// Sounds wait for a running context, they would pile up and burst out on
	// resume otherwise. The start cue follows a click, which resumes it.
	const play = (sound, ...args) => {
		const ctx = !muted && audio(sound === SOUNDS.start);
		if (ctx && (ctx.state === "running" || sound === SOUNDS.start)) {
			sound(ctx, ctx.currentTime, ...args);
		}
	};

	return {
		tick(reel = 0) {
			const now = context?.currentTime ?? 0;
			if (muted || now - lastTick < TICK_MIN_GAP) {
				return;
			}
			lastTick = now;
			play(SOUNDS.tick, reel);
		},
		land(rarity) {
			play(SOUNDS.land[rarity] ?? SOUNDS.land.common);
		},
		start() {
			play(SOUNDS.start);
		},
	};
}
