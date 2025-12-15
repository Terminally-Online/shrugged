-- name: GetUsers :rows
SELECT id, email, role, status, display_name, avatar_url, mailing_address,
       preferences, email_verified_at, created_at, updated_at
FROM users
WHERE (id = @id OR @id IS NULL)
  AND (role = @role OR @role IS NULL)
  AND (status = @status OR @status IS NULL)
ORDER BY created_at DESC;

-- name: CreateUser :row
INSERT INTO users (email, role, status, display_name, avatar_url, mailing_address, preferences)
VALUES (@email, @role, @status, @display_name, @avatar_url, @mailing_address, @preferences)
RETURNING id, email, role, status, display_name, avatar_url, mailing_address, preferences, email_verified_at, created_at, updated_at;

-- name: UpdateUserRole :exec
UPDATE users
SET role = @role,
    updated_at = NOW()
WHERE id = @id;

-- name: UpdateUserStatus :exec
UPDATE users
SET status = @status,
    updated_at = NOW()
WHERE id = @id;

-- name: VerifyUserEmail :exec
UPDATE users
SET status = 'active',
    email_verified_at = NOW(),
    updated_at = NOW()
WHERE id = @id;

-- name: GetTickets :rows
SELECT id, user_id, assignee_id, priority, status, title, description,
       tags, due_date, created_at, updated_at, resolved_at
FROM tickets
WHERE (id = @id OR @id IS NULL)
  AND (user_id = @user_id OR @user_id IS NULL)
  AND (assignee_id = @assignee_id OR @assignee_id IS NULL)
  AND (priority = @priority OR @priority IS NULL)
  AND (status = @status OR @status IS NULL)
ORDER BY
    CASE priority
        WHEN 'critical' THEN 1
        WHEN 'high' THEN 2
        WHEN 'medium' THEN 3
        WHEN 'low' THEN 4
    END,
    created_at DESC;

-- name: CreateTicket :row
INSERT INTO tickets (user_id, assignee_id, priority, status, title, description, tags, due_date)
VALUES (@user_id, @assignee_id, @priority, @status, @title, @description, @tags, @due_date)
RETURNING id, user_id, assignee_id, priority, status, title, description, tags, due_date, created_at, updated_at, resolved_at;

-- name: AssignTicket :exec
UPDATE tickets
SET assignee_id = @assignee_id,
    updated_at = NOW()
WHERE id = @id;

-- name: ResolveTicket :exec
UPDATE tickets
SET status = 'deleted',
    resolved_at = NOW(),
    updated_at = NOW()
WHERE id = @id;

-- name: GetInvoices :rows
SELECT id, user_id, amount, status, issued_at, due_at, paid_at
FROM invoices
WHERE (id = @id OR @id IS NULL)
  AND (user_id = @user_id OR @user_id IS NULL)
  AND (status = @status OR @status IS NULL)
ORDER BY due_at ASC;

-- name: CreateInvoice :row
INSERT INTO invoices (user_id, amount, status, due_at)
VALUES (@user_id, @amount, @status, @due_at)
RETURNING id, user_id, amount, status, issued_at, due_at, paid_at;

-- name: MarkInvoicePaid :exec
UPDATE invoices
SET status = 'active',
    paid_at = NOW()
WHERE id = @id;

-- name: GetUserWithTickets :row
SELECT
    u.id,
    u.email,
    u.display_name,
    u.role,
    (SELECT json_agg(t.*) FROM tickets t WHERE t.user_id = u.id) as tickets
FROM users u
WHERE u.id = @id;

-- name: GetAuditLog :rows
SELECT id, user_id, action, resource_type, resource_id, old_values, new_values,
       ip_address, user_agent, created_at
FROM audit_log
WHERE (user_id = @user_id OR @user_id IS NULL)
  AND (resource_type = @resource_type OR @resource_type IS NULL)
  AND (resource_id = @resource_id OR @resource_id IS NULL)
ORDER BY created_at DESC
LIMIT 100;

-- name: CreateAuditLog :row
INSERT INTO audit_log (user_id, action, resource_type, resource_id, old_values, new_values, ip_address, user_agent)
VALUES (@user_id, @action, @resource_type, @resource_id, @old_values, @new_values, @ip_address, @user_agent)
RETURNING id, user_id, action, resource_type, resource_id, old_values, new_values, ip_address, user_agent, created_at;
