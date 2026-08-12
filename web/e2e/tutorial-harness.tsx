import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createRoot } from "react-dom/client";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { TutorialPage } from "../src/pages/Secondary";
import { usePreferencesStore, useSessionStore } from "../src/store";
import "../src/styles.css";

const userId = "tutorial-browser-user";
const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false, staleTime: Infinity } } });
queryClient.setQueryData(["decks"], { decks: [] });
queryClient.setQueryData(["ruleset", "current"], { version: "alpha-0.13.0" });

useSessionStore.getState().setAuth(
  { id: userId, email: "tutorial@example.test", display_name: "Viajante", role: "player", password_set: true, created_at: "2026-08-12T00:00:00Z" },
  { access_token: "local", refresh_token: "local", access_expires_at: "2099-01-01T00:00:00Z", refresh_expires_at: "2099-01-02T00:00:00Z" },
);
usePreferencesStore.setState({ locale: "pt-BR", tutorialByUser: {} });

function Destination() {
  const completed = usePreferencesStore((state) => state.tutorialByUser[userId]?.completed ?? []);
  return <main className="page"><h1>Destino Avatares</h1><output data-testid="completed">{completed.join(",")}</output></main>;
}

createRoot(document.getElementById("root")!).render(
  <QueryClientProvider client={queryClient}>
    <MemoryRouter initialEntries={["/tutorial"]}>
      <Routes>
        <Route path="/tutorial" element={<TutorialPage />} />
        <Route path="/champions" element={<Destination />} />
      </Routes>
    </MemoryRouter>
  </QueryClientProvider>,
);
