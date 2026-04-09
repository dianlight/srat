#!/usr/bin/env python3
"""Generate HA brand images for custom_components/srat.

Reads source assets from frontend/src/img and produces all required brand-image
variants in custom_components/srat/ according to the Home Assistant brands spec:
https://github.com/home-assistant/brands/blob/master/README.md

Required outputs
----------------
icon.png      256×256  – standard square icon
icon@2x.png   512×512  – hDPI square icon
logo.png       shortest side 256 px – standard logo
logo@2x.png    shortest side 512 px – hDPI logo

Usage
-----
Run from the repository root:
    python3 scripts/generate-brand-images.py
"""

from __future__ import annotations

import argparse
import sys
from pathlib import Path

try:
    from PIL import Image
except ImportError:
    print("ERROR: Pillow is required. Install it with: pip install pillow", file=sys.stderr)
    sys.exit(1)

# ---------------------------------------------------------------------------
# Paths (relative to repository root)
# ---------------------------------------------------------------------------

REPO_ROOT = Path(__file__).resolve().parent.parent
SRC_ICON = REPO_ROOT / "frontend" / "src" / "img" / "icon.png"
SRC_LOGO = REPO_ROOT / "frontend" / "src" / "img" / "logo.png"
DEST_DIR = REPO_ROOT / "custom_components" / "srat"


def _resize_icon(src: Path, size: int) -> Image.Image:
    """Return a square icon resized to *size* × *size*."""
    with Image.open(src) as img:
        img = img.convert("RGBA")
        return img.resize((size, size), Image.LANCZOS)


def _resize_logo(src: Path, shortest_side: int) -> Image.Image:
    """Return the logo resized so its shortest side equals *shortest_side*."""
    with Image.open(src) as img:
        img = img.convert("RGBA")
        w, h = img.size
        if w <= h:
            # width is the shortest side
            new_w = shortest_side
            new_h = round(h * shortest_side / w)
        else:
            # height is the shortest side
            new_h = shortest_side
            new_w = round(w * shortest_side / h)
        return img.resize((new_w, new_h), Image.LANCZOS)


def _save(image: Image.Image, dest: Path) -> None:
    """Save *image* as an optimised PNG to *dest*."""
    dest.parent.mkdir(parents=True, exist_ok=True)
    image.save(
        dest,
        format="PNG",
        optimize=True,
        # Interlaced ("Adam7") PNGs are preferred by the brands spec
        interlace=True,
    )
    print(f"  ✔  {dest.relative_to(REPO_ROOT)}  ({image.size[0]}×{image.size[1]})")


def generate(dry_run: bool = False) -> None:
    """Generate all brand-image variants."""
    for src in (SRC_ICON, SRC_LOGO):
        if not src.exists():
            print(f"ERROR: source file not found: {src}", file=sys.stderr)
            sys.exit(1)

    print(f"Source icon : {SRC_ICON.relative_to(REPO_ROOT)}")
    print(f"Source logo : {SRC_LOGO.relative_to(REPO_ROOT)}")
    print(f"Destination : {DEST_DIR.relative_to(REPO_ROOT)}")
    print()

    variants: list[tuple[str, Image.Image]] = [
        ("icon.png", _resize_icon(SRC_ICON, 256)),
        ("icon@2x.png", _resize_icon(SRC_ICON, 512)),
        ("logo.png", _resize_logo(SRC_LOGO, 256)),
        ("logo@2x.png", _resize_logo(SRC_LOGO, 512)),
    ]

    if dry_run:
        print("Dry-run mode – no files written.")
        for name, img in variants:
            print(f"  would write {DEST_DIR / name}  ({img.size[0]}×{img.size[1]})")
        return

    print("Writing brand images …")
    for name, img in variants:
        _save(img, DEST_DIR / name)

    print()
    print("Done. All brand images generated.")


def main() -> None:
    """Entry point."""
    parser = argparse.ArgumentParser(
        description=__doc__,
        formatter_class=argparse.RawDescriptionHelpFormatter,
    )
    parser.add_argument(
        "--dry-run",
        action="store_true",
        help="Print what would be done without writing any files.",
    )
    args = parser.parse_args()
    generate(dry_run=args.dry_run)


if __name__ == "__main__":
    main()
