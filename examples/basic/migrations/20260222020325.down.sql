COMMENT ON SCHEMA public IS 'standard public schema';

REVOKE USAGE ON TYPE users FROM PUBLIC;

REVOKE USAGE ON TYPE posts FROM PUBLIC;

REVOKE USAGE ON TYPE comments FROM PUBLIC;

DROP INDEX idx_posts_user_id;

DROP INDEX idx_posts_published;

DROP INDEX idx_comments_user_id;

DROP INDEX idx_comments_post_id;

DROP TABLE comments;

DROP TABLE posts;

DROP TABLE users;

DROP SEQUENCE users_id_seq;

DROP SEQUENCE posts_id_seq;

DROP SEQUENCE comments_id_seq;

