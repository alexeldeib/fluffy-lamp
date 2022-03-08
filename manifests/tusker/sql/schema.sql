CREATE TABLE users (
    id varchar(27) PRIMARY KEY,
    first_name TEXT,
    last_name TEXT,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone
);
