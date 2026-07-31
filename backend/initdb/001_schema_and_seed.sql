CREATE TABLE IF NOT EXISTS movies (
    id SERIAL PRIMARY KEY,
    title TEXT NOT NULL,
    poster_url TEXT
);

CREATE TABLE IF NOT EXISTS showtimes (
    id SERIAL PRIMARY KEY,
    movie_id INT NOT NULL REFERENCES movies(id) ON DELETE CASCADE,
    starts_at TIMESTAMPTZ NOT NULL,
    hall VARCHAR(50)
);

CREATE TABLE IF NOT EXISTS seats (
    id SERIAL PRIMARY KEY,
    showtime_id INT NOT NULL REFERENCES showtimes(id) ON DELETE CASCADE,
    label VARCHAR(10) NOT NULL,
    status VARCHAR(10) NOT NULL DEFAULT 'available' CHECK (status IN ('available', 'booked'))
);

CREATE TABLE IF NOT EXISTS bookings (
    id VARCHAR(50) PRIMARY KEY,
    showtime_id INT NOT NULL REFERENCES showtimes(id),
    seat_ids INT[] NOT NULL,
    user_email VARCHAR(255) NOT NULL,
    status VARCHAR(50) DEFAULT 'confirmed',
    created_at TIMESTAMPTZ DEFAULT NOW()
);

DO $$
DECLARE
    v_movie_id INT;
    v_showtime_id INT;
    showtime_record RECORD;
    seat_number INT;
BEGIN
    INSERT INTO movies (title, poster_url)
    SELECT 'Iron Man', 'https://image.tmdb.org/t/p/original/78lPtwv72eTNqFW9COBYI0dWDJa.jpg'
    WHERE NOT EXISTS (
        SELECT 1 FROM movies WHERE title = 'Iron Man'
    )
    RETURNING id INTO v_movie_id;

    IF v_movie_id IS NULL THEN
        SELECT id INTO v_movie_id
        FROM movies
        WHERE title = 'Iron Man'
        ORDER BY id
        LIMIT 1;
    END IF;

    FOR showtime_record IN
        SELECT *
        FROM (VALUES
            ('2026-08-01T10:00:00Z'::timestamptz, 'Hall A'),
            ('2026-08-01T13:00:00Z'::timestamptz, 'Hall B'),
            ('2026-08-01T16:00:00Z'::timestamptz, 'Hall A'),
            ('2026-08-01T19:00:00Z'::timestamptz, 'Hall C'),
            ('2026-08-01T22:00:00Z'::timestamptz, 'Hall B')
        ) AS values_table(starts_at, hall)
    LOOP
        INSERT INTO showtimes (movie_id, starts_at, hall)
        SELECT v_movie_id, showtime_record.starts_at, showtime_record.hall
        WHERE NOT EXISTS (
            SELECT 1
            FROM showtimes existing
            WHERE existing.movie_id = v_movie_id
              AND existing.starts_at = showtime_record.starts_at
              AND existing.hall = showtime_record.hall
        )
        RETURNING id INTO v_showtime_id;

        IF v_showtime_id IS NULL THEN
            SELECT id INTO v_showtime_id
            FROM showtimes existing
            WHERE existing.movie_id = v_movie_id
              AND existing.starts_at = showtime_record.starts_at
              AND existing.hall = showtime_record.hall
            ORDER BY id
            LIMIT 1;
        END IF;

        FOR seat_number IN 1..100 LOOP
            INSERT INTO seats (showtime_id, label)
            SELECT v_showtime_id, 'S' || seat_number
            WHERE NOT EXISTS (
                SELECT 1
                FROM seats existing
                WHERE existing.showtime_id = v_showtime_id
                  AND existing.label = 'S' || seat_number
            );
        END LOOP;
    END LOOP;
END $$;