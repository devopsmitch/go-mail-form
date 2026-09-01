# 📨 go-mail-form

A lightweight, self-hosted email relay for contact forms. Zero external dependencies — just a single Go binary.

## Features

- HTML form and API access with JSON or multipart form data
- Configurable CORS and origin restriction
- Per-target rate limiting
- Optional API key authentication
- File attachments
- Honeypot antispam (hidden `_gotcha` field)
- Optional Cloudflare Turnstile CAPTCHA verification
- Redirect support for form submissions

## Quick Start

### 1. Create a target

Create a `targets/` directory and add a JSON file for each endpoint:

```shell
mkdir targets
cp targets/example.json targets/my-site.json
# Edit with your SMTP credentials and recipients
```

See [targets/example.json](targets/example.json) for reference.

### 2. Run with Docker

A public image is available on GitHub Container Registry:

```shell
docker run -d \
  -p 3000:3000 \
  -v $(pwd)/targets:/targets \
  -e TARGETS_DIR=/targets \
  ghcr.io/devopsmitch/go-mail-form:main
```

Or build it yourself:

```shell
docker build -t go-mail-form .
docker run -d \
  -p 3000:3000 \
  -v $(pwd)/targets:/targets \
  -e TARGETS_DIR=/targets \
  go-mail-form
```

### 3. Send a test email

```shell
curl -X POST http://localhost:3000/my-site \
  -d "from=user@example.com" \
  -d "subject=Hello" \
  -d "body=Test message"
```

## Configuration

Each setting can be provided as a command-line flag or an environment variable. When both are set, the flag takes precedence. Precedence is: **flag > environment variable > default**.

| Flag | Environment variable | Default | Description |
|---|---|---|---|
| `--port` | `PORT` | `3000` | Server port |
| `--targets-dir` | `TARGETS_DIR` | `targets` | Path to target JSON files |
| `--trusted-header` | `TRUSTED_HEADER` | *(empty)* | HTTP header to trust for client IP (e.g. `CF-Connecting-IP`, `X-Forwarded-For`). If unset, uses the direct connection IP |

Other flags:

| Flag | Description |
|---|---|
| `--version` | Print the version and exit |

## Target File Reference

| Field | Required | Description |
|---|---|---|
| `transport` | no | Delivery transport: `smtp` (default) or `ses` |
| `smtp` | conditional | SMTP(S) URL, e.g. `smtps://user:pass@smtp.example.com`. Required when `transport` is `smtp` |
| `ses.region` | conditional | AWS region for Amazon SES, e.g. `us-east-1`. Required when `transport` is `ses` |
| `from` | yes | Sender address. Must be a verified identity when using SES |
| `recipients` | yes | Array of recipient email addresses |
| `rateLimit.timespan` | yes | Rate limit window in seconds |
| `rateLimit.requests` | yes | Max requests per window per IP |
| `origin` | no | Allowed HTTP origin (CORS). Default `*` |
| `subjectPrefix` | no | Prefix prepended to all subjects |
| `key` | no | API key (sent as `Authorization: Bearer <key>`) |
| `redirect.success` | no | URL to redirect on success |
| `redirect.error` | no | URL to redirect on error |
| `turnstile.secretKey` | no | Cloudflare Turnstile secret key. Enables CAPTCHA verification when set |

> **Rate limiting IP detection:** The client IP is resolved from the `TRUSTED_HEADER` if configured, otherwise from the direct connection address. Set `TRUSTED_HEADER=CF-Connecting-IP` behind Cloudflare, or `TRUSTED_HEADER=X-Forwarded-For` behind other reverse proxies. Leave unset if clients connect directly.

> **Note:** The `origin` check relies on the `Origin` header, which is only sent by browsers. Non-browser API clients (e.g. `curl`) won't send it, so requests will be rejected if `origin` is set. If you need both browser form submissions and API access on the same target, use `key` for API clients and `origin` for browser CORS — or create separate targets.

### Example with Turnstile

