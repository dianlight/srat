#!/usr/bin/env bash
# Windows/WSL with HTTP_PROXY setup script for SRAT
# This script runs ONLY on Windows or WSL with HTTP_PROXY environment variable set.
# On other platforms or without HTTP_PROXY, it does nothing.

set -euo pipefail

# Check if running on Windows (native) or WSL
is_windows_or_wsl() {
	case "$(uname -s)" in
	CYGWIN* | MINGW* | MSYS* | Windows_NT)
		return 0
		;;
	Linux)
		# Check if running in WSL
		if grep -qi microsoft /proc/version 2>/dev/null || grep -qi microsoft /proc/sys/kernel/osrelease 2>/dev/null; then
			return 0
		fi
		return 1
		;;
	*)
		return 1
		;;
	esac
}

# Check if HTTP_PROXY or HTTPS_PROXY is set
has_proxy() {
	[[ -n "${HTTP_PROXY:-}" ]] || [[ -n "${HTTPS_PROXY:-}" ]] || [[ -n "${http_proxy:-}" ]] || [[ -n "${https_proxy:-}" ]]
}

main() {
	# Exit early if not on Windows or WSL
	if ! is_windows_or_wsl; then
		echo "Not running on Windows or WSL - skipping proxy setup"
		exit 0
	fi

	# Exit early if no proxy is configured
	if ! has_proxy; then
		echo "HTTP_PROXY/HTTPS_PROXY not set - skipping proxy setup"
		exit 0
	fi

	echo "Running on Windows/WSL with HTTP_PROXY configured"

	# Export proxy variables (normalize to uppercase)
	export HTTP_PROXY="${HTTP_PROXY:-${http_proxy:-}}"
	export HTTPS_PROXY="${HTTPS_PROXY:-${https_proxy:-}}"
	export NO_PROXY="${NO_PROXY:-${no_proxy:-}}"

	# Export for Go, Bun, and other tools
	export GO_PROXY="${HTTP_PROXY}"
	export GO_PROXY_AUTH="${GO_PROXY_AUTH:-}"
	export NPM_CONFIG_PROXY="${HTTP_PROXY}"
	export NPM_CONFIG_HTTPS_PROXY="${HTTPS_PROXY}"
	export BUN_PROXY="${HTTP_PROXY}"

	echo "Proxy configured:"
	echo "  HTTP_PROXY=${HTTP_PROXY}"
	echo "  HTTPS_PROXY=${HTTPS_PROXY}"
	[[ -n "${NO_PROXY}" ]] && echo "  NO_PROXY=${NO_PROXY}"

	# Export for subprocesses
	export HTTP_PROXY HTTPS_PROXY NO_PROXY
}

main "$@"
