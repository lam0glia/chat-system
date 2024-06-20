CREATE KEYSPACE IF NOT EXISTS chat WITH REPLICATION = {'class': 'SimpleStrategy', 'replication_factor': 1};

USE chat;

CREATE TABLE messages (
    id bigint,
    content text,
    created_at TIMESTAMP,
    from_id bigint,
    to_id bigint,
    pair varchar,
    PRIMARY KEY ((pair), id)
) WITH CLUSTERING ORDER BY (id DESC);