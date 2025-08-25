-- Complete production database schema for Nugs Collection API
-- Users table
CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT UNIQUE NOT NULL,
    email TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    role TEXT NOT NULL DEFAULT 'user' CHECK (role IN ('admin', 'user')),
    active BOOLEAN DEFAULT true,
    last_login TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Artists table  
CREATE TABLE IF NOT EXISTS artists (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT UNIQUE NOT NULL,
    slug TEXT UNIQUE NOT NULL,
    show_count INTEGER DEFAULT 0,
    is_active BOOLEAN DEFAULT true,
    last_updated TIMESTAMP,
    genres TEXT, -- JSON array
    description TEXT,
    image_url TEXT,
    nugs_artist_id INTEGER UNIQUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Shows table  
CREATE TABLE IF NOT EXISTS shows (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    artist_id INTEGER NOT NULL,
    date DATE NOT NULL,
    venue TEXT NOT NULL,
    city TEXT,
    state TEXT,
    country TEXT DEFAULT 'USA',
    container_id INTEGER UNIQUE,
    duration_minutes INTEGER,
    set_list TEXT, -- JSON array
    notes TEXT,
    download_count INTEGER DEFAULT 0,
    rating REAL,
    is_available BOOLEAN DEFAULT true,
    formats TEXT, -- JSON object
    tags TEXT, -- JSON array
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (artist_id) REFERENCES artists(id) ON DELETE CASCADE
);

-- Downloads table
CREATE TABLE IF NOT EXISTS downloads (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    show_id INTEGER,
    container_id INTEGER NOT NULL,
    artist_name TEXT NOT NULL,
    show_date DATE NOT NULL,
    venue TEXT NOT NULL,
    format TEXT NOT NULL CHECK (format IN ('FLAC', 'MP3', 'ALAC')),
    quality TEXT NOT NULL,
    size_mb REAL,
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'queued', 'downloading', 'completed', 'failed', 'cancelled')),
    progress INTEGER DEFAULT 0 CHECK (progress >= 0 AND progress <= 100),
    download_path TEXT,
    error_message TEXT,
    queue_position INTEGER,
    retry_count INTEGER DEFAULT 0,
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (show_id) REFERENCES shows(id) ON DELETE SET NULL
);

-- Monitors table (for artist monitoring)
CREATE TABLE IF NOT EXISTS monitors (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    artist_id INTEGER NOT NULL,
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'paused', 'disabled', 'error')),
    settings TEXT NOT NULL, -- JSON object
    last_check TIMESTAMP,
    next_check TIMESTAMP,
    shows_found INTEGER DEFAULT 0,
    alerts_sent INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (artist_id) REFERENCES artists(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    UNIQUE(user_id, artist_id)
);

-- Monitor alerts table
CREATE TABLE IF NOT EXISTS monitor_alerts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    monitor_id INTEGER NOT NULL,
    artist_id INTEGER NOT NULL,
    type TEXT NOT NULL CHECK (type IN ('new_show', 'show_update', 'show_available')),
    title TEXT NOT NULL,
    message TEXT NOT NULL,
    data TEXT, -- JSON object with alert details
    severity TEXT NOT NULL DEFAULT 'info' CHECK (severity IN ('info', 'warning', 'high', 'critical')),
    acknowledged BOOLEAN DEFAULT false,
    acknowledged_at TIMESTAMP,
    acknowledged_by INTEGER,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (monitor_id) REFERENCES monitors(id) ON DELETE CASCADE,
    FOREIGN KEY (artist_id) REFERENCES artists(id) ON DELETE CASCADE,
    FOREIGN KEY (acknowledged_by) REFERENCES users(id) ON DELETE SET NULL
);

-- Webhooks table
CREATE TABLE IF NOT EXISTS webhooks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    url TEXT NOT NULL,
    events TEXT NOT NULL, -- JSON array of subscribed events
    secret TEXT,
    headers TEXT, -- JSON object of custom headers
    active BOOLEAN DEFAULT true,
    retry_count INTEGER DEFAULT 3,
    timeout_seconds INTEGER DEFAULT 30,
    last_triggered TIMESTAMP,
    total_deliveries INTEGER DEFAULT 0,
    successful_deliveries INTEGER DEFAULT 0,
    failed_deliveries INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Webhook deliveries table
