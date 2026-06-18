#!/bin/bash
# Pre-populate PKL's local cache with the hk package referenced in hk.pkl
# so that `hk install --mise` works on corporate networks where PKL cannot
# reach github.com directly (MITM proxy with a CA not in PKL's bundled store).
#
# On Windows / WSL (where powershell.exe is reachable), it also exports the
# Windows trusted root CAs to a PEM bundle and passes it to pkl download-package.
#
# Idempotent: a sentinel file per package version prevents re-downloading.
# Run with --force to clear the sentinel and re-download.
# Honors HTTP_PROXY / HTTPS_PROXY from the calling environment; if unset,
# auto-detects the Windows proxy from the registry.

set -e

# --- configuration -----------------------------------------------------------

HK_PKL_FILE="${HK_PKL_FILE:-hk.pkl}"
CA_BUNDLE_MAX_AGE_DAYS="${CA_BUNDLE_MAX_AGE_DAYS:-30}"

# --- colors / print_status ---------------------------------------------------

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

print_status() {
	local status=$1
	local message=$2
	case $status in
	"success") echo -e "${GREEN}✅ $message${NC}" ;;
	"warning") echo -e "${YELLOW}⚠️  $message${NC}" ;;
	"error") echo -e "${RED}❌ $message${NC}" ;;
	"info") echo -e "ℹ️  $message" ;;
	esac
}

# --- helpers -----------------------------------------------------------------

# Returns the directory that holds PKL's cache and the CA bundle.
# pkl is installed by mise as a Linux ELF binary; it always uses $HOME/.pkl
# on the Linux filesystem regardless of $USERPROFILE.
resolve_pkl_home() {
	echo "${HOME}/.pkl"
}

# True when running on a Windows host (native or WSL with powershell.exe on PATH).
is_windows_host() {
	command -v powershell.exe >/dev/null 2>&1
}

# Parse the hk PKL package version from the `amends` line in hk.pkl.
# Returns e.g. "1.43.0".
parse_pkl_version() {
	grep -m1 -oE 'hk@[0-9]+\.[0-9]+\.[0-9]+' "$HK_PKL_FILE" 2>/dev/null |
		head -1 |
		sed 's/^hk@//'
}

# Read the Windows proxy server from the registry and echo "host:port", or
# echo nothing if not configured / not detectable.
# Writes a temporary .ps1 file to avoid bash → PowerShell escaping issues.
resolve_windows_proxy() {
	if ! is_windows_host; then
		return 0
	fi
	local tmp_ps
	tmp_ps="$(mktemp /tmp/srat-proxy-XXXXXX.ps1)"
	cat >"$tmp_ps" <<'PSEOF'
[Console]::OutputEncoding = [System.Text.Encoding]::ASCII
$s = Get-ItemProperty 'HKCU:\Software\Microsoft\Windows\CurrentVersion\Internet Settings' -ErrorAction SilentlyContinue
if ($s.ProxyEnable -eq 1 -and $s.ProxyServer) {
    $raw = $s.ProxyServer
    if ($raw -match 'https=([^;]+)') { Write-Output $Matches[1] }
    elseif ($raw -match 'http=([^;]+)') { Write-Output $Matches[1] }
    else { Write-Output $raw }
}
PSEOF
	# Convert the Linux temp path to a Windows path for PowerShell.
	local win_tmp
	win_tmp="$(wslpath -w "$tmp_ps" 2>/dev/null || echo "$tmp_ps")"
	local result
	result="$(powershell.exe -NoProfile -ExecutionPolicy Bypass -File "$win_tmp" 2>/dev/null | tr -d '\r\n')"
	rm -f "$tmp_ps"
	echo "$result"
}

# --- steps -------------------------------------------------------------------

