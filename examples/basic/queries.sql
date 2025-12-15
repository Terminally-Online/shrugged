-- name: GetUsers :rows
SELECT id, email, name, bio, created_at, updated_at
FROM users
WHERE (id = @id OR @id IS NULL)
  AND (email = @email OR @email IS NULL)
ORDER BY created_at DESC;

-- name: CreateUser :row
INSERT INTO users (email, name, bio)
VALUES (@email, @name, @bio)
RETURNING id, email, name, bio, created_at, updated_at;

-- name: UpdateUser :exec
UPDATE users
SET name = COALESCE(@name, name),
    bio = COALESCE(@bio, bio),
    updated_at = NOW()
WHERE id = @id;

-- name: DeleteUser :execrows
DELETE FROM users WHERE id = @id;

-- name: GetPosts :rows
SELECT id, user_id, title, slug, content, published, published_at, created_at, updated_at
FROM posts
WHERE (id = @id OR @id IS NULL)
  AND (user_id = @user_id OR @user_id IS NULL)
  AND (published = @published OR @published IS NULL)
ORDER BY created_at DESC;

-- name: CreatePost :row
INSERT INTO posts (user_id, title, slug, content)
VALUES (@user_id, @title, @slug, @content)
RETURNING id, user_id, title, slug, content, published, published_at, created_at, updated_at;

-- name: PublishPost :exec
UPDATE posts
SET published = true, published_at = NOW(), updated_at = NOW()
WHERE id = @id;

-- name: GetUserWithPosts :row
SELECT
    u.id,
    u.email,
    u.name,
    (SELECT json_agg(p.*) FROM posts p WHERE p.user_id = u.id) as posts
FROM users u
WHERE u.id = @id;

-- name: GetPostWithComments :row
SELECT
    p.id,
    p.title,
    p.content,
    (SELECT json_agg(c.*) FROM comments c WHERE c.post_id = p.id) as comments
FROM posts p
WHERE p.id = @id;
