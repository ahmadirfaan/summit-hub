CREATE EXTENSION IF NOT EXISTS postgis;

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