# Export Windows trusted root CAs to a PEM bundle understood by the Linux pkl binary.
# Writes the bundle to $HOME/.pkl/windows-ca-bundle.pem (Linux filesystem) by
# piping powershell.exe stdout — never using Set-Content, which would write to the
# Windows FS and be invisible to the Linux pkl process.
# Skips if the bundle is fresh (< CA_BUNDLE_MAX_AGE_DAYS days old).
ensure_ca_bundle() {
	if ! is_windows_host; then
		print_status "info" "Not a Windows host — skipping CA bundle export"
		return 0
	fi

	local pkl_home bundle
	pkl_home="$(resolve_pkl_home)"
	bundle="${pkl_home}/windows-ca-bundle.pem"

	if [ -f "$bundle" ]; then
		if find "$bundle" -mtime -"${CA_BUNDLE_MAX_AGE_DAYS}" -print -quit 2>/dev/null | grep -q .; then
			print_status "info" "CA bundle is fresh (< ${CA_BUNDLE_MAX_AGE_DAYS} days) — skipping export"
			return 0
		fi
		print_status "info" "CA bundle is stale — re-exporting..."
	else
		print_status "info" "Exporting Windows trusted root CAs to PEM bundle..."
	fi

	mkdir -p "$pkl_home"

	# Write the PowerShell script to a temp file to avoid bash escaping issues.
	local tmp_ps
	tmp_ps="$(mktemp /tmp/srat-ca-XXXXXX.ps1)"
	cat >"$tmp_ps" <<'PSEOF'
[Console]::OutputEncoding = [System.Text.Encoding]::ASCII
Get-ChildItem Cert:\LocalMachine\Root | ForEach-Object {
    $b64 = [System.Convert]::ToBase64String($_.RawData, 'InsertLineBreaks')
    Write-Output '-----BEGIN CERTIFICATE-----'
    Write-Output $b64
    Write-Output '-----END CERTIFICATE-----'
}
PSEOF

	local win_tmp tmp_bundle
	win_tmp="$(wslpath -w "$tmp_ps" 2>/dev/null || echo "$tmp_ps")"
	tmp_bundle="${bundle}.tmp"

	powershell.exe -NoProfile -ExecutionPolicy Bypass -File "$win_tmp" 2>/dev/null |
		tr -d '\r' >"$tmp_bundle"
	rm -f "$tmp_ps"

	if [ -s "$tmp_bundle" ]; then
		mv "$tmp_bundle" "$bundle"
		local cert_count
		cert_count=$(grep -c 'BEGIN CERTIFICATE' "$bundle" 2>/dev/null || echo "?")
		print_status "success" "CA bundle written to $bundle ($cert_count certificates)"
	else
		rm -f "$tmp_bundle"
		print_status "warning" "CA bundle export produced empty output (continuing without it)"
	fi
}

# Download the hk PKL package into PKL's local cache using curl (not pkl itself).
#
# Why curl and not `pkl download-package`?
# pkl download-package honours HTTP_PROXY/HTTPS_PROXY but its GraalVM SSL layer
# does NOT trust the corporate CA, so the TLS handshake to github.com fails even
# when the proxy forwards the CONNECT tunnel correctly. curl uses the system
# OpenSSL and accepts --cacert, so it always works through the corporate proxy.
#
# PKL cache layout (from PackageResolvers.java):
#   ~/.pkl/cache/package-2/<authority>/<path>/<name>.json  ← metadata
#   ~/.pkl/cache/package-2/<authority>/<path>/<name>.zip   ← package zip
# For package://github.com/jdx/hk/releases/download/v1.43.0/hk@1.43.0 :
#   authority = github.com
#   path      = jdx/hk/releases/download/v1.43.0/hk@1.43.0
#   name      = hk@1.43.0
ensure_pkl_cache() {
	local version="$1"
	local pkl_home
	pkl_home="$(resolve_pkl_home)"

	local pkg_name="hk@${version}"
	local pkg_dir="${pkl_home}/cache/package-2/github.com/jdx/hk/releases/download/v${version}/${pkg_name}"
	local sentinel="${pkl_home}/cache/.srat-hk-${version}.ok"

	if [ -f "$sentinel" ]; then
		print_status "info" "PKL package ${pkg_name} already cached (sentinel found)"
		return 0
	fi

	if ! command -v curl >/dev/null 2>&1; then
		print_status "error" "curl is required but not found on PATH"
		return 1
	fi

	# Build curl proxy and CA cert args.
	local curl_args=(--silent --location --max-time "${PKL_DOWNLOAD_TIMEOUT:-120}" --fail)

	local bundle="${pkl_home}/windows-ca-bundle.pem"
	if [ -f "$bundle" ]; then
		curl_args+=(--cacert "$bundle")
		print_status "info" "Using CA bundle: $bundle"
	fi

	# Use HTTPS_PROXY/HTTP_PROXY from env if set; otherwise auto-detect from Windows registry.
	local proxy_url="${HTTPS_PROXY:-${https_proxy:-}}"
	if [ -z "$proxy_url" ]; then
		local win_proxy
		win_proxy="$(resolve_windows_proxy)"
		if [ -n "$win_proxy" ]; then
			proxy_url="http://${win_proxy}"
			print_status "info" "Detected Windows proxy: $win_proxy"
		fi
	fi
	if [ -n "$proxy_url" ]; then
		curl_args+=(--proxy "$proxy_url")
	fi

	local base_url="https://github.com/jdx/hk/releases/download/v${version}"
	mkdir -p "$pkg_dir"

	# 1. Metadata JSON (bare package URL — no fragment).
	print_status "info" "Downloading PKL package metadata for ${pkg_name}..."
	local meta_file="${pkg_dir}/${pkg_name}.json"
	if ! curl "${curl_args[@]}" "${base_url}/${pkg_name}" -o "${meta_file}.tmp"; then
		print_status "error" "Failed to download package metadata"
		rm -f "${meta_file}.tmp"
		return 1
	fi
	mv "${meta_file}.tmp" "$meta_file"

	# 2. Package zip (URL taken from the metadata to stay in sync with PKL's own logic).
	local zip_url
	zip_url="$(python3 -c "import sys,json; print(json.load(open('${meta_file}'))['packageZipUrl'])" 2>/dev/null ||
		echo "${base_url}/${pkg_name}.zip")"
	print_status "info" "Downloading PKL package zip..."
	local zip_file="${pkg_dir}/${pkg_name}.zip"
	if ! curl "${curl_args[@]}" "$zip_url" -o "${zip_file}.tmp"; then
		print_status "error" "Failed to download package zip"
		rm -f "${zip_file}.tmp"
		return 1
	fi
	mv "${zip_file}.tmp" "$zip_file"

	touch "$sentinel"
	print_status "success" "PKL package ${pkg_name} cached at ${pkg_dir}"
}

