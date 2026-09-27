/* eslint-disable */
import { screen } from "@testing-library/react";
import React from "react";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { DashboardActions } from "../DashboardActions";
import { renderWithTestStore } from "/test/testing";

// Controllable useVolume mock: drives the Dashboard render chain without a
// live WebSocket, so the test asserts the user-visible 006.002 expectation
// (Actionable Items row flips on unmount without reload).
const volumeMock = vi.hoisted(() => ({
  disks: [] as unknown[],
  isLoading: false,
  error: undefined as unknown,
}));

vi.mock("../../../hooks/volumeHook", () => ({
  useVolume: () => ({
    disks: volumeMock.disks,
    isLoading: volumeMock.isLoading,
    error: volumeMock.error,
  }),
}));

const mountedDisk = {
  id: "disk-fmtui01",
  partitions: {
    sda1: {
      id: "part-1",
      name: "FMTUI01",
      mount_point_data: {
        mnt1: { path: "/mnt/FMTUI01", is_mounted: true },
      },
    },
  },
};

const unmountedDisk = {
  id: "disk-fmtui01",
  partitions: {
    sda1: {
      id: "part-1",
      name: "FMTUI01",
      mount_point_data: {
        mnt1: { path: "/mnt/FMTUI01", is_mounted: false },
      },
    },
  },
};

describe("DashboardActions unmount refresh (issue #1253)", () => {
  beforeEach(() => {
    volumeMock.disks = [];
    volumeMock.isLoading = false;
    volumeMock.error = undefined;
  });

  it("flips the Actionable Items row back to not-mounted on unmount without reload", async () => {
    volumeMock.disks = [mountedDisk] as unknown[];
    const rendered = await renderWithTestStore(
      React.createElement(
        MemoryRouter,
        null,
        React.createElement(DashboardActions as any),
      ),
    );

    // Mount half: mounted partition offers the share action.
    expect(
      await screen.findByText("This partition is mounted but not shared."),
    ).toBeTruthy();
    expect(await screen.findByText("Create Share")).toBeTruthy();

    // Unmount half: the same list must flip back without a reload.
    volumeMock.disks = [unmountedDisk] as unknown[];
    const { Provider } = await import("react-redux");
    rendered.rerender(
      React.createElement(
        Provider,
        {
          store: rendered.store,
          children: React.createElement(
            MemoryRouter,
            null,
            React.createElement(DashboardActions as any),
          ),
        } as any,
      ),
    );

    expect(
      await screen.findByText("This partition is not mounted."),
    ).toBeTruthy();
    expect(await screen.findByText("Mount")).toBeTruthy();
  });
});
