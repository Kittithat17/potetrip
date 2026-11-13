-- +goose Up
-- Keeps receipt lines in the order they were entered.
ALTER TABLE expense_items ADD COLUMN position int NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE expense_items DROP COLUMN position;
