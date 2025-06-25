import { useState, useEffect } from 'react';

export interface NewsItem {
    id: number;
    title: string;
    url: string;
    published_at: string;
}

const REPO_OWNER = 'dianlight';
const REPO_NAME = 'srat';
const GITHUB_API_URL = `https://api.github.com/repos/${REPO_OWNER}/${REPO_NAME}/discussions?category=announcements`;
const MAX_NEWS_ITEMS = 5;
const MAX_NEWS_AGE_MONTHS = 3;

export function useGithubNews() {
    const [news, setNews] = useState<NewsItem[]>([]);
    const [isLoading, setIsLoading] = useState(true);
    const [error, setError] = useState<Error | null>(null);

    useEffect(() => {
        const fetchNews = async () => {
            setIsLoading(true);
            setError(null);
            try {
                const response = await fetch(GITHUB_API_URL);
                if (!response.ok) {
                    throw new Error(`GitHub API request failed: ${response.statusText}`);
                }
                const data = await response.json();

                const threeMonthsAgo = new Date();
                threeMonthsAgo.setMonth(threeMonthsAgo.getMonth() - MAX_NEWS_AGE_MONTHS);

                const filteredNews = data
                    .filter((discussion: any) => new Date(discussion.created_at) > threeMonthsAgo)
                    .sort((a: any, b: any) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime())
                    .slice(0, MAX_NEWS_ITEMS)
                    .map((discussion: any): NewsItem => ({
                        id: discussion.id,
                        title: discussion.title,
                        url: discussion.html_url,
                        published_at: discussion.created_at,
                    }));
                setNews(filteredNews);
            } catch (e) {
                setError(e instanceof Error ? e : new Error('An unknown error occurred'));
                console.error("Failed to fetch GitHub news:", e);
            } finally {
                setIsLoading(false);
            }
        };

        fetchNews();
    }, []);

    return { news, isLoading, error };
}