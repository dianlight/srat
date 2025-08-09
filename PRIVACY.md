# Privacy Policy

**Effective Date:** August 6, 2025
**Last Updated:** August 6, 2025

## Overview

SRAT (Samba Resource Administration Tool) respects your privacy and is committed to protecting your personal data. This privacy policy explains how we collect, use, and protect information when you use our software.

## Data Collection

### Telemetry and Error Reporting

SRAT includes optional telemetry and error reporting functionality powered by [Rollbar](https://rollbar.com). This system operates under user control with four distinct modes:

- **Ask** (Default): No data is collected until you make a choice
- **All**: Error reports and usage analytics are collected
- **Errors**: Only error reports are collected
- **Disabled**: No data is collected or transmitted

### What Data We Collect

When telemetry is enabled, we may collect the following information:

#### Error Information (Errors and All modes)

- Error messages and stack traces
- Application version and environment (development/production)
- Timestamp when the error occurred
- Browser user agent string (frontend errors only)
- Current page URL when error occurred (frontend errors only)
- Component stack traces (React component errors)

#### Usage Analytics (All mode only)

- Application initialization events
- Telemetry configuration changes
- Feature usage statistics
- Technical performance metrics
- Session information (no personal identifiers)

### What Data We DO NOT Collect

SRAT never collects:

- Personal files or file contents
- Passwords or authentication credentials
- Network share contents or file names
- Personal identification information
- IP addresses or location data
- User's personal data or sensitive information

## How We Use Your Data

Data collected through Rollbar telemetry is used exclusively to:

- Identify and fix software bugs
- Improve application stability and performance
- Understand how features are being used
- Prioritize development efforts
- Provide better user experience

## Data Transmission and Storage

- All telemetry data is transmitted securely over HTTPS
- Data is stored and processed by Rollbar in accordance with their [Privacy Policy](https://rollbar.com/privacy/)
- No data is transmitted when internet connectivity is unavailable
- No data is transmitted in "Ask" or "Disabled" modes

## Your Control and Choices

### Telemetry Configuration

- **First Launch**: You will be prompted to choose your telemetry preference
- **Settings Page**: Change your telemetry mode at any time in the application settings
- **Internet Requirement**: Telemetry requires internet connectivity to function
- **Complete Control**: You can disable all data collection at any time

### Data Subject Rights

You have the right to:

- Enable or disable telemetry collection at any time
- Access information about what data has been collected (via Rollbar's systems)
- Request deletion of your data (contact Rollbar directly)
- Withdraw consent for data processing

## Third-Party Services

### Rollbar

SRAT uses Rollbar for error tracking and telemetry. When telemetry is enabled:

- Data is transmitted to and processed by Rollbar's servers
- Rollbar's [Privacy Policy](https://rollbar.com/privacy/) and [Terms of Service](https://docs.rollbar.com/docs/terms-of-service) apply
- Rollbar may process data in accordance with applicable privacy laws (GDPR, CCPA, etc.)

## Technical Implementation

### Data Security

- All telemetry data is transmitted over encrypted HTTPS connections
- No sensitive application data (passwords, file contents) is ever included in reports
- Rollbar access tokens are embedded at build time and not user-configurable
- Error context is limited to technical debugging information only

### Internet Connectivity

- Telemetry functionality requires active internet connection
- Connection checks are performed against `https://api.rollbar.com`
- No data is transmitted when offline
- Internet requirement prevents accidental data transmission

## Children's Privacy

SRAT is not designed for or directed at children under the age of 13. We do not knowingly collect personal information from children under 13.

## Changes to This Privacy Policy

We may update this privacy policy from time to time. Any changes will be reflected in the "Last Updated" date above. Significant changes will be communicated through the application or project documentation.

## Data Retention

- Error data is retained according to Rollbar's data retention policies
- Users can disable telemetry at any time to stop future data collection
- Historical data previously collected remains subject to Rollbar's retention policies

## Geographic Considerations

SRAT is distributed globally and telemetry data may be processed in various jurisdictions where Rollbar operates. By enabling telemetry, you consent to this international data transfer.

## Contact Information

For privacy-related questions or concerns about SRAT:

- **Project Repository**: [https://github.com/dianlight/srat](https://github.com/dianlight/srat)
- **Issues**: Report privacy concerns via GitHub Issues
- **Rollbar Privacy**: For data subject requests, contact Rollbar directly at [https://rollbar.com/privacy/](https://rollbar.com/privacy/)

## Compliance

This privacy policy is designed to comply with applicable privacy regulations including:

- General Data Protection Regulation (GDPR)
- California Consumer Privacy Act (CCPA)
- Other applicable local privacy laws

---

## Summary

SRAT's telemetry is:

- **Optional**: Completely user-controlled with clear choices
- **Transparent**: This policy clearly explains what data is collected
- **Limited**: Only technical error and usage data, never personal files or sensitive information
- **Secure**: All data transmitted over HTTPS to trusted third-party service
- **Controlled**: Can be disabled at any time through application settings

By using SRAT with telemetry enabled, you acknowledge that you have read and understood this privacy policy.
