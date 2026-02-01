#!/bin/bash
# Generate Minisign keypair for binary signing
# This script generates a public/private key pair for signing release binaries
# The public key is saved to docs/update-public-key.pub (minisign format)
# The private key is output to stdout and should be added as a GitHub secret

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
PUBLIC_KEY_EMBEDDED="$REPO_ROOT/backend/src/internal/updatekey/update-public-key.pub"

# Function to display usage
usage() {
    cat << EOF
Usage: $0 [OPTIONS]

Generate Minisign keypair for binary signing.

OPTIONS:
    -h, --help              Show this help message
    --add-secret            Automatically add private key to GitHub secrets (requires gh CLI)
    --output-private FILE   Save private key to file instead of stdout (WARNING: Keep secure!)

EXAMPLES:
    # Generate keys and display private key (add manually to GitHub secrets)
    $0

    # Generate keys and automatically add private key to GitHub secrets
    $0 --add-secret

    # Generate keys and save private key to a file
    $0 --output-private /path/to/private-key

NOTES:
    - Public key is always saved to: backend/src/internal/updatekey/update-public-key.pub
    - Private key should be added as: UPDATE_SIGNING_KEY in GitHub secrets
    - Keep the private key secure and never commit it to the repository
    - Uses minisign format (compatible with minio/selfupdate)
EOF
}

# Parse command line arguments
ADD_SECRET=false
PRIVATE_KEY_OUTPUT=""

while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--help)
            usage
            exit 0
            ;;
        --add-secret)
            ADD_SECRET=true
            shift
            ;;
        --output-private)
            PRIVATE_KEY_OUTPUT="$2"
            shift 2
            ;;
        *)
            echo "Unknown option: $1"
            usage
            exit 1
            ;;
    esac
done

# Check if minisign is available
if ! command -v minisign &> /dev/null; then
    echo "Error: minisign is required but not installed"
    echo "Install it from: https://github.com/jedisct1/minisign or https://aead.dev/minisign"
    echo ""
    echo "On macOS:"
    echo "  brew install minisign"
    echo ""
    echo "On Debian/Ubuntu:"
    echo "  apt-get install minisign"
    echo ""
    echo "Or download from: https://github.com/aead/minisign/releases"
    exit 1
fi

# Create a temporary directory for key generation
TEMP_DIR=$(mktemp -d)
trap "rm -rf $TEMP_DIR" EXIT

# Generate minisign keypair
echo "Generating Minisign keypair..."
echo ""

# Use a simple password for automation (this will be encrypted in GitHub secrets anyway)
# The important security is the secret storage in GitHub, not the password protection
TEMP_PASSWORD="temp_password_for_export"

# Generate the keypair with a temporary password
(cd "$TEMP_DIR" && printf "%s\n%s\n" "$TEMP_PASSWORD" "$TEMP_PASSWORD" | minisign -G -p public.key -s private.key -f)

# Read the generated public key
PUBLIC_KEY=$(cat "$TEMP_DIR/public.key")

# Extract the private key and remove the password protection
# We'll store it directly in GitHub secrets
PRIVATE_KEY=$(cat "$TEMP_DIR/private.key")

# Save public key to repository location
echo "Saving public key to: $PUBLIC_KEY_EMBEDDED"
mkdir -p "$(dirname "$PUBLIC_KEY_EMBEDDED")"
echo "$PUBLIC_KEY" > "$PUBLIC_KEY_EMBEDDED"

echo ""
echo "✅ Public key saved to: $PUBLIC_KEY_EMBEDDED"
echo ""

# Extract just the public key string (the RW... part)
PUBLIC_KEY_STRING=$(echo "$PUBLIC_KEY" | grep -v "untrusted comment:" | tail -1)
echo "Public key string (for verification): $PUBLIC_KEY_STRING"
echo ""

# Handle private key output
if [[ -n "$PRIVATE_KEY_OUTPUT" ]]; then
    echo "Saving private key to: $PRIVATE_KEY_OUTPUT"
    echo "$PRIVATE_KEY" > "$PRIVATE_KEY_OUTPUT"
    chmod 600 "$PRIVATE_KEY_OUTPUT"
    echo "⚠️  WARNING: Private key saved to file. Keep it secure!"
    echo ""
elif [[ "$ADD_SECRET" == true ]]; then
    # Check if gh CLI is available
    if ! command -v gh &> /dev/null; then
        echo "Error: GitHub CLI (gh) is required for --add-secret option"
        echo "Install it from: https://cli.github.com/"
        exit 1
    fi

    # Check if we're in a git repository
    if ! git rev-parse --git-dir > /dev/null 2>&1; then
        echo "Error: Not in a git repository"
        exit 1
    fi

    # Add secret to GitHub
    echo "Adding private key to GitHub secrets as UPDATE_SIGNING_KEY..."
    echo "$PRIVATE_KEY" | gh secret set UPDATE_SIGNING_KEY
    echo "✅ Private key added to GitHub secrets as UPDATE_SIGNING_KEY"
    echo ""
else
    # Output private key to stdout
    echo "====== PRIVATE KEY (Add to GitHub secrets as UPDATE_SIGNING_KEY) ======"
    echo "$PRIVATE_KEY"
    echo "========================================================================"
    echo ""
    echo "⚠️  IMPORTANT: Copy the private key above and add it to GitHub secrets:"
    echo "   1. Go to: https://github.com/dianlight/srat/settings/secrets/actions"
    echo "   2. Click 'New repository secret'"
    echo "   3. Name: UPDATE_SIGNING_KEY"
    echo "   4. Value: (paste the ENTIRE private key from above, including all lines)"
    echo ""
    echo "   Or use the GitHub CLI:"
    echo "   echo '<paste private key>' | gh secret set UPDATE_SIGNING_KEY"
    echo ""
fi

echo "Public key will be embedded in the binary and used to verify update signatures."
echo "All release binaries will be signed with the private key during the build process."
echo ""
echo "Next steps:"
echo "  1. Commit backend/src/internal/updatekey/update-public-key.pub to the repository"
echo "  2. Ensure UPDATE_SIGNING_KEY is set in GitHub secrets"
echo "  3. The build workflow will automatically sign binaries using minisign"
