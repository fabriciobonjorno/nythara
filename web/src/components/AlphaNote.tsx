import { useEffect, useRef, useState } from "react";
import { api } from "../api";
import { UiIcon } from "./UiIcon";
import { hasFinePointer } from "../mobileViewport";

// Convite, não formulário. O Alpha precisa de leitura de quem joga, mas quem
// acabou de perder um duelo não deve encontrar uma cobrança na frente: o bloco
// começa fechado, cabe em uma linha, some quando dispensado e nunca bloqueia o
// caminho para a próxima partida.

const MAX_LENGTH = 2000;

// Quem já respondeu ou dispensou não é perguntado de novo sobre a mesma
// partida. A marca é local: é preferência de quem joga, não dado de produto.
function noteKey(matchId?: string) {
  return matchId ? `nythara-alpha-note:${matchId}` : "";
}

function alreadyAnswered(matchId?: string) {
  const key = noteKey(matchId);
  if (!key) return false;
  try {
    return Boolean(window.localStorage.getItem(key));
  } catch {
    return false;
  }
}

function remember(matchId: string | undefined, value: "sent" | "dismissed") {
  const key = noteKey(matchId);
  if (!key) return;
  try {
    window.localStorage.setItem(key, value);
  } catch {
    // Armazenamento bloqueado apenas faz o convite reaparecer; nada quebra.
  }
}

export function AlphaNote({ matchId }: { matchId?: string }) {
  const [open, setOpen] = useState(false);
  const [dismissed, setDismissed] = useState(() => alreadyAnswered(matchId));
  const [message, setMessage] = useState("");
  const [status, setStatus] = useState<"idle" | "sending" | "sent" | "error">("idle");
  const [error, setError] = useState("");
  const fieldRef = useRef<HTMLTextAreaElement>(null);

  useEffect(() => {
    if (open && hasFinePointer()) fieldRef.current?.focus({ preventScroll: true });
  }, [open]);

  const dismiss = () => { remember(matchId, "dismissed"); setDismissed(true); };

  if (dismissed) return null;

  if (status === "sent") {
    return <section className="alpha-note is-sent" role="status">
      <span className="alpha-note__mark"><UiIcon name="check" /></span>
      <div><strong>Anotado — obrigado mesmo.</strong><small>Sua leitura vai direto para quem ajusta o jogo.</small></div>
    </section>;
  }

  const send = async () => {
    const trimmed = message.trim();
    if (!trimmed) return;
    setStatus("sending");
    setError("");
    try {
      await api<void>("/v1/feedback", {
        method: "POST",
        body: JSON.stringify({ match_id: matchId, message: trimmed }),
      });
      remember(matchId, "sent");
      setStatus("sent");
    } catch (caught) {
      setStatus("error");
      setError(caught instanceof Error ? caught.message : "Não foi possível enviar agora.");
    }
  };

  return <section className="alpha-note" aria-labelledby="alpha-note-title">
    <header>
      <span className="alpha-note__mark"><UiIcon name="guide" /></span>
      <div>
        <p className="eyebrow">NYTHARA EM ALPHA</p>
        <h3 id="alpha-note-title">Tem algo que daria para melhorar?</h3>
        <p>Se quiser contar como foi este duelo, a gente lê. É totalmente opcional — pode seguir direto para a próxima partida.</p>
      </div>
      <button type="button" className="alpha-note__close" onClick={dismiss} aria-label="Dispensar o convite">
        <UiIcon name="close" />
      </button>
    </header>

    {open
      ? <div className="alpha-note__form">
          <label className="sr-only" htmlFor="alpha-note-field">O que podemos melhorar</label>
          <textarea
            ref={fieldRef}
            id="alpha-note-field"
            value={message}
            maxLength={MAX_LENGTH}
            rows={4}
            placeholder="O que atrapalhou, o que agradou, o que faltou…"
            onChange={(event) => setMessage(event.target.value)}
          />
          <div className="alpha-note__actions">
            <small>{message.length}/{MAX_LENGTH}</small>
            <button type="button" className="ghost-button" onClick={dismiss}>Agora não</button>
            <button type="button" className="primary-button" disabled={!message.trim() || status === "sending"} onClick={send}>
              {status === "sending" ? "Enviando…" : "Enviar"}
            </button>
          </div>
          {status === "error" && <p className="form-error" role="alert">{error}</p>}
        </div>
      : <button type="button" className="alpha-note__open" onClick={() => setOpen(true)}>
          Deixar um recado<UiIcon name="arrow-right" />
        </button>}
  </section>;
}
