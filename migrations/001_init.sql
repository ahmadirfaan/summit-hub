CREATE EXTENSION IF NOT EXISTS postgis;
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE users (
    id UUID PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    username VARCHAR(100) UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    full_name VARCHAR(150),
    avatar_url TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE trips (
    id UUID PRIMARY KEY,
    name VARCHAR(200) NOT NULL,
    mountain_name VARCHAR(200),
    start_date DATE,
    end_date DATE,
    description TEXT,
    created_by UUID REFERENCES users(id),
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE trip_members (
    trip_id UUID REFERENCES trips(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    role VARCHAR(50) DEFAULT 'member',
    joined_at TIMESTAMP DEFAULT NOW(),
    PRIMARY KEY (trip_id, user_id)
);

CREATE TABLE gpx_routes (
    id UUID PRIMARY KEY,
    trip_id UUID REFERENCES trips(id) ON DELETE CASCADE,
    name VARCHAR(200),
    description TEXT,
    total_distance_m DOUBLE PRECISION,
    total_elevation_gain_m DOUBLE PRECISION,
    route GEOGRAPHY(LINESTRING, 4326),
    uploaded_by UUID REFERENCES users(id),
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE track_sessions (
    id UUID PRIMARY KEY,
    trip_id UUID REFERENCES trips(id),
    user_id UUID REFERENCES users(id),
    started_at TIMESTAMP NOT NULL,
    ended_at TIMESTAMP,
    total_distance_m DOUBLE PRECISION,
    total_elevation_gain_m DOUBLE PRECISION,
    status VARCHAR(50) DEFAULT 'active'
);

CREATE TABLE track_points (
    id BIGSERIAL PRIMARY KEY,
    session_id UUID REFERENCES track_sessions(id) ON DELETE CASCADE,
    location GEOGRAPHY(POINT, 4326),
    elevation_m DOUBLE PRECISION,
    recorded_at TIMESTAMP NOT NULL,
    speed_mps DOUBLE PRECISION,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_track_points_session ON track_points(session_id);
CREATE INDEX idx_track_points_location ON track_points USING GIST(location);
CREATE INDEX idx_gpx_route_geom ON gpx_routes USING GIST(route);

CREATE TABLE waypoints (
    id UUID PRIMARY KEY,
    name VARCHAR(200),
    description TEXT,
    type VARCHAR(100),
    location GEOGRAPHY(POINT, 4326),
    elevation_m DOUBLE PRECISION,
    created_by UUID REFERENCES users(id),
    is_verified BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE waypoint_reviews (
    id UUID PRIMARY KEY,
    waypoint_id UUID REFERENCES waypoints(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id),
    rating INTEGER CHECK (rating BETWEEN 1 AND 5),
    comment TEXT,
    created_at TIMESTAMP DEFAULT NOW(),
    UNIQUE (waypoint_id, user_id)
);

CREATE TABLE waypoint_photos (
    id UUID PRIMARY KEY,
    waypoint_id UUID REFERENCES waypoints(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id),
    photo_url TEXT,
    caption TEXT,
    location GEOGRAPHY(POINT, 4326),
    taken_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE posts (
    id UUID PRIMARY KEY,
    user_id UUID REFERENCES users(id),
    content TEXT,
    location GEOGRAPHY(POINT, 4326),
    visibility VARCHAR(50) DEFAULT 'public',
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE post_photos (
    id UUID PRIMARY KEY,
    post_id UUID REFERENCES posts(id) ON DELETE CASCADE,
    photo_url TEXT,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE user_follows (
    follower_id UUID REFERENCES users(id),
    following_id UUID REFERENCES users(id),
    created_at TIMESTAMP DEFAULT NOW(),
    PRIMARY KEY (follower_id, following_id)
);

CREATE TABLE refresh_tokens (
    id UUID PRIMARY KEY,
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    token TEXT UNIQUE NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    revoked_at TIMESTAMP
);

CREATE INDEX idx_refresh_tokens_user ON refresh_tokens(user_id);

CREATE TABLE storage_objects (
    id UUID PRIMARY KEY,
    user_id UUID REFERENCES users(id),
    url TEXT NOT NULL,
    kind VARCHAR(50),
    created_at TIMESTAMP DEFAULT NOW()
);

-- Seed 1000 synthetic users for local/dev usage
INSERT INTO users (id, email, username, password_hash, full_name, avatar_url, created_at, updated_at)
SELECT
    gen_random_uuid(),
    format('user%04s@example.com', gs),
    format('user%04s', gs),
    'dev-hash',
    format('User %s', gs),
    format('https://picsum.photos/seed/user%04s/200/200', gs),
    NOW() - (gs || ' hours')::interval,
    NOW() - (gs || ' hours')::interval
FROM generate_series(1, 1000) AS gs
ON CONFLICT (email) DO NOTHING;

-- Seed trips
INSERT INTO trips (id, name, mountain_name, start_date, end_date, description, created_by, created_at)
SELECT
    gen_random_uuid(),
    format('Trip %s', gs),
    (ARRAY['Merbabu','Semeru','Rinjani','Kerinci','Gede','Pangrango','Sindoro','Sumbing'])[1 + (gs % 8)],
    CURRENT_DATE - (gs % 30),
    CURRENT_DATE - (gs % 30) + ((gs % 7) + 1),
    'Auto-generated trip for dev data',
    (SELECT id FROM users ORDER BY random() LIMIT 1),
    NOW() - (gs || ' days')::interval
FROM generate_series(1, 200) AS gs;

-- Seed trip members
INSERT INTO trip_members (trip_id, user_id, role, joined_at)
SELECT
    t.id,
    u.id,
    CASE WHEN random() < 0.1 THEN 'admin' ELSE 'member' END,
    NOW() - ((random() * 30)::int || ' days')::interval
FROM trips t
JOIN LATERAL (
    SELECT id FROM users ORDER BY random() LIMIT 5
) u ON TRUE
ON CONFLICT DO NOTHING;

-- Seed waypoints (clustered around Indonesian mountains)
INSERT INTO waypoints (id, name, description, type, location, elevation_m, created_by, is_verified, created_at)
SELECT
    gen_random_uuid(),
    format('Waypoint %s', gs),
    'Auto-generated waypoint for dev data',
    (ARRAY['camp','peak','viewpoint','spring','hut'])[1 + (gs % 5)],
    ST_SetSRID(ST_MakePoint(m.lon + (random() - 0.5) * 0.12, m.lat + (random() - 0.5) * 0.12), 4326)::geography,
    100 + (random() * 3000),
    (SELECT id FROM users ORDER BY random() LIMIT 1),
    random() < 0.3,
    NOW() - (gs || ' days')::interval
FROM generate_series(1, 5000) AS gs
JOIN LATERAL (
    SELECT * FROM (VALUES
        ('Semeru', 112.922, -8.108),
        ('Rinjani', 116.457, -8.411),
        ('Kerinci', 101.264, -1.697),
        ('Merbabu', 110.440, -7.454),
        ('Gede', 106.984, -6.783),
        ('Pangrango', 106.993, -6.772),
        ('Sindoro', 109.992, -7.300),
        ('Sumbing', 110.071, -7.384)
    ) AS t(name, lon, lat)
    ORDER BY random()
    LIMIT 1
) AS m ON TRUE;

-- Seed waypoint reviews
INSERT INTO waypoint_reviews (id, waypoint_id, user_id, rating, comment, created_at)
SELECT
    gen_random_uuid(),
    w.id,
    (SELECT id FROM users ORDER BY random() LIMIT 1),
    1 + (random() * 4)::int,
    'Great spot for a break!',
    NOW() - ((random() * 20)::int || ' days')::interval
FROM waypoints w
ORDER BY random()
LIMIT 800
ON CONFLICT (waypoint_id, user_id) DO NOTHING;

-- Seed waypoint photos
INSERT INTO waypoint_photos (id, waypoint_id, user_id, photo_url, caption, location, taken_at, created_at)
SELECT
    gen_random_uuid(),
    w.id,
    (SELECT id FROM users ORDER BY random() LIMIT 1),
    format('https://picsum.photos/seed/waypoint%04s/800/600', gs),
    'Auto-generated photo',
    w.location,
    NOW() - ((random() * 20)::int || ' days')::interval,
    NOW() - ((random() * 20)::int || ' days')::interval
FROM (SELECT id, location FROM waypoints ORDER BY random() LIMIT 1200) w
JOIN generate_series(1, 1200) gs ON TRUE;

-- Seed posts (clustered around Indonesian mountains)
INSERT INTO posts (id, user_id, content, location, visibility, created_at)
SELECT
    gen_random_uuid(),
    (SELECT id FROM users ORDER BY random() LIMIT 1),
    'Auto-generated post content',
    ST_SetSRID(ST_MakePoint(m.lon + (random() - 0.5) * 0.15, m.lat + (random() - 0.5) * 0.15), 4326)::geography,
    CASE WHEN random() < 0.85 THEN 'public' ELSE 'followers' END,
    NOW() - ((random() * 30)::int || ' days')::interval
FROM generate_series(1, 10000) AS gs
JOIN LATERAL (
    SELECT * FROM (VALUES
        ('Semeru', 112.922, -8.108),
        ('Rinjani', 116.457, -8.411),
        ('Kerinci', 101.264, -1.697),
        ('Merbabu', 110.440, -7.454),
        ('Gede', 106.984, -6.783),
        ('Pangrango', 106.993, -6.772),
        ('Sindoro', 109.992, -7.300),
        ('Sumbing', 110.071, -7.384)
    ) AS t(name, lon, lat)
    ORDER BY random()
    LIMIT 1
) AS m ON TRUE;

-- Seed post photos
INSERT INTO post_photos (id, post_id, photo_url, created_at)
SELECT
    gen_random_uuid(),
    p.id,
    format('https://picsum.photos/seed/post%04s/1000/800', gs),
    NOW() - ((random() * 30)::int || ' days')::interval
FROM (SELECT id FROM posts ORDER BY random() LIMIT 3000) p
JOIN generate_series(1, 3000) gs ON TRUE;

-- Seed follows
INSERT INTO user_follows (follower_id, following_id, created_at)
SELECT
    u1.id,
    u2.id,
    NOW() - ((random() * 60)::int || ' days')::interval
FROM (SELECT id FROM users ORDER BY random() LIMIT 1500) u1
JOIN LATERAL (SELECT id FROM users ORDER BY random() LIMIT 1) u2 ON TRUE
WHERE u1.id <> u2.id
ON CONFLICT DO NOTHING;

-- Seed refresh tokens
INSERT INTO refresh_tokens (id, user_id, token, expires_at, created_at, revoked_at)
SELECT
    gen_random_uuid(),
    u.id,
    encode(gen_random_bytes(32), 'hex'),
    NOW() + INTERVAL '30 days',
    NOW() - ((random() * 10)::int || ' days')::interval,
    NULL
FROM (SELECT id FROM users ORDER BY random() LIMIT 400) u;

-- Seed storage objects
INSERT INTO storage_objects (id, user_id, url, kind, created_at)
SELECT
    gen_random_uuid(),
    (SELECT id FROM users ORDER BY random() LIMIT 1),
    format('https://picsum.photos/seed/storage%04s/1200/900', gs),
    (ARRAY['photo','gpx','avatar'])[1 + (gs % 3)],
    NOW() - ((random() * 30)::int || ' days')::interval
FROM generate_series(1, 500) gs;

-- Seed track sessions
INSERT INTO track_sessions (id, trip_id, user_id, started_at, ended_at, total_distance_m, total_elevation_gain_m, status)
SELECT
    gen_random_uuid(),
    (SELECT id FROM trips ORDER BY random() LIMIT 1),
    (SELECT id FROM users ORDER BY random() LIMIT 1),
    NOW() - ((random() * 20)::int || ' days')::interval,
    NOW() - ((random() * 20)::int || ' days')::interval + INTERVAL '2 hours',
    1000 + (random() * 20000),
    100 + (random() * 2000),
    'completed'
FROM generate_series(1, 200) gs;

-- Seed track points
INSERT INTO track_points (session_id, location, elevation_m, recorded_at, speed_mps, created_at)
SELECT
    s.id,
    ST_SetSRID(ST_MakePoint(m.lon + (random() - 0.5) * 0.12, m.lat + (random() - 0.5) * 0.12), 4326)::geography,
    100 + (random() * 3000),
    s.started_at + (gs || ' minutes')::interval,
    0.5 + (random() * 4),
    s.started_at + (gs || ' minutes')::interval
FROM (SELECT id, started_at FROM track_sessions ORDER BY random() LIMIT 100) s
JOIN generate_series(1, 50) gs ON TRUE
JOIN LATERAL (
    SELECT * FROM (VALUES
        ('Semeru', 112.922, -8.108),
        ('Rinjani', 116.457, -8.411),
        ('Kerinci', 101.264, -1.697),
        ('Merbabu', 110.440, -7.454),
        ('Gede', 106.984, -6.783),
        ('Pangrango', 106.993, -6.772),
        ('Sindoro', 109.992, -7.300),
        ('Sumbing', 110.071, -7.384)
    ) AS t(name, lon, lat)
    ORDER BY random()
    LIMIT 1
) AS m ON TRUE;

-- Seed GPX routes (synthetic lines near mountain clusters)
INSERT INTO gpx_routes (id, trip_id, name, description, total_distance_m, total_elevation_gain_m, route, uploaded_by, created_at)
SELECT
    gen_random_uuid(),
    (SELECT id FROM trips ORDER BY random() LIMIT 1),
    format('Route %s', gs),
    'Auto-generated GPX route',
    1000 + (random() * 20000),
    100 + (random() * 2000),
    ST_MakeLine(ARRAY[
        ST_SetSRID(ST_MakePoint(m.lon - 0.03, m.lat - 0.02), 4326),
        ST_SetSRID(ST_MakePoint(m.lon, m.lat), 4326),
        ST_SetSRID(ST_MakePoint(m.lon + 0.03, m.lat + 0.02), 4326)
    ])::geography,
    (SELECT id FROM users ORDER BY random() LIMIT 1),
    NOW() - ((random() * 20)::int || ' days')::interval
FROM generate_series(1, 150) gs
JOIN LATERAL (
    SELECT * FROM (VALUES
        ('Semeru', 112.922, -8.108),
        ('Rinjani', 116.457, -8.411),
        ('Kerinci', 101.264, -1.697),
        ('Merbabu', 110.440, -7.454),
        ('Gede', 106.984, -6.783),
        ('Pangrango', 106.993, -6.772),
        ('Sindoro', 109.992, -7.300),
        ('Sumbing', 110.071, -7.384)
    ) AS t(name, lon, lat)
    ORDER BY random()
    LIMIT 1
) AS m ON TRUE;
