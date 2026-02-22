COMMENT ON SCHEMA public IS 'standard public schema';

REVOKE USAGE ON TYPE users FROM PUBLIC;

REVOKE USAGE ON TYPE tickets FROM PUBLIC;

REVOKE USAGE ON TYPE money_amount FROM PUBLIC;

REVOKE USAGE ON TYPE invoices FROM PUBLIC;

REVOKE USAGE ON TYPE audit_log FROM PUBLIC;

REVOKE USAGE ON TYPE address FROM PUBLIC;

DROP INDEX idx_users_status;

DROP INDEX idx_users_role;

DROP INDEX idx_tickets_user_id;

DROP INDEX idx_tickets_status;

DROP INDEX idx_tickets_priority;

DROP INDEX idx_tickets_assignee_id;

DROP INDEX idx_invoices_user_id;

DROP INDEX idx_invoices_status;

DROP INDEX idx_audit_log_user_id;

DROP INDEX idx_audit_log_resource;

DROP INDEX idx_audit_log_created_at;

DROP TABLE invoices;

DROP TABLE audit_log;

DROP TABLE tickets;

DROP TABLE users;

DROP SEQUENCE users_id_seq;

DROP SEQUENCE tickets_id_seq;

DROP SEQUENCE invoices_id_seq;

DROP SEQUENCE audit_log_id_seq;

DROP TYPE money_amount;

DROP TYPE address;

DROP TYPE user_role;

DROP TYPE priority_level;

DROP TYPE account_status;

