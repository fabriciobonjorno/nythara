package engine_test

import (
	"encoding/json"
	"testing"

	"veurubro/backend/internal/engine"
)

func TestRestoreSnapshotContinuesDeterministically(t *testing.T) {
	h := newHarness(t, "CH-VH-01", "CH-SO-01", deckWith("VR-001"), deckWith("VR-014"), 0)
	h.must(engine.Command{Player: 0, Kind: engine.CmdKindMulligan})
	snapshot, err := h.g.SnapshotJSON()
	if err != nil {
		t.Fatal(err)
	}
	restored, err := engine.RestoreSnapshot(h.g.Cfg, snapshot, h.g.Log, h.g.CommandLog)
	if err != nil {
		t.Fatal(err)
	}
	command := engine.Command{Player: 1, Kind: engine.CmdKindMulligan}
	want, err := h.g.Apply(command)
	if err != nil {
		t.Fatal(err)
	}
	got, err := restored.Apply(command)
	if err != nil {
		t.Fatal(err)
	}
	wantJSON, _ := json.Marshal(want)
	gotJSON, _ := json.Marshal(got)
	if string(wantJSON) != string(gotJSON) {
		t.Fatalf("continuação divergiu:\nwant=%s\ngot=%s", wantJSON, gotJSON)
	}
	wantState, _ := h.g.SnapshotJSON()
	gotState, _ := restored.SnapshotJSON()
	if string(wantState) != string(gotState) {
		t.Fatal("estado restaurado divergiu após catch-up")
	}
}
