// Sound effects of the pack opening, synthesised with the Web Audio API so no
// file has to be downloaded: a tick as the cards roll past the payline, a
// power-up when a reel lands (by rarity), and a cue when another opening is
// started.

// Not the former selector's key, its "off" would mute the sound for good
const storageKey = 'pack-sound';
const masterGain = 0.9;
// Ticks closer together than this are merged, five reels can cross at once
const tickMinGap = 0.03;
// How long a resume may take before the sounds count as blocked, in ms
const allowedWaitMs = 300;
// Samples of the looped noise, the most getRandomValues fills at once
const noiseLength = 32_768;
const unlockEvents = ['pointerdown', 'click', 'keydown', 'touchend'];

// The chiptune is played an octave down, for a chunkier sound
const octave = 0.5;
const notes = {
	c5: 523.25,
	e5: 659.25,
	g5: 783.99,
	b5: 987.77,
	c6: 1046.5,
	e6: 1318.5,
	g6: 1568,
	c7: 2093,
};

// One audio graph for the page, built on first use
const audio = {
	context: undefined,
	master: undefined,
	noise: undefined,
	lastTick: 0,
};

// Browsers keep a context suspended until the page got a click or a key
// press, a suspended one resumes on the next. It is created on load, or by
// the start cue with shouldCreate set.
function getContext(shouldCreate = false) {
	const AudioContext = globalThis.AudioContext ?? globalThis.webkitAudioContext;
	if (shouldCreate && AudioContext && !audio.context) {
		audio.context = new AudioContext();
		// Loud overall, a compressor keeping the peaks from clipping
		const limiter = audio.context.createDynamicsCompressor();
		limiter.threshold.value = -12;
		limiter.ratio.value = 8;
		limiter.connect(audio.context.destination);
		audio.master = audio.context.createGain();
		audio.master.gain.value = masterGain;
		audio.master.connect(limiter);
	}
	if (audio.context?.state === 'suspended') {
		audio.context.resume();
	}
	return audio.context;
}

// White noise, looped by the sounds using it
function noiseBuffer() {
	if (!audio.noise) {
		const { context } = audio;
		const random = crypto.getRandomValues(new Int16Array(noiseLength));
		audio.noise = context.createBuffer(1, noiseLength, context.sampleRate);
		audio.noise.getChannelData(0).set(Float32Array.from(random, (value) => value / 32_768));
	}
	return audio.noise;
}

// Gain that rises in attack seconds and decays exponentially to silence
function envelope(at, { gain, attack = 0.002, duration }) {
	const node = audio.context.createGain();
	node.gain.setValueAtTime(0.0001, at);
	node.gain.exponentialRampToValueAtTime(gain, at + attack);
	node.gain.exponentialRampToValueAtTime(0.0001, at + duration);
	node.connect(audio.master);
	return node;
}

// A lowpass in front of output, its cutoff optionally sweeping to filterTo
function lowpass(at, output, { filter, filterTo, duration }) {
	const node = audio.context.createBiquadFilter();
	node.type = 'lowpass';
	node.frequency.setValueAtTime(filter, at);
	if (filterTo) {
		node.frequency.exponentialRampToValueAtTime(filterTo, at + duration);
	}
	node.connect(output);
	return node;
}

// One oscillator note, optionally gliding to `to` Hz and through a lowpass
function tone(at, { type = 'sine', freq, to, glide, filter, filterTo, ...shape }) {
	const output = envelope(at, shape);
	const oscillator = audio.context.createOscillator();
	oscillator.type = type;
	oscillator.frequency.setValueAtTime(freq, at);
	if (to) {
		oscillator.frequency.exponentialRampToValueAtTime(to, at + (glide ?? shape.duration));
	}
	oscillator.connect(filter ? lowpass(at, output, { filter, filterTo, duration: shape.duration }) : output);
	oscillator.start(at);
	oscillator.stop(at + shape.duration + 0.05);
}

// A band of white noise
function hiss(at, { freq, quality = 1, ...shape }) {
	const output = envelope(at, shape);
	const source = audio.context.createBufferSource();
	source.buffer = noiseBuffer();
	source.loop = true;
	const band = audio.context.createBiquadFilter();
	band.type = 'bandpass';
	band.Q.value = quality;
	band.frequency.setValueAtTime(freq, at);
	source.connect(band);
	band.connect(output);
	source.start(at);
	source.stop(at + shape.duration + 0.05);
}

// Square notes one after the other, every `step` seconds, an octave down
function chiptune(at, frequencies, step, note) {
	for (const [index, freq] of frequencies.entries()) {
		tone(at + (index * step), { type: 'square', ...note, freq: freq * octave });
	}
}

