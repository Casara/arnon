# Security Policy

## Supported Versions

Before `v1.0.0`, only the latest released version is supported with
security fixes. Once `v1.0.0` ships, this section will list which
major versions still receive patches.

## Reporting a Vulnerability

Please **do not** open a public issue for a security vulnerability.

Use GitHub's private vulnerability reporting instead: go to the
[Security tab](https://github.com/casara/arnon/security) of this
repository and select "Report a vulnerability." This opens a private
advisory visible only to the maintainers until a fix is ready.

Include, if possible:

- The affected version/commit.
- A minimal reproduction (a failing request against a minimal
  `httpx.Endpoint`/middleware setup is usually enough).
- The impact you believe it has (e.g., does it affect request
  validation, error responses leaking data, authentication/authorization
  if used alongside arnon, denial of service).

We'll acknowledge the report and work with you on a disclosure
timeline before any public advisory is published.
