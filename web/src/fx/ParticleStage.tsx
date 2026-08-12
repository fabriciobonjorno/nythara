import { useEffect, useRef } from "react";
import { onFx, type FxSpawn } from "./bus";

// Palco de partículas da mesa: um único canvas sobre o tabuleiro, aditivo,
// pointer-events none. Regras de casa:
// - Nada aqui lê estado de jogo; só reage ao barramento.
// - O loop dorme quando não há partícula viva — zero custo em mesa parada.
// - Teto rígido de partículas: pico de eventos degrada bonito, não trava.
// - Coordenadas resolvidas na hora do disparo via [data-fx=...]: se o layout
//   mudar, o efeito segue o elemento, não um número mágico.

interface Particle {
  x: number; y: number;
  vx: number; vy: number;
  life: number; ttl: number;
  size: number;
  color: string;
  kind: "spark" | "shard" | "mote" | "glint";
  spin: number; angle: number;
  gravity: number; drag: number;
}

interface Ring {
  x: number; y: number;
  life: number; ttl: number;
  radius: number; to: number;
  width: number;
  color: string;
}

const MAX_PARTICLES = 420;

const PALETTE = {
  damage: "#ff5d54",
  block: "#9fd2ef",
  shatter: "#f2dcae",
  heal: "#7ee2a1",
  ward: "#7fb7ff",
  gold: "#f4cf7a",
};

