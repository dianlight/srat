#!/usr/bin/env python3
"""Setup script for SRAT Companion development environment."""

import subprocess
import sys
from pathlib import Path


def setup_development() -> bool:
    """Set up development environment."""
    print("Setting up SRAT Companion development environment...")

    # Install development dependencies
    print("Installing development dependencies...")
    try:
        subprocess.run(
            [sys.executable, "-m", "pip", "install", "-r", "requirements.txt"],
            check=True,
        )
        print("✓ Development dependencies installed")
    except subprocess.CalledProcessError as e:
        print(f"✗ Failed to install dependencies: {e}")
        return False

    # Generate OpenAPI client
    print("Generating OpenAPI client...")
    try:
        from generate_client import generate_client

        generate_client()
        print("✓ OpenAPI client generated")
    except Exception as e:
        print(f"✗ Failed to generate client: {e}")
        return False

    # Setup pre-commit hooks
    print("Setting up pre-commit hooks...")
    try:
        subprocess.run(["pre-commit", "install"], check=True)
        print("✓ Pre-commit hooks installed")
    except subprocess.CalledProcessError:
        print("⚠ Pre-commit not available, skipping hooks setup")

    print("\n✓ Development environment setup complete!")
    print("\nNext steps:")
    print(
        "1. Copy custom_components/srat_companion to your HA config/custom_components/"
    )
    print("2. Restart Home Assistant")
    print("3. Add the SRAT Companion integration")

    return True


if __name__ == "__main__":
    setup_development()
