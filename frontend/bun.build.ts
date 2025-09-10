//import copy from 'bun-copy-plugin';

import { watch } from "node:fs";
import { parseArgs } from "node:util";
import { withHtmlLiveReload, reloadClients } from "bun-html-live-reload";
import { type BuildConfig, type BuildOutput, Glob, type Serve, $ } from "bun";
import path from "node:path";

const { values, positionals } = parseArgs({
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
		apiContextUrl: {
			type: "string",
			short: "c",
			default: "'dynamic'",
			description:
				"Specify the URL of the API context (in watching mode) (default: dynamic)",
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

const APIURL = values.watch ? values.apiContextUrl || "" : "'dynamic'";
console.log(`API URL: ${APIURL}`);

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
	sourcemap: "inline",
	minify: values.watch ? false : true,
	plugins: [
		//copy("src/index.html", "out/index.html")
		//  html({})
	],
	define: {
		"process.env.APIURL": APIURL,
		"process.env.NODE_ENV": values.watch ? "'development'" : "'production'",
		"process.env.ROLLBAR_CLIENT_ACCESS_TOKEN": `"${process.env.ROLLBAR_CLIENT_ACCESS_TOKEN || 'disabled'}"`,
		"process.env.SERVER_EVENT_BACKEND": "'WS'", // SSE or WS
	},
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

		Bun.build(buildConfig);

		watch(path.join(import.meta.dir, "src"), {
			recursive: true
		}).on("change", async (event, filename) => {
			console.log(`Change file: ${filename}`);
			//await $`rm -r  ${import.meta.dir}/${values.outDir}`;
			await Bun.build(buildConfig);
			reloadClients();
		});

		Bun.serve({
			fetch: withHtmlLiveReload(async (request) => {
				const url = new URL(request.url);
				if (
					url.pathname === "/.well-known/appspecific/com.chrome.devtools.json"
				) {
					const { getDevtoolData } = require("./src/devtool/server");
					return new Response(JSON.stringify(getDevtoolData()), {
						headers: {
							"Content-Type": "application/json",
						},
					});
				}
				if (url.pathname === "/") {
					url.pathname = "/index.html";
				}
				console.log(`Values ${import.meta.dir} ${values.outDir} ${url.pathname}`);
				const tpath = path.normalize(path.join(import.meta.dir, values.outDir!, url.pathname));
				const afile = Bun.file(tpath)
				console.log(`Request ${request.mode} ${url.pathname} ==> ${tpath} ${await afile.exists()} [${afile.size}]`);
				return new Response(await afile.arrayBuffer(), {
					headers: {
						"Content-Type": afile.type,
					}
				});
			}),
			port: 3080,
			idleTimeout: 60, // Set idle timeout to 60 seconds (configurable)
			development: {
				console: true,
				hmr: true,
			},
		});
		console.log("Serving http://localhost:3080/index.html");
	} else if (values.watch) {
		console.log(`Build Watch ${import.meta.dir}/src -> ${values.outDir}`);
		async function rebuild(event: string, filename: string | null) {
			console.log(`Detected ${event} in ${filename}`);
			const glob = new Glob(`index-*`);

			for await (const file of glob.scan(values.outDir)) {
				console.log(`D ${values.outDir}/${file}`);
				Bun.file(`${values.outDir}/${file}`)
					.delete()
					.catch((_err) => { });
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
}

await build();
console.log(`Build complete ✅ [:${values.watch ? "👁️:watching" : "🧻:build"}]`);

/*
if (values.watch) {
}
	*/
