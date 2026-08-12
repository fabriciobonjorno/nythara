import type { Locale } from "./locales";

const publicMetadata: Record<Locale, { title: string; description: string; ogLocale: string }> = {
  "pt-BR": {
    title: "Nythara — Jogo de cartas online PvP",
    description: "Nythara é um jogo de cartas online PvP gratuito e original. Monte um baralho de 30 cartas, treine contra um adversário virtual e dispute duelos 1v1 em tempo real.",
    ogLocale: "pt_BR",
  },
  es: {
    title: "Nythara — Juego de cartas online PvP",
    description: "Nythara es un juego de cartas online PvP gratuito y original. Arma un mazo de 30 cartas, entrena contra el bot y disputa duelos 1v1 en tiempo real.",
    ogLocale: "es_ES",
  },
  en: {
    title: "Nythara — Online PvP card game",
    description: "Nythara is a free, original online PvP card game. Build a 30-card deck, train against the bot, and play real-time 1v1 duels.",
    ogLocale: "en_US",
  },
};

const privateTitles: Record<string, Record<Locale, string>> = {
  "/forgot-password": { "pt-BR": "Recuperar acesso", es: "Recuperar acceso", en: "Recover access" },
  "/reset-password": { "pt-BR": "Redefinir senha", es: "Restablecer contraseña", en: "Reset password" },
};

const privateDescriptions: Record<Locale, string> = {
  "pt-BR": "Acesse sua conta Nythara para continuar.",
  es: "Accede a tu cuenta de Nythara para continuar.",
  en: "Sign in to your Nythara account to continue.",
};

function setMeta(selector: string, value: string) {
  document.querySelector<HTMLMetaElement>(selector)?.setAttribute("content", value);
}

export function applyRouteMetadata(pathname: string, search: string, locale: Locale) {
  const publicPage = pathname === "/" && search === "";
  const metadata = publicMetadata[locale];
  document.title = publicPage ? metadata.title : `Nythara · ${privateTitles[pathname]?.[locale] ?? "Arena"}`;
  setMeta('meta[name="description"]', publicPage ? metadata.description : privateDescriptions[locale]);
  setMeta('meta[property="og:title"]', metadata.title);
  setMeta('meta[property="og:description"]', metadata.description);
  setMeta('meta[property="og:locale"]', metadata.ogLocale);
  setMeta('meta[name="twitter:title"]', metadata.title);
  setMeta('meta[name="twitter:description"]', metadata.description);

  let robots = document.querySelector<HTMLMetaElement>('meta[name="robots"]');
  if (!robots) {
    robots = document.createElement("meta");
    robots.name = "robots";
    document.head.append(robots);
  }
  robots.content = publicPage
    ? "index, follow, max-image-preview:large, max-snippet:-1, max-video-preview:-1"
    : "noindex, nofollow, noarchive";
}
