# Perform security review (HMAC secret rotation, OAuth scopes, firewall rules) before launch.

**Parent Section:** 11. Security & Compliance
**Task ID:** 121

## Goal
Conduct comprehensive security review before launch covering secrets, OAuth scopes, firewall rules, and threat modeling.

## Plan
- Run dependency vulnerability scans (govulncheck, Snyk) and patch findings.
- Review firewall/IP restrictions, Cloud Armor rules, and IAP configuration.
- Validate secret rotation process, key management, and audit trails.
- Perform threat modeling session; capture mitigations in security documentation.
