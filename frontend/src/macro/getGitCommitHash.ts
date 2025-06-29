export function getGitCommitHash() {
	const { stdout } = Bun.spawnSync({
		cmd: ["git", "rev-parse", "HEAD"],
		stdout: "pipe",
	});

	return stdout.toString();
}
