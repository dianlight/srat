import "../../test/setup";
import { useEffect, useState } from "react";

const IGNORED_ISSUES_KEY = "srat_ignored_issues";

export function useIgnoredIssues() {
	const [ignoredIssues, setIgnoredIssues] = useState<(number | string)[]>(
		() => {
			const saved = localStorage.getItem(IGNORED_ISSUES_KEY);
			return saved ? JSON.parse(saved) : [];
		},
	);

	useEffect(() => {
		localStorage.setItem(IGNORED_ISSUES_KEY, JSON.stringify(ignoredIssues));
	}, [ignoredIssues]);

	const ignoreIssue = (id: number | string) => {
		setIgnoredIssues((prev) => [...prev, id]);
	};

	const unignoreIssue = (id: number | string) => {
		setIgnoredIssues((prev) => prev.filter((issueId) => issueId !== id));
	};

	const isIssueIgnored = (id: number | string) => {
		return ignoredIssues.includes(id);
	};

	return {
		ignoredIssues,
		ignoreIssue,
		unignoreIssue,
		isIssueIgnored,
	};
}
