#!/bin/bash
# Generate Ed25519 keypair for binary signing
# This script generates a public/private key pair for signing release binaries
# The public key is saved to docs/update-public-key.pem
# The private key is output to stdout and should be added as a GitHub secret

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
PUBLIC_KEY_FILE="$REPO_ROOT/docs/update-public-key.pem"

# Function to display usage
usage() {
    cat << EOF
Usage: $0 [OPTIONS]

Generate Ed25519 keypair for binary signing.

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
    $0 --output-private /path/to/private-key.pem

NOTES:
    - Public key is always saved to: docs/update-public-key.pem
    - Private key should be added as: UPDATE_SIGNING_KEY in GitHub secrets
    - Keep the private key secure and never commit it to the repository
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

# Check if openssl is available
if ! command -v openssl &> /dev/null; then
    echo "Error: openssl is required but not installed"
    exit 1
fi

# Generate Ed25519 private key
echo "Generating Ed25519 private key..."
PRIVATE_KEY=$(openssl genpkey -algorithm ED25519 2>/dev/null)

# Extract public key from private key
echo "Extracting public key..."
PUBLIC_KEY=$(echo "$PRIVATE_KEY" | openssl pkey -pubout 2>/dev/null)

# Save public key to file
echo "Saving public key to: $PUBLIC_KEY_FILE"
mkdir -p "$(dirname "$PUBLIC_KEY_FILE")"
echo "$PUBLIC_KEY" > "$PUBLIC_KEY_FILE"

echo ""
echo "✅ Public key saved to: $PUBLIC_KEY_FILE"
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
    echo "   4. Value: (paste the private key from above)"
    echo ""
    echo "   Or use the GitHub CLI:"
    echo "   gh secret set UPDATE_SIGNING_KEY"
    echo ""
fi

echo "Public key will be embedded in the binary and used to verify update signatures."
echo "All release binaries will be signed with the private key during the build process."
echo ""
echo "Next steps:"
echo "  1. Commit docs/update-public-key.pem to the repository"
echo "  2. Ensure UPDATE_SIGNING_KEY is set in GitHub secrets"
echo "  3. The build workflow will automatically sign binaries"
