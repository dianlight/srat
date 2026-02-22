import '../../../test/setup';

import { describe, it, expect, afterEach } from "bun:test";
import { getCurrentEnv } from "../Environment";

const getEnv = () => {
	const proc = (globalThis as any).process || ((globalThis as any).process = { env: {} });
	proc.env = proc.env || {};
	return proc.env as Record<string, string | undefined>;
};

const originalEnv = { ...getEnv() };

const setEnv = (nodeEnv?: string, pkgVersion?: string) => {
	const env = getEnv();

	if (nodeEnv === undefined) {
		delete env.NODE_ENV;
	} else {
		env.NODE_ENV = nodeEnv;
	}

	if (pkgVersion === undefined) {
		delete env.npm_package_version;
	} else {
		env.npm_package_version = pkgVersion;
	}
};

afterEach(() => {
	const env = getEnv();
	Object.keys(env).forEach((key) => {
		delete env[key];
	});
	Object.assign(env, originalEnv);
});

describe("getCurrentEnv", () => {
	it("returns NODE_ENV when not production", () => {
		setEnv("development", "2025.12.0-rc.1");

		expect(getCurrentEnv()).toBe("development");
	});

	it("maps production to development when version is a dev build", () => {
		setEnv("production", "2025.12.0-dev.3");

		expect(getCurrentEnv()).toBe("development");
	});

	it("maps production to prerelease when version is a release candidate", () => {
		setEnv("production", "2025.12.0-rc.4");

		expect(getCurrentEnv()).toBe("prerelease");
	});

	it("returns production when version has no prerelease suffix", () => {
		setEnv("production", "2025.12.0");

		expect(getCurrentEnv()).toBe("production");
	});
});
