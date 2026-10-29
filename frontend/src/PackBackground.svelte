<script>
	import { onMount } from "svelte";

	let { target } = $props();

	let canvas;

	// The shader runs on a small buffer stretched to the screen: cheap, and
	// the pixelated look is part of the style
	const PIXEL_SIZE = 6;
	const MAX_WIDTH = 480;
	// Seconds of animation per real second, slower when motion is reduced
	const SPEED = 1;
	const REDUCED_SPEED = 0.3;
	// The swirl is slow, 30 frames per second are enough
	const FRAME_MS = 1000 / 30;

	// Swirling paint: the plane spins faster towards the center, then is
	// folded onto itself a few times. The length of the folded point picks
	// between three colors, with sharp edges.
	const VERTEX = `
		attribute vec2 position;
		void main() {
			gl_Position = vec4(position, 0.0, 1.0);
		}
	`;
	const FRAGMENT = `
		// The time keeps growing, mediump would turn it to steps on phones
		#ifdef GL_FRAGMENT_PRECISION_HIGH
		precision highp float;
		#else
		precision mediump float;
		#endif

		uniform vec2 resolution;
		uniform float time;
		uniform vec3 color1;
		uniform vec3 color2;
		uniform vec3 color3;

		void main() {
			vec2 uv = (gl_FragCoord.xy - 0.5 * resolution) / length(resolution);

			float radius = length(uv);
			float angle = atan(uv.y, uv.x) + 0.08 * time - 4.0 * radius;
			uv = radius * vec2(cos(angle), sin(angle)) * 60.0;

			float t = 0.35 * time;
			vec2 p = uv;
			for (int i = 0; i < 5; i++) {
				p += vec2(
					sin(0.42 * p.y + t + 0.7 * float(i)),
					cos(0.38 * p.x - 0.8 * t)
				) * 1.6;
				p += vec2(
					cos(0.21 * p.x + 0.29 * p.y + 0.5 * t),
					sin(0.31 * p.x - 0.23 * p.y - 0.4 * t)
				);
			}

			float paint = 0.5 + 0.5 * sin(0.25 * length(p) + 0.15 * (p.x - p.y));
			float band1 = smoothstep(0.33, 0.37, paint);
			float band2 = smoothstep(0.66, 0.7, paint);

			vec3 color = mix(color3, color1, band1);
			color = mix(color, color2, band2);
			// Lighter edge at each band, like wet paint
			color += 0.1 * (1.0 - smoothstep(0.0, 0.04, abs(paint - 0.35)));

			gl_FragColor = vec4(color, 1.0);
		}
	`;

	// "#rrggbb" to normalized RGB, null when not parsable
	function parseColor(value) {
		const match = /^#?([0-9a-f]{6})$/i.exec(value?.trim() ?? "");
		if (!match) {
			return null;
		}
		const n = Number.parseInt(match[1], 16);
		return [(n >> 16) / 255, ((n >> 8) & 255) / 255, (n & 255) / 255];
	}

	function compile(gl, type, source) {
		const shader = gl.createShader(type);
		gl.shaderSource(shader, source);
		gl.compileShader(shader);
		if (!gl.getShaderParameter(shader, gl.COMPILE_STATUS)) {
			console.warn("pack background:", gl.getShaderInfoLog(shader));
			return null;
		}
		return shader;
	}

	onMount(() => {
		const primary = parseColor(target.dataset.colorPrimary) ?? [1, 0.5, 0.37];
		const secondary = parseColor(target.dataset.colorSecondary) ?? [0.5, 0.18, 1];
		// Dark base the two colors are painted over
		const base = secondary.map((c) => c * 0.25 + 0.04);

		// Also used by the CSS fallback
		target.style.setProperty("--pack-c1", target.dataset.colorPrimary);
		target.style.setProperty("--pack-c2", target.dataset.colorSecondary);

		// The canvas gives way to the drifting CSS gradient
		function fallback() {
			canvas.hidden = true;
			target.classList.add("is-fallback");
		}

		const gl = canvas.getContext("webgl", { antialias: false, depth: false, alpha: false });
		if (!gl) {
			console.warn("pack background: WebGL unavailable, using the CSS fallback");
			fallback();
			return;
		}

		const vertex = compile(gl, gl.VERTEX_SHADER, VERTEX);
		const fragment = compile(gl, gl.FRAGMENT_SHADER, FRAGMENT);
		if (!vertex || !fragment) {
			fallback();
			return;
		}
		const program = gl.createProgram();
		gl.attachShader(program, vertex);
		gl.attachShader(program, fragment);
		gl.linkProgram(program);
		if (!gl.getProgramParameter(program, gl.LINK_STATUS)) {
			fallback();
			return;
		}
		gl.useProgram(program);

		// One triangle covering the screen
		gl.bindBuffer(gl.ARRAY_BUFFER, gl.createBuffer());
		gl.bufferData(gl.ARRAY_BUFFER, new Float32Array([-1, -1, 3, -1, -1, 3]), gl.STATIC_DRAW);
		const position = gl.getAttribLocation(program, "position");
		gl.enableVertexAttribArray(position);
		gl.vertexAttribPointer(position, 2, gl.FLOAT, false, 0, 0);

		gl.uniform3fv(gl.getUniformLocation(program, "color1"), primary);
		gl.uniform3fv(gl.getUniformLocation(program, "color2"), secondary);
		gl.uniform3fv(gl.getUniformLocation(program, "color3"), base);
		const resolution = gl.getUniformLocation(program, "resolution");
		const time = gl.getUniformLocation(program, "time");

		function resize() {
			const width = Math.max(1, Math.min(MAX_WIDTH, Math.round(window.innerWidth / PIXEL_SIZE)));
			const height = Math.max(1, Math.round(width * window.innerHeight / window.innerWidth));
			canvas.width = width;
			canvas.height = height;
			gl.viewport(0, 0, width, height);
			gl.uniform2f(resolution, width, height);
		}

		function draw(seconds) {
			gl.uniform1f(time, seconds);
			gl.drawArrays(gl.TRIANGLES, 0, 3);
		}

		// Reduced motion only slows the swirl down to a drift
		const reduced = window.matchMedia("(prefers-reduced-motion: reduce)");
		let elapsed = 0;
		let last = performance.now();
		let frame = 0;

		function loop(now) {
			frame = requestAnimationFrame(loop);
			const delta = now - last;
			if (delta < FRAME_MS) {
				return;
			}
			last = now;
			// Capped so coming back to a hidden tab doesn't jump ahead
			elapsed += (Math.min(delta, 100) / 1000) * (reduced.matches ? REDUCED_SPEED : SPEED);
			draw(elapsed);
		}

		resize();
		draw(elapsed);
		frame = requestAnimationFrame(loop);
		window.addEventListener("resize", resize);

		return () => {
			cancelAnimationFrame(frame);
			window.removeEventListener("resize", resize);
		};
	});
</script>

<canvas bind:this={canvas}></canvas>
