//import { $ } from "bun";
import fs from "node:fs";
import { parseArgs } from "node:util";
import semver from "semver";
import packageJson from "./package.json";

const { values } = parseArgs({
	args: Bun.argv,
	options: {
		releaseType: {
			default: "prerelease",
			type: "string",
			short: "r",
		},
		identifier: {
			type: "string",
			short: "i",
		},
		overwite: {
			type: "string",
			short: "o",
		},
	},
	strict: true,
	allowPositionals: true,
});
//const currentVersion = semver.parse(packageJson.version);
let newVersion: string | null = packageJson.version;
if (values.overwite && values.overwite?.trim() !== "TEST_INTERNAL") {
	newVersion = values.overwite;
} else if (values.overwite?.trim() !== "TEST_INTERNAL") {
	newVersion = semver.inc(
		packageJson.version,
		values.releaseType as semver.ReleaseType,
		true,
		values.identifier,
	);
}

if (newVersion && newVersion !== packageJson.version) {
	packageJson.version = newVersion;
	fs.writeFileSync("./package.json", JSON.stringify(packageJson, null, 2));
	console.log(`Version updated to ${newVersion}`);
	//await $`git add ./package.json`;
	//await $`git commit -m "chore(webapp): :bookmark: Mark frontend version ${newVersion}"`;
} else {
	console.log("No version update required");
}
