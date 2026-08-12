// SFX sintetizados via WebAudio — sem assets, nunca lança, nunca decide regra.
// O contexto nasce no primeiro uso (sempre após um gesto: a batalha só emite
// sons em resposta a eventos de uma partida iniciada por clique).
//
// Desenho de som (ADR de apresentação, não de regra):
// - Cadeia mestre com compressor: picos de impacto não estouram nem somem.
// - Toda fonte passa por envelope exponencial; nada corta com clique.
// - Camadas: tom (osciladores com detune) + textura (ruído filtrado). É o que
//   separa "bip" de "golpe".
// - Variação: cada disparo sofre ±12 cents e ±10% de ganho. Ouvido humano
//   percebe repetição exata como barato; variação pequena lê como orgânico.
// - Música: sons "positivos" usam um modo menor fixo (ré), então qualquer
//   sequência de cura/ward/compra soa intencional. Dano é atonal de propósito.
// - Ducking: um golpe pesado abaixa a ambiência por ~700ms, como numa mixagem
//   de verdade — o impacto ganha palco.

let context: AudioContext | null = null;
let master: GainNode | null = null;
let compressor: DynamicsCompressorNode | null = null;
let noiseBuffer: AudioBuffer | null = null;
let enabled = true;
let ambienceEnabled = true;
let ambientMood: AmbientMood = { ownTurn: false, guard: false, danger: false };
let heartbeatTimer: number | null = null;
let ambient: {
  bus: GainNode;
  drone: OscillatorNode;
  droneGain: GainNode;
  fifth: OscillatorNode;
  fifthGain: GainNode;
  air: AudioBufferSourceNode;
  airGain: GainNode;
  filter: BiquadFilterNode;
  lfo: OscillatorNode;
  lfoGain: GainNode;
} | null = null;

export interface AmbientMood {
  ownTurn: boolean;
  guard: boolean;
  danger: boolean;
}

// Ré menor como âncora: D F A C — intervalos usados pelos sons "musicais".
const NOTE = {
  d2: 73.42, a2: 110, d3: 146.83, f3: 174.61, a3: 220, c4: 261.63,
  d4: 293.66, f4: 349.23, a4: 440, c5: 523.25, d5: 587.33, f5: 698.46, a5: 880,
};

export function setSfxEnabled(value: boolean) {
  enabled = value;
  if (master) master.gain.value = value ? 0.16 : 0;
  if (!value) stopAmbience();
}

export function setAmbienceEnabled(value: boolean) {
  ambienceEnabled = value;
  if (!value) stopAmbience();
}

function ensureContext(): AudioContext | null {
  try {
    if (!context) {
      const Ctor = window.AudioContext ?? (window as unknown as { webkitAudioContext?: typeof AudioContext }).webkitAudioContext;
      if (!Ctor) return null;
      context = new Ctor();
      compressor = context.createDynamicsCompressor();
      compressor.threshold.value = -22;
      compressor.knee.value = 18;
      compressor.ratio.value = 5;
      compressor.attack.value = 0.004;
      compressor.release.value = 0.16;
      master = context.createGain();
      master.gain.value = enabled ? 0.16 : 0;
      master.connect(compressor).connect(context.destination);
    }
    if (context.state === "suspended") void context.resume();
    return context;
  } catch {
    return null;
  }
}

/** Buffer único de ruído branco; cada uso filtra e envelopa a própria fatia. */
function ensureNoise(ctx: AudioContext): AudioBuffer {
  if (!noiseBuffer) {
    const length = Math.floor(ctx.sampleRate * 1.2);
    noiseBuffer = ctx.createBuffer(1, length, ctx.sampleRate);
    const data = noiseBuffer.getChannelData(0);
    for (let i = 0; i < length; i++) data[i] = Math.random() * 2 - 1;
  }
  return noiseBuffer;
}

/** ±cents de variação para o disparo não soar carimbado. */
function vary(freq: number, cents = 12) {
  return freq * Math.pow(2, ((Math.random() * 2 - 1) * cents) / 1200);
}

interface Tone {
  freq: number;
  to?: number;
  at?: number;
  dur?: number;
  type?: OscillatorType;
  gain?: number;
  detune?: number; // segundo oscilador levemente desafinado (chorus barato)
  steady?: boolean; // sem variação de afinação (motivos musicais exatos)
}

