import copy from 'bun-copy-plugin';
import { watch } from "fs"
import { parseArgs } from "util";
import { file, type BuildOutput, type Serve } from "bun";
import { htmlLiveReload } from '@gtramontina.com/bun-html-live-reload';

const { values, positionals } = parseArgs({
    args: Bun.argv,
    options: {
        watch: {
            type: 'boolean',
            default: false,
            short: 'w'
        },
        serve: {
            type: 'string',
            short: 's'
        },
    },
    strict: true,
    allowPositionals: true
});


async function build(): Promise<BuildOutput | void> {
    return Bun.build({
        entrypoints: [/*'src/index.html',*/ 'src/index.tsx'],
        outdir: './out',  // Specify the output directory
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
            "process.env.APIURL": values.watch ? '"http://localhost:8080"' : "window.location.href.substring(0,window.location.href.lastIndexOf('/')+1)",
        }
    }).then((result) => {
        if (!result.success) {
            console.error("Build failed");
            for (const message of result.logs) {
                // Bun will pretty print the message object
                console.error(message);
            }
        }
        return result
    })
}

console.log(`Build ${import.meta.dir}/src`)
await build();
console.log('Build complete ✅')


if (values.watch) {
    console.log(`Build Watch ${import.meta.dir}/src`)
    const srcwatch = watch(
        `${import.meta.dir}/src`,
        { recursive: true },
        async (event, filename) => {
            console.log(`Detected ${event} in ${filename}`)
            await build();
            console.log('Build complete ✅')
        }
    )


    process.on('SIGINT', () => {
        srcwatch.close();
        process.exit(0);
    })
}

if (values.serve) {
    console.log(`Serving ${values.serve}`);
    const serve: Serve = {
        fetch(req: Request) {
            const url = new URL(req.url)
            const path = values.serve + url.pathname;
            console.log(`Request ${req.mode} ${url.pathname} ==> ${path}`)
            return new Response(Bun.file(path))
        },
        port: 3000
    }

    Bun.serve(htmlLiveReload(serve, { watchPath: import.meta.dir + "/src" }));
    console.log("Serving http://localhost:3000/index.html");
}



