/**
 * Vitest happy-dom test setup entrypoint.
 */

import { GlobalRegistrator } from "@happy-dom/global-registrator";

if (!(globalThis as any).window || !(globalThis as any).document) {
	GlobalRegistrator.register({
		settings: {
			enableJavaScriptEvaluation: true,
			suppressCodeGenerationFromStringsWarning: true,
		},
		url: "http://localhost:3000/",
	});
}

await import("./common-setup");
await import("./msw-happydom-setup");
