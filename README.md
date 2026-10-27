# Bento Go backend

The Go backend currently implements the API foundation, Google Sign-In, and user session endpoints. Feature migration scope and completion checkboxes are tracked in [docs/migration-plan.md](docs/migration-plan.md).

## Local API startup

Run MongoDB and Redis. Google sign-in receives the frontend's OAuth access token in the existing `idToken` request field. Set `APP_AUTH_GOOGLE_CLIENT_ID` to the OAuth client ID used by the frontend, and set `APP_SERVER_CORS_ALLOWED_ORIGINS` to exact frontend origins when the UI and API use different origins. The Go backend checks the token with Google's tokeninfo endpoint and requires its audience and authorized party to match that client, plus a live token, an OpenID Connect subject, and a verified email. It does not silently link an existing account based only on a matching email; those accounts need an explicit provider-link flow.

Google access tokens are bearer credentials, so never log or expose them. Google documents access-token introspection through tokeninfo for diagnostic use ([token types](https://cloud.google.com/docs/authentication/token-types)). This Go-only migration keeps the current frontend contract; for a stronger long-term production flow, migrate the client to send an ID token or use an authorization-code flow and validate/exchange it server-side ([backend authentication guidance](https://developers.google.com/identity/sign-in/web/backend-auth)). The local config points to the existing NestJS RS256 key files under `../backend/secure/`, so both backends can validate the same access and refresh tokens during a staged cutover. Keep using those keys while both services are active. Override the four key path variables in `.env` if those files are stored elsewhere.

If you do not already have a `.env`, copy `.env.example` and adjust the settings as needed. From this directory, start the API with:

```sh
make run
```

`make run` sources `.env` when present and starts the API. The local config listens on port `4001`, connects to MongoDB at `mongodb://localhost:27017`, uses database `bento`, and connects to Redis at `localhost:6379`. To use another config file, set `APP_CONFIG` in `.env` or export it in your shell. Environment overrides use the `APP_` prefix.

## Implemented endpoints

- `POST /api/v2/sign-in/google` — accepts `{"idToken":"..."}` (the frontend's Google OAuth access token), verifies it against the configured Google OAuth client, and sets the session cookies.
- `POST /api/v1/auth/refresh` — rotates the access and refresh cookies.
- `POST /api/v1/auth/logout` — clears session cookies.
- `GET /api/v1/auth/check` — returns the authenticated Bento user; accepts a Bearer token or access cookie.
- `POST /api/v1/auth/admin-login` — optional admin login, enabled when admin credentials are configured.
- `GET /api/health` — API and MongoDB readiness.
- `GET /metrics` — Prometheus metrics.

In production, provide MongoDB settings, RS256 key paths, the exact browser origins, and—when CSRF is enabled—a dedicated CSRF secret of at least 32 bytes through environment or deployment secrets. Keep private key files and secrets out of source control.
