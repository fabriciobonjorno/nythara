import type { ReactNode } from "react";

const sigilMarks: Record<string, ReactNode> = {
  Presa: (
    <>
      <path className="card-sigil-icon__soft" d="M10 11c13 4 20 12 22 25 2-13 9-21 22-25-3 21-10 35-22 43C20 46 13 32 10 11Z" />
      <path className="card-sigil-icon__line" d="M15 14c2 15 6 27 14 35M49 14c-2 15-6 27-14 35" />
      <path className="card-sigil-icon__accent" d="m27 42 5 10 5-10-5-7-5 7Z" />
    </>
  ),
  Sol: (
    <>
      <circle className="card-sigil-icon__soft" cx="32" cy="32" r="17" />
      <circle className="card-sigil-icon__line" cx="32" cy="32" r="10" />
      <path className="card-sigil-icon__accent" d="M32 7v9M32 48v9M7 32h9M48 32h9M14.3 14.3l6.3 6.3M43.4 43.4l6.3 6.3M49.7 14.3l-6.3 6.3M20.6 43.4l-6.3 6.3" />
      <path className="card-sigil-icon__fine" d="M26 32a6 6 0 0 1 10.7-3.7A7 7 0 1 0 32 39a7 7 0 0 0 5.5-2.7A6 6 0 0 1 26 32Z" />
    </>
  ),
  Espelho: (
    <>
      <path className="card-sigil-icon__soft" d="m32 7 22 25-22 25L10 32 32 7Z" />
      <path className="card-sigil-icon__line" d="m32 8 21 24-21 24-21-24L32 8Z" />
      <path className="card-sigil-icon__fine" d="m32 8-7 20 7 4-4 11 4 13M25 28l-14 4M32 32l21 0M28 43l-10 2" />
      <path className="card-sigil-icon__accent" d="m32 17 8 15-8-3-6 3 6-15Z" />
    </>
  ),
  Garra: (
    <>
      <path className="card-sigil-icon__soft" d="M12 49c12-7 14-25 20-38 1 15-1 32-12 43l-8-5Zm16 5c10-9 11-26 16-38 2 14 1 30-8 41l-8-3Zm17-2c5-8 4-19 7-28 4 10 5 21 0 30l-7-2Z" />
      <path className="card-sigil-icon__line" d="M14 50c11-8 13-24 18-38M30 54c9-10 10-25 14-37M46 52c5-8 4-18 6-27" />
      <path className="card-sigil-icon__accent" d="m13 50 7 4m10 0 7 3m9-5 6 2" />
    </>
  ),
  Cinza: (
    <>
      <path className="card-sigil-icon__soft" d="M34 7c3 11-6 15-2 24 2 4 6 5 8 1 3-6-1-11 0-16 9 8 14 17 11 27-3 10-12 15-21 14-10-1-18-9-17-19 1-9 7-14 13-21-1 9 1 14 5 16-1-10 8-15 3-26Z" />
      <path className="card-sigil-icon__line" d="M34 8c3 11-6 15-2 24 2 4 6 5 8 1 3-6-1-11 0-16 9 8 14 17 11 27-3 10-12 15-21 14-10-1-18-9-17-19 1-9 7-14 13-21" />
      <path className="card-sigil-icon__fine" d="M30 56c-5-4-7-9-5-14 1-4 5-7 7-11 0 7 8 8 6 15-1 5-4 8-8 10Z" />
      <circle className="card-sigil-icon__accent-fill" cx="17" cy="22" r="2" />
      <circle className="card-sigil-icon__accent-fill" cx="48" cy="10" r="1.5" />
    </>
  ),
  Coroa: (
    <>
      <path className="card-sigil-icon__soft" d="m10 22 13 9 9-21 9 21 13-9-5 27H15l-5-27Z" />
      <path className="card-sigil-icon__line" d="m10 22 13 9 9-21 9 21 13-9-5 27H15l-5-27ZM17 55h30" />
      <path className="card-sigil-icon__fine" d="m17 37 15-9 15 9M16 43h32" />
      <circle className="card-sigil-icon__accent-fill" cx="10" cy="21" r="3" />
      <circle className="card-sigil-icon__accent-fill" cx="32" cy="9" r="3" />
      <circle className="card-sigil-icon__accent-fill" cx="54" cy="21" r="3" />
    </>
  ),
};

export function CardSigil({ sigil }: { sigil: string }) {
  return (
    <svg className={`card-sigil-icon card-sigil-icon--${sigil.toLocaleLowerCase("pt-BR")}`} viewBox="0 0 64 64" aria-hidden="true">
      {sigilMarks[sigil] ?? sigilMarks.Espelho}
    </svg>
  );
}

export function CardZoomIcon() {
  return (
    <svg className="card-zoom-icon" viewBox="0 0 24 24" aria-hidden="true">
      <circle cx="10.5" cy="10.5" r="5.5" />
      <path d="m14.7 14.7 5 5M7.5 3.8H3.8v3.7M13.5 3.8h3.7v3.7" />
    </svg>
  );
}
