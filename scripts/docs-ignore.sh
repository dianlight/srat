#!/usr/bin/env bash

set -euo pipefail

DOCS_IGNORE_FILE="${DOCS_IGNORE_FILE:-.docsignore}"

trim() {
	local value="$1"
	value="${value#"${value%%[![:space:]]*}"}"
	value="${value%"${value##*[![:space:]]}"}"
	printf '%s' "$value"
}

should_include_pattern() {
	local tool="$1"
	local metadata="$2"

	if [[ -z "$metadata" ]]; then
		return 0
	fi

	if [[ "$metadata" =~ tools: ]]; then
		local scoped
		scoped="$(trim "${metadata#*tools:}")"
		IFS=',' read -ra tools <<<"$scoped"
		for token in "${tools[@]}"; do
			token="$(trim "$token")"
			if [[ "$token" == "$tool" ]]; then
				return 0
			fi
		done
		return 1
	fi

	return 0
}

load_patterns() {
	local tool="$1"

	if [[ ! -f "$DOCS_IGNORE_FILE" ]]; then
		echo "docs-ignore: file not found: $DOCS_IGNORE_FILE" >&2
		exit 1
	fi

	local raw line pattern metadata
	while IFS= read -r raw || [[ -n "$raw" ]]; do
		line="${raw%$'\r'}"
		line="$(trim "$line")"

		[[ -z "$line" ]] && continue
		[[ "$line" == \#* ]] && continue

		pattern="$line"
		metadata=""
		if [[ "$line" == *'#'* ]]; then
			pattern="$(trim "${line%%#*}")"
			metadata="$(trim "${line#*#}")"
		fi

		[[ -z "$pattern" ]] && continue

		if should_include_pattern "$tool" "$metadata"; then
			printf '%s\n' "$pattern"
		fi
	done <"$DOCS_IGNORE_FILE"
}

list_md_files() {
	local tool="$1"
	local find_args=(.)
	local pattern normalized

	while IFS= read -r pattern; do
		normalized="${pattern#./}"
		normalized="${normalized%/}"

		[[ -z "$normalized" ]] && continue

		if [[ "$pattern" == */ ]]; then
			find_args+=(-not -path "./${normalized}" -not -path "./${normalized}/*")
		else
			find_args+=(-not -path "./${normalized}")
		fi
	done < <(load_patterns "$tool")

	find "${find_args[@]}" -type f -name "*.md" -print | sort
}

print_usage() {
	cat <<'EOF'
Usage:
  scripts/docs-ignore.sh patterns [tool]
  scripts/docs-ignore.sh markdownlint-args
  scripts/docs-ignore.sh list-md-files [tool]

Tools:
  markdownlint | vale | hk | toc

Defaults:
  tool defaults to "markdownlint" where omitted.
EOF
}

command="${1:-}"
case "$command" in
patterns)
	tool="${2:-markdownlint}"
	load_patterns "$tool"
	;;
markdownlint-args)
	while IFS= read -r pattern; do
		printf '#%s\n' "$pattern"
	done < <(load_patterns "markdownlint")
	;;
list-md-files)
	tool="${2:-markdownlint}"
	list_md_files "$tool"
	;;
"" | --help | -h)
	print_usage
	;;
*)
	echo "docs-ignore: unknown command '$command'" >&2
	print_usage >&2
	exit 1
	;;
esac
