import type { ReactNode } from "react";

const championMarks: Record<string, ReactNode> = {
  "CH-VH-01": <>
    <path className="champion-emblem__soft" d="M90 39c18 25 30 39 30 59a30 30 0 0 1-60 0c0-20 12-34 30-59Z" />
    <path className="champion-emblem__line" d="M90 39c18 25 30 39 30 59a30 30 0 0 1-60 0c0-20 12-34 30-59Z" />
    <path className="champion-emblem__accent" d="M64 101c14 7 38 7 52-7-1 19-12 34-29 34-14 0-24-11-23-27Z" />
    <path className="champion-emblem__fine" d="M74 75c7-6 16-8 27-5M90 45V28M82 35l8-7 8 7" />
  </>,
  "CH-VH-02": <>
    <path className="champion-emblem__soft" d="M90 130S47 105 47 72c0-19 25-27 43-7 18-20 43-12 43 7 0 33-43 58-43 58Z" />
    <path className="champion-emblem__line" d="M90 130S47 105 47 72c0-19 25-27 43-7 18-20 43-12 43 7 0 33-43 58-43 58Z" />
    <path className="champion-emblem__accent-line" d="M61 96h18l8-19 9 33 9-21h18" />
    <path className="champion-emblem__fine" d="M69 49 58 35M111 49l11-14M90 52V30" />
  </>,
  "CH-SO-01": <>
    <circle className="champion-emblem__soft" cx="90" cy="72" r="32" />
    <circle className="champion-emblem__line" cx="90" cy="72" r="32" />
    <path className="champion-emblem__fine" d="M90 25v14M90 105v17M43 72h15M122 72h15M57 39l11 11M112 94l12 12M57 105l11-11M112 50l12-12" />
    <path className="champion-emblem__accent" d="m83 55 13 7-6 11 20 32-9 6-20-32-12 6-5-14 19-16Z" />
  </>,
  "CH-SO-02": <>
    <circle className="champion-emblem__soft" cx="90" cy="80" r="40" />
    <path className="champion-emblem__line" d="M90 34v92M50 80h80M62 52l56 56M118 52l-56 56" />
    <path className="champion-emblem__accent-line" d="M104 43 84 69l10 11-18 25" />
    <path className="champion-emblem__fine" d="M90 27V16M90 144v-11M43 80H30M150 80h-13" />
  </>,
  "CH-MI-01": <>
    <path className="champion-emblem__soft" d="m90 37 39 43-39 43-39-43 39-43Z" />
    <path className="champion-emblem__line" d="m90 37 39 43-39 43-39-43 39-43Z" />
    <path className="champion-emblem__fine" d="m90 52 24 28-24 28-24-28 24-28ZM90 37v86M51 80h78" />
    <circle className="champion-emblem__accent" cx="90" cy="80" r="10" />
    <path className="champion-emblem__fine champion-emblem__rays" d="m56 43-7-8M124 43l7-8M43 80H31M149 80h-12M56 117l-7 8M124 117l7 8" />
  </>,
  "CH-MI-02": <>
    <path className="champion-emblem__soft" d="M40 80c14-25 31-37 50-37s36 12 50 37c-14 25-31 37-50 37S54 105 40 80Z" />
    <path className="champion-emblem__line" d="M40 80c14-25 31-37 50-37s36 12 50 37c-14 25-31 37-50 37S54 105 40 80Z" />
    <circle className="champion-emblem__line" cx="90" cy="80" r="22" />
    <path className="champion-emblem__accent" d="M90 58a22 22 0 0 1 0 44 16 22 0 0 0 0-44Z" />
    <path className="champion-emblem__fine" d="M90 35V21M58 45l-9-12M122 45l9-12" />
  </>,
  "CH-VA-01": <>
    <path className="champion-emblem__soft" d="M48 116c9-23 13-46 7-69l24 18 11-34 12 34 23-18c-6 23-2 46 7 69-27 19-57 19-84 0Z" />
    <path className="champion-emblem__line" d="M48 116c9-23 13-46 7-69l24 18 11-34 12 34 23-18c-6 23-2 46 7 69-27 19-57 19-84 0Z" />
    <path className="champion-emblem__fine" d="M67 91c8-8 15-8 23 0 8-8 16-8 24 0M77 110l13 7 13-7" />
    <circle className="champion-emblem__accent" cx="73" cy="87" r="5" />
    <circle className="champion-emblem__accent" cx="107" cy="87" r="5" />
  </>,
  "CH-VA-02": <>
    <path className="champion-emblem__soft" d="M122 38a49 49 0 1 0 0 84 39 39 0 1 1 0-84Z" />
    <path className="champion-emblem__line" d="M122 38a49 49 0 1 0 0 84 39 39 0 1 1 0-84Z" />
    <path className="champion-emblem__line" d="M65 112c3-22 11-35 25-41l7-20 13 15 18-4-8 18c3 16-5 34-22 44" />
    <path className="champion-emblem__accent" d="M98 85c7-4 13-3 18 1-8 6-13 6-18-1Z" />
  </>,
  "CH-CI-01": <>
    <path className="champion-emblem__line" d="M72 35h36M81 35v35l-31 51c-5 9 1 17 12 17h56c11 0 17-8 12-17L99 70V35" />
    <path className="champion-emblem__soft" d="M61 111c18 6 36-10 58-2l11 20c3 6-2 9-12 9H62c-10 0-15-4-12-10l11-17Z" />
    <path className="champion-emblem__accent-line" d="M61 111c18 6 36-10 58-2" />
    <circle className="champion-emblem__accent" cx="74" cy="94" r="6" />
    <circle className="champion-emblem__accent" cx="102" cy="88" r="4" />
    <circle className="champion-emblem__fine" cx="119" cy="60" r="5" />
  </>,
  "CH-CI-02": <>
    <path className="champion-emblem__soft" d="M48 40h73c7 0 12 5 12 12v77H60c-7 0-12-5-12-12V40Z" />
    <path className="champion-emblem__line" d="M48 40h73c7 0 12 5 12 12v77H60c-7 0-12-5-12-12V40ZM60 40v77c0 7 5 12 12 12" />
    <path className="champion-emblem__fine" d="M76 67h39M76 83h29M76 99h22" />
    <path className="champion-emblem__accent" d="M119 42c7-11 16-17 26-18-2 11-7 20-17 28l-20 42-8-4 19-48Z" />
  </>,
};

const fallback = <>
  <circle className="champion-emblem__soft" cx="90" cy="80" r="39" />
  <path className="champion-emblem__line" d="m90 39 36 41-36 41-36-41 36-41Z" />
  <circle className="champion-emblem__accent" cx="90" cy="80" r="11" />
</>;

export function ChampionEmblem({ id, faction }: { id: string; faction: string }) {
  const factionClass = faction.normalize("NFD").replace(/[\u0300-\u036f]/g, "").replaceAll(" ", "-").toLowerCase();
  return <svg className={`champion-emblem champion-emblem--${factionClass}`} viewBox="0 0 180 160" aria-hidden="true" focusable="false">
    <circle className="champion-emblem__halo" cx="90" cy="80" r="68" />
    <circle className="champion-emblem__orbit" cx="90" cy="80" r="61" />
    <path className="champion-emblem__ticks" d="M90 8v8M90 144v8M18 80h8M154 80h8" />
    <g>{championMarks[id] ?? fallback}</g>
  </svg>;
}