interface Noise {
  at?: number;
  dur?: number;
  gain?: number;
  freq?: number;   // centro do filtro
  to?: number;     // glide do filtro
  q?: number;
  kind?: "band" | "low" | "high";
}

function play(tones: Tone[], noises: Noise[] = []) {
  if (!enabled) return;
  const ctx = ensureContext();
  if (!ctx || !master) return;
  try {
    for (const tone of tones) {
      const start = ctx.currentTime + (tone.at ?? 0);
      const dur = tone.dur ?? 0.18;
      const base = tone.steady ? tone.freq : vary(tone.freq);
      const level = (tone.gain ?? 0.5) * (0.9 + Math.random() * 0.2);
      const voices = tone.detune ? [0, tone.detune] : [0];
      for (const det of voices) {
        const osc = ctx.createOscillator();
        const gain = ctx.createGain();
        osc.type = tone.type ?? "sine";
        osc.frequency.setValueAtTime(base, start);
        osc.detune.setValueAtTime(det, start);
        if (tone.to) osc.frequency.exponentialRampToValueAtTime(Math.max(28, tone.to), start + dur);
        gain.gain.setValueAtTime(0.0001, start);
        gain.gain.exponentialRampToValueAtTime(level / voices.length, start + 0.012);
        gain.gain.exponentialRampToValueAtTime(0.0001, start + dur);
        osc.connect(gain).connect(master);
        osc.start(start);
        osc.stop(start + dur + 0.05);
      }
    }
    for (const noise of noises) {
      const start = ctx.currentTime + (noise.at ?? 0);
      const dur = noise.dur ?? 0.12;
      const src = ctx.createBufferSource();
      src.buffer = ensureNoise(ctx);
      src.loop = true;
      src.playbackRate.value = 0.9 + Math.random() * 0.2;
      const filter = ctx.createBiquadFilter();
      filter.type = noise.kind === "low" ? "lowpass" : noise.kind === "high" ? "highpass" : "bandpass";
      filter.frequency.setValueAtTime(vary(noise.freq ?? 1200, 40), start);
      if (noise.to) filter.frequency.exponentialRampToValueAtTime(Math.max(60, noise.to), start + dur);
      filter.Q.value = noise.q ?? 0.9;
      const gain = ctx.createGain();
      gain.gain.setValueAtTime(0.0001, start);
      gain.gain.exponentialRampToValueAtTime((noise.gain ?? 0.3) * (0.9 + Math.random() * 0.2), start + 0.008);
      gain.gain.exponentialRampToValueAtTime(0.0001, start + dur);
      src.connect(filter).connect(gain).connect(master);
      src.start(start);
      src.stop(start + dur + 0.05);
    }
  } catch {
    // áudio jamais derruba a mesa
  }
}

/** Abaixa a ambiência por um instante para o impacto ter palco. */
function duck(amount = 0.25, seconds = 0.7) {
  if (!ambient || !context) return;
  try {
    const now = context.currentTime;
    const current = Math.max(0.0001, ambient.bus.gain.value);
    ambient.bus.gain.cancelScheduledValues(now);
    ambient.bus.gain.setValueAtTime(current, now);
    ambient.bus.gain.exponentialRampToValueAtTime(Math.max(0.0001, current * amount), now + 0.05);
    ambient.bus.gain.exponentialRampToValueAtTime(current, now + seconds);
  } catch {
    // ducking é cosmético
  }
}

function applyAmbientMood() {
  if (!ambient || !context) return;
  const now = context.currentTime;
  const base = ambientMood.danger ? NOTE.d2 * 0.84 : ambientMood.guard ? NOTE.a2 * 0.75 : ambientMood.ownTurn ? NOTE.d2 : NOTE.d2 * 0.92;
  const volume = ambientMood.danger ? 0.085 : ambientMood.guard ? 0.066 : ambientMood.ownTurn ? 0.055 : 0.04;
  ambient.drone.frequency.cancelScheduledValues(now);
  ambient.fifth.frequency.cancelScheduledValues(now);
  ambient.bus.gain.cancelScheduledValues(now);
  ambient.drone.frequency.linearRampToValueAtTime(base, now + 1.1);
  ambient.fifth.frequency.linearRampToValueAtTime(base * (ambientMood.danger ? 1.4142 : 1.4983), now + 1.1);
  ambient.bus.gain.setValueAtTime(Math.max(0.0001, ambient.bus.gain.value), now);
  ambient.bus.gain.linearRampToValueAtTime(volume, now + 0.9);
  syncHeartbeat();
}

