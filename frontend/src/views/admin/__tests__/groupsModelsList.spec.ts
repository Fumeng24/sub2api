import { describe, expect, it } from "vitest";

import {
  buildModelsListConfig,
  createModelsListState,
  hydrateModelsListState,
  invertModelsListSelection,
  moveModelsListItem,
  selectAllModelsListItems,
  setModelsListCandidates,
  toggleModelsListItem,
} from "../groupsModelsList";

describe("groupsModelsList", () => {
  it("selects all default candidates for a new disabled config", () => {
    const state = createModelsListState();

    setModelsListCandidates(state, ["gpt-5.5", "gpt-5.4"]);

    expect(state.enabled).toBe(false);
    expect(state.items).toEqual([
      { id: "gpt-5.5", selected: true },
      { id: "gpt-5.4", selected: true },
    ]);
  });

  it("keeps saved selections that are still backed by account candidates", () => {
    const state = createModelsListState({
      enabled: true,
      models: ["gpt-5.5", "gpt-5.4"],
    });

    setModelsListCandidates(state, ["gpt-5.4", "legacy-gpt", "gpt-5.5"]);

    expect(state.enabled).toBe(true);
    expect(state.items).toEqual([
      { id: "gpt-5.5", selected: true },
      { id: "gpt-5.4", selected: true },
      { id: "legacy-gpt", selected: false },
    ]);
  });

  it("drops saved models that are no longer supported by group accounts", () => {
    const state = createModelsListState({
      enabled: true,
      models: ["gpt-5.5", "stale-pro-model"],
    });

    setModelsListCandidates(state, ["gpt-5.5", "gpt-5.4"]);

    expect(state.items).toEqual([
      { id: "gpt-5.5", selected: true },
      { id: "gpt-5.4", selected: false },
    ]);
    expect(buildModelsListConfig(state)).toEqual({
      enabled: true,
      models: ["gpt-5.5"],
    });
  });

  it("clears saved models after an empty account candidate list has loaded", () => {
    const state = createModelsListState({
      enabled: true,
      models: ["stale-pro-model"],
    });

    setModelsListCandidates(state, []);

    expect(state.items).toEqual([]);
    expect(buildModelsListConfig(state)).toEqual({
      enabled: true,
      models: [],
    });
  });

  it("preserves explicitly unselected saved candidates when candidates refresh", () => {
    const state = createModelsListState({
      enabled: true,
      models: ["gpt-5.5"],
    });

    setModelsListCandidates(state, ["gpt-5.5", "gpt-5.4"]);

    expect(state.items).toEqual([
      { id: "gpt-5.5", selected: true },
      { id: "gpt-5.4", selected: false },
    ]);
  });

  it("builds config with selected models in current display order", () => {
    const state = hydrateModelsListState({
      enabled: true,
      models: ["gpt-5.5", "gpt-5.4", "legacy-gpt"],
    }, ["gpt-5.5", "gpt-5.4", "legacy-gpt"]);

    toggleModelsListItem(state, "legacy-gpt");
    moveModelsListItem(state, 1, 0);

    expect(buildModelsListConfig(state)).toEqual({
      enabled: true,
      models: ["gpt-5.4", "gpt-5.5"],
    });
  });

  it("keeps selected models in payload even when disabled so reopening can restore choices", () => {
    const state = hydrateModelsListState({
      enabled: false,
      models: ["gpt-5.5"],
    }, ["gpt-5.5", "gpt-5.4"]);

    expect(buildModelsListConfig(state)).toEqual({
      enabled: false,
      models: ["gpt-5.5"],
    });
  });

  it("preserves saved models when candidates have not loaded yet", () => {
    const state = createModelsListState({
      enabled: true,
      models: ["gpt-5.5", "gpt-5.4"],
    });

    expect(buildModelsListConfig(state)).toEqual({
      enabled: true,
      models: ["gpt-5.5", "gpt-5.4"],
    });
  });

  it("selects all candidate models from the toolbar action", () => {
    const state = hydrateModelsListState({
      enabled: true,
      models: ["gpt-5.5"],
    }, ["gpt-5.5", "gpt-5.4", "gpt-5.4-mini"]);

    selectAllModelsListItems(state);

    expect(state.items).toEqual([
      { id: "gpt-5.5", selected: true },
      { id: "gpt-5.4", selected: true },
      { id: "gpt-5.4-mini", selected: true },
    ]);
  });

  it("inverts selected models from the toolbar action", () => {
    const state = hydrateModelsListState({
      enabled: true,
      models: ["gpt-5.5"],
    }, ["gpt-5.5", "gpt-5.4", "gpt-5.4-mini"]);

    invertModelsListSelection(state);

    expect(state.items).toEqual([
      { id: "gpt-5.5", selected: false },
      { id: "gpt-5.4", selected: true },
      { id: "gpt-5.4-mini", selected: true },
    ]);
  });
});
