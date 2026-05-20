#!/usr/bin/env bash

ENV_FILE="${PWD}/.devcontainer/.env"

prompt_for_var() {
	VAR_NAME=$1
	PROMPT_MSG=$2
	DEFAULT_VALUE=$3
	TIMEOUT=$4

	INPUT_VALUE=""

	EXISTING_VALUE=$(grep "^${VAR_NAME}=" "${ENV_FILE}" 2>/dev/null | cut -d'=' -f2- | sed 's/\\"//g')

	if [ -n "${!VAR_NAME}" ]; then
		INPUT_VALUE="${!VAR_NAME}"
		echo "${VAR_NAME} is already set in current shell: ${INPUT_VALUE}"
	elif [ -n "$EXISTING_VALUE" ]; then
		INPUT_VALUE="${EXISTING_VALUE}"
		echo "${VAR_NAME} is already set in ${ENV_FILE}: ${INPUT_VALUE}"
	fi

	if [ -z "$INPUT_VALUE" ]; then
		PROMPT_TEXT="$PROMPT_MSG"
		if [ -n "$DEFAULT_VALUE" ]; then
			PROMPT_TEXT="$PROMPT_TEXT [$DEFAULT_VALUE]"
		fi
		PROMPT_TEXT="$PROMPT_TEXT (Timeout ${TIMEOUT}s): "

		read -r -t "$TIMEOUT" -p "$PROMPT_TEXT" USER_INPUT_TEMP
		READ_STATUS=$?

		if [ $READ_STATUS -eq 0 ]; then
			if [ -n "$USER_INPUT_TEMP" ]; then
				INPUT_VALUE="${USER_INPUT_TEMP}"
			elif [ -n "$DEFAULT_VALUE" ]; then
				INPUT_VALUE="${DEFAULT_VALUE}"
			fi
		elif [ $READ_STATUS -gt 128 ]; then
			echo "Timeout for ${VAR_NAME}. Using default or skipping."
			if [ -n "$DEFAULT_VALUE" ]; then
				INPUT_VALUE="${DEFAULT_VALUE}"
			fi
		fi
	fi

	if [ -n "$INPUT_VALUE" ]; then
		if grep -q "^${VAR_NAME}=" "${ENV_FILE}" 2>/dev/null; then
			grep -v "^${VAR_NAME}=" "${ENV_FILE}" >"${ENV_FILE}.tmp"
			mv "${ENV_FILE}.tmp" "${ENV_FILE}"
		fi
		echo "${VAR_NAME}=${INPUT_VALUE}" >>"${ENV_FILE}"
	fi
}

# Prompt for environment variables
prompt_for_var HOMEASSISTANT_IP "Enter the IP address of your Home Assistant instance" "192.168.0.68" 10
prompt_for_var HOMEASSISTANT_SSH_USER "Enter the SSH username for your Home Assistant instance" "root" 10
prompt_for_var SUPERVISOR_TOKEN "Enter your Home Assistant Supervisor Token" "" 60
prompt_for_var SENTRY_DSN "Enter your Sentry DSN (backend)" "" 60
prompt_for_var VITE_SENTRY_DSN "Enter your Sentry DSN (frontend)" "" 60
prompt_for_var GIST_TOKEN "Enter your GitHub Gist Token (with 'gist' scope)" "" 60

echo "Environment variables processed. Check ${ENV_FILE}"

# Trust mise configuration
WORKSPACE_DIR="${WORKSPACE_DIR:-${PWD}}"
MISE_FILE="$WORKSPACE_DIR/.mise.toml"

if [[ -f "$MISE_FILE" ]]; then
	mise trust "$MISE_FILE"
else
	echo "WARN: $MISE_FILE not found. Skipping mise trust."
fi

# Ensure zsh is default shell
if command -v zsh >/dev/null 2>&1; then
	sudo chsh -s "$(command -v zsh)" "$(whoami)" || true
fi
