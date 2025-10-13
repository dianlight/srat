//import copy from 'bun-copy-plugin';

import { watch } from "node:fs";
import { parseArgs } from "node:util";
import type { BuildConfig, BuildOutput } from "bun";
import { Glob, $ } from "bun";
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
	minify: values.watch || values.serve ? false : true,
	plugins: [
		//copy("src/index.html", "out/index.html")
		//  html({})
	],
	define: {
		"process.env.APIURL": APIURL,
		"process.env.NODE_ENV": values.watch || values.serve ? "'development'" : "'production'",
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

		// Track last build time for live reload
		let lastBuildTime = Date.now();

		// Initial build
		await Bun.build(buildConfig);

		// Watch for file changes and rebuild
		watch(path.join(import.meta.dir, "src"), {
			recursive: true
		}).on("change", async (event, filename) => {
			console.log(`Change file: ${filename}`);
			try {
				const result = await Bun.build(buildConfig);
				if (!result.success) {
					console.error("Build failed during hot reload");
					for (const message of result.logs) {
						console.error(message);
					}
				} else {
					lastBuildTime = Date.now();
					console.log("Build successful, clients will reload");
				}
			} catch (error) {
				console.error("Hot reload: build error", error);
			}
		});

		Bun.serve({
			fetch: async (request, server) => {
				const url = new URL(request.url);

				// Server-Sent Events endpoint for live reload
				if (url.pathname === "/__hmr") {
					const clientBuildTime = parseInt(url.searchParams.get("t") || "0");
					if (clientBuildTime < lastBuildTime) {
						return new Response("reload", {
							headers: {
								"Content-Type": "text/plain",
								"Cache-Control": "no-cache",
							},
						});
					}
					// Long-polling: wait for changes
					return new Promise((resolve) => {
						const checkInterval = setInterval(() => {
							if (clientBuildTime < lastBuildTime) {
								clearInterval(checkInterval);
								resolve(new Response("reload", {
									headers: {
										"Content-Type": "text/plain",
										"Cache-Control": "no-cache",
									},
								}));
							}
						}, 100);
						// Timeout after 30 seconds
						setTimeout(() => {
							clearInterval(checkInterval);
							resolve(new Response("ok", {
								headers: {
									"Content-Type": "text/plain",
									"Cache-Control": "no-cache",
								},
							}));
						}, 30000);
					});
				}

				// DevTools endpoint
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

				// Serve index.html for root
				if (url.pathname === "/") {
					url.pathname = "/index.html";
				}

				console.log(`Values ${import.meta.dir} ${values.outDir} ${url.pathname}`);
				const tpath = path.normalize(path.join(import.meta.dir, values.outDir!, url.pathname));
				const afile = Bun.file(tpath);
				console.log(`Request ${request.mode} ${url.pathname} ==> ${tpath} ${await afile.exists()} [${afile.size}]`);

				// Inject live reload script into HTML files
				if (afile.type.startsWith("text/html") && await afile.exists()) {
					const html = await afile.text();
					const injectedHtml = html.replace(
						"</body>",
						`<script>
							(function() {
								let buildTime = Date.now();
								function checkForUpdates() {
									fetch("/__hmr?t=" + buildTime)
										.then(r => r.text())
										.then(msg => {
											if (msg === "reload") {
												location.reload();
											} else {
												// Continue polling
												setTimeout(checkForUpdates, 1000);
											}
										})
										.catch(() => setTimeout(checkForUpdates, 5000));
								}
								checkForUpdates();
							})();
						</script></body>`
					);
					return new Response(injectedHtml, {
						headers: {
							"Content-Type": "text/html",
						}
					});
				}

				return new Response(await afile.arrayBuffer(), {
					headers: {
						"Content-Type": afile.type,
					}
				});
			},
			port: 3080,
			idleTimeout: 60,
			development: true,
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
}

await build();
console.log(`Build complete ✅ [:${values.watch ? "👁️:watching" : "🧻:build"}]`);

/*
if (values.watch) {
}
	*/