// A low triangle under the bigger landings
function bass(at, gain, duration) {
	tone(at, { type: 'triangle', freq: (notes.c5 / 2) * octave, gain, duration });
}

// The classic coin: a short B, then a held E
function coin(at, pitch = 1) {
	const gain = 0.15;
	tone(at, { type: 'square', freq: notes.b5 * pitch, gain, duration: 0.08 });
	tone(at + 0.08, { type: 'square', freq: notes.e6 * pitch, gain, duration: 0.45 });
}

const sounds = {
	// A low knock with a little noise
	tick(at, reel) {
		tone(at, { type: 'square', freq: 700 + (reel * 40), to: 300, gain: 0.3, duration: 0.05, filter: 2500 });
		hiss(at, { freq: 1500, quality: 2, gain: 0.6, duration: 0.02 });
	},
	// Arpeggios climbing higher and longer with the rarity
	land: {
		common(at) {
			chiptune(at, [notes.c5, notes.e5, notes.g5], 0.06, { gain: 0.1, duration: 0.09 });
		},
		rare(at) {
			chiptune(at, [notes.c5, notes.e5, notes.g5, notes.c6, notes.e6], 0.05, { gain: 0.11, duration: 0.1 });
			bass(at + 0.25, 0.25, 0.35);
		},
		legendary(at) {
			const run = [notes.c5, notes.e5, notes.g5, notes.c6, notes.e6, notes.g6, notes.c7];
			chiptune(at, run, 0.045, { gain: 0.11, duration: 0.1 });
			// Every note of the chord at once
			const chord = { gain: 0.07, attack: 0.02, duration: 0.8, filter: 6000, filterTo: 1500 };
			chiptune(at + 0.32, [notes.c6, notes.e6, notes.g6], 0, chord);
			bass(at + 0.32, 0.3, 0.8);
		},
	},
	// Two coins dropped in, a low one then a high one
	start(at) {
		coin(at, 0.5);
		coin(at + 0.15);
	},
};

// Whether the sounds play now, without waiting for a click on this page
export async function soundsAllowed() {
	const context = getContext(true);
	if (!context || context.state === 'running') {
		return true;
	}
	if (navigator.getAutoplayPolicy?.('audiocontext') === 'disallowed') {
		return false;
	}
	// A blocked resume neither resolves nor fails until the click
	const resumed = (async () => {
		try {
			await context.resume();
		} catch {
			// Blocked for good, the state tells
		}
	})();
	const timeout = new Promise((resolve) => {
		setTimeout(resolve, allowedWaitMs);
	});
	await Promise.race([resumed, timeout]);
	return context.state === 'running';
}

// Opening straight from the inventory starts without a click on this page.
// Chromium browsers let it play right away, the click that led here counting;
// others wait for one here, every interaction tries until the sounds play.
export function unlockSounds() {
	getContext(true);
	const unlock = async () => {
		try {
			await getContext(true)?.resume();
		} catch {
			// Still blocked, the next interaction tries again
		}
		if (audio.context?.state !== 'running') {
			return;
		}
		for (const type of unlockEvents) {
			removeEventListener(type, unlock);
		}
	};
	for (const type of unlockEvents) {
		addEventListener(type, unlock);
	}
}

export function loadMuted() {
	try {
		return localStorage.getItem(storageKey) === 'off';
	} catch {
		// Storage blocked: sound on
		return false;
	}
}

export function saveMuted(isMuted) {
	try {
		localStorage.setItem(storageKey, isMuted ? 'off' : 'on');
	} catch {
		// Not remembered, still used for this visit
	}
}

// Plays the opening's sounds, doing nothing when muted or without audio
export function packSounds(isMuted) {
	// Sounds wait for a running context, they would pile up and burst out on
	// resume otherwise. The start cue follows a click, which resumes it.
	const play = (sound, reel) => {
		const isStart = sound === sounds.start;
		const context = isMuted ? undefined : getContext(isStart);
		if (context && (isStart || context.state === 'running')) {
			sound(context.currentTime, reel);
		}
	};

	return {
		tick(reel = 0) {
			const now = audio.context?.currentTime ?? 0;
			if (isMuted || now - audio.lastTick < tickMinGap) {
				return;
			}
			audio.lastTick = now;
			play(sounds.tick, reel);
		},
		land(rarity) {
			play(sounds.land[rarity] ?? sounds.land.common);
		},
		start() {
			play(sounds.start);
		},
	};
}
