-- +goose Up
CREATE TABLE feeds (
    id SERIAL PRIMARY KEY,         
    name TEXT NOT NULL,   
    url TEXT NOT NULL UNIQUE,       
    user_id UUID NOT NULL,      
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP, 
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- +goose Down
DROP TABLE feeds;