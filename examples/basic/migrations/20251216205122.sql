CREATE SEQUENCE comments_id_seq START 1 INCREMENT 1 MINVALUE 1 MAXVALUE 9223372036854775807 CACHE 1;

CREATE SEQUENCE posts_id_seq START 1 INCREMENT 1 MINVALUE 1 MAXVALUE 9223372036854775807 CACHE 1;

CREATE SEQUENCE users_id_seq START 1 INCREMENT 1 MINVALUE 1 MAXVALUE 9223372036854775807 CACHE 1;

CREATE TABLE users (
    id bigint NOT NULL DEFAULT nextval('users_id_seq'::regclass),
    email character varying NOT NULL,
    name text NOT NULL,
    bio text,
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    updated_at timestamp with time zone,
    CONSTRAINT users_email_key UNIQUE (email),
    CONSTRAINT users_pkey PRIMARY KEY (id)
);

CREATE TABLE posts (
    id bigint NOT NULL DEFAULT nextval('posts_id_seq'::regclass),
    user_id bigint NOT NULL,
    title text NOT NULL,
    slug character varying NOT NULL,
    content text,
    published boolean NOT NULL DEFAULT false,
    published_at timestamp with time zone,
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    updated_at timestamp with time zone,
    CONSTRAINT posts_pkey PRIMARY KEY (id),
    CONSTRAINT posts_slug_key UNIQUE (slug),
    CONSTRAINT posts_user_id_fkey FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);

CREATE TABLE comments (
    id bigint NOT NULL DEFAULT nextval('comments_id_seq'::regclass),
    post_id bigint NOT NULL,
    user_id bigint NOT NULL,
    content text NOT NULL,
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    CONSTRAINT comments_pkey PRIMARY KEY (id),
    CONSTRAINT comments_post_id_fkey FOREIGN KEY (post_id) REFERENCES posts (id) ON DELETE CASCADE,
    CONSTRAINT comments_user_id_fkey FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);

CREATE INDEX idx_comments_post_id ON comments (post_id);

CREATE INDEX idx_comments_user_id ON comments (user_id);

CREATE INDEX idx_posts_published ON posts (published) WHERE (published = true);

CREATE INDEX idx_posts_user_id ON posts (user_id);

GRANT USAGE ON TYPE comments TO PUBLIC;

GRANT USAGE ON TYPE posts TO PUBLIC;

GRANT USAGE ON TYPE users TO PUBLIC;

COMMENT ON SCHEMA public IS NULL;

