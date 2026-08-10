import { useEffect, useRef } from "react";

export function EclipseStage({ value, reducedMotion }: { value: number; reducedMotion: boolean }) {
  const host = useRef<HTMLDivElement>(null);

  useEffect(() => {
    let disposed = false;
    let cleanup: () => void = () => {};
    void import("pixi.js").then(async ({ Application, Container, Graphics }) => {
      if (!host.current || disposed) return;
      const app = new Application();
      await app.init({ resizeTo: host.current, backgroundAlpha: 0, antialias: true, resolution: Math.min(devicePixelRatio, 2) });
      if (disposed || !host.current) { app.destroy(true); return; }
      host.current.replaceChildren(app.canvas);
      const group = new Container();
      const halo = new Graphics();
      const moon = new Graphics();
      const crimson = new Graphics();
      group.addChild(halo, moon, crimson);
      app.stage.addChild(group);

      const draw = () => {
        const width = app.screen.width;
        const height = app.screen.height;
        const radius = Math.max(38, Math.min(width, height) * 0.27);
        group.position.set(width / 2, height / 2);
        halo.clear().circle(0, 0, radius * 1.2).fill({ color: value > 0 ? 0x8f1d36 : 0xd6bc76, alpha: 0.12 + Math.abs(value) * 0.07 });
        moon.clear().circle(0, 0, radius).fill({ color: 0xdacbaf, alpha: 0.9 });
        const cover = (value + 3) / 6;
        crimson.clear().rect(-radius, -radius, radius * 2 * cover, radius * 2).fill({ color: 0x531020, alpha: 0.96 });
        const mask = new Graphics().circle(0, 0, radius).fill(0xffffff);
        group.addChild(mask);
        crimson.mask = mask;
      };
      draw();
      app.renderer.on("resize", draw);
      if (!reducedMotion) {
        app.ticker.add((ticker) => { group.rotation += 0.00012 * ticker.deltaTime; });
      }
      cleanup = () => app.destroy(true, { children: true });
    });
    return () => { disposed = true; cleanup(); };
  }, [value, reducedMotion]);

  return <div className="eclipse-canvas" ref={host} aria-hidden="true" />;
}
