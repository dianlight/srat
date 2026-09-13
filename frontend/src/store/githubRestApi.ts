import { createApi, fetchBaseQuery } from "@reduxjs/toolkit/query/react";

export type NewsType = "release" | "announcement";

export interface NewsItem {
  id: number;
  title: string;
  url: string;
  published_at: string;
  abstract: string;
  type: NewsType;
  version?: string;
}

interface Discussion {
  id: number;
  title: string;
  html_url: string;
  created_at: string;
  body?: string;
}

const REPO_OWNER = "dianlight";
const REPO_NAME = "srat";
const ANNOUNCEMENTS_CATEGORY = "announcements";

/** Full archive of older releases and announcements. */
export const GITHUB_ANNOUNCEMENTS_URL = `https://github.com/${REPO_OWNER}/${REPO_NAME}/discussions/categories/${ANNOUNCEMENTS_CATEGORY}`;
const MAX_NEWS_ITEMS = 5;
const MAX_NEWS_AGE_MONTHS = 3;
const MAX_ABSTRACT_LENGTH = 200;

/**
 * Extract a release version from a discussion title or body.
 * Titles look like "Release 2026.8.0-rc12"; release-created discussions
 * also carry a footer link to `/releases/tag/<version>`.
 */
export function extractReleaseVersion(
  title: string,
  body?: string,
): string | undefined {
  const fromTitle = /^Release\s+(\S+)/i.exec(title.trim());
  if (fromTitle?.[1]) {
    return fromTitle[1].replace(/[),.\]]+$/, "");
  }
  if (body) {
    const fromBody = /releases\/tag\/([^\s"'<>)]+)/.exec(body);
    if (fromBody?.[1]) {
      return fromBody[1].replace(/[),.\]]+$/, "");
    }
  }
  return undefined;
}

export function detectNewsType(title: string, body?: string): NewsType {
  return extractReleaseVersion(title, body) !== undefined
    ? "release"
    : "announcement";
}

/**
 * Build a short plain-text abstract from a discussion markdown body.
 * Strips HTML tags, markdown links/images, headings and emphasis,
 * collapses whitespace and truncates at a word boundary.
 */
export function makeAbstract(
  body: string | undefined,
  maxLength = MAX_ABSTRACT_LENGTH,
): string {
  if (!body) {
    return "";
  }
  const plain = body
    .replace(/<[^>]*>/g, " ")
    .replace(/!\[([^\]]*)\]\([^)]*\)/g, "$1")
    .replace(/\[([^\]]*)\]\([^)]*\)/g, "$1")
    .replace(/[#>*_`~|-]+/g, " ")
    .replace(/\s+/g, " ")
    .trim();
  if (plain.length <= maxLength) {
    return plain;
  }
  const cut = plain.slice(0, maxLength);
  const lastSpace = cut.lastIndexOf(" ");
  return `${(lastSpace > maxLength * 0.5 ? cut.slice(0, lastSpace) : cut).trim()}…`;
}

export const githubRestApi = createApi({
  reducerPath: "githubRestApi",
  baseQuery: fetchBaseQuery({
    baseUrl: "https://api.github.com",
    prepareHeaders: (headers) => {
      // Use the GitHub v3 media type; discussions may require previews in some cases
      headers.set("Accept", "application/vnd.github.v3+json");
      return headers;
    },
  }),
  endpoints: (builder) => ({
    getDiscussions: builder.query<NewsItem[], void>({
      query: () =>
        `/repos/${REPO_OWNER}/${REPO_NAME}/discussions?category=announcements`,
      transformResponse: (response: Discussion[]) => {
        const threeMonthsAgo = new Date();
        threeMonthsAgo.setMonth(
          threeMonthsAgo.getMonth() - MAX_NEWS_AGE_MONTHS,
        );

        return (
          (response || [])
            .filter((d) => new Date(d.created_at) > threeMonthsAgo)
            .sort(
              (a, b) =>
                new Date(b.created_at).getTime() -
                new Date(a.created_at).getTime(),
            )
            .map((d) => {
              const type = detectNewsType(d.title, d.body);
              const version =
                type === "release"
                  ? extractReleaseVersion(d.title, d.body)
                  : undefined;
              return {
                id: d.id,
                title: d.title,
                url: d.html_url,
                published_at: d.created_at,
                abstract: makeAbstract(d.body),
                type,
                version,
              } satisfies NewsItem;
            })
            // Only real news: must carry a non-empty abstract.
            .filter((item) => item.abstract.length > 0)
            .slice(0, MAX_NEWS_ITEMS)
        );
      },
      keepUnusedDataFor: 3600, // cache for 1 hour
    }),
  }),
});

export const { useGetDiscussionsQuery } = githubRestApi;
