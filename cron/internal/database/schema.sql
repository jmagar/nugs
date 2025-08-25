-- Users table
CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT UNIQUE NOT NULL,
    email TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    role TEXT NOT NULL DEFAULT 'user',
    active BOOLEAN DEFAULT true,
    last_login TIMESTAMP,
    login_count INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Artists table
CREATE TABLE IF NOT EXISTS artists (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    slug TEXT UNIQUE,
    show_count INTEGER DEFAULT 0,
    first_show_date DATE,
    last_show_date DATE,
    is_active BOOLEAN DEFAULT true,
    genres TEXT, -- JSON array
    bio TEXT,
    image_url TEXT,
    social_links TEXT, -- JSON object
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
    FOREIGN KEY (artist_id) REFERENCES artists(id)
);

-- Downloads table
CREATE TABLE IF NOT EXISTS downloads (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    show_id INTEGER NOT NULL,
    container_id INTEGER NOT NULL,
    artist_name TEXT NOT NULL,
    show_date DATE NOT NULL,
    venue TEXT NOT NULL,
    format TEXT NOT NULL CHECK (format IN ('MP3', 'FLAC', 'ALAC')),
    quality TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'in_progress', 'completed', 'failed', 'cancelled')),
    progress INTEGER DEFAULT 0,
    file_path TEXT,
    file_size INTEGER DEFAULT 0,
    error_message TEXT,
    downloaded_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (show_id) REFERENCES shows(id)
);

-- Artist monitors table
CREATE TABLE IF NOT EXISTS artist_monitors (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    artist_id INTEGER NOT NULL,
    user_id INTEGER NOT NULL,
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'paused', 'disabled', 'error')),
    settings TEXT NOT NULL, -- JSON object
    last_check TIMESTAMP,
    next_check TIMESTAMP,
    shows_found INTEGER DEFAULT 0,
    alerts_sent INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (artist_id) REFERENCES artists(id),
    FOREIGN KEY (user_id) REFERENCES users(id)
);

-- Monitor alerts table
CREATE TABLE IF NOT EXISTS monitor_alerts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    monitor_id INTEGER NOT NULL,
    artist_id INTEGER NOT NULL,
    type TEXT NOT NULL,
    title TEXT NOT NULL,
    message TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'unread' CHECK (status IN ('unread', 'read', 'acknowledged')),
    data TEXT, -- JSON object
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    acknowledged_at TIMESTAMP,
    FOREIGN KEY (monitor_id) REFERENCES artist_monitors(id),
    FOREIGN KEY (artist_id) REFERENCES artists(id)
);

-- Webhooks table
CREATE TABLE IF NOT EXISTS webhooks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    url TEXT NOT NULL,
    events TEXT NOT NULL, -- JSON array
    secret TEXT,
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'disabled', 'failed')),
    timeout INTEGER DEFAULT 30,
    retry_count INTEGER DEFAULT 3,
    description TEXT,
    total_deliveries INTEGER DEFAULT 0,
    successful_deliveries INTEGER DEFAULT 0,
    failed_deliveries INTEGER DEFAULT 0,
    last_delivery TIMESTAMP,
    last_success TIMESTAMP,
    last_failure TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Webhook deliveries table
CREATE TABLE IF NOT EXISTS webhook_deliveries (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    webhook_id INTEGER NOT NULL,
    event TEXT NOT NULL,
    url TEXT NOT NULL,
    payload TEXT NOT NULL,
    headers TEXT,
    status_code INTEGER,
    response TEXT,
    error TEXT,
    duration_ms INTEGER,
    attempt INTEGER DEFAULT 1,
    success BOOLEAN,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (webhook_id) REFERENCES webhooks(id)
);

-- Schedules table
CREATE TABLE IF NOT EXISTS schedules (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    description TEXT,
    type TEXT NOT NULL,
    cron_expr TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'paused', 'disabled', 'error')),
    parameters TEXT, -- JSON object
    next_run TIMESTAMP,
    last_run TIMESTAMP,
    last_job_id TEXT,
    last_status TEXT,
    run_count INTEGER DEFAULT 0,
    fail_count INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_by TEXT NOT NULL
);

