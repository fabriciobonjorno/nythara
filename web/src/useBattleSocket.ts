import { useCallback, useEffect, useRef, useState } from "react";
import { api, websocketURL } from "./api";
import { useSessionStore } from "./store";
import type { BattleEvent, BattleIntent, BattleServerMessage, BattleState, BattleTicket } from "./types";

interface PendingCommand { sequence: number; intent: BattleIntent }

function readPending(matchId: string): PendingCommand | null {
  try { return JSON.parse(sessionStorage.getItem(`vr-pending-${matchId}`) ?? "null"); } catch { return null; }
}

function writePending(matchId: string, pending: PendingCommand | null) {
  if (pending) sessionStorage.setItem(`vr-pending-${matchId}`, JSON.stringify(pending));
  else sessionStorage.removeItem(`vr-pending-${matchId}`);
}

export function useBattleSocket(matchId: string) {
  const [state, setState] = useState<BattleState | null>(null);
  const [events, setEvents] = useState<BattleEvent[]>([]);
  const [slot, setSlot] = useState<number | null>(null);
  const [status, setStatus] = useState<"connecting" | "connected" | "reconnecting" | "closed">("connecting");
  const [ready, setReady] = useState<[boolean, boolean]>([false, false]);
  const [deadline, setDeadline] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [pending, setPending] = useState<PendingCommand | null>(() => readPending(matchId));
  const socketRef = useRef<WebSocket | null>(null);
  const closedRef = useRef(false);
  const retryRef = useRef(0);
  const lastEventRef = useRef(-1);
  const confirmedRef = useRef(0);
  const pendingRef = useRef<PendingCommand | null>(pending);
  const setActiveMatch = useSessionStore((store) => store.setActiveMatch);
  const setLastBattle = useSessionStore((store) => store.setLastBattle);

  const rememberPending = useCallback((next: PendingCommand | null) => {
    pendingRef.current = next;
    setPending(next);
    writePending(matchId, next);
  }, [matchId]);

  const mergeEvents = useCallback((incoming: BattleEvent[] = []) => {
    if (!incoming.length) return;
    lastEventRef.current = Math.max(lastEventRef.current, ...incoming.map((event) => event.seq));
    setEvents((current) => {
      const merged = new Map(current.map((event) => [event.seq, event]));
      incoming.forEach((event) => merged.set(event.seq, event));
      return [...merged.values()].sort((a, b) => a.seq - b.seq);
    });
  }, []);

  useEffect(() => {
    closedRef.current = false;
    setActiveMatch(matchId);
    let reconnectTimer: number | undefined;

    const connect = async () => {
      if (closedRef.current) return;
      setStatus(retryRef.current ? "reconnecting" : "connecting");
      try {
        const ticket = await api<BattleTicket>(`/v1/battles/${matchId}/tickets`, { method: "POST", body: JSON.stringify({ mode: "player" }) });
        if (closedRef.current) return;
        const ws = new WebSocket(websocketURL(`/v1/battles/${matchId}/ws?ticket=${encodeURIComponent(ticket.ticket)}&after_event=${lastEventRef.current}`));
        socketRef.current = ws;
        ws.onopen = () => {
          retryRef.current = 0;
          setStatus("connected");
          setError(null);
          ws.send(JSON.stringify({ type: "ready" }));
        };
        ws.onmessage = (event) => {
          const message = JSON.parse(event.data) as BattleServerMessage;
          if (message.slot !== undefined) setSlot(message.slot);
          if (message.ready) setReady(message.ready);
          if (message.deadline !== undefined) setDeadline(message.deadline ?? null);
          if (message.state) setState(message.state);
          mergeEvents(message.events);

          if (message.type === "sync") {
            const confirmed = message.client_sequence ?? 0;
            confirmedRef.current = confirmed;
            const waiting = pendingRef.current;
            if (waiting && waiting.sequence <= confirmed) {
              rememberPending(null);
            } else if (waiting && waiting.sequence === confirmed + 1) {
              ws.send(JSON.stringify({ type: "command", client_sequence: waiting.sequence, command: waiting.intent }));
            } else if (waiting) {
              rememberPending(null);
            }
          }
          if (message.type === "events" && message.client_sequence) {
            confirmedRef.current = Math.max(confirmedRef.current, message.client_sequence);
            if (pendingRef.current?.sequence === message.client_sequence) rememberPending(null);
          }
          if (message.type === "error" && message.error) {
            setError(message.error.message);
            if (message.error.code === "sequence_gap" || message.error.code === "sequence_reuse") {
              ws.close(4000, "resync");
            } else {
              rememberPending(null);
            }
          }
        };
        ws.onclose = () => {
          if (socketRef.current === ws) socketRef.current = null;
          if (closedRef.current) return;
          retryRef.current += 1;
          const wait = Math.min(8000, 500 * 2 ** Math.min(retryRef.current, 4));
          setStatus("reconnecting");
          reconnectTimer = window.setTimeout(connect, wait);
        };
        ws.onerror = () => setError("A conexão oscilou. Reconectando à partida…");
      } catch (caught) {
        if (closedRef.current) return;
        setError(caught instanceof Error ? caught.message : "Não foi possível entrar na partida.");
        retryRef.current += 1;
        reconnectTimer = window.setTimeout(connect, Math.min(8000, 700 * 2 ** Math.min(retryRef.current, 4)));
      }
    };
    void connect();
    return () => {
      closedRef.current = true;
      if (reconnectTimer) window.clearTimeout(reconnectTimer);
      socketRef.current?.close(1000, "leaving");
      socketRef.current = null;
    };
  }, [matchId, mergeEvents, rememberPending, setActiveMatch]);

  useEffect(() => {
    if (!state?.over || slot === null) return;
    const snapshot = { matchId, slot, state, events, finishedAt: new Date().toISOString() };
    setLastBattle(snapshot);
    writePending(matchId, null);
  }, [events, matchId, setLastBattle, slot, state]);

  const send = useCallback((intent: BattleIntent) => {
    const ws = socketRef.current;
    if (!ws || ws.readyState !== WebSocket.OPEN) { setError("Aguarde a reconexão antes de agir."); return false; }
    if (pendingRef.current) { setError("Sua ação anterior ainda está sendo confirmada."); return false; }
    const command = { sequence: confirmedRef.current + 1, intent };
    rememberPending(command);
    setError(null);
    ws.send(JSON.stringify({ type: "command", client_sequence: command.sequence, command: intent }));
    return true;
  }, [rememberPending]);

  return { state, events, slot, status, ready, deadline, error, pending, send };
}
