CREATE KEYSPACE IF NOT EXISTS urlshortener
            WITH replication = {'class': 'SimpleStrategy', 'replication_factor': 1};

CREATE TABLE IF NOT EXISTS urlshortener.urls
(
    code       TEXT PRIMARY KEY,
    original   TEXT,
    created_at TIMESTAMP,
    expires_at TIMESTAMP
);

CREATE TABLE IF NOT EXISTS urlshortener.clicks
(
    code       TEXT,
    bucket     TEXT,
    clicked_at TIMESTAMP,
    click_id   UUID,
    country    TEXT,
    device     TEXT,
    referer    TEXT,
    PRIMARY KEY ((code, bucket), clicked_at, click_id)
) WITH CLUSTERING ORDER BY (clicked_at DESC)
   AND default_time_to_live = 7776000;

CREATE TABLE IF NOT EXISTS urlshortener.click_counts
(
    code   TEXT,
    bucket TEXT,
    total  COUNTER,
    PRIMARY KEY (code, bucket)
);