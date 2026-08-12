import type { ReactNode, SVGProps } from "react";

export type UiIconName =
  | "home" | "collection" | "champion" | "deck" | "duel"
  | "guide" | "settings" | "logout" | "fragment" | "rank"
  | "mastery" | "check" | "heart" | "ward" | "essence"
  | "clock" | "history" | "ultimate" | "balance" | "info"
  | "arrow-left" | "arrow-right" | "close" | "minus" | "warning"
  | "reaction" | "sigil" | "versus";

const marks: Record<UiIconName, ReactNode> = {
  home: <><path d="M4 11.5 12 4l8 7.5v8H4v-8Z" /><path className="ui-icon__fine" d="M8 19v-5.5h8V19M9 9.2a3.4 3.4 0 0 0 5.5 2.7A3.3 3.3 0 1 1 9 9.2Z" /></>,
  collection: <><path d="M4 5.5h5l1.5 2H20v11H4v-13Z" /><path className="ui-icon__fine" d="M7 11h10M7 14h10M7 17h6" /><path className="ui-icon__accent" d="M15.5 5.5 17 3l1.5 2.5" /></>,
  champion: <><path d="M6 19c.5-4 2.7-6.2 6-6.2s5.5 2.2 6 6.2H6Z" /><path d="M8.3 9.2C8.3 6.2 9.8 4 12 4s3.7 2.2 3.7 5.2c0 2.2-1.5 4-3.7 4s-3.7-1.8-3.7-4Z" /><path className="ui-icon__fine" d="M9.2 6.2c1.9 1 3.8 1 5.6 0M9.5 17.2h5" /></>,
  deck: <><rect x="5" y="6" width="12" height="14" rx="1" /><path className="ui-icon__fine" d="m8 6 2-2h9v13l-2 2M8 10h6M8 13h6M8 16h4" /><path className="ui-icon__accent" d="m14.5 3.8 1 1.8 1.8 1-1.8 1-1 1.8-1-1.8-1.8-1 1.8-1 1-1.8Z" /></>,
  duel: <><path d="m5 4 5.5 6.5-2 2L3 7V4h2ZM19 4l-5.5 6.5 2 2L21 7V4h-2ZM8.5 12.5l7 7M15.5 12.5l-7 7" /><path className="ui-icon__accent" d="M5.7 17.7 3.5 20M18.3 17.7l2.2 2.3" /></>,
  guide: <><path d="M4 5.5c2.8-.7 5.5-.1 8 1.7v12c-2.5-1.8-5.2-2.4-8-1.7v-12ZM20 5.5c-2.8-.7-5.5-.1-8 1.7v12c2.5-1.8 5.2-2.4 8-1.7v-12Z" /><path className="ui-icon__fine" d="M7 9h2.5M7 12h2.5M14.5 9H17M14.5 12H17" /></>,
  settings: <><circle cx="12" cy="12" r="3.2" /><path d="M12 3.5v2M12 18.5v2M3.5 12h2M18.5 12h2M6 6l1.5 1.5M16.5 16.5 18 18M18 6l-1.5 1.5M7.5 16.5 6 18" /><circle className="ui-icon__accent-fill" cx="12" cy="12" r="1" /></>,
  logout: <><path d="M10 4H5v16h5M13 8l4 4-4 4M8 12h9" /><path className="ui-icon__accent" d="M17 8v8" /></>,
  fragment: <><path className="ui-icon__soft" d="m12 2 2.6 7.4L22 12l-7.4 2.6L12 22l-2.6-7.4L2 12l7.4-2.6L12 2Z" /><path d="m12 2 2.6 7.4L22 12l-7.4 2.6L12 22l-2.6-7.4L2 12l7.4-2.6L12 2Z" /><path className="ui-icon__fine" d="M12 6v12M6 12h12" /></>,
  rank: <><path d="M5 20h14M7 20v-4h10v4M9 16v-4h6v4M11 12V7h2v5" /><path className="ui-icon__accent" d="m12 3 1.2 2.3 2.5.4-1.8 1.8.4 2.5-2.3-1.2L9.7 10l.4-2.5-1.8-1.8 2.5-.4L12 3Z" /></>,
  mastery: <><circle cx="12" cy="9" r="4.5" /><path d="M9.3 13 7 21l5-2.7 5 2.7-2.3-8" /><path className="ui-icon__fine" d="m12 6.2.8 1.7 1.9.3-1.4 1.3.3 1.9-1.6-.9-1.6.9.3-1.9-1.4-1.3 1.9-.3.8-1.7Z" /></>,
  check: <><path d="m5 12.5 4.2 4.2L19 7" /><path className="ui-icon__fine" d="m12 2.8 8.2 4.6v9.2L12 21.2l-8.2-4.6V7.4L12 2.8Z" /></>,
  heart: <><path className="ui-icon__soft" d="M12 20S4 15.4 4 9.2C4 5.5 8.7 4 12 8c3.3-4 8-2.5 8 1.2 0 6.2-8 10.8-8 10.8Z" /><path d="M12 20S4 15.4 4 9.2C4 5.5 8.7 4 12 8c3.3-4 8-2.5 8 1.2 0 6.2-8 10.8-8 10.8Z" /><path className="ui-icon__accent" d="M7 12h3l1.3-3 1.8 6 1.3-3H17" /></>,
  ward: <><path className="ui-icon__soft" d="m12 3 7.5 4.3v8.4L12 21l-7.5-5.3V7.3L12 3Z" /><path d="m12 3 7.5 4.3v8.4L12 21l-7.5-5.3V7.3L12 3Z" /><path className="ui-icon__fine" d="m8 14.5 4-8 4 8-4 2-4-2Z" /></>,
  essence: <><path className="ui-icon__soft" d="m12 2.5 7 8.5-7 10.5L5 11l7-8.5Z" /><path d="m12 2.5 7 8.5-7 10.5L5 11l7-8.5Z" /><path className="ui-icon__fine" d="m12 6 3.5 5-3.5 5-3.5-5L12 6Z" /></>,
  clock: <><circle cx="12" cy="12" r="8.5" /><path d="M12 7v5l3.5 2" /><path className="ui-icon__accent" d="M12 3.5v2M20.5 12h-2" /></>,
  history: <><path d="M5 6.5h14M5 12h14M5 17.5h9" /><circle className="ui-icon__accent-fill" cx="3" cy="6.5" r="1" /><circle className="ui-icon__accent-fill" cx="3" cy="12" r="1" /><circle className="ui-icon__accent-fill" cx="3" cy="17.5" r="1" /></>,
  ultimate: <><path className="ui-icon__soft" d="m12 2.5 2.3 6.2 6.2 2.3-6.2 2.3-2.3 8.2-2.3-8.2L3.5 11l6.2-2.3L12 2.5Z" /><path d="m12 2.5 2.3 6.2 6.2 2.3-6.2 2.3-2.3 8.2-2.3-8.2L3.5 11l6.2-2.3L12 2.5Z" /><circle className="ui-icon__accent-fill" cx="12" cy="11" r="1.5" /></>,
  balance: <><path d="M12 4v16M7 20h10M5 7h14" /><path d="m5 7-3 6h6L5 7ZM19 7l-3 6h6l-3-6Z" /><path className="ui-icon__accent" d="M9 4h6" /></>,
  info: <><circle cx="12" cy="12" r="9" /><path d="M12 10v7" /><circle className="ui-icon__accent-fill" cx="12" cy="6.8" r="1.2" /></>,
  "arrow-left": <><path d="M20 12H5M10 6l-6 6 6 6" /><path className="ui-icon__accent" d="M20 9v6" /></>,
  "arrow-right": <><path d="M4 12h15M14 6l6 6-6 6" /><path className="ui-icon__accent" d="M4 9v6" /></>,
  close: <><path d="m5 5 14 14M19 5 5 19" /><circle className="ui-icon__fine" cx="12" cy="12" r="9.5" /></>,
  minus: <><path d="M5 12h14" /><circle className="ui-icon__fine" cx="12" cy="12" r="9.5" /></>,
  warning: <><path className="ui-icon__soft" d="m12 3 9 17H3L12 3Z" /><path d="m12 3 9 17H3L12 3ZM12 8v6" /><circle className="ui-icon__accent-fill" cx="12" cy="17" r="1" /></>,
  reaction: <><path d="m5 5 7-2 7 2-2 7-5 5-5-5-2-7Z" /><path className="ui-icon__fine" d="m8 10 3 3 5-6M12 17v4" /></>,
  sigil: <><path className="ui-icon__soft" d="m12 3 8 9-8 9-8-9 8-9Z" /><path d="m12 3 8 9-8 9-8-9 8-9Z" /><path className="ui-icon__fine" d="M12 6v12M6.5 12h11" /><circle className="ui-icon__accent-fill" cx="12" cy="12" r="1.5" /></>,
  versus: <><path d="m5 4 14 16M19 4 5 20" /><path className="ui-icon__accent" d="M3.5 7.5 5 4l3.5.5M20.5 7.5 19 4l-3.5.5" /></>,
};

type UiIconProps = Omit<SVGProps<SVGSVGElement>, "name"> & { name: UiIconName };

export function UiIcon({ name, className = "", ...props }: UiIconProps) {
  return <svg className={`ui-icon ui-icon--${name} ${className}`.trim()} viewBox="0 0 24 24" aria-hidden="true" focusable="false" {...props}>{marks[name]}</svg>;
}
