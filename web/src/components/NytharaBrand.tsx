import { NytharaMark } from "./NytharaMark";

export function NytharaBrand({ className = "" }: { className?: string }) {
  return <span className={`nythara-brand ${className}`.trim()} aria-hidden="true">
    <NytharaMark />
    <img className="nythara-brand__wordmark" src="/assets/nythara-apocalypse-wordmark.webp" alt="" draggable={false} />
  </span>;
}
