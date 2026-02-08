import { useState, useEffect } from "react";
import { useGetApiIssuesTemplateQuery } from "../store/sratApi";
import type { IssueTemplate } from "../store/sratApi";

interface UseIssueTemplateReturn {
    template: IssueTemplate | null;
    isLoading: boolean;
    error: string | null;
    isAvailable: boolean;
}

/**
 * Hook to fetch and manage the GitHub issue template at app startup.
 * The template is fetched once and cached for the session.
 * @returns Template data, loading state, error state, and availability flag
 */
export function useIssueTemplate(): UseIssueTemplateReturn {
    const [template, setTemplate] = useState<IssueTemplate | null>(null);
    const [error, setError] = useState<string | null>(null);

    const { data, isLoading, error: queryError } = useGetApiIssuesTemplateQuery();

    useEffect(() => {
        if (data) {
            if ("template" in data && data.template) {
                setTemplate(data.template);
                setError(null);
            } else if ("error" in data && data.error) {
                setError(data.error);
                setTemplate(null);
            }
        } else if (queryError) {
            setError("Failed to fetch issue template");
            setTemplate(null);
        }
    }, [data, queryError]);

    return {
        template,
        isLoading,
        error,
        isAvailable: template !== null && !isLoading && !error,
    };
}