-- Schedule executions table
CREATE TABLE IF NOT EXISTS schedule_executions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    schedule_id INTEGER NOT NULL,
    job_id TEXT NOT NULL,
    status TEXT NOT NULL,
    started_at TIMESTAMP NOT NULL,
    completed_at TIMESTAMP,
    duration_ms INTEGER,
    error TEXT,
    result TEXT, -- JSON object
    FOREIGN KEY (schedule_id) REFERENCES schedules(id)
);

-- System config table
CREATE TABLE IF NOT EXISTS system_config (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    description TEXT,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Audit logs table
CREATE TABLE IF NOT EXISTS audit_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER,
    username TEXT,
    action TEXT NOT NULL,
    resource TEXT,
    resource_id INTEGER,
    details TEXT, -- JSON object
    ip_address TEXT,
    user_agent TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id)
);

-- Indexes for better performance
CREATE INDEX IF NOT EXISTS idx_shows_artist_id ON shows(artist_id);
CREATE INDEX IF NOT EXISTS idx_shows_date ON shows(date);
CREATE INDEX IF NOT EXISTS idx_shows_container_id ON shows(container_id);
CREATE INDEX IF NOT EXISTS idx_downloads_status ON downloads(status);
CREATE INDEX IF NOT EXISTS idx_downloads_created_at ON downloads(created_at);
CREATE INDEX IF NOT EXISTS idx_monitors_artist_id ON artist_monitors(artist_id);
CREATE INDEX IF NOT EXISTS idx_monitors_status ON artist_monitors(status);
CREATE INDEX IF NOT EXISTS idx_alerts_monitor_id ON monitor_alerts(monitor_id);
CREATE INDEX IF NOT EXISTS idx_alerts_status ON monitor_alerts(status);
CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_webhook_id ON webhook_deliveries(webhook_id);
CREATE INDEX IF NOT EXISTS idx_schedule_executions_schedule_id ON schedule_executions(schedule_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_user_id ON audit_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_action ON audit_logs(action);

-- Insert default admin user
INSERT OR IGNORE INTO users (username, email, password_hash, role, active) 
VALUES ('admin', 'admin@example.com', '240be518fabd2724ddb6f04eeb1da5967448d7e831c08c8fa822809f74c720a9', 'admin', true);

-- Insert default system config
INSERT OR IGNORE INTO system_config (key, value, description) VALUES 
('max_concurrent_downloads', '5', 'Maximum concurrent downloads'),
('default_download_path', '/downloads', 'Default download directory'),
('auto_retry_failed', 'true', 'Auto retry failed downloads'),
('retry_count', '3', 'Number of retry attempts'),
('default_check_frequency', 'daily', 'Default monitor check frequency'),
('max_monitors_per_user', '100', 'Maximum monitors per user'),
('alert_retention_days', '30', 'Alert retention period in days'),
('log_retention_days', '30', 'Log retention period in days'),
('backup_frequency', 'daily', 'Database backup frequency'),
('maintenance_window', '02:00-04:00', 'System maintenance window'),
('rate_limit_enabled', 'true', 'Enable API rate limiting'),
('max_requests_per_hour', '1000', 'Maximum API requests per hour'),
('jwt_expiry_hours', '24', 'JWT token expiry in hours');

-- Insert sample artists for testing
INSERT OR IGNORE INTO artists (name, slug, show_count, is_active, genres) VALUES 
('Grateful Dead', 'grateful-dead', 2847, false, '["Rock", "Psychedelic"]'),
('Phish', 'phish', 1800, true, '["Jam", "Rock"]'),
('Dead & Company', 'dead-and-company', 150, true, '["Rock", "Jam"]');

-- Insert sample shows for testing
INSERT OR IGNORE INTO shows (artist_id, date, venue, city, state, container_id, duration_minutes, is_available) VALUES 
(1, '1977-05-08', 'Barton Hall, Cornell University', 'Ithaca', 'NY', 67890, 180, true),
(1, '1989-07-07', 'JFK Stadium', 'Philadelphia', 'PA', 67891, 210, true),
(2, '1997-12-31', 'Madison Square Garden', 'New York', 'NY', 67892, 240, true);