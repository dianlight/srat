#!/usr/bin/env bash
#set -x

# Define the .env file path relative to the workspace root
ENV_FILE_PATH="${CONTAINER_WORKSPACE_FOLDER}/.devcontainer/.env"

echo "Dev Container Post-Start Setup:"

# Check if the .env file exists and has content (meaning init-env.sh has run)
if [ -s "${ENV_FILE_PATH}" ]; then
    echo "Environment variables are configured in ${ENV_FILE_PATH}."
    echo "If this is your *first time* setting up, please 'Rebuild Container' or 'Reopen in Container' for variables to take effect in new terminals."
    echo "You can do this by opening the Command Palette (F1 or Ctrl+Shift+P) and searching for 'Dev Containers: Rebuild Container' or 'Dev Containers: Reopen in Container'."
    echo ""
    echo "Current values (from .env file and possibly loaded by Docker):"
    grep -E '^(HOMEASSISTANT_IP|HOMEASSISTANT_SSH_USER|SUPERVISOR_TOKEN)=' "${ENV_FILE_PATH}" 2>/dev/null || echo "No values found in .env file yet."
else
    echo "The .devcontainer/.env file is empty or does not exist."
    echo "Please ensure you respond to the prompts during 'Rebuild Container' to set your Home Assistant variables."
fi

echo "Post-start setup script finished."

# Optional: Add a small delay for the user to see the message if needed
# sleep 5
