import { beforeEach, describe, expect, it } from "vitest";
import { needsFirstLoginTutorial, TUTORIAL_STEP_IDS, usePreferencesStore } from "./store";

describe("jornada do primeiro duelo", () => {
  beforeEach(() => {
    usePreferencesStore.setState({ onboardingUserId: null, tutorialByUser: {} });
  });

  it("começa uma vez e só termina depois dos seis marcos", () => {
    expect(needsFirstLoginTutorial("u1")).toBe(true);

    usePreferencesStore.getState().beginTutorial("u1");
    expect(needsFirstLoginTutorial("u1")).toBe(false);
    expect(usePreferencesStore.getState().tutorialByUser.u1.completed).toEqual([]);

    for (const step of TUTORIAL_STEP_IDS) usePreferencesStore.getState().completeTutorialStep("u1", step);
    expect(usePreferencesStore.getState().tutorialByUser.u1).toEqual({
      started: true,
      completed: [...TUTORIAL_STEP_IDS],
      finished: true,
    });
  });

  it("isola o progresso por usuário e permite recomeçar", () => {
    const preferences = usePreferencesStore.getState();
    preferences.beginTutorial("u1");
    preferences.completeTutorialStep("u1", "goal");
    preferences.beginTutorial("u2");

    expect(usePreferencesStore.getState().tutorialByUser.u1.completed).toEqual(["goal"]);
    expect(usePreferencesStore.getState().tutorialByUser.u2.completed).toEqual([]);

    usePreferencesStore.getState().restartTutorial("u1");
    expect(usePreferencesStore.getState().tutorialByUser.u1.completed).toEqual([]);
  });
});
