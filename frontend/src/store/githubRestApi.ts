import { createApi, fetchBaseQuery } from "@reduxjs/toolkit/query/react";

export interface NewsItem {
  id: number;
  title: string;
  url: string;
  published_at: string;
}

interface Discussion {
  id: number;
  title: string;
  html_url: string;
  created_at: string;
}

const REPO_OWNER = "dianlight";
const REPO_NAME = "srat";
const MAX_NEWS_ITEMS = 5;
const MAX_NEWS_AGE_MONTHS = 3;

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

        return (response || [])
          .filter((d) => new Date(d.created_at) > threeMonthsAgo)
          .sort(
            (a, b) =>
              new Date(b.created_at).getTime() -
              new Date(a.created_at).getTime(),
          )
          .slice(0, MAX_NEWS_ITEMS)
          .map((d) => ({
            id: d.id,
            title: d.title,
            url: d.html_url,
            published_at: d.created_at,
          }));
      },
      keepUnusedDataFor: 3600, // cache for 1 hour
    }),
  }),
});

export const { useGetDiscussionsQuery } = githubRestApi;