CREATE TABLE IF NOT EXISTS webhook_deliveries (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    webhook_id INTEGER NOT NULL,
    event_type TEXT NOT NULL,
    payload TEXT NOT NULL, -- JSON object
    response_status INTEGER,
    response_body TEXT,
    response_time_ms INTEGER,
    error_message TEXT,
    attempts INTEGER DEFAULT 1,
    delivered_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (webhook_id) REFERENCES webhooks(id) ON DELETE CASCADE
);

-- System config table
CREATE TABLE IF NOT EXISTS system_config (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    key TEXT UNIQUE NOT NULL,
    value TEXT NOT NULL,
    description TEXT,
    data_type TEXT DEFAULT 'string' CHECK (data_type IN ('string', 'integer', 'boolean', 'json')),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Schedules table (for background job scheduling)
CREATE TABLE IF NOT EXISTS schedules (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT UNIQUE NOT NULL,
    description TEXT,
    cron TEXT NOT NULL,
    job_type TEXT NOT NULL,
    config TEXT, -- JSON object with job-specific configuration
    enabled BOOLEAN DEFAULT true,
    last_run TIMESTAMP,
    next_run TIMESTAMP,
    run_count INTEGER DEFAULT 0,
    error_count INTEGER DEFAULT 0,
    last_error TEXT,
    created_by INTEGER,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL
);

-- Schedule executions table
CREATE TABLE IF NOT EXISTS schedule_executions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    schedule_id INTEGER NOT NULL,
    job_id TEXT, -- UUID from job manager
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'running', 'completed', 'failed')),
    started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP,
    duration_seconds INTEGER,
    output TEXT,
    error_message TEXT,
    FOREIGN KEY (schedule_id) REFERENCES schedules(id) ON DELETE CASCADE
);

-- Audit logs table
CREATE TABLE IF NOT EXISTS audit_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER,
    action TEXT NOT NULL,
    resource_type TEXT NOT NULL,
    resource_id TEXT,
    details TEXT, -- JSON object
    ip_address TEXT,
    user_agent TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL
);

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_artists_active ON artists(is_active);
CREATE INDEX IF NOT EXISTS idx_artists_name ON artists(name);
CREATE INDEX IF NOT EXISTS idx_shows_artist ON shows(artist_id);
CREATE INDEX IF NOT EXISTS idx_shows_date ON shows(date);
CREATE INDEX IF NOT EXISTS idx_shows_container ON shows(container_id);
CREATE INDEX IF NOT EXISTS idx_downloads_user ON downloads(user_id);
CREATE INDEX IF NOT EXISTS idx_downloads_status ON downloads(status);
CREATE INDEX IF NOT EXISTS idx_downloads_queue ON downloads(queue_position) WHERE status = 'queued';
CREATE INDEX IF NOT EXISTS idx_monitors_user ON monitors(user_id);
CREATE INDEX IF NOT EXISTS idx_monitors_artist ON monitors(artist_id);
CREATE INDEX IF NOT EXISTS idx_monitors_status ON monitors(status);
CREATE INDEX IF NOT EXISTS idx_alerts_monitor ON monitor_alerts(monitor_id);
CREATE INDEX IF NOT EXISTS idx_alerts_unacked ON monitor_alerts(acknowledged) WHERE acknowledged = false;
CREATE INDEX IF NOT EXISTS idx_webhooks_user ON webhooks(user_id);
CREATE INDEX IF NOT EXISTS idx_webhooks_active ON webhooks(active) WHERE active = true;
CREATE INDEX IF NOT EXISTS idx_deliveries_webhook ON webhook_deliveries(webhook_id);
CREATE INDEX IF NOT EXISTS idx_schedules_enabled ON schedules(enabled) WHERE enabled = true;
CREATE INDEX IF NOT EXISTS idx_executions_schedule ON schedule_executions(schedule_id);
CREATE INDEX IF NOT EXISTS idx_audit_user ON audit_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_audit_created ON audit_logs(created_at);

