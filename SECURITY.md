# Security Policy

## Supported versions

Latest minor release receives security fixes.

## Reporting a vulnerability

**Do not open a public issue.** Email **linderhassinger00@gmail.com** with:

- Description and impact
- Reproduction steps
- Affected version
- Suggested fix (optional)

Acknowledgment within 72 hours. Coordinated disclosure preferred.

## Scope

Especially relevant: anything that lets cleanmac delete a path the safety
model is supposed to block — symlink escape, path-traversal in scanner output,
config injection, race conditions between scan and delete, etc.
