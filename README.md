# 📨 go-mail-form

A lightweight, self-hosted email relay for contact forms. Zero external dependencies — just a single Go binary.

## Features

- HTML form and API access with JSON or multipart form data
- Configurable CORS and origin restriction
- Per-target rate limiting
- Optional API key authentication
- File attachments
- Honeypot antispam (hidden `_gotcha` field)
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

Environment variables:

| Variable | Default | Description |
|---|---|---|
| `PORT` | `3000` | Server port |
| `TARGETS_DIR` | `targets` | Path to target JSON files |
| `TRUSTED_HEADER` | *(empty)* | HTTP header to trust for client IP (e.g. `CF-Connecting-IP`, `X-Forwarded-For`). If unset, uses the direct connection IP |

## Target File Reference

| Field | Required | Description |
|---|---|---|
| `smtp` | yes | SMTP(S) URL, e.g. `smtps://user:pass@smtp.example.com` |
| `recipients` | yes | Array of recipient email addresses |
| `rateLimit.timespan` | yes | Rate limit window in seconds |
| `rateLimit.requests` | yes | Max requests per window per IP |
| `origin` | no | Allowed HTTP origin (CORS). Default `*` |
| `from` | no | Default sender address |
| `subjectPrefix` | no | Prefix prepended to all subjects |
| `key` | no | API key (sent as `Authorization: Bearer <key>`) |
| `redirect.success` | no | URL to redirect on success |
| `redirect.error` | no | URL to redirect on error |

> **Rate limiting IP detection:** The client IP is resolved from the `TRUSTED_HEADER` if configured, otherwise from the direct connection address. Set `TRUSTED_HEADER=CF-Connecting-IP` behind Cloudflare, or `TRUSTED_HEADER=X-Forwarded-For` behind other reverse proxies. Leave unset if clients connect directly.

> **Note:** The `origin` check relies on the `Origin` header, which is only sent by browsers. Non-browser API clients (e.g. `curl`) won't send it, so requests will be rejected if `origin` is set. If you need both browser form submissions and API access on the same target, use `key` for API clients and `origin` for browser CORS — or create separate targets.

## Request Fields

| Field | Required | Description |
|---|---|---|
| `from` | no | Sender email address |
| `name` | no | Sender name |
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
  <button type="submit">Send</button>
</form>

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
