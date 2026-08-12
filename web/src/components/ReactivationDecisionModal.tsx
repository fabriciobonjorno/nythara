import { useEffect, useRef, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { useNavigate } from "react-router-dom";
import { api } from "../api";
import { usePreferencesStore, useSessionStore } from "../store";
import type { Principal } from "../types";

export function ReactivationDecisionModal() {
  const principal = useSessionStore((state) => state.principal);
  const setPrincipal = useSessionStore((state) => state.setPrincipal);
  const resetGameplay = useSessionStore((state) => state.resetGameplay);
  const resetAccount = usePreferencesStore((state) => state.resetAccount);
  const beginTutorial = usePreferencesStore((state) => state.beginTutorial);
  const queryClient = useQueryClient();
  const navigate = useNavigate();
  const [busyChoice, setBusyChoice] = useState<"keep" | "reset" | null>(null);
  const [error, setError] = useState("");
	const dialogRef = useRef<HTMLElement>(null);

	useEffect(() => {
		if (!principal?.reactivation_reset_pending || !dialogRef.current) return;
		const dialog = dialogRef.current;
		const previousOverflow = document.body.style.overflow;
		document.body.style.overflow = "hidden";
		dialog.querySelector<HTMLElement>("button:not(:disabled)")?.focus();
		const keepFocusInside = (event: KeyboardEvent) => {
			if (event.key !== "Tab") return;
			const focusable = Array.from(dialog.querySelectorAll<HTMLElement>("button:not(:disabled)"));
			if (focusable.length === 0) return;
			const first = focusable[0];
			const last = focusable[focusable.length - 1];
			if (event.shiftKey && document.activeElement === first) { event.preventDefault(); last.focus(); }
			else if (!event.shiftKey && document.activeElement === last) { event.preventDefault(); first.focus(); }
		};
		document.addEventListener("keydown", keepFocusInside);
		return () => {
			document.removeEventListener("keydown", keepFocusInside);
			document.body.style.overflow = previousOverflow;
		};
	}, [principal?.reactivation_reset_pending]);

  if (!principal?.reactivation_reset_pending) return null;

  const decide = async (resetData: boolean) => {
    setBusyChoice(resetData ? "reset" : "keep");
    setError("");
    try {
      const updated = await api<Principal>("/v1/me/reactivation", {
        method: "POST",
        body: JSON.stringify({ reset_data: resetData }),
      });
      setPrincipal(updated);
      queryClient.setQueryData(["me"], updated);
      if (resetData) {
        resetGameplay();
        resetAccount(principal.user_id);
        beginTutorial(principal.user_id);
        navigate("/tutorial", { replace: true });
      }
	  await queryClient.invalidateQueries();
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : "Não foi possível concluir a reativação.");
    } finally {
      setBusyChoice(null);
    }
  };

  return <div className="account-modal-backdrop">
    <section ref={dialogRef} className="account-modal" role="dialog" aria-modal="true" aria-labelledby="reactivation-title" aria-describedby="reactivation-copy">
      <p className="eyebrow">CONTA REATIVADA</p>
      <h2 id="reactivation-title">Como você quer retornar?</h2>
      <p id="reactivation-copy">Sua conta voltou a ficar ativa. Você pode continuar exatamente de onde parou ou recomeçar com nível, coleção, baralho e histórico pessoal limpos.</p>
      <div className="reactivation-choices">
        <button className="secondary-button" type="button" disabled={busyChoice !== null} onClick={() => decide(false)}>
          <strong>Manter meus dados</strong><span>Preserva progresso, cartas, baralho e partidas.</span>
        </button>
        <button className="danger-button" type="button" disabled={busyChoice !== null} onClick={() => decide(true)}>
          <strong>{busyChoice === "reset" ? "Recomeçando…" : "Sim, quero recomeçar"}</strong><span>O reset não poderá ser desfeito depois da confirmação.</span>
        </button>
      </div>
      {error && <p className="form-error" role="alert">{error}</p>}
    </section>
  </div>;
}
