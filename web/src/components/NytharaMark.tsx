import type { ImgHTMLAttributes } from "react";

export function NytharaMark({ className = "", alt = "", ...props }: ImgHTMLAttributes<HTMLImageElement>) {
  return <img
    className={`nythara-mark ${className}`.trim()}
    src="/assets/nythara-apocalypse-emblem-v2.webp"
    alt={alt}
    aria-hidden={alt ? undefined : true}
    draggable={false}
    {...props}
  />;
}
