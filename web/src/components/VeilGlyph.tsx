import type { ReactNode } from "react";

export type VeilGlyphVariant =
  | "veil"
  | "journey"
  | "champion"
  | "deck"
  | "validation"
  | "duel"
  | "mulligan"
  | "stance"
  | "rite"
  | "guard"
  | "decision"
  | "eclipse";

const symbols: Record<VeilGlyphVariant, ReactNode> = {
  veil: <>
    <path className="veil-glyph__soft" d="M96 39a43 43 0 1 0 0 82 35 35 0 1 1 0-82Z" />
    <path className="veil-glyph__line" d="M96 39a43 43 0 1 0 0 82 35 35 0 1 1 0-82Z" />
    <path className="veil-glyph__accent" d="m116 47 3 7 7 3-7 3-3 7-3-7-7-3 7-3 3-7Z" />
  </>,
  journey: <>
    <path className="veil-glyph__line veil-glyph__dash" d="M34 110C51 77 68 102 80 74s29-10 46-29" />
    <path className="veil-glyph__line" d="m115 43 12 1-2 12" />
    <circle className="veil-glyph__node" cx="34" cy="110" r="8" />
    <circle className="veil-glyph__node" cx="80" cy="74" r="8" />
    <circle className="veil-glyph__accent" cx="126" cy="44" r="7" />
    <path className="veil-glyph__fine" d="M34 99V86M23 110H10M80 63V51M69 74H56" />
  </>,
  champion: <>
    <circle className="veil-glyph__soft" cx="80" cy="56" r="17" />
    <circle className="veil-glyph__line" cx="80" cy="56" r="17" />
    <path className="veil-glyph__line" d="M48 119c2-25 15-39 32-39s30 14 32 39M61 86l19 17 19-17M80 103v23" />
    <path className="veil-glyph__accent" d="m55 48 8-18 17 10 17-10 8 18-10-6-15 8-15-8-10 6Z" />
    <path className="veil-glyph__fine" d="M57 119h46" />
  </>,
  deck: <>
    <rect className="veil-glyph__soft" x="39" y="43" width="55" height="76" rx="4" transform="rotate(-10 66 81)" />
    <rect className="veil-glyph__line" x="39" y="43" width="55" height="76" rx="4" transform="rotate(-10 66 81)" />
    <rect className="veil-glyph__soft" x="66" y="38" width="55" height="76" rx="4" transform="rotate(8 94 76)" />
    <rect className="veil-glyph__line" x="66" y="38" width="55" height="76" rx="4" transform="rotate(8 94 76)" />
    <path className="veil-glyph__accent" d="m93 59 11 17-16 13-11-17 16-13Z" />
    <path className="veil-glyph__fine" d="M87 49 109 52M75 101l24 3" />
  </>,
  validation: <>
    <path className="veil-glyph__soft" d="m80 32 14 9 17 1 6 16 11 13-7 16-1 17-17 5-13 11-15-8-17-1-5-17-11-13 8-15 1-17 16-5 13-12Z" />
    <path className="veil-glyph__line" d="m80 32 14 9 17 1 6 16 11 13-7 16-1 17-17 5-13 11-15-8-17-1-5-17-11-13 8-15 1-17 16-5 13-12Z" />
    <circle className="veil-glyph__fine" cx="80" cy="77" r="29" />
    <path className="veil-glyph__accent-line" d="m62 78 12 12 25-29" />
  </>,
  duel: <>
    <path className="veil-glyph__soft" d="m43 39 49 63-16 13-49-63 16-13ZM117 39 68 102l16 13 49-63-16-13Z" />
    <path className="veil-glyph__line" d="m43 39 49 63-16 13-49-63 16-13ZM117 39 68 102l16 13 49-63-16-13Z" />
    <path className="veil-glyph__accent-line" d="M36 105h27M97 105h27" />
    <circle className="veil-glyph__accent" cx="80" cy="80" r="9" />
  </>,
  mulligan: <>
    <rect className="veil-glyph__soft" x="55" y="49" width="50" height="66" rx="4" />
    <rect className="veil-glyph__line" x="55" y="49" width="50" height="66" rx="4" />
    <path className="veil-glyph__accent" d="m80 68 10 14-10 14-10-14 10-14Z" />
    <path className="veil-glyph__line" d="M45 65c-13 10-14 31-3 43M39 96l3 12 12-3M115 99c13-10 14-31 3-43M121 68l-3-12-12 3" />
  </>,
  stance: <>
    <path className="veil-glyph__soft" d="m80 31 46 83H34l46-83Z" />
    <path className="veil-glyph__line" d="m80 31 46 83H34l46-83Z" />
    <path className="veil-glyph__fine" d="M80 52v40M58 103l22-11 22 11" />
    <circle className="veil-glyph__accent" cx="80" cy="48" r="7" />
    <circle className="veil-glyph__node" cx="49" cy="104" r="7" />
    <circle className="veil-glyph__node" cx="111" cy="104" r="7" />
  </>,
  rite: <>
    <path className="veil-glyph__soft" d="m80 40 30 40-30 40-30-40 30-40Z" />
    <path className="veil-glyph__line" d="m80 40 30 40-30 40-30-40 30-40Z" />
    <path className="veil-glyph__accent" d="m80 60 14 20-14 20-14-20 14-20Z" />
    <path className="veil-glyph__fine" d="M80 29V15M80 145v-14M120 80h15M25 80h15M108 52l11-11M41 119l11-11M108 108l11 11M41 41l11 11" />
  </>,
  guard: <>
    <path className="veil-glyph__soft" d="M80 29c15 12 29 14 41 16v31c0 24-16 39-41 53-25-14-41-29-41-53V45c12-2 26-4 41-16Z" />
    <path className="veil-glyph__line" d="M80 29c15 12 29 14 41 16v31c0 24-16 39-41 53-25-14-41-29-41-53V45c12-2 26-4 41-16Z" />
    <path className="veil-glyph__fine" d="M80 45v65M51 73h58" />
    <circle className="veil-glyph__accent" cx="80" cy="74" r="13" />
  </>,
  decision: <>
    <circle className="veil-glyph__accent" cx="80" cy="80" r="10" />
    <path className="veil-glyph__line" d="M80 70V43M71 82 48 96M89 82l23 14M80 90v27" />
    <path className="veil-glyph__soft" d="m80 28 13 15-13 15-13-15 13-15ZM35 92l17-8 10 16-17 8-10-16ZM125 92l-17-8-10 16 17 8 10-16ZM80 105l13 15-13 15-13-15 13-15Z" />
    <path className="veil-glyph__fine" d="m80 28 13 15-13 15-13-15 13-15ZM35 92l17-8 10 16-17 8-10-16ZM125 92l-17-8-10 16 17 8 10-16ZM80 105l13 15-13 15-13-15 13-15Z" />
  </>,
  eclipse: <>
    <circle className="veil-glyph__soft" cx="80" cy="80" r="43" />
    <path className="veil-glyph__accent" d="M80 37a43 43 0 0 1 0 86 33 43 0 0 0 0-86Z" />
    <circle className="veil-glyph__line" cx="80" cy="80" r="43" />
    <path className="veil-glyph__fine" d="M80 27V15M80 145v-12M27 80H15M145 80h-12" />
  </>,
};

export function VeilGlyph({ variant }: { variant: VeilGlyphVariant }) {
  return <svg className={`veil-glyph veil-glyph--${variant}`} viewBox="0 0 160 160" aria-hidden="true" focusable="false">
    <circle className="veil-glyph__ring" cx="80" cy="80" r="68" />
    <circle className="veil-glyph__orbit" cx="80" cy="80" r="61" />
    <path className="veil-glyph__ticks" d="M80 8v8M80 144v8M8 80h8M144 80h8" />
    <g className="veil-glyph__symbol">{symbols[variant]}</g>
  </svg>;
}
