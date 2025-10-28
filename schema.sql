-- Create database and users table
CREATE DATABASE sftpdb;

-- Connect to the database
\c sftpdb;

-- Create users table
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(50) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    is_active BOOLEAN DEFAULT TRUE
);

-- Create an index on username for faster lookups
CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);

-- Create incoming_files table for /in/ directory files (PostgreSQL storage)
CREATE TABLE IF NOT EXISTS incoming_files (
    id SERIAL PRIMARY KEY,
    username VARCHAR(50) NOT NULL REFERENCES users(username) ON DELETE CASCADE,
    filename VARCHAR(255) NOT NULL,
    file_content TEXT NOT NULL,
    file_size INTEGER NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(username, filename)
);

-- Create indexes for better performance
CREATE INDEX IF NOT EXISTS idx_incoming_files_username ON incoming_files(username);
CREATE INDEX IF NOT EXISTS idx_incoming_files_filename ON incoming_files(username, filename);

-- Note: This schema only creates the table structure
-- Users are managed externally in existing database
