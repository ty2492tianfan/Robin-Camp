CREATE TABLE IF NOT EXISTS movies (
    id  BIGSERIAL PRIMARY KEY,
    title TEXT NOT NULL UNIQUE,
    genre TEXT NOT NULL,
    distributor TEXT ,
    release_date  DATE NOT NULL,
    budget BIGINT,
    mpa_rating TEXT,
    box_office  JSONB
);


CREATE TABLE IF NOT EXISTS ratings (
    id  BIGSERIAL PRIMARY KEY,
    movie_id  BIGINT NOT NULL REFERENCES movies(id) ON DELETE CASCADE,
    rater_id  TEXT NOT NULL,
    rating  NUMERIC(2,1) NOT NULL CHECK (rating >= 0.5 AND rating <= 5.0), 
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (rating IN (0.5,1.0,1.5,2.0,2.5,3.0,3.5,4.0,4.5,5.0)),
    UNIQUE(movie_id, rater_id)
);