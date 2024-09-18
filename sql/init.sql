CREATE TYPE UNITS AS ENUM ('Поддержка', 'IT', 'Billing');

CREATE TABLE tickets(
    id SERIAL PRIMARY KEY NOT NULL,
    chat_id BIGINT NOT NULL,
    unit UNITS NOT NULL,
    name TEXT NOT NULL,
    description TEXT NOT NULL
);

CREATE INDEX tickets_chat_id_idx ON tickets(chat_id);

CREATE TABLE admins(
    id SERIAL PRIMARY KEY NOT NULL,
    chat_id BIGINT NOT NULL,
    unit UNITS NOT NULL,
    UNIQUE(chat_id, unit)
);