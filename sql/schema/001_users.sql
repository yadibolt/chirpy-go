-- +goose Up
CREATE TABLE users (
	id INT PRIMARY KEY,
	created_at TIMESTAMP NOT NULL,
	updated_at TIMESTAMP NOT NULL,
	email TEXT,
	UNIQUE (email)
);

-- +goose Down
DROP TABLE users;
