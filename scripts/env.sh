#!/bin/bash
# Environment setup for SRAT build and test scripts
# Sets common environment variables and functions

#  API_URL
if [ -n "${SUPERVISOR_URL}" ]; then
	export API_URL="${SUPERVISOR_URL%/}:3000/"
else
	echo "WARN: Defaulting API_URL to http://localhost:3000/ - ensure this is correct for your environment"
	export API_URL="http://localhost:3000/"
fi

# VERSION
if [ -n "${VERSION}" ]; then
	export VERSION="${VERSION}"
else
	VERSION=$(./scripts/next_version.sh)
	export VERSION="${VERSION}"
	echo "WARN: VERSION not set - defaulting to ${VERSION}"
fi

# ARCH
if [ -n "${ARCH}" ]; then
	export ARCH="${ARCH}"
else
	echo "WARN: ARCH not set - defaulting to host architecture"
	if [[ "$(uname -m)" == "x86_64" ]]; then
		export ARCH="x86_64"
	elif [[ "$(uname -m)" == "aarch64" || "$(uname -m)" == "arm64" ]]; then
		export ARCH="aarch64"
	else
		echo "ERROR: Unsupported architecture $(uname -m). Please set ARCH manually." >&2
		exit 1
	fi
fi
