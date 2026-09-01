# Security Policy

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| main    | :white_check_mark: |

## Reporting a Vulnerability

**Do not open a public issue** for security vulnerabilities.

Email **rodrigopizarro1234@gmail.com** with:

- Description of the issue
- Steps to reproduce
- Estimated impact
- Affected version or commit (if applicable)

We aim to respond within 72 hours. We will let you know when a fix is published.

## Good practices in this repo

- Do not commit secrets (`.env`, keys, tokens). Use `.env.example` as a template.
- Relevant security changes should have a spec in `docs/sdd/` when the change is not trivial.
