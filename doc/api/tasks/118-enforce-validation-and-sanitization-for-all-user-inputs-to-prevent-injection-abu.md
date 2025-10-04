# Enforce validation and sanitization for all user inputs to prevent injection/abuse.

**Parent Section:** 11. Security & Compliance
**Task ID:** 118

## Goal
Ensure all inputs validated and sanitised to prevent injection, XSS, or data corruption.

## Plan
- Create validation utilities (regex, length checks) and HTML sanitizer for content endpoints.
- Enforce validation in handlers/services with descriptive errors.
- Add automated tests for typical attack payloads (script tags, SQL-like injection, path traversal).
- Document validation rules per endpoint.