```json
{
    "smtp": "smtps://user:pass@smtp.example.com",
    "recipients": ["you@example.com"],
    "from": "noreply@example.com",
    "rateLimit": { "timespan": 60, "requests": 5 },
    "turnstile": {
        "secretKey": "0x4AAAAAAA..."
    }
}
```

### Sending via Amazon SES

Set `transport` to `ses` and provide a region. The message is delivered through
the SES `SendEmail` API over HTTPS instead of SMTP, so no SMTP credentials are
needed. The `from` address must be a verified SES identity (email or domain).

```json
{
    "transport": "ses",
    "ses": { "region": "us-east-1" },
    "recipients": ["you@example.com"],
    "from": "noreply@example.com",
    "rateLimit": { "timespan": 60, "requests": 5 }
}
```

AWS credentials are resolved from the standard credential chain: environment
variables (`AWS_ACCESS_KEY_ID` / `AWS_SECRET_ACCESS_KEY` / `AWS_SESSION_TOKEN`),
shared config (`~/.aws`), or an attached IAM role / task role. The role needs
the `ses:SendEmail` permission. SES clients are created per distinct region only
when at least one target uses the SES transport, so SMTP-only deployments do not
require AWS credentials.

## Request Fields

| Field | Required | Description |
|---|---|---|
| `from` | no | Visitor's email address. Used as the `Reply-To` so replies go to the visitor; the email's `From` is always the target's configured `from` |
| `name` | no | Visitor's name, combined with `from` in the `Reply-To` |
| `subject` | yes | Email subject (2-255 chars) |
| `body` | yes | Email body, supports HTML (5-32000 chars) |
| `subjectPrefix` | no | Per-request subject prefix |
| `_gotcha` | no | Honeypot field — leave empty. Bots that fill this in are silently rejected |

## HTML Form Example

```html
<form method="post" action="https://mailform.example.com/my-site">
  <input type="email" name="from" placeholder="Your email" />
  <input type="text" name="name" placeholder="Your name" />
  <input type="text" name="subject" placeholder="Subject" />
  <textarea name="body" placeholder="Your message"></textarea>
  <!-- Honeypot: hide with CSS, invisible to screen readers and keyboard -->
  <div class="hp-field" aria-hidden="true">
    <input type="text" name="_gotcha" tabindex="-1" autocomplete="off" />
  </div>
  <!-- Optional: Cloudflare Turnstile CAPTCHA -->
  <div class="cf-turnstile" data-sitekey="YOUR_SITE_KEY"></div>
  <button type="submit">Send</button>
</form>

<!-- Include only if using Turnstile -->
<script src="https://challenges.cloudflare.com/turnstile/v0/api.js" async defer></script>

<style>
  .hp-field { position: absolute; left: -9999px; }
</style>
```

## API Response Codes

| Code | Meaning |
|---|---|
| `200` | Email sent |
| `400` | Bad request |
| `401` | Unauthorized (wrong or missing API key) |
| `403` | Forbidden (origin mismatch) |
| `404` | Target not found |
| `405` | Method not allowed |
| `422` | Validation error (details in JSON body) |
| `429` | Rate limited |
| `500` | Email sending failed |

## Health Check

```
GET /healthz → 200 OK
```

## Versioning

This project follows [Semantic Versioning](https://semver.org/). Given a version `MAJOR.MINOR.PATCH`:

- **MAJOR** — Changes that may break existing deployments, such as changes to target-file fields, request fields, or the HTTP request/response contract. Review the release notes before upgrading.
- **MINOR** — Backwards-compatible new features (e.g. a new optional target field).
- **PATCH** — Backwards-compatible bug fixes.

While the version is `0.x`, the target-file schema and API should be considered unstable and may change between minor releases. Version `1.0.0` marks the point where the target-file format and API contract are considered stable.

Released binaries and tagged Docker images report their version via `--version` and in the startup log. Docker images built from `main` (untagged) report the commit SHA instead.