# Create or update .mise.local.toml so that `mise install` works behind a
# corporate proxy where bun breaks against Nexus registries.
#
# Why bun breaks:
#   bun constructs NPM URLs by appending the package name directly to the last
#   path segment of the registry URL, stripping intermediate path segments.
#   e.g. https://nexus.host/repository/npm-all → bun sends requests to
#        https://nexus.host/repository/<package>         (400 — wrong path)
#   instead of:
#        https://nexus.host/repository/npm-all/<package> (correct)
#
# Fix: switch mise npm tool installs from bun → system npm, point it at the
# official registry (corporate Nexus often lacks preview/nightly packages),
# and supply NODE_EXTRA_CA_CERTS so Node trusts the corporate CA.
#
# The file is gitignored (.mise.local.toml) and idempotent: if the required
# keys are already present, the file is left unchanged.
ensure_mise_local_toml() {
	local toml=".mise.local.toml"

	# Keys we need to be present.
	local need_pkg_mgr need_bun need_ca_certs need_registry need_node
	need_pkg_mgr=1
	need_bun=1
	need_ca_certs=1
	need_registry=1
	need_node=1

	if [ -f "$toml" ]; then
		grep -q 'npm\.package_manager' "$toml" 2>/dev/null && need_pkg_mgr=0
		grep -q 'npm\.bun' "$toml" 2>/dev/null && need_bun=0
		grep -q 'NODE_EXTRA_CA_CERTS' "$toml" 2>/dev/null && need_ca_certs=0
		grep -q 'npm_config_registry' "$toml" 2>/dev/null && need_registry=0
		grep -q '^node ' "$toml" 2>/dev/null && need_node=0

		if [ $need_pkg_mgr -eq 0 ] && [ $need_bun -eq 0 ] &&
			[ $need_ca_certs -eq 0 ] && [ $need_registry -eq 0 ] && [ $need_node -eq 0 ]; then
			print_status "info" ".mise.local.toml already configured — skipping"
			return 0
		fi
		print_status "info" "Updating existing .mise.local.toml with missing keys..."
	else
		print_status "info" "Creating .mise.local.toml for corporate proxy compatibility..."
	fi

	# Append only the missing sections to avoid clobbering user customisations.
	# shellcheck disable=SC2094
	{
		if [ ! -f "$toml" ]; then
			echo "# Machine-local mise overrides — NOT committed to git."
			echo ""
		fi

		if [ $need_pkg_mgr -eq 1 ] || [ $need_bun -eq 1 ]; then
			echo "[settings]"
			[ $need_pkg_mgr -eq 1 ] && echo 'npm.package_manager = "npm"'
			[ $need_bun -eq 1 ] && echo "npm.bun = false"
			echo ""
		fi

		if [ $need_ca_certs -eq 1 ] || [ $need_registry -eq 1 ]; then
			echo "[env]"
			# Use $HOME so the path stays valid for any user without hardcoding.
			[ $need_ca_certs -eq 1 ] &&
				echo "NODE_EXTRA_CA_CERTS = \"\$HOME/.pkl/windows-ca-bundle.pem\""
			[ $need_registry -eq 1 ] &&
				echo 'npm_config_registry = "https://registry.npmjs.org"'
			echo ""
		fi

		if [ $need_node -eq 1 ]; then
			# Install a Linux Node.js via mise so that `npm` resolves to a Linux
			# binary, not the Windows npm on the WSL PATH. Windows npm cannot
			# install packages to Linux paths (stack overflow on long paths).
			echo "[tools]"
			echo 'node = "22"'
			echo ""
		fi
	} >>"$toml"

	print_status "success" ".mise.local.toml written — bun/Nexus issues resolved"
	print_status "info" "  npm.package_manager = npm          (bun bypassed)"
	print_status "info" "  NODE_EXTRA_CA_CERTS  = \$HOME/.pkl/windows-ca-bundle.pem"
	print_status "info" "  npm_config_registry  = https://registry.npmjs.org"
	print_status "info" "  node = 22                          (Linux npm, shadows Windows npm)"
}

