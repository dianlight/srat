import { useGetDiscussionsQuery } from "../store/githubRestApi";

export interface NewsItem {
  id: number;
  title: string;
  url: string;
  published_at: string;
}

// Hook wrapper around the RTK Query endpoint. Keeping the same return
// shape as the legacy hook for compatibility with existing consumers.
export function useGithubNews() {
  const { data, isLoading, error } = useGetDiscussionsQuery();

  let mappedError: Error | null = null;
  if (error) {
    try {
      // error may be a serialized object from fetchBaseQuery
      const anyErr = error as unknown as { status?: number; error?: unknown };
      if (anyErr?.status) {
        mappedError = new Error(
          `GitHub API request failed: ${String(anyErr.status)}`,
        );
      } else if (anyErr?.error) {
        mappedError = new Error(String(anyErr.error));
      } else {
        mappedError = new Error(JSON.stringify(anyErr));
      }
    } catch {
      mappedError = new Error("Failed to fetch GitHub news");
    }
  }

  return { news: data ?? [], isLoading, error: mappedError };
}
