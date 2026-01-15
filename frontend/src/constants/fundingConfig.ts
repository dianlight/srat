/**
 * Funding configuration parsed from .github/funding.yaml
 * This file should be updated when funding.yaml changes
 */

export interface FundingPlatform {
	platform: string;
	identifier: string;
	url: string;
	label: string;
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
		custom: (id) => id, // Custom URLs are used as-is
	};

	const urlBuilder = platformUrls[platform];
	return urlBuilder ? urlBuilder(identifier) : identifier;
}

/**
 * Funding platforms from .github/funding.yaml
 * Updated based on repository funding configuration
 */
export const FUNDING_PLATFORMS: FundingPlatform[] = [
	{
		platform: "github",
		identifier: "dianlight",
		url: buildFundingUrl("github", "dianlight"),
		label: "GitHub Sponsors",
	},
	{
		platform: "buy_me_a_coffee",
		identifier: "ypKZ2I0",
		url: buildFundingUrl("buy_me_a_coffee", "ypKZ2I0"),
		label: "Buy Me a Coffee",
	},
];
