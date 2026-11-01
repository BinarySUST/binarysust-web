# Security policy

## Reporting a vulnerability

Email **binary.sust@gmail.com** with "Security" in the subject. Please do not open a public issue, pull request
or discussion for a vulnerability.

Include what you found, where (URL, file or endpoint), how to reproduce it, and what an attacker could do with it.
A committee member will acknowledge your report within 7 days and keep you updated until it is fixed. We credit
reporters in the fix's release notes unless you ask us not to.

## Scope

- The site at `https://binarysust.org`, its `/api/*` proxy and the Go API (`apps/api`).
- Sign-in (Google OAuth), sessions, roles and permissions, uploads (`media.binarysust.org`).
- This repository's GitHub Actions workflows and their secrets handling.

Out of scope: denial of service, findings that need a compromised device or account, missing security headers
without a demonstrated impact, and reports produced by automated scanners without a working proof of concept.

## Supported versions

Only the `main` branch, which is what is deployed, receives fixes.