-- Insert default admin user (password: admin123)
INSERT OR IGNORE INTO users (username, email, password_hash, role, active) 
VALUES ('admin', 'admin@example.com', '240be518fabd2724ddb6f04eeb1da5967448d7e831c08c8fa822809f74c720a9', 'admin', true);

-- Insert default system config
INSERT OR IGNORE INTO system_config (key, value, description, data_type) VALUES 
('max_concurrent_downloads', '5', 'Maximum concurrent downloads', 'integer'),
('default_download_path', '/downloads', 'Default download directory', 'string'),
('auto_retry_failed', 'true', 'Auto retry failed downloads', 'boolean'),
('retry_count', '3', 'Number of retry attempts', 'integer'),
('default_check_frequency', 'daily', 'Default monitor check frequency', 'string'),
('max_monitors_per_user', '100', 'Maximum monitors per user', 'integer'),
('alert_retention_days', '30', 'Alert retention period in days', 'integer'),
('log_retention_days', '30', 'Log retention period in days', 'integer'),
('backup_frequency', 'daily', 'Database backup frequency', 'string'),
('maintenance_window', '02:00-04:00', 'System maintenance window', 'string'),
('rate_limit_enabled', 'true', 'Enable API rate limiting', 'boolean'),
('max_requests_per_hour', '1000', 'Maximum API requests per hour', 'integer'),
('jwt_expiry_hours', '24', 'JWT token expiry in hours', 'integer'),
('webhook_timeout_seconds', '30', 'Webhook delivery timeout', 'integer'),
('auto_refresh_enabled', 'true', 'Auto refresh catalog', 'boolean'),
('refresh_interval_hours', '24', 'Catalog refresh interval', 'integer');

-- Insert sample artists for testing/demo
INSERT OR IGNORE INTO artists (name, slug, show_count, is_active, genres, nugs_artist_id) VALUES 
('Grateful Dead', 'grateful-dead', 2847, false, '["Rock", "Psychedelic", "Jam"]', 83),
('Phish', 'phish', 1800, true, '["Jam", "Rock", "Progressive"]', 26),
('Dead & Company', 'dead-and-company', 150, true, '["Rock", "Jam"]', 205),
('Widespread Panic', 'widespread-panic', 500, true, '["Rock", "Southern Rock", "Jam"]', 45),
('String Cheese Incident', 'string-cheese-incident', 400, true, '["Bluegrass", "Electronic", "Jam"]', 67);

-- Insert sample shows for testing/demo
INSERT OR IGNORE INTO shows (artist_id, date, venue, city, state, container_id, duration_minutes, is_available) VALUES 
(1, '1977-05-08', 'Barton Hall, Cornell University', 'Ithaca', 'NY', 67890, 180, true),
(1, '1989-07-07', 'JFK Stadium', 'Philadelphia', 'PA', 67891, 210, true),
(2, '1997-12-31', 'Madison Square Garden', 'New York', 'NY', 67892, 240, true),
(2, '2023-07-14', 'Merriweather Post Pavilion', 'Columbia', 'MD', 67893, 195, true),
(3, '2023-07-01', 'Wrigley Field', 'Chicago', 'IL', 67894, 225, true);

-- Insert sample schedules for demo
INSERT OR IGNORE INTO schedules (name, description, cron, job_type, config, enabled) VALUES
('Daily Catalog Refresh', 'Refresh artist and show catalog daily at 2 AM', '0 2 * * *', 'catalog_refresh', '{"force": false}', true),
('Hourly Monitor Check', 'Check all active monitors every hour', '0 * * * *', 'monitor_check', '{"check_all": true}', true),
('Weekly Analytics Report', 'Generate weekly analytics report', '0 9 * * 1', 'analytics_report', '{"report_type": "weekly", "format": "json"}', true);