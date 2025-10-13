# Security Policy

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->

- [Supported Versions](#supported-versions)
- [Reporting a Vulnerability](#reporting-a-vulnerability)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

## Supported Versions

Use this section to tell people about which versions of your project are
currently being supported with security updates.

| Version     | Supported          |
| ----------- | ------------------ |
| 2025.10.x   | :white_check_mark: |
| < 2025.10.0 | :x:                |

## Reporting a Vulnerability

We take security seriously. If you discover a vulnerability, please follow these steps:

1. Do not open a public issue. Instead, report privately via GitHub Security Advisories:
   - Navigate to the repository Security tab → Report a vulnerability
   - Or go directly: https://github.com/dianlight/srat/security/advisories/new
2. Provide a clear description with reproduction steps, affected versions, and impact.
3. If possible, include a minimal proof of concept. Avoid sharing sensitive data.

Response expectations:

- We will acknowledge receipt within 5 business days.
- We will investigate and provide a status update within 10 business days.
- If confirmed, we will work on a fix and coordinate a disclosure window.

Disclosure policy:

- We prefer responsible disclosure. We will publish an advisory with credit once a fix is released.
- Security fixes will target the latest supported series (see table above). Older, unsupported releases may not receive patches.

Out-of-scope examples:

- Issues requiring privileged local access without a clear escalation path
- Denial of service via unrealistic resource constraints
- Vulnerabilities in third-party dependencies without a practical exploit path in SRAT
