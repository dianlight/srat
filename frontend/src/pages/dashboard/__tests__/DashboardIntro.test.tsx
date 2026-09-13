import { render, screen } from "@testing-library/react";
import { Provider } from "react-redux";
import { MemoryRouter } from "react-router";
import { describe, expect, it, vi } from "vitest";
import type { NewsItem } from "../../../hooks/githubNewsHook";
import { GITHUB_ANNOUNCEMENTS_URL } from "../../../store/githubRestApi";
import { testIds } from "../../../testIds";
import { DashboardIntro } from "../DashboardIntro";
import { createTestStore } from "/test/testing";

const releaseItem: NewsItem = {
  id: 1,
  title: "Release 2026.9.0-rc14",
  url: "https://github.com/dianlight/srat/discussions/1111",
  published_at: "2026-09-06T16:38:25.000Z",
  abstract: "Release abstract with changelog highlights.",
  type: "release",
  version: "2026.9.0-rc14",
};

const announcementItem: NewsItem = {
  id: 2,
  title: "Maintenance window",
  url: "https://github.com/dianlight/srat/discussions/2",
  published_at: "2026-09-01T12:00:00.000Z",
  abstract: "Announcement abstract with details.",
  type: "announcement",
};

async function renderIntro(news: NewsItem[], error: Error | null = null) {
  const store = await createTestStore();
  return render(
    <Provider store={store}>
      <MemoryRouter>
        <DashboardIntro
          isCollapsed={false}
          onToggleCollapse={vi.fn()}
          news={news}
          isLoading={false}
          error={error}
        />
      </MemoryRouter>
    </Provider>,
  );
}

describe("DashboardIntro news", () => {
  it("renders per-type icons with distinct testids", async () => {
    await renderIntro([releaseItem, announcementItem]);

    expect(
      screen.getByTestId(testIds.dashboard.newsReleaseIcon),
    ).toBeInTheDocument();
    expect(
      screen.getByTestId(testIds.dashboard.newsAnnouncementIcon),
    ).toBeInTheDocument();
  });

  it("renders the abstract under each title link", async () => {
    await renderIntro([releaseItem, announcementItem]);

    expect(
      screen.getByRole("link", { name: "Release 2026.9.0-rc14" }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("link", { name: "Maintenance window" }),
    ).toBeInTheDocument();

    const abstracts = screen.getAllByTestId(testIds.dashboard.newsAbstract);
    expect(abstracts).toHaveLength(2);
    expect(abstracts[0]).toHaveTextContent(
      "Release abstract with changelog highlights.",
    );
    expect(abstracts[1]).toHaveTextContent("Announcement abstract with details.");
  });

  it("keeps the archive link when there is no news", async () => {
    await renderIntro([]);

    expect(screen.getByText("Latest News:")).toBeInTheDocument();
    expect(
      screen.queryByTestId(testIds.dashboard.newsReleaseIcon),
    ).toBeNull();
    const link = screen.getByTestId(testIds.dashboard.newsSeeAll);
    expect(link).toBeInTheDocument();
    expect(link.getAttribute("href")).toBe(GITHUB_ANNOUNCEMENTS_URL);
  });

  it("links to the full announcements archive in a new tab", async () => {
    await renderIntro([releaseItem, announcementItem]);

    const link = screen.getByRole("link", { name: "See all news" });
    expect(link.getAttribute("href")).toBe(GITHUB_ANNOUNCEMENTS_URL);
    expect(link.getAttribute("target")).toBe("_blank");
  });

  it("keeps the archive link when loading news fails", async () => {
    await renderIntro([], new Error("boom"));

    expect(screen.getByText("Could not load project news.")).toBeInTheDocument();
    expect(screen.getByTestId(testIds.dashboard.newsSeeAll)).toBeInTheDocument();
  });
});
