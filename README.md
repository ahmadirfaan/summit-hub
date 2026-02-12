# SummitHub Backend (Fiber + Go)

Backend skeleton for hiking trips, tracking, waypoints, and social feed. Uses Fiber, PostgreSQL (with PostGIS), and optional Redis Pub/Sub for live tracking.

## Features covered
- Auth: login, JWT, refresh tokens
- Trips: CRUD trips, invite members, upload GPX routes
- Tracking: start session, track points, summary, WebSocket broadcast
- Waypoints: CRUD, visit check, reviews, geo search
- Social: posts, follow, feed, geo photo
- Storage: placeholder upload endpoint

## Quick start

1) Configure environment variables (see `.env.example`).
2) Ensure PostgreSQL has PostGIS enabled (migration in `migrations/001_init.sql`).

## API overview

Base URL: `http://localhost:8080`

### Auth
- `POST /auth/register`
- `POST /auth/login`
- `POST /auth/refresh`
- `GET /auth/jwt/verify`

### Trips
- `POST /trips`
- `GET /trips/:id`
- `PUT /trips/:id`
- `DELETE /trips/:id`
- `POST /trips/:id/members`
- `GET /trips/:id/members`
- `POST /trips/:id/routes`
- `GET /trips/:id/routes`

### Tracking
- `POST /tracking/sessions`
- `POST /tracking/sessions/:id/points`
- `GET /tracking/sessions/:id/summary`
- `GET /tracking/sessions/:id/points`
- WebSocket: `GET /stream/ws/:sessionID`

### Waypoints
- `POST /waypoints`
- `GET /waypoints/:id`
- `PUT /waypoints/:id`
- `DELETE /waypoints/:id`
- `POST /waypoints/:id/visit`
- `POST /waypoints/:id/reviews`
- `GET /waypoints/:id/reviews`
- `POST /waypoints/:id/photos`
- `GET /waypoints/:id/photos`
- `GET /waypoints/search?lat=...&lng=...&radius_km=...`

### Social
- `POST /social/posts`
- `POST /social/posts/:id/photos`
- `POST /social/follow`
- `GET /social/feed?user_id=...`
- `GET /social/posts/nearby?lat=...&lng=...&radius_km=...`

### Storage
- `POST /storage/upload`

## Notes
- The implementation uses Postgres with PostGIS and stores refresh tokens in `refresh_tokens`.
- For geo queries, PostGIS tables are provided in `migrations/001_init.sql`.
