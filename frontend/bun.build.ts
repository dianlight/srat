import copy from 'bun-copy-plugin';
import { watch } from "fs"
import { parseArgs } from "util";
import { file, type BuildConfig, type BuildOutput, type Serve } from "bun";
import { htmlLiveReload } from '@gtramontina.com/bun-html-live-reload';

const { values, positionals } = parseArgs({
    args: Bun.argv,
    options: {
        watch: {
            type: 'boolean',
            default: false,
            short: 'w',
            description: 'Watch for changes and rebuild automatically'
        },
        serve: {
            type: 'string',
            short: 's',
            description: 'Specify the host and port to serve the application'
        },
        apiContextUrl: {
            type: 'string',
            short: 'c',
            default: 'http://localhost:8080',
            description: 'Specify the URL of the API context (in watching mode) (default: http://localhost:8080)'
        },
    },
    strict: true,
    allowPositionals: true
});

const APIURL = values.watch ? values.apiContextUrl || "" : "'dynamic'"
console.log(`API URL: ${APIURL}`)

const buildConfig: BuildConfig = {
    entrypoints: [/*'src/index.html',*/ 'src/index.tsx'],
    outdir: './out',  // Specify the output directory
    experimentalCss: true,
    naming: {
        entry: "[dir]/[name].[ext]",
        chunk: '[name]-[hash].[ext]',
        asset: '[name].[ext]',
    },
    target: "browser",
    sourcemap: "inline",
    minify: true,
    plugins: [
        copy("src/index.html", "out/index.html")
        //  html({})
    ],
    define: {
        "process.env.APIURL": APIURL,
        "process.env.NODE_ENV": values.watch ? "'development'" : "'production'"
    }
}

async function build(): Promise<BuildOutput | void> {
    if (!values.serve) {
        console.log(`Build ${import.meta.dir}/src`)
        return Bun.build(buildConfig).then((result) => {
            if (!result.success) {
                console.error("Build failed");
                for (const message of result.logs) {
                    // Bun will pretty print the message object
                    console.error(message);
                }
            }
            return result
        })
    } else {
        console.log(`Serving ${values.serve}`);
        const serve: Serve = {
            fetch(req: Request) {
                const url = new URL(req.url)
                if (url.pathname === "/") {
                    url.pathname = "/index.html"
                }
                const path = values.serve + url.pathname;
                console.log(`Request ${req.mode} ${url.pathname} ==> ${path}`)
                return new Response(Bun.file(path))
            },
            port: 3000
        }

        Bun.serve(htmlLiveReload(serve, { buildConfig, watchPath: import.meta.dir + "/src" }));
        console.log("Serving http://localhost:3000/index.html");
    }
}

await build();
console.log(`Build complete ✅ [:${values.watch ? '👁️:watching' : '🧻:build'}]`)

/*
if (values.watch) {
    console.log(`Build Watch ${import.meta.dir}/src`)
    const srcwatch = watch(
        `${import.meta.dir}/src`,
        { recursive: true },
        async (event, filename) => {
            console.log(`Detected ${event} in ${filename}`)
            //await build();
            await htmlLiveReload.
            console.log('Build complete ✅')
        }
    )


    process.on('SIGINT', () => {
        srcwatch.close();
        process.exit(0);
    })
}
    */


