CREATE TABLE IF NOT EXISTS todos (
    id bigserial PRIMARY KEY,
    created_at timestamp(0) with time zone NOT NULL DEFAULT NOW(),
    title text NOT NULL,
    description text NOT NULL,
    due_date timestamp(0) with time zone NOT NULL,
    is_completed bool NOT NULL
);



