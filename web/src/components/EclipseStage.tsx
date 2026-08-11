import { useEffect, useRef } from "react";
import type { Application, Container, Graphics } from "pixi.js";

// Palco PixiJS da mesa. O app nasce uma única vez por partida; mudanças de
// Eclipse interpolam a cobertura da lua em vez de recriar o canvas, e o
// Eclipse Total dispara uma explosão de partículas. Com reducedMotion, tudo
// vira atualização instantânea sem rotação nem partículas.

interface Particle {
  gfx: Graphics;
  vx: number;
  vy: number;
  life: number;
}

interface Stage {
  app: Application;
  group: Container;
  halo: Graphics;
  moon: Graphics;
  crimson: Graphics;
  particles: Particle[];
  displayed: number;
  dirty: boolean;
}

export function EclipseStage({ value, burst, reducedMotion }: {
  value: number;
  burst?: { key: number; night: boolean } | null;
  reducedMotion: boolean;
}) {
  const host = useRef<HTMLDivElement>(null);
  const stageRef = useRef<Stage | null>(null);
  const targetRef = useRef(value);
  const burstKeyRef = useRef(burst?.key ?? 0);
  targetRef.current = value;

  useEffect(() => {
    let disposed = false;
    let cleanup: () => void = () => {};
    // O pacote mantém o nome histórico `unsafe-eval`, mas este entrypoint
    // instala justamente os polyfills que evitam `eval` sob uma CSP estrita.
    void import("pixi.js/unsafe-eval").then(() => import("pixi.js")).then(async ({ Application, Container, Graphics }) => {
      if (!host.current || disposed) return;
      const app = new Application();
      await app.init({ resizeTo: host.current, backgroundAlpha: 0, antialias: true, resolution: Math.min(devicePixelRatio, 2) });
      if (disposed || !host.current) { app.destroy(true); return; }
      host.current.replaceChildren(app.canvas);
      const group = new Container();
      const halo = new Graphics();
      const moon = new Graphics();
      const crimson = new Graphics();
      const mask = new Graphics();
      group.addChild(halo, moon, crimson, mask);
      crimson.mask = mask;
      app.stage.addChild(group);
      const stage: Stage = { app, group, halo, moon, crimson, particles: [], displayed: targetRef.current, dirty: true };
      stageRef.current = stage;

      const radius = () => Math.max(38, Math.min(app.screen.width, app.screen.height) * 0.27);
      const draw = () => {
        const r = radius();
        group.position.set(app.screen.width / 2, app.screen.height / 2);
        const intensity = Math.abs(stage.displayed);
        halo.clear().circle(0, 0, r * 1.2).fill({ color: stage.displayed > 0 ? 0x8f1d36 : 0xd6bc76, alpha: 0.12 + intensity * 0.07 });
        moon.clear().circle(0, 0, r).fill({ color: 0xdacbaf, alpha: 0.9 });
        const cover = (stage.displayed + 3) / 6;
        crimson.clear().rect(-r, -r, r * 2 * cover, r * 2).fill({ color: 0x531020, alpha: 0.96 });
        mask.clear().circle(0, 0, r).fill(0xffffff);
        stage.dirty = false;
      };
      draw();
      app.renderer.on("resize", () => { stage.dirty = true; });

      app.ticker.add((ticker) => {
        if (!reducedMotion) group.rotation += 0.00012 * ticker.deltaTime;
        const target = targetRef.current;
        if (stage.displayed !== target) {
          if (reducedMotion) {
            stage.displayed = target;
          } else {
            const step = (target - stage.displayed) * Math.min(1, 0.055 * ticker.deltaTime);
            stage.displayed = Math.abs(target - stage.displayed) < 0.004 ? target : stage.displayed + step;
          }
          stage.dirty = true;
        }
        if (stage.dirty) draw();
        if (stage.particles.length) {
          const r = radius();
          for (let index = stage.particles.length - 1; index >= 0; index--) {
            const particle = stage.particles[index];
            particle.life -= ticker.deltaTime / 60;
            particle.gfx.position.x += particle.vx * ticker.deltaTime;
            particle.gfx.position.y += particle.vy * ticker.deltaTime;
            particle.vy += 0.012 * ticker.deltaTime;
            particle.gfx.alpha = Math.max(0, particle.life / 1.4);
            particle.gfx.scale.set(Math.max(0.2, particle.life));
            const distance = Math.hypot(particle.gfx.position.x, particle.gfx.position.y);
            if (particle.life <= 0 || distance > r * 6) {
              group.removeChild(particle.gfx);
              particle.gfx.destroy();
              stage.particles.splice(index, 1);
            }
          }
        }
      });

      cleanup = () => {
        stageRef.current = null;
        app.destroy(true, { children: true });
      };
    });
    return () => { disposed = true; cleanup(); };
  }, [reducedMotion]);

  useEffect(() => {
    const stage = stageRef.current;
    if (stage) stage.dirty = true; // o ticker interpola até targetRef
  }, [value]);

  useEffect(() => {
    if (!burst || burst.key === burstKeyRef.current) return;
    burstKeyRef.current = burst.key;
    const stage = stageRef.current;
    if (!stage || reducedMotion) return;
    void import("pixi.js").then(({ Graphics }) => {
      const current = stageRef.current;
      if (!current || current !== stage) return;
      const colors = burst.night ? [0x8f1d36, 0x531020, 0xd6bc76] : [0xd6bc76, 0xf4e3b2, 0x8f1d36];
      for (let index = 0; index < 30; index++) {
        const angle = (index / 30) * Math.PI * 2 + (index % 3) * 0.17;
        const speed = 1.6 + (index % 5) * 0.75;
        const gfx = new Graphics().circle(0, 0, 2.4 + (index % 3)).fill({ color: colors[index % colors.length], alpha: 0.9 });
        current.group.addChild(gfx);
        current.particles.push({ gfx, vx: Math.cos(angle) * speed, vy: Math.sin(angle) * speed, life: 1.15 + (index % 4) * 0.14 });
      }
    });
  }, [burst, reducedMotion]);

  return <div className="eclipse-canvas" ref={host} aria-hidden="true" />;
}
