import { useEffect } from "react";
import { Navigate, Route, Routes, useLocation } from "react-router-dom";
import { Shell } from "./components/Shell";
import { Landing } from "./pages/Landing";
import { Home } from "./pages/Home";
import { ChampionsPage, CollectionPage } from "./pages/Catalog";
import { DeckBuilderPage } from "./pages/DeckBuilder";
import { QueuePage } from "./pages/Queue";
import { ArenaPage } from "./pages/Arena";
import { CronicaPage } from "./pages/Cronica";
import { AdminPage } from "./pages/Admin";
import { BattlePage } from "./pages/Battle";
import { ReplayPage } from "./pages/Replay";
import { Missing, ProfilePage, ResultPage, SettingsPage, TutorialPage } from "./pages/Secondary";
import { usePreferencesStore, useSessionStore } from "./store";
import { installDocumentLocalization } from "./documentLocalization";

function Protected({ children }: { children: React.ReactNode }) {
  const authenticated = useSessionStore((state) => Boolean(state.tokens));
  return authenticated ? children : <Navigate to="/" replace />;
}

function ScrollToTop() {
  const { pathname } = useLocation();
  useEffect(() => { window.scrollTo(0, 0); }, [pathname]);
  return null;
}

export default function App() {
  const preferences = usePreferencesStore();
  useEffect(() => installDocumentLocalization(preferences.locale), [preferences.locale]);
  useEffect(() => {
    document.documentElement.dataset.motion = preferences.reducedMotion ? "reduced" : "full";
    document.documentElement.dataset.contrast = preferences.highContrast ? "high" : "normal";
    document.documentElement.dataset.text = preferences.largeText ? "large" : "normal";
    document.documentElement.dataset.hints = preferences.combatHints ? "shown" : "hidden";
    document.documentElement.dataset.pace = preferences.animationPace;
  }, [preferences.animationPace, preferences.combatHints, preferences.highContrast, preferences.largeText, preferences.reducedMotion]);

  return <><ScrollToTop /><Routes>
    <Route path="/" element={<Landing />} />
    <Route path="/battle/:matchId" element={<Protected><BattlePage /></Protected>} />
    <Route element={<Protected><Shell /></Protected>}>
      <Route path="/app" element={<Home />} />
      <Route path="/collection" element={<CollectionPage />} />
      <Route path="/champions" element={<ChampionsPage />} />
      <Route path="/decks" element={<DeckBuilderPage />} />
      <Route path="/queue" element={<QueuePage />} />
      <Route path="/arena" element={<ArenaPage />} />
      <Route path="/cronica/:matchId" element={<CronicaPage />} />
      <Route path="/salao" element={<AdminPage />} />
      <Route path="/result" element={<ResultPage />} />
      <Route path="/replay" element={<ReplayPage />} />
      <Route path="/replay/:matchId" element={<ReplayPage />} />
      <Route path="/profile" element={<ProfilePage />} />
      <Route path="/tutorial" element={<TutorialPage />} />
      <Route path="/settings" element={<SettingsPage />} />
      <Route path="*" element={<Missing />} />
    </Route>
  </Routes></>;
}
