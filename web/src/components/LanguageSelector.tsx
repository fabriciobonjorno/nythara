import { localeNames, supportedLocales, type Locale } from "../locales";
import { usePreferencesStore } from "../store";
import { useEffect } from "react";
import { localizeDocument } from "../documentLocalization";

export function LanguageSelector({ compact = false }: { compact?: boolean }) {
  const locale = usePreferencesStore((state) => state.locale);
  const setLocale = usePreferencesStore((state) => state.setLocale);
  useEffect(() => { localizeDocument(locale); }, [locale]);
  return <label className={`language-selector ${compact ? "is-compact" : ""}`}>
    {!compact && <span>Idioma</span>}
    <select aria-label="Escolher idioma" value={locale} onChange={(event) => setLocale(event.target.value as Locale)}>
      {supportedLocales.map((item) => <option value={item} key={item}>{localeNames[item]}</option>)}
    </select>
  </label>;
}