// Batimento em perigo: dois toques surdos, ~63bpm. É informação (Vitalidade
// baixa) entregue por um canal que não ocupa os olhos.
function syncHeartbeat() {
  const wantsBeat = enabled && ambienceEnabled && ambientMood.danger && ambient !== null;
  if (wantsBeat && heartbeatTimer === null) {
    heartbeatTimer = window.setInterval(() => {
      play([
        { freq: 58, dur: 0.11, type: "sine", gain: 0.5, steady: true },
        { freq: 52, dur: 0.13, at: 0.16, type: "sine", gain: 0.38, steady: true },
      ]);
    }, 950);
  } else if (!wantsBeat && heartbeatTimer !== null) {
    window.clearInterval(heartbeatTimer);
    heartbeatTimer = null;
  }
}

/** Inicia somente quando chamado por uma ação explícita do jogador. */
export function engageAmbience() {
  if (!enabled || !ambienceEnabled || ambient) return;
  const ctx = ensureContext();
  if (!ctx || !master) return;
  try {
    const bus = ctx.createGain();
    const filter = ctx.createBiquadFilter();
    const drone = ctx.createOscillator();
    const droneGain = ctx.createGain();
    const fifth = ctx.createOscillator();
    const fifthGain = ctx.createGain();
    const air = ctx.createBufferSource();
    const airGain = ctx.createGain();
    const lfo = ctx.createOscillator();
    const lfoGain = ctx.createGain();
    bus.gain.value = 0.0001;
    filter.type = "lowpass";
    filter.frequency.value = 380;
    filter.Q.value = 0.6;
    drone.type = "sine";
    fifth.type = "triangle";
    droneGain.gain.value = 0.72;
    fifthGain.gain.value = 0.14;
    air.buffer = ensureNoise(ctx);
    air.loop = true;
    airGain.gain.value = 0.05; // um sopro de sala; sem ele o drone soa de laboratório
    lfo.type = "sine";
    lfo.frequency.value = 0.07; // respiração lenta do filtro
    lfoGain.gain.value = 90;
    lfo.connect(lfoGain).connect(filter.frequency);
    drone.connect(droneGain).connect(filter);
    fifth.connect(fifthGain).connect(filter);
    air.connect(airGain).connect(filter);
    filter.connect(bus).connect(master);
    drone.start();
    fifth.start();
    air.start();
    lfo.start();
    ambient = { bus, drone, droneGain, fifth, fifthGain, air, airGain, filter, lfo, lfoGain };
    applyAmbientMood();
  } catch {
    stopAmbience();
  }
}

export function updateAmbience(mood: AmbientMood) {
  ambientMood = mood;
  applyAmbientMood();
}

export function stopAmbience() {
  if (heartbeatTimer !== null) {
    window.clearInterval(heartbeatTimer);
    heartbeatTimer = null;
  }
  if (!ambient) return;
  try {
    for (const node of [ambient.drone, ambient.fifth, ambient.air, ambient.lfo]) node.stop();
    for (const node of [ambient.drone, ambient.droneGain, ambient.fifth, ambient.fifthGain,
      ambient.air, ambient.airGain, ambient.lfo, ambient.lfoGain, ambient.filter, ambient.bus]) node.disconnect();
  } catch {
    // limpeza de áudio é sempre best effort
  }
  ambient = null;
}

