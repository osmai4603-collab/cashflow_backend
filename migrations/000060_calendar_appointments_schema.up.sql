CREATE TABLE IF NOT EXISTS calendar_recurrences (
    id BIGSERIAL PRIMARY KEY,
    rrule VARCHAR(256) NOT NULL,
    count INT,
    until TIMESTAMPTZ,
    company_id BIGINT NOT NULL REFERENCES res_companies(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS calendar_events (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(256) NOT NULL,
    description TEXT,
    start_date TIMESTAMPTZ NOT NULL,
    stop_date TIMESTAMPTZ NOT NULL,
    duration NUMERIC(6,2) NOT NULL DEFAULT 1.0,
    allday BOOLEAN NOT NULL DEFAULT FALSE,
    location VARCHAR(256),
    video_url VARCHAR(512),
    privacy VARCHAR(32) NOT NULL DEFAULT 'public',
    show_as VARCHAR(16) NOT NULL DEFAULT 'busy',
    user_id BIGINT NOT NULL REFERENCES res_users(id),
    res_model VARCHAR(64),
    res_id BIGINT,
    recurrence_id BIGINT REFERENCES calendar_recurrences(id) ON DELETE SET NULL,
    company_id BIGINT NOT NULL REFERENCES res_companies(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (stop_date > start_date)
);
CREATE INDEX IF NOT EXISTS idx_calendar_events_dates ON calendar_events(start_date, stop_date);
CREATE INDEX IF NOT EXISTS idx_calendar_events_user ON calendar_events(user_id);
CREATE INDEX IF NOT EXISTS idx_calendar_events_ref ON calendar_events(res_model, res_id);

CREATE TABLE IF NOT EXISTS calendar_attendees (
    id BIGSERIAL PRIMARY KEY,
    event_id BIGINT NOT NULL REFERENCES calendar_events(id) ON DELETE CASCADE,
    partner_id BIGINT REFERENCES res_partners(id),
    email VARCHAR(128) NOT NULL,
    name VARCHAR(128) NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'needs_action',
    is_owner BOOLEAN NOT NULL DEFAULT FALSE,
    token VARCHAR(64) NOT NULL UNIQUE
);

CREATE TABLE IF NOT EXISTS calendar_event_alarms (
    id BIGSERIAL PRIMARY KEY,
    event_id BIGINT NOT NULL REFERENCES calendar_events(id) ON DELETE CASCADE,
    alarm_type VARCHAR(32) NOT NULL DEFAULT 'notification',
    duration_minutes INT NOT NULL DEFAULT 15,
    message TEXT
);

CREATE TABLE IF NOT EXISTS appointment_types (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(128) NOT NULL,
    slug VARCHAR(64) NOT NULL UNIQUE,
    duration_minutes INT NOT NULL DEFAULT 30,
    min_schedule_hours INT NOT NULL DEFAULT 2,
    max_schedule_days INT NOT NULL DEFAULT 30,
    assignation_method VARCHAR(32) NOT NULL DEFAULT 'round_robin',
    location VARCHAR(256),
    active BOOLEAN NOT NULL DEFAULT TRUE,
    company_id BIGINT NOT NULL REFERENCES res_companies(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS appointment_type_users (
    appointment_type_id BIGINT NOT NULL REFERENCES appointment_types(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES res_users(id) ON DELETE CASCADE,
    PRIMARY KEY (appointment_type_id, user_id)
);

CREATE TABLE IF NOT EXISTS appointment_slots (
    id BIGSERIAL PRIMARY KEY,
    appointment_type_id BIGINT NOT NULL REFERENCES appointment_types(id) ON DELETE CASCADE,
    day_of_week SMALLINT NOT NULL CHECK (day_of_week BETWEEN 0 AND 6),
    hour_from NUMERIC(4,2) NOT NULL,
    hour_to NUMERIC(4,2) NOT NULL,
    CHECK (hour_to > hour_from)
);

CREATE TABLE IF NOT EXISTS appointment_bookings (
    id BIGSERIAL PRIMARY KEY,
    appointment_type_id BIGINT NOT NULL REFERENCES appointment_types(id),
    event_id BIGINT NOT NULL REFERENCES calendar_events(id),
    staff_id BIGINT NOT NULL REFERENCES res_users(id),
    customer_name VARCHAR(128) NOT NULL,
    customer_email VARCHAR(128) NOT NULL,
    customer_phone VARCHAR(64),
    start_time TIMESTAMPTZ NOT NULL,
    end_time TIMESTAMPTZ NOT NULL,
    notes TEXT,
    status VARCHAR(32) NOT NULL DEFAULT 'confirmed',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (end_time > start_time)
);
CREATE INDEX IF NOT EXISTS idx_appointment_bookings_staff_time ON appointment_bookings(staff_id, start_time, end_time);