export function ParticleStage() {
  const canvasRef = useRef<HTMLCanvasElement>(null);

  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;
    const ctx = canvas.getContext("2d");
    if (!ctx) return;

    const particles: Particle[] = [];
    const rings: Ring[] = [];
    let raf = 0;
    let running = false;
    let last = 0;

    const dpr = Math.min(2, window.devicePixelRatio || 1);
    const resize = () => {
      const rect = canvas.getBoundingClientRect();
      canvas.width = Math.max(1, Math.round(rect.width * dpr));
      canvas.height = Math.max(1, Math.round(rect.height * dpr));
    };
    resize();
    const observer = new ResizeObserver(resize);
    observer.observe(canvas);

    /** Centro de um alvo data-fx em coordenadas do canvas. */
    const locate = (target: string): { x: number; y: number } | null => {
      const el = document.querySelector(`[data-fx="${target}"]`);
      if (!el) return null;
      const a = el.getBoundingClientRect();
      const b = canvas.getBoundingClientRect();
      return { x: (a.left + a.width / 2 - b.left) * dpr, y: (a.top + a.height / 2 - b.top) * dpr };
    };

    const wake = () => {
      if (running) return;
      running = true;
      last = performance.now();
      raf = requestAnimationFrame(tick);
    };

    const push = (p: Particle) => {
      if (particles.length >= MAX_PARTICLES) particles.shift();
      particles.push(p);
    };

    const spawn = (s: FxSpawn) => {
      const at = locate(s.target);
      if (!at) return;
      const power = Math.max(0.2, Math.min(1, s.power ?? 0.5));
      const u = dpr; // unidade: px lógicos → físicos
      switch (s.kind) {
        case "sparks": {
          const count = Math.round(10 + power * 26);
          for (let i = 0; i < count; i++) {
            const angle = Math.random() * Math.PI * 2;
            const speed = (60 + Math.random() * 340 * power) * u;
            push({
              x: at.x, y: at.y,
              vx: Math.cos(angle) * speed, vy: Math.sin(angle) * speed - 40 * u * power,
              life: 0, ttl: 0.35 + Math.random() * 0.45,
              size: (1 + Math.random() * 2.2) * u,
              color: s.color ?? PALETTE.damage,
              kind: "spark", spin: 0, angle: 0,
              gravity: 620 * u, drag: 2.6,
            });
          }
          rings.push({ x: at.x, y: at.y, life: 0, ttl: 0.4, radius: 6 * u, to: (60 + 90 * power) * u, width: 2.5 * u, color: s.color ?? PALETTE.damage });
          break;
        }
        case "ring":
          rings.push({ x: at.x, y: at.y, life: 0, ttl: 0.55, radius: 8 * u, to: (52 + 70 * power) * u, width: 3 * u, color: s.color ?? PALETTE.block });
          break;
        case "shards": {
          const count = Math.round(12 + power * 14);
          for (let i = 0; i < count; i++) {
            const angle = -Math.PI / 2 + (Math.random() - 0.5) * 2.4;
            const speed = (120 + Math.random() * 260) * u;
            push({
              x: at.x + (Math.random() - 0.5) * 40 * u, y: at.y + (Math.random() - 0.5) * 56 * u,
              vx: Math.cos(angle) * speed, vy: Math.sin(angle) * speed,
              life: 0, ttl: 0.6 + Math.random() * 0.5,
              size: (2.2 + Math.random() * 3.4) * u,
              color: s.color ?? PALETTE.shatter,
              kind: "shard", spin: (Math.random() - 0.5) * 14, angle: Math.random() * Math.PI,
              gravity: 980 * u, drag: 0.6,
            });
          }
          break;
        }
        case "motes": {
          const count = Math.round(8 + power * 10);
          for (let i = 0; i < count; i++) {
            push({
              x: at.x + (Math.random() - 0.5) * 46 * u, y: at.y + (Math.random() - 0.3) * 20 * u,
              vx: (Math.random() - 0.5) * 26 * u, vy: -(30 + Math.random() * 60) * u,
              life: 0, ttl: 0.9 + Math.random() * 0.7,
              size: (1.4 + Math.random() * 2) * u,
              color: s.color ?? PALETTE.heal,
              kind: "mote", spin: 0, angle: Math.random() * Math.PI * 2,
              gravity: -30 * u, drag: 1.2,
            });
          }
          break;
        }
        case "glint": {
          const count = Math.round(6 + power * 8);
          for (let i = 0; i < count; i++) {
            const angle = Math.random() * Math.PI * 2;
            const speed = (30 + Math.random() * 90) * u;
            push({
              x: at.x, y: at.y,
              vx: Math.cos(angle) * speed, vy: Math.sin(angle) * speed,
              life: 0, ttl: 0.5 + Math.random() * 0.3,
              size: (1.2 + Math.random() * 1.8) * u,
              color: s.color ?? PALETTE.gold,
              kind: "glint", spin: 0, angle: 0,
              gravity: 0, drag: 3.2,
            });
          }
          rings.push({ x: at.x, y: at.y, life: 0, ttl: 0.5, radius: 4 * u, to: 40 * u, width: 1.6 * u, color: s.color ?? PALETTE.gold });
          break;
        }
      }
      wake();
    };

    const tick = (now: number) => {
      // rAF entrega o timestamp do início do frame: no primeiro tick ele pode
      // ser ANTERIOR ao performance.now() do wake(), e dt negativo faria o
      // easing extrapolar (raio negativo em arc()). Nunca ande para trás.
      const dt = Math.max(0, Math.min(0.05, (now - last) / 1000));
      last = now;
      ctx.clearRect(0, 0, canvas.width, canvas.height);
      ctx.globalCompositeOperation = "lighter";

      for (let i = particles.length - 1; i >= 0; i--) {
        const p = particles[i];
        p.life += dt;
        if (p.life >= p.ttl) { particles.splice(i, 1); continue; }
        const fade = 1 - p.life / p.ttl;
        p.vx *= 1 - p.drag * dt;
        p.vy = p.vy * (1 - p.drag * dt) + p.gravity * dt;
        p.x += p.vx * dt;
        p.y += p.vy * dt;
        p.angle += p.spin * dt;
        ctx.globalAlpha = p.kind === "mote" ? fade * 0.8 : fade;
        ctx.fillStyle = p.color;
        if (p.kind === "shard") {
          ctx.save();
          ctx.translate(p.x, p.y);
          ctx.rotate(p.angle);
          ctx.fillRect(-p.size, -p.size * 0.4, p.size * 2, p.size * 0.8);
          ctx.restore();
        } else {
          ctx.beginPath();
          ctx.arc(p.x, p.y, Math.max(0.1, p.size * (p.kind === "glint" ? 0.6 + 0.4 * Math.sin(p.life * 26) : 1)), 0, Math.PI * 2);
          ctx.fill();
        }
      }

      for (let i = rings.length - 1; i >= 0; i--) {
        const r = rings[i];
        r.life += dt;
        if (r.life >= r.ttl) { rings.splice(i, 1); continue; }
        const t = r.life / r.ttl;
        ctx.globalAlpha = (1 - t) * 0.9;
        ctx.strokeStyle = r.color;
        ctx.lineWidth = r.width * (1 - t * 0.6);
        ctx.beginPath();
        ctx.arc(r.x, r.y, Math.max(0.1, r.radius + (r.to - r.radius) * (1 - Math.pow(1 - t, 3))), 0, Math.PI * 2);
        ctx.stroke();
      }

      ctx.globalAlpha = 1;
      if (particles.length || rings.length) {
        raf = requestAnimationFrame(tick);
      } else {
        running = false;
        ctx.clearRect(0, 0, canvas.width, canvas.height);
      }
    };

    const off = onFx(spawn);
    return () => {
      off();
      observer.disconnect();
      cancelAnimationFrame(raf);
    };
  }, []);

  return <canvas ref={canvasRef} className="fx-particles" aria-hidden="true" />;
}
