#!/usr/bin/env python3
"""Script to generate OpenAPI client for SRAT Companion."""

import subprocess
import sys
import os
from pathlib import Path


def generate_client():
    """Generate the OpenAPI client."""
    # Path to the OpenAPI spec
    openapi_spec = Path(__file__).parent.parent / "backend" / "docs" / "openapi.yaml"

    if not openapi_spec.exists():
        print(f"Error: OpenAPI spec not found at {openapi_spec}")
        sys.exit(1)

    # Output directory
    output_dir = (
        Path(__file__).parent.parent / "custom_components" / "srat_companion" / "client"
    )

    # Command to generate client
    cmd = [
        "openapi-python-client",
        "generate",
        "--path",
        str(openapi_spec),
        "--output-path",
        str(output_dir),
        "--meta",
        "none",
        # "--client-name",
        # "srat_client",
        # "--package-name",
        # "client",
        "--overwrite",
    ]

    try:
        subprocess.run(cmd, check=True)
        print(f"Client generated successfully at {output_dir}")
    except subprocess.CalledProcessError as e:
        print(f"Error generating client: {e}")
        sys.exit(1)
    except FileNotFoundError:
        print("Error: openapi-python-client not found. Install it with:")
        print("pip install openapi-python-client")
        sys.exit(1)


if __name__ == "__main__":
    generate_client()