export const sfx = {
  /** Compra da própria carta: papel deslizando, curtíssimo. */
  drawCard() {
    play([], [{ dur: 0.09, gain: 0.16, freq: 2600, to: 1100, kind: "band", q: 0.7 }]);
  },
  /** Carta enviada ao centro: lufada + baque de feltro. */
  playCard() {
    play(
      [{ freq: 120, to: 62, dur: 0.16, at: 0.05, type: "sine", gain: 0.5 }],
      [{ dur: 0.14, gain: 0.22, freq: 900, to: 2400, kind: "band", q: 0.6 },
        { dur: 0.07, at: 0.06, gain: 0.3, freq: 320, kind: "low" }],
    );
  },
  /** Assalto declarado: tambor + lâmina saindo da bainha. */
  confront() {
    play(
      [{ freq: NOTE.d3, to: NOTE.d2, dur: 0.22, type: "triangle", gain: 0.5 },
        { freq: NOTE.a3, to: NOTE.a2, dur: 0.18, at: 0.03, type: "sawtooth", gain: 0.16 }],
      [{ dur: 0.2, gain: 0.2, freq: 1800, to: 5200, kind: "high", q: 0.5 },
        { dur: 0.1, gain: 0.28, freq: 240, kind: "low" }],
    );
  },
  /** Guarda comprometida: escudo erguido, metal curto. */
  prevented() {
    play(
      [{ freq: NOTE.a4, dur: 0.07, type: "square", gain: 0.14 },
        { freq: NOTE.d5, dur: 0.1, at: 0.045, type: "square", gain: 0.12 }],
      [{ dur: 0.08, gain: 0.18, freq: 3400, kind: "band", q: 2.2 }],
    );
  },
  /** Golpe bloqueado por completo: sino de bigorna, limpo e satisfatório. */
  block() {
    play(
      [{ freq: 1244, dur: 0.16, type: "triangle", gain: 0.3, detune: 9 },
        { freq: 1865, dur: 0.24, at: 0.012, type: "sine", gain: 0.2 },
        { freq: 415, dur: 0.1, type: "square", gain: 0.12 }],
      [{ dur: 0.05, gain: 0.3, freq: 5200, kind: "high", q: 0.6 }],
    );
    duck(0.45, 0.4);
  },
  /** Dano que entra: sub + estalo. Pesado dobra o corpo e abaixa a ambiência. */
  damage(heavy: boolean) {
    if (heavy) {
      play(
        [{ freq: 150, to: 46, dur: 0.34, type: "triangle", gain: 0.9 },
          { freq: 72, to: 38, dur: 0.5, at: 0.02, type: "sine", gain: 0.6 },
          { freq: 210, to: 90, dur: 0.12, type: "sawtooth", gain: 0.2 }],
        [{ dur: 0.16, gain: 0.5, freq: 700, to: 160, kind: "low" },
          { dur: 0.1, at: 0.01, gain: 0.32, freq: 2600, to: 900, kind: "band", q: 0.8 }],
      );
      duck(0.2, 0.85);
    } else {
      play(
        [{ freq: 190, to: 88, dur: 0.16, type: "triangle", gain: 0.6 }],
        [{ dur: 0.08, gain: 0.26, freq: 1900, to: 700, kind: "band", q: 0.9 }],
      );
    }
  },
  /** Carta estilhaçada: vidro real — ataque, cacos, poeira caindo. */
  shatter() {
    play(
      [{ freq: 1980, to: 640, dur: 0.1, type: "square", gain: 0.14 },
        { freq: 2637, dur: 0.06, at: 0.02, type: "triangle", gain: 0.12 },
        { freq: 660, to: 140, dur: 0.26, at: 0.03, type: "sawtooth", gain: 0.16 }],
      [{ dur: 0.06, gain: 0.42, freq: 4800, kind: "high", q: 0.5 },
        { dur: 0.22, at: 0.05, gain: 0.24, freq: 3400, to: 1200, kind: "band", q: 1.4 },
        { dur: 0.3, at: 0.1, gain: 0.12, freq: 900, to: 300, kind: "low" }],
    );
    duck(0.35, 0.6);
  },
  /** Cura: terça menor subindo, quente. */
  heal() {
    play([
      { freq: NOTE.d4, dur: 0.14, gain: 0.3, steady: true },
      { freq: NOTE.f4, dur: 0.16, at: 0.07, gain: 0.3, steady: true },
      { freq: NOTE.a4, dur: 0.26, at: 0.15, gain: 0.26, steady: true, detune: 6 },
    ]);
  },
  /** Ward: cristal — parciais não harmônicos curtos. */
  ward() {
    play([
      { freq: 1318, dur: 0.09, type: "triangle", gain: 0.2 },
      { freq: 1975, dur: 0.14, at: 0.04, type: "sine", gain: 0.16 },
      { freq: 2793, dur: 0.2, at: 0.08, type: "sine", gain: 0.1 },
    ]);
  },
  /** Poder de Avatar: assinatura curta, dourada, sempre a mesma (é marca). */
  power() {
    play([
      { freq: NOTE.a4, dur: 0.07, type: "triangle", gain: 0.2, steady: true },
      { freq: NOTE.d5, dur: 0.09, at: 0.05, type: "triangle", gain: 0.22, steady: true },
      { freq: NOTE.a5, dur: 0.2, at: 0.1, type: "sine", gain: 0.16, steady: true, detune: 7 },
    ]);
  },
  sigil(chain: number) {
    const base = NOTE.a4 * Math.pow(1.122, Math.min(chain, 5));
    play([{ freq: base, dur: 0.1, type: "triangle", gain: 0.3 }, { freq: base * 1.5, dur: 0.14, at: 0.05, type: "sine", gain: 0.2 }]);
  },
  chain() {
    play([
      { freq: NOTE.c5, dur: 0.09, gain: 0.28, steady: true },
      { freq: NOTE.f5, dur: 0.09, at: 0.07, gain: 0.28, steady: true },
      { freq: NOTE.a5, dur: 0.2, at: 0.14, gain: 0.3, steady: true },
    ]);
  },
  eclipseShift() {
    play([{ freq: 240, to: 300, dur: 0.12, type: "sine", gain: 0.18 }]);
  },
  eclipseTotal(night: boolean) {
    play(night
      ? [{ freq: 90, to: 38, dur: 1.1, type: "sawtooth", gain: 0.5 }, { freq: 180, to: 60, dur: 0.9, at: 0.08, type: "triangle", gain: 0.3 }, { freq: 55, dur: 1.3, at: 0.15, type: "sine", gain: 0.45 }]
      : [{ freq: 523, to: 1046, dur: 0.8, type: "sine", gain: 0.32 }, { freq: 659, to: 1318, dur: 0.9, at: 0.1, type: "triangle", gain: 0.22 }, { freq: 392, dur: 1.1, at: 0.05, type: "sine", gain: 0.25 }]);
  },
  stances() {
    play([{ freq: NOTE.a3, dur: 0.1, type: "triangle", gain: 0.28 }, { freq: NOTE.d4, dur: 0.16, at: 0.09, type: "triangle", gain: 0.3 }]);
  },
  /** Virada de turno: tambor surdo + tique de madeira. Discreto — acontece muito. */
  round() {
    play(
      [{ freq: NOTE.d3, dur: 0.09, type: "triangle", gain: 0.22 }],
      [{ dur: 0.05, at: 0.06, gain: 0.14, freq: 2100, kind: "band", q: 2.5 }],
    );
  },
  ultimate() {
    play(
      [{ freq: 110, to: 220, dur: 0.5, type: "sawtooth", gain: 0.36 },
        { freq: 440, to: 880, dur: 0.4, at: 0.12, type: "triangle", gain: 0.24 }],
      [{ dur: 0.5, gain: 0.2, freq: 500, to: 4000, kind: "band", q: 0.7 }],
    );
  },
  countered() {
    play(
      [{ freq: 700, to: 180, dur: 0.24, type: "square", gain: 0.2 }],
      [{ dur: 0.12, gain: 0.2, freq: 1600, to: 400, kind: "band", q: 1.1 }],
    );
  },
  /** Fim de partida: motivos exatos — vitória sobe o modo, derrota desce. */
  ended(won: boolean) {
    if (won) {
      play([
        { freq: NOTE.d4, dur: 0.16, gain: 0.3, steady: true },
        { freq: NOTE.f4, dur: 0.16, at: 0.13, gain: 0.3, steady: true },
        { freq: NOTE.a4, dur: 0.18, at: 0.26, gain: 0.32, steady: true },
        { freq: NOTE.d5, dur: 0.5, at: 0.4, gain: 0.36, steady: true, detune: 8 },
        { freq: NOTE.a5, dur: 0.6, at: 0.42, gain: 0.14, steady: true },
      ]);
    } else {
      play([
        { freq: NOTE.a3, to: NOTE.a2, dur: 0.7, type: "triangle", gain: 0.32, steady: true },
        { freq: NOTE.f3, to: NOTE.d3 * 0.5, dur: 0.95, at: 0.14, type: "sine", gain: 0.3, steady: true },
      ], [{ dur: 0.8, gain: 0.1, freq: 400, to: 120, kind: "low" }]);
    }
  },
};
