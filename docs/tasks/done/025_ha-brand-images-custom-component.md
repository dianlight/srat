# [FEATURE]: HA Brand Images for Custom Component

**Target Repo:** `srat`  **Status:** ✅ Complete  **Issue Link:** https://github.com/dianlight/srat/issues/552

## 🎯 Objective

Follow the [HA Brand Images specification](https://developers.home-assistant.io/docs/core/integration/brand_images)
and the [home-assistant/brands README](https://github.com/home-assistant/brands/blob/master/README.md)
to create the brand folder inside `custom_components/srat`.

Since HA 2026.3.0, custom components can bundle brand icons directly in their folder (see the
[Brands Proxy API announcement](https://developers.home-assistant.io/blog/2026/02/24/brands-proxy-api)).

Generate icons and logos in all required formats starting from `frontend/src/img` source assets,
using an automated conversion script integrated with mise.

## 🛠️ Technical Specifications

- **Inputs:** `frontend/src/img/icon.png` (128×128), `frontend/src/img/logo.png` (129×128)
- **Outputs:** Brand images placed directly in `custom_components/srat/`
  - `icon.png` — 256×256 square icon
  - `icon@2x.png` — 512×512 hDPI square icon
  - `logo.png` — logo, shortest side 256 px
  - `logo@2x.png` — logo, shortest side 512 px
- **Script:** `scripts/generate-brand-images.py` (Python + Pillow)
- **Mise task:** `brand-images` in root `.mise.toml`
- **Dependencies:** Python 3.12+, Pillow

## 📝 Task List

- [x] Task 1: Explore source assets and HA brand spec requirements
- [x] Task 2: Create this task document and GitHub issue
- [x] Task 3: Write `scripts/generate-brand-images.py` script
- [x] Task 4: Add `brand-images` mise task in root `.mise.toml`
- [x] Task 5: Run script and verify generated images land in `custom_components/srat/`
- [x] Task 6: Open PR

## 🧠 Implementation Notes

**Image requirements (from brands README):**

- PNG format, lossless compression, with transparency.
- Icon: square, 256×256 (normal) and 512×512 (hDPI).
- Logo: landscape preferred, shortest side 128–256 px (normal) and 256–512 px (hDPI).
- Dark variants are optional; fall back to light versions if absent.

**Plan agreed:**

1. Write `scripts/generate-brand-images.py` using Pillow to resize source images
   to all required dimensions and save as optimised PNGs.
2. Add `[tasks.brand-images]` entry in root `.mise.toml` pointing to the script.
3. Run the script; commit generated images alongside the script.

## 🔗 Code References & TODOs

- `frontend/src/img/icon.png` — source 128×128 icon
- `frontend/src/img/logo.png` — source 129×128 logo
- `custom_components/srat/` — destination for brand images
- `scripts/generate-brand-images.py` — new script
- `.mise.toml` — root mise configuration (add `brand-images` task)