# --- main --------------------------------------------------------------------

main() {
	echo "🔑 SRAT PKL proxy / CA cache setup"
	echo "==================================="
	echo ""

	local version
	version="$(parse_pkl_version)" || true
	if [ -z "$version" ]; then
		print_status "error" "Could not parse hk PKL version from ${HK_PKL_FILE}"
		print_status "info" "Expected a line like: amends \"package://github.com/jdx/hk/releases/download/v1.43.0/hk@1.43.0#/Config.pkl\""
		exit 2
	fi
	print_status "info" "Detected hk PKL package version: ${version}"
	echo ""

	ensure_ca_bundle || true

	echo ""
	ensure_pkl_cache "$version"

	echo ""
	ensure_mise_local_toml

	echo ""
	print_status "success" "Setup complete — \`mise install\` should now work"
}

# --- argument parsing --------------------------------------------------------

case "${1:-}" in
"--help" | "-h")
	echo "PKL proxy / CA cache setup script for SRAT project"
	echo ""
	echo "Usage: $0 [option]"
	echo ""
	echo "Options:"
	echo "  --help, -h       Show this help message"
	echo "  --version-only   Print the hk PKL package version parsed from hk.pkl and exit"
	echo "  --force          Clear the cached-download sentinel and re-run"
	echo ""
	echo "Environment variables:"
	echo "  HK_PKL_FILE              Path to hk.pkl (default: hk.pkl)"
	echo "  CA_BUNDLE_MAX_AGE_DAYS   Days before re-exporting the Windows CA bundle (default: 30)"
	echo "  HTTP_PROXY / HTTPS_PROXY If unset, auto-detected from the Windows registry"
	echo "  PKL_DOWNLOAD_TIMEOUT     curl --max-time in seconds for each download (default: 120)"
	echo ""
	echo "What it does:"
	echo "  1. Parses the hk PKL package version from hk.pkl."
	echo "  2. On Windows/WSL: exports trusted root CAs to ~/.pkl/windows-ca-bundle.pem."
	echo "  3. Downloads the PKL package metadata+zip via curl (bypasses PKL's GraalVM SSL"
	echo "     layer) into ~/.pkl/cache/ so \`hk install --mise\` never needs the network."
	echo "  4. Creates/updates .mise.local.toml (gitignored) to:"
	echo "       - switch mise npm tool installs from bun to system npm"
	echo "       - add node=22 so mise installs a Linux Node (shadows Windows npm)"
	echo "       - point npm at registry.npmjs.org (bypasses broken Nexus)"
	echo "       - set NODE_EXTRA_CA_CERTS for corporate CA trust"
	echo ""
	echo "Idempotent: a sentinel prevents re-downloading; .mise.local.toml is only"
	echo "appended when keys are missing. Run with --force to re-download the PKL package."
	exit 0
	;;
"--version-only")
	version="$(parse_pkl_version)"
	if [ -z "$version" ]; then
		echo "ERROR: could not parse hk PKL version from ${HK_PKL_FILE}" >&2
		exit 2
	fi
	echo "$version"
	exit 0
	;;
"--force")
	pkl_home="$(resolve_pkl_home)"
	sentinels=("${pkl_home}/cache/.srat-hk-"*.ok)
	if [ "${sentinels[0]}" != "${pkl_home}/cache/.srat-hk-*.ok" ]; then
		print_status "info" "Clearing ${#sentinels[@]} sentinel(s)..."
		rm -f "${sentinels[@]}"
	fi
	main
	;;
"")
	main
	;;
*)
	print_status "error" "Unknown option: $1"
	echo "Run '$0 --help' for usage."
	exit 1
	;;
esac
