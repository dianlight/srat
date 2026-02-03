#!/bin/sh
# Extract version information from SRAT binaries
# Tries strings method first, falls back to readelf ELF section method
#
# Usage: extract-binary-version.sh <binary_path>
# Returns: Version string on success, exits with error code on failure

set -e

BINARY_PATH="${1}"

if [ -z "${BINARY_PATH}" ]; then
    echo "ERROR: No binary path provided" >&2
    echo "Usage: $0 <binary_path>" >&2
    exit 1
fi

if [ ! -f "${BINARY_PATH}" ]; then
    echo "ERROR: Binary not found: ${BINARY_PATH}" >&2
    exit 1
fi

if [ ! -r "${BINARY_PATH}" ]; then
    echo "ERROR: Binary not readable: ${BINARY_PATH}" >&2
    exit 1
fi

# Method 1: Try strings method (works for both CGO and no-CGO builds)
try_strings_method() {
    if ! command -v strings >/dev/null 2>&1; then
        return 1
    fi
    
    if ! command -v jq >/dev/null 2>&1; then
        return 1
    fi
    
    META_LINE=$(strings -a "${BINARY_PATH}" | grep -oE 'SRAT_METADATA:\{[^}]*\}' | head -1)
    
    if [ -n "${META_LINE}" ]; then
        EXTRACTED_JSON=$(echo "${META_LINE}" | grep -oE '\{[^}]+\}')
        EXTRACTED_VERSION=$(echo "${EXTRACTED_JSON}" | jq -r '.version' 2>/dev/null)
        
        if [ -n "${EXTRACTED_VERSION}" ] && [ "${EXTRACTED_VERSION}" != "null" ]; then
            echo "${EXTRACTED_VERSION}"
            return 0
        fi
    fi
    
    return 1
}

# Method 2: Try readelf method (works for CGO builds with ELF metadata section)
try_readelf_method() {
    if ! command -v readelf >/dev/null 2>&1; then
        return 1
    fi
    
    if ! command -v jq >/dev/null 2>&1; then
        return 1
    fi
    
    # Check if .note.metadata section exists and extract version
    if readelf -p .note.metadata "${BINARY_PATH}" 2>/dev/null | grep -q "version"; then
        EMBEDDED_VERSION=$(readelf -p .note.metadata "${BINARY_PATH}" 2>/dev/null | grep -oE '\{[^}]+\}' | jq -r '.version' | head -1)
        
        if [ -n "${EMBEDDED_VERSION}" ] && [ "${EMBEDDED_VERSION}" != "null" ]; then
            echo "${EMBEDDED_VERSION}"
            return 0
        fi
    fi
    
    return 1
}

# Try methods in order
if try_strings_method; then
    exit 0
fi

if try_readelf_method; then
    exit 0
fi

# If both methods failed
echo "ERROR: Could not extract version from binary: ${BINARY_PATH}" >&2
echo "Ensure the binary was built with metadata embedding enabled" >&2
exit 1
