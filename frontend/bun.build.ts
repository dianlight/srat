//import copy from 'bun-copy-plugin';

import { Glob } from "bun";
import type { BuildConfig, BuildOutput } from "bun";
import { watch } from "node:fs";
import { parseArgs } from "node:util";
//import path from "node:path";
import App from "./src/index.html";

const { values } = parseArgs({
	args: Bun.argv,
	options: {
		watch: {
			type: "boolean",
			default: false,
			short: "w",
			description: "Watch for changes and rebuild automatically",
		},
		serve: {
			type: "boolean",
			short: "s",
			description: "Start the HTTP Server",
		},
		outDir: {
			type: "string",
			short: "o",
			default: "./out",
			description: "Specify the output directory (default: ./out)",
		},
	},
	strict: true,
	allowPositionals: true,
});

const buildConfig: BuildConfig = {
	entrypoints: ["src/index.html" /*, 'src/index.tsx' */],
	outdir: values.outDir, // Specify the output directory
	//experimentalCss: true,
	naming: {
		entry: "[dir]/[name].[ext]",
		chunk: "[name]-[hash].[ext]",
		asset: "[name].[ext]",
	},
	target: "browser",
	// Use inline sourcemaps during watch/serve for better DX, external in production to avoid bloating bundles
	sourcemap: "inline", // values.watch || values.serve ? "inline" : "external",
	minify: values.watch || values.serve ? false : true,
	// Avoid bundling optional prism-based highlighters that reference unsupported subpath exports
	external: [
		"refractor",
		"refractor/*",
		"react-syntax-highlighter/dist/esm/prism*",
	],
	plugins: [
		//copy("src/index.html", "out/index.html")
		//  html({})
	],
	tsconfig: values.watch || values.serve ? "tsconfig.json" : "tsconfig.prod.json",
	/*
	define: {
		"process.env.APIURL": APIURL,
		"process.env.NODE_ENV": values.watch || values.serve ? "'development'" : "'production'",
		"process.env.ROLLBAR_CLIENT_ACCESS_TOKEN": `"${process.env.ROLLBAR_CLIENT_ACCESS_TOKEN || 'disabled'}"`,
		"process.env.SERVER_EVENT_BACKEND": "'WS'", // SSE or WS
	},
	*/
};

async function build(): Promise<BuildOutput | undefined> {
	if (!values.serve && !values.watch) {
		console.log(`Build ${import.meta.dir}/src -> ${values.outDir}`);
		return Bun.build(buildConfig).then((result) => {
			if (!result.success) {
				console.error("Build failed");
				for (const message of result.logs) {
					// Bun will pretty print the message object
					console.error(message);
				}
			}
			return result;
		});
	} else if (values.serve) {
		console.log(`Serving ${values.outDir}`);
		Bun.serve({
			routes:
			{
				"/*": App,
			},
			port: 3080,
			idleTimeout: 60,
			development: {
				chromeDevToolsAutomaticWorkspaceFolders: true,
				console: true,
				hmr: true,
			},
		});
		console.log("Serving http://localhost:3080/index.html");
	} else if (values.watch) {
		console.log(`Build Watch ${import.meta.dir}/src -> ${values.outDir}`);
		async function rebuild(event: string, filename: string | null) {
			console.log(`Detected ${event} in ${filename}`);

			// Only clean up old index files if the output directory exists
			try {
				const glob = new Glob(`index-*`);
				for await (const file of glob.scan(values.outDir)) {
					console.log(`D ${values.outDir}/${file}`);
					Bun.file(`${values.outDir}/${file}`)
						.delete()
						.catch((_err) => { });
				}
			} catch (err) {
				// Directory might not exist yet on first build
			}

			Bun.build(buildConfig).then((result) => {
				if (!result.success) {
					console.error("Build failed");
					for (const message of result.logs) {
						// Bun will pretty print the message object
						console.error(message);
					}
				}
				return result;
			});
			console.log("ReBuild complete ✅");
		}
		const srcwatch = watch(
			`${import.meta.dir}/src`,
			{ recursive: true },
			async (event, filename) => {
				rebuild(event, filename);
			},
		);

		process.on("SIGINT", () => {
			srcwatch.close();
			process.exit(0);
		});
		rebuild("initial build", null);
	}
	return undefined;
}

await build();
console.log(`Build complete ✅ [:${values.watch ? "👁️:watching" : "🧻:build"}]`);
