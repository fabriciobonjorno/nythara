import { NytharaMark } from "../components/NytharaMark";
import { UiIcon } from "../components/UiIcon";
import type { EclipseCinematicData, Floater, FxBannerData, StanceRevealData } from "./useBattleFx";

// Overlays de juice — todos decorativos (aria-hidden ou role=status), nunca
// interativos: pointer-events none para jamais atrapalhar uma janela de ação.

export function FloaterLayer({ floaters, mySlot }: { floaters: Floater[]; mySlot: number }) {
  return <div className="fx-floaters" aria-hidden="true">
    {floaters.map((floater) => {
      const drift = ((floater.id * 53) % 88) - 44;
      return <span
        key={floater.id}
        className={`fx-floater tone-${floater.tone} ${floater.slot === mySlot ? "is-own" : "is-opponent"}`}
        style={{ marginLeft: `${drift}px` }}
      >{floater.icon && <UiIcon name={floater.icon} />}{floater.text}</span>;
    })}
  </div>;
}

export function FxBanner({ banner }: { banner: FxBannerData | null }) {
  if (!banner) return null;
  return <div className={`fx-banner tone-${banner.tone}`} key={banner.id} role="status">
    <strong>{banner.title}</strong>
    {banner.sub && <small>{banner.sub}</small>}
  </div>;
}

export function EclipseTotalOverlay({ cine }: { cine: EclipseCinematicData | null }) {
  if (!cine) return null;
  return <div className={`eclipse-total-overlay ${cine.night ? "is-night" : "is-aurora"}`} key={cine.id} role="status">
    <div className="eclipse-total-veil" />
    <div className="eclipse-total-core">
      <span className="eclipse-total-glyph"><NytharaMark /></span>
      <p className="eyebrow">O CÉU DECIDIU</p>
      <strong>{cine.title.toLocaleUpperCase("pt-BR")}</strong>
      <small>{cine.night ? "A Noite cobra seus tributos." : "A Aurora consome as sombras."}</small>
    </div>
  </div>;
}

export function StanceRevealOverlay({ reveal }: { reveal: StanceRevealData | null }) {
  if (!reveal) return null;
  return <div className="stance-reveal-overlay" key={reveal.id} role="status">
    <div className="stance-flip is-own"><small>VOCÊ</small><strong>{reveal.mine.toLocaleUpperCase("pt-BR")}</strong></div>
    <span className="stance-clash" aria-hidden="true"><UiIcon name="duel" /></span>
    <div className="stance-flip is-opponent"><small>RIVAL</small><strong>{reveal.theirs.toLocaleUpperCase("pt-BR")}</strong></div>
  </div>;
}
