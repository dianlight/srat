import { createApi, fetchBaseQuery } from "@reduxjs/toolkit/query/react";
import * as yaml from "js-yaml";

/**
 * Interface representing a funding platform from funding.yaml
 */
export interface FundingPlatform {
	platform: string;
	identifier: string;
	url: string;
	label: string;
}

/**
 * Raw structure of funding.yaml from GitHub
 */
interface FundingYaml {
	github?: string | string[];
	patreon?: string;
	open_collective?: string;
	ko_fi?: string;
	tidelift?: string;
	community_bridge?: string;
	liberapay?: string;
	issuehunt?: string;
	otechie?: string;
	lfx_crowdfunding?: string;
	custom?: string | string[];
	buy_me_a_coffee?: string;
}

/**
 * Maps funding platform identifiers to their full URLs
 */
function buildFundingUrl(platform: string, identifier: string): string {
	const platformUrls: Record<string, (id: string) => string> = {
		github: (id) => `https://github.com/sponsors/${id}`,
		buy_me_a_coffee: (id) => `https://www.buymeacoffee.com/${id}`,
		patreon: (id) => `https://www.patreon.com/${id}`,
		ko_fi: (id) => `https://ko-fi.com/${id}`,
		open_collective: (id) => `https://opencollective.com/${id}`,
		liberapay: (id) => `https://liberapay.com/${id}`,
		issuehunt: (id) => `https://issuehunt.io/r/${id}`,
		otechie: (id) => `https://otechie.com/${id}`,
		custom: (id) => id, // Custom URLs are used as-is
	};

	const urlBuilder = platformUrls[platform];
	return urlBuilder ? urlBuilder(identifier) : identifier;
}

/**
 * Maps platform names to human-readable labels
 */
function getPlatformLabel(platform: string): string {
	const labels: Record<string, string> = {
		github: "GitHub Sponsors",
		buy_me_a_coffee: "Buy Me a Coffee",
		patreon: "Patreon",
		ko_fi: "Ko-fi",
		open_collective: "Open Collective",
		liberapay: "Liberapay",
		issuehunt: "IssueHunt",
		otechie: "Otechie",
		tidelift: "Tidelift",
		community_bridge: "Community Bridge",
		lfx_crowdfunding: "LFX Crowdfunding",
		custom: "Support",
	};

	return labels[platform] || platform;
}

/**
 * Parses funding.yaml content and transforms it into FundingPlatform array
 */
function parseFundingYaml(yamlContent: string): FundingPlatform[] {
	try {
		const parsed = yaml.load(yamlContent) as FundingYaml;
		const platforms: FundingPlatform[] = [];

		// Process each platform in the YAML
		Object.entries(parsed).forEach(([platform, value]) => {
			if (!value) return;

			// Handle both single values and arrays
			const identifiers = Array.isArray(value) ? value : [value];

			identifiers.forEach((identifier) => {
				platforms.push({
					platform,
					identifier,
					url: buildFundingUrl(platform, identifier),
					label: getPlatformLabel(platform),
				});
			});
		});

		return platforms;
	} catch (error) {
		console.error("Failed to parse funding.yaml:", error);
		return [];
	}
}

/**
 * GitHub API for fetching repository metadata
 * Uses caching to minimize API calls
 */
export const githubApi = createApi({
	reducerPath: "githubApi",
	baseQuery: fetchBaseQuery({
		baseUrl: "https://raw.githubusercontent.com",
	}),
	// Cache for 24 hours (86400 seconds)
	keepUnusedDataFor: 86400,
	endpoints: (builder) => ({
		getFundingConfig: builder.query<FundingPlatform[], void>({
			query: () => "/dianlight/srat/main/.github/funding.yaml",
			transformResponse: (response: string) => parseFundingYaml(response),
		}),
	}),
});

export const { useGetFundingConfigQuery } = githubApi;
