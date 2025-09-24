-- +goose Up
CREATE TABLE chirps (
	id UUID PRIMARY KEY,
	created_at TIMESTAMP NOT NULL,
	updated_at TIMESTAMP NOT NULL,
	body TEXT NOT NULL,
	user_id UUID REFERENCES users NOT NULL
);

-- +goose Down
DROP TABLE chirps;
