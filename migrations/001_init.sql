CREATE TABLE IF NOT EXISTS companies (
    id UUID PRIMARY KEY,
    name VARCHAR(15) NOT NULL UNIQUE,
    description VARCHAR(3000),
    employees INT NOT NULL,
    registered BOOLEAN NOT NULL,
    type VARCHAR(50) NOT NULL
);