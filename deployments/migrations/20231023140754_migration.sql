-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS metrics (
	name CHARACTER VARYING(256),
	kind SMALLINT,
	counter BIGINT,
	gauge DOUBLE PRECISION,
	PRIMARY KEY(name, kind)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS metrics;
-- +goose StatementEnd
