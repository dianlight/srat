import fs from "fs";
import packageJson from "./package.json";
import semver from "semver";
import { parseArgs } from "util";

const { values, positionals } = parseArgs({
    args: Bun.argv,
    options: {
        releaseType: {
            default: "prerelease",
            type: 'string',
            short: 'r'
        },
        identifier: {
            type: 'string',
            short: 'i'
        },
    },
    strict: true,
    allowPositionals: true
});
//const currentVersion = semver.parse(packageJson.version);
let newVersion = semver.inc(packageJson.version, values.releaseType as semver.ReleaseType, values.identifier);

if (newVersion && newVersion !== packageJson.version) {
    packageJson.version = newVersion;
    fs.writeFileSync("./package.json", JSON.stringify(packageJson, null, 2));
    console.log(`Version updated to ${newVersion}`);
} else {
    console.log("No version update required");
}