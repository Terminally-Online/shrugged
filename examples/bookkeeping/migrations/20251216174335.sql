CREATE TYPE account_status AS ENUM ('active', 'suspended', 'pending_verification', 'deleted');

CREATE TYPE priority_level AS ENUM ('low', 'medium', 'high', 'critical');

CREATE TYPE user_role AS ENUM ('admin', 'moderator', 'member', 'guest');

CREATE TYPE address AS (street text, city text, state text, postal_code text, country text);

CREATE TYPE money_amount AS (amount numeric(15,2), currency character varying(3));

CREATE SEQUENCE audit_log_id_seq START 1 INCREMENT 1 MINVALUE 1 MAXVALUE 9223372036854775807 CACHE 1;

CREATE SEQUENCE invoices_id_seq START 1 INCREMENT 1 MINVALUE 1 MAXVALUE 9223372036854775807 CACHE 1;

CREATE SEQUENCE tickets_id_seq START 1 INCREMENT 1 MINVALUE 1 MAXVALUE 9223372036854775807 CACHE 1;

CREATE SEQUENCE users_id_seq START 1 INCREMENT 1 MINVALUE 1 MAXVALUE 9223372036854775807 CACHE 1;

CREATE TABLE users (
    id bigint NOT NULL DEFAULT nextval('users_id_seq'::regclass),
    email character varying NOT NULL,
    role user_role NOT NULL DEFAULT 'member'::user_role,
    status account_status NOT NULL DEFAULT 'pending_verification'::account_status,
    display_name text NOT NULL,
    avatar_url text,
    mailing_address address,
    preferences jsonb NOT NULL DEFAULT '{}'::jsonb,
    email_verified_at timestamp with time zone,
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    updated_at timestamp with time zone,
    CONSTRAINT users_email_key UNIQUE (email),
    CONSTRAINT users_pkey PRIMARY KEY (id)
);

CREATE TABLE audit_log (
    id bigint NOT NULL DEFAULT nextval('audit_log_id_seq'::regclass),
    user_id bigint,
    action text NOT NULL,
    resource_type text NOT NULL,
    resource_id bigint,
    old_values jsonb,
    new_values jsonb,
    ip_address inet,
    user_agent text,
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    CONSTRAINT audit_log_pkey PRIMARY KEY (id),
    CONSTRAINT audit_log_user_id_fkey FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE SET NULL
);

CREATE TABLE invoices (
    id bigint NOT NULL DEFAULT nextval('invoices_id_seq'::regclass),
    user_id bigint NOT NULL,
    amount money_amount NOT NULL,
    status account_status NOT NULL DEFAULT 'pending_verification'::account_status,
    issued_at timestamp with time zone NOT NULL DEFAULT now(),
    due_at timestamp with time zone NOT NULL,
    paid_at timestamp with time zone,
    CONSTRAINT invoices_pkey PRIMARY KEY (id),
    CONSTRAINT invoices_user_id_fkey FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);

CREATE TABLE tickets (
    id bigint NOT NULL DEFAULT nextval('tickets_id_seq'::regclass),
    user_id bigint NOT NULL,
    assignee_id bigint,
    priority priority_level NOT NULL DEFAULT 'medium'::priority_level,
    status account_status NOT NULL DEFAULT 'active'::account_status,
    title text NOT NULL,
    description text,
    tags text[],
    due_date date,
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    updated_at timestamp with time zone,
    resolved_at timestamp with time zone,
    CONSTRAINT tickets_assignee_id_fkey FOREIGN KEY (assignee_id) REFERENCES users (id) ON DELETE SET NULL,
    CONSTRAINT tickets_pkey PRIMARY KEY (id),
    CONSTRAINT tickets_user_id_fkey FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);

CREATE INDEX idx_audit_log_created_at ON audit_log (created_at);

CREATE INDEX idx_audit_log_resource ON audit_log (resource_type, resource_id);

CREATE INDEX idx_audit_log_user_id ON audit_log (user_id);

CREATE INDEX idx_invoices_status ON invoices (status);

CREATE INDEX idx_invoices_user_id ON invoices (user_id);

CREATE INDEX idx_tickets_assignee_id ON tickets (assignee_id);

CREATE INDEX idx_tickets_priority ON tickets (priority);

CREATE INDEX idx_tickets_status ON tickets (status);

CREATE INDEX idx_tickets_user_id ON tickets (user_id);

CREATE INDEX idx_users_role ON users (role);

CREATE INDEX idx_users_status ON users (status);

GRANT USAGE ON TYPE address TO PUBLIC;

GRANT USAGE ON TYPE audit_log TO PUBLIC;

GRANT USAGE ON TYPE invoices TO PUBLIC;

GRANT USAGE ON TYPE money_amount TO PUBLIC;

GRANT USAGE ON TYPE tickets TO PUBLIC;

GRANT USAGE ON TYPE users TO PUBLIC;

COMMENT ON SCHEMA public IS NULL;

