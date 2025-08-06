#!/usr/bin/env bash
#set -x

# Define the .env file path relative to the workspace root
ENV_FILE="${PWD}/.devcontainer/.env"

# Function to prompt for a variable with a timeout and default, if applicable
prompt_for_var() {
    VAR_NAME=$1
    PROMPT_MSG=$2
    DEFAULT_VALUE=$3 # Can be empty for no default
    TIMEOUT=$4       # Timeout in seconds

    INPUT_VALUE="" # Initialize to empty

    # Check if the variable is already set in the target env file
    EXISTING_VALUE=$(grep "^${VAR_NAME}=" "${ENV_FILE}" 2>/dev/null | cut -d'=' -f2- | sed 's/\\"//g')

    # Determine initial value from environment, then existing file
    if [ -n "${!VAR_NAME}" ]; then
        INPUT_VALUE="${!VAR_NAME}"
        echo "${VAR_NAME} is already set in current shell: ${INPUT_VALUE}"
    elif [ -n "$EXISTING_VALUE" ]; then
        INPUT_VALUE="${EXISTING_VALUE}"
        echo "${VAR_NAME} is already set in ${ENV_FILE}: ${INPUT_VALUE}"
    fi

    # If the variable is still empty (not set in env or file), prompt the user
    if [ -z "$INPUT_VALUE" ]; then
        PROMPT_TEXT="$PROMPT_MSG"
        if [ -n "$DEFAULT_VALUE" ]; then
            PROMPT_TEXT="$PROMPT_TEXT [$DEFAULT_VALUE]"
        fi
        PROMPT_TEXT="$PROMPT_TEXT (Timeout ${TIMEOUT}s): "

        read -t "$TIMEOUT" -p "$PROMPT_TEXT" USER_INPUT_TEMP

        READ_STATUS=$? # Capture the exit status of the read command

        if [ $READ_STATUS -eq 0 ]; then # User provided input within timeout
            if [ -n "$USER_INPUT_TEMP" ]; then
                INPUT_VALUE="${USER_INPUT_TEMP}"
            elif [ -n "$DEFAULT_VALUE" ]; then
                # User pressed Enter, use default if provided
                INPUT_VALUE="${DEFAULT_VALUE}"
            else
                # User pressed Enter, no default, leave empty
                INPUT_VALUE=""
            fi
        elif [ $READ_STATUS -gt 128 ]; then # Timeout or other error occurred
            if [ $READ_STATUS -eq 142 ]; then
                echo "Timeout for ${VAR_NAME}. Using default or skipping."
            else
                echo "Error reading input for ${VAR_NAME} (status: $READ_STATUS). Using default or skipping."
            fi
        fi
    fi

    # Write to the .env file, only if a value is determined
    if [ -n "$INPUT_VALUE" ]; then
        echo "Exporting ${VAR_NAME}=${INPUT_VALUE}" # Debug print
        # To avoid issues with special characters in sed, we'll remove the old line and append the new one.
        if grep -q "^${VAR_NAME}=" "${ENV_FILE}" 2>/dev/null; then
            grep -v "^${VAR_NAME}=" "${ENV_FILE}" > "${ENV_FILE}.tmp"
            mv "${ENV_FILE}.tmp" "${ENV_FILE}"
        fi
        echo "${VAR_NAME}=${INPUT_VALUE}" >> "${ENV_FILE}"
    else
        echo "Variable ${VAR_NAME} is empty and will not be added to ${ENV_FILE}."
        # If the variable was previously in the .env file but is now skipped/empty,
        # you might want to remove it from the .env file.
        # This can be done by uncommenting the next line:
        # sed -i "/^${VAR_NAME}=/d" "${ENV_FILE}" 2>/dev/null
    fi
}

# --- Main script execution ---

# Prompt for Home Assistant IP (default: 192.168.1.100, timeout: 10s)
prompt_for_var HOMEASSISTANT_IP "Enter the IP address of your Home Assistant instance" "192.168.0.68" 10

# Prompt for Home Assistant SSH User (default: homeassistant, timeout: 10s)
prompt_for_var HOMEASSISTANT_SSH_USER "Enter the SSH username for your Home Assistant instance" "root" 10

# Prompt for SUPERVISOR_TOKEN (no default, timeout: 30s)
prompt_for_var SUPERVISOR_TOKEN "Enter your Home Assistant Supervisor Token" "" 60

# Prompt for ROLLBAR_CLIENT_ACCESS_TOKEN (no default, timeout: 30s)
prompt_for_var ROLLBAR_CLIENT_ACCESS_TOKEN "Enter your Rollbar Client Access Token" "" 60

echo "Environment variables processed. Check ${ENV_FILE}"