export const supportedLocales = ["pt-BR", "es", "en"] as const;

export type Locale = typeof supportedLocales[number];

export const localeNames: Record<Locale, string> = {
  "pt-BR": "Português (Brasil)",
  es: "Español",
  en: "English",
};

export function normalizeLocale(value?: string | null): Locale | null {
  const locale = value?.trim().toLocaleLowerCase();
  if (!locale) return null;
  if (locale === "pt" || locale.startsWith("pt-")) return "pt-BR";
  if (locale === "es" || locale.startsWith("es-")) return "es";
  if (locale === "en" || locale.startsWith("en-")) return "en";
  return null;
}

export function detectLocale(languages?: readonly string[]): Locale {
  const preferred = languages ?? (typeof navigator === "undefined" ? [] : navigator.languages);
  for (const language of preferred) {
    const supported = normalizeLocale(language);
    if (supported) return supported;
  }
  return "pt-BR";
}
