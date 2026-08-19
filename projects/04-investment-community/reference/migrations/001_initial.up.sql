CREATE TABLE IF NOT EXISTS users (
    id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    email VARCHAR(254) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_bin NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    display_name VARCHAR(80) NOT NULL,
    role ENUM('user', 'admin') NOT NULL DEFAULT 'user',
    status ENUM('active', 'disabled') NOT NULL DEFAULT 'active',
    created_at DATETIME(6) NOT NULL DEFAULT (UTC_TIMESTAMP(6)),
    updated_at DATETIME(6) NOT NULL DEFAULT (UTC_TIMESTAMP(6)),
    CONSTRAINT chk_users_schema_v1 CHECK (
        CHAR_LENGTH(email) BETWEEN 3 AND 254 AND CHAR_LENGTH(display_name) BETWEEN 1 AND 80
    ),
    UNIQUE KEY uq_users_email (email)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS circles (
    id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    slug VARCHAR(64) NOT NULL,
    name VARCHAR(80) NOT NULL,
    description VARCHAR(500) NOT NULL DEFAULT '',
    status ENUM('active', 'archived') NOT NULL DEFAULT 'active',
    created_at DATETIME(6) NOT NULL DEFAULT (UTC_TIMESTAMP(6)),
    updated_at DATETIME(6) NOT NULL DEFAULT (UTC_TIMESTAMP(6)),
    CONSTRAINT chk_circles_schema_v1 CHECK (CHAR_LENGTH(slug) >= 1 AND CHAR_LENGTH(name) >= 1),
    UNIQUE KEY uq_circles_slug (slug),
    UNIQUE KEY uq_circles_name (name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS circle_memberships (
    circle_id BIGINT NOT NULL,
    user_id BIGINT NOT NULL,
    joined_at DATETIME(6) NOT NULL DEFAULT (UTC_TIMESTAMP(6)),
    CONSTRAINT chk_circle_memberships_schema_v1 CHECK (circle_id > 0 AND user_id > 0),
    PRIMARY KEY (circle_id, user_id),
    KEY idx_circle_memberships_user (user_id, joined_at),
    CONSTRAINT fk_circle_memberships_circle FOREIGN KEY (circle_id) REFERENCES circles (id),
    CONSTRAINT fk_circle_memberships_user FOREIGN KEY (user_id) REFERENCES users (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS securities (
    id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    market VARCHAR(16) NOT NULL,
    code VARCHAR(16) NOT NULL,
    name VARCHAR(80) NOT NULL,
    status ENUM('active', 'inactive') NOT NULL DEFAULT 'active',
    created_at DATETIME(6) NOT NULL DEFAULT (UTC_TIMESTAMP(6)),
    CONSTRAINT chk_securities_schema_v1 CHECK (CHAR_LENGTH(market) >= 1 AND CHAR_LENGTH(code) >= 1),
    UNIQUE KEY uq_securities_market_code (market, code),
    KEY idx_securities_name (name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS posts (
    id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    circle_id BIGINT NOT NULL,
    author_id BIGINT NOT NULL,
    title VARCHAR(120) NOT NULL,
    body TEXT NOT NULL,
    visibility ENUM('visible', 'hidden') NOT NULL DEFAULT 'visible',
    moderation_version BIGINT NOT NULL DEFAULT 1,
    version BIGINT NOT NULL DEFAULT 1,
    idempotency_key VARCHAR(128) CHARACTER SET ascii COLLATE ascii_bin NULL,
    request_hash CHAR(64) CHARACTER SET ascii COLLATE ascii_bin NULL,
    created_at DATETIME(6) NOT NULL DEFAULT (UTC_TIMESTAMP(6)),
    updated_at DATETIME(6) NOT NULL DEFAULT (UTC_TIMESTAMP(6)),
    deleted_at DATETIME(6) NULL,
    CONSTRAINT chk_posts_schema_v1 CHECK (
        circle_id > 0 AND author_id > 0 AND CHAR_LENGTH(title) >= 1 AND CHAR_LENGTH(body) >= 1
    ),
    CONSTRAINT chk_posts_versions CHECK (version >= 1 AND moderation_version >= 1),
    CONSTRAINT chk_posts_idempotency_pair CHECK (
        (idempotency_key IS NULL AND request_hash IS NULL) OR
        (idempotency_key IS NOT NULL AND request_hash IS NOT NULL)
    ),
    UNIQUE KEY uq_posts_author_idempotency (author_id, idempotency_key),
    KEY idx_posts_feed (circle_id, visibility, deleted_at, created_at DESC, id DESC),
    KEY idx_posts_global_feed (visibility, deleted_at, created_at DESC, id DESC),
    CONSTRAINT fk_posts_circle FOREIGN KEY (circle_id) REFERENCES circles (id),
    CONSTRAINT fk_posts_author FOREIGN KEY (author_id) REFERENCES users (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS post_securities (
    post_id BIGINT NOT NULL,
    security_id BIGINT NOT NULL,
    created_at DATETIME(6) NOT NULL DEFAULT (UTC_TIMESTAMP(6)),
    CONSTRAINT chk_post_securities_schema_v1 CHECK (post_id > 0 AND security_id > 0),
    PRIMARY KEY (post_id, security_id),
    KEY idx_post_securities_security (security_id, post_id),
    CONSTRAINT fk_post_securities_post FOREIGN KEY (post_id) REFERENCES posts (id),
    CONSTRAINT fk_post_securities_security FOREIGN KEY (security_id) REFERENCES securities (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS comments (
    id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    post_id BIGINT NOT NULL,
    author_id BIGINT NOT NULL,
    parent_id BIGINT NULL,
    body VARCHAR(2000) NOT NULL,
    visibility ENUM('visible', 'hidden') NOT NULL DEFAULT 'visible',
    moderation_version BIGINT NOT NULL DEFAULT 1,
    idempotency_key VARCHAR(128) CHARACTER SET ascii COLLATE ascii_bin NULL,
    request_hash CHAR(64) CHARACTER SET ascii COLLATE ascii_bin NULL,
    created_at DATETIME(6) NOT NULL DEFAULT (UTC_TIMESTAMP(6)),
    updated_at DATETIME(6) NOT NULL DEFAULT (UTC_TIMESTAMP(6)),
    deleted_at DATETIME(6) NULL,
    CONSTRAINT chk_comments_schema_v1 CHECK (
        post_id > 0 AND author_id > 0 AND (parent_id IS NULL OR parent_id > 0) AND CHAR_LENGTH(body) >= 1
    ),
    CONSTRAINT chk_comments_moderation_version CHECK (moderation_version >= 1),
    CONSTRAINT chk_comments_idempotency_pair CHECK (
        (idempotency_key IS NULL AND request_hash IS NULL) OR
        (idempotency_key IS NOT NULL AND request_hash IS NOT NULL)
    ),
    UNIQUE KEY uq_comments_author_idempotency (author_id, idempotency_key),
    KEY idx_comments_list (post_id, visibility, deleted_at, created_at, id),
    KEY idx_comments_parent (parent_id),
    CONSTRAINT fk_comments_post FOREIGN KEY (post_id) REFERENCES posts (id),
    CONSTRAINT fk_comments_author FOREIGN KEY (author_id) REFERENCES users (id),
    CONSTRAINT fk_comments_parent FOREIGN KEY (parent_id) REFERENCES comments (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS reports (
    id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    reporter_id BIGINT NOT NULL,
    post_id BIGINT NULL,
    comment_id BIGINT NULL,
    reason_code ENUM('spam', 'harassment', 'misleading', 'illegal', 'other') NOT NULL,
    details VARCHAR(1000) NOT NULL DEFAULT '',
    status ENUM('pending', 'dismissed', 'resolved') NOT NULL DEFAULT 'pending',
    resolution_action ENUM('dismiss', 'hide', 'author_deleted') NULL,
    handled_by BIGINT NULL,
    handled_at DATETIME(6) NULL,
    resolution_note VARCHAR(1000) NULL,
    created_at DATETIME(6) NOT NULL DEFAULT (UTC_TIMESTAMP(6)),
    updated_at DATETIME(6) NOT NULL DEFAULT (UTC_TIMESTAMP(6)),
    CONSTRAINT chk_reports_schema_v1 CHECK (reporter_id > 0),
    CONSTRAINT chk_reports_one_target CHECK ((post_id IS NOT NULL) <> (comment_id IS NOT NULL)),
    CONSTRAINT chk_reports_state_shape CHECK (
        (status = 'pending' AND resolution_action IS NULL AND handled_by IS NULL AND handled_at IS NULL AND resolution_note IS NULL) OR
        (status = 'dismissed' AND resolution_action = 'dismiss' AND handled_by IS NOT NULL AND handled_at IS NOT NULL) OR
        (status = 'resolved' AND resolution_action = 'hide' AND handled_by IS NOT NULL AND handled_at IS NOT NULL) OR
        (status = 'resolved' AND resolution_action = 'author_deleted' AND handled_by IS NULL AND handled_at IS NOT NULL)
    ),
    UNIQUE KEY uq_reports_reporter_post (reporter_id, post_id),
    UNIQUE KEY uq_reports_reporter_comment (reporter_id, comment_id),
    KEY idx_reports_queue (status, created_at, id),
    KEY idx_reports_post_pending (post_id, status, id),
    KEY idx_reports_comment_pending (comment_id, status, id),
    CONSTRAINT fk_reports_reporter FOREIGN KEY (reporter_id) REFERENCES users (id),
    CONSTRAINT fk_reports_post FOREIGN KEY (post_id) REFERENCES posts (id),
    CONSTRAINT fk_reports_comment FOREIGN KEY (comment_id) REFERENCES comments (id),
    CONSTRAINT fk_reports_handler FOREIGN KEY (handled_by) REFERENCES users (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS notifications (
    id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    user_id BIGINT NOT NULL,
    type ENUM('comment', 'reply', 'content_hidden', 'content_restored') NOT NULL,
    actor_user_id BIGINT NULL,
    post_id BIGINT NOT NULL,
    comment_id BIGINT NULL,
    read_at DATETIME(6) NULL,
    created_at DATETIME(6) NOT NULL DEFAULT (UTC_TIMESTAMP(6)),
    CONSTRAINT chk_notifications_schema_v1 CHECK (user_id > 0 AND post_id > 0),
    CONSTRAINT chk_notifications_shape CHECK (
        (type IN ('comment', 'reply') AND actor_user_id IS NOT NULL AND comment_id IS NOT NULL) OR
        (type IN ('content_hidden', 'content_restored') AND actor_user_id IS NULL)
    ),
    KEY idx_notifications_inbox (user_id, read_at, created_at DESC, id DESC),
    KEY idx_notifications_timeline (user_id, created_at DESC, id DESC),
    CONSTRAINT fk_notifications_user FOREIGN KEY (user_id) REFERENCES users (id),
    CONSTRAINT fk_notifications_actor FOREIGN KEY (actor_user_id) REFERENCES users (id),
    CONSTRAINT fk_notifications_post FOREIGN KEY (post_id) REFERENCES posts (id),
    CONSTRAINT fk_notifications_comment FOREIGN KEY (comment_id) REFERENCES comments (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS admin_audit_logs (
    id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    admin_user_id BIGINT NOT NULL,
    action ENUM('report_dismissed', 'content_hidden', 'content_restored') NOT NULL,
    report_id BIGINT NULL,
    post_id BIGINT NULL,
    comment_id BIGINT NULL,
    before_status VARCHAR(32) NOT NULL,
    after_status VARCHAR(32) NOT NULL,
    reason VARCHAR(1000) NOT NULL,
    request_id VARCHAR(64) NOT NULL,
    created_at DATETIME(6) NOT NULL DEFAULT (UTC_TIMESTAMP(6)),
    CONSTRAINT chk_admin_audit_schema_v1 CHECK (admin_user_id > 0),
    CONSTRAINT chk_admin_audit_target CHECK ((post_id IS NOT NULL) <> (comment_id IS NOT NULL)),
    CONSTRAINT chk_admin_audit_shape CHECK (
        (action = 'report_dismissed' AND report_id IS NOT NULL AND before_status = 'pending' AND after_status = 'dismissed') OR
        (action = 'content_hidden' AND report_id IS NOT NULL AND before_status = 'visible' AND after_status = 'hidden') OR
        (action = 'content_restored' AND report_id IS NULL AND before_status = 'hidden' AND after_status = 'visible')
    ),
    UNIQUE KEY uq_admin_audit_report (report_id),
    KEY idx_admin_audit_created (created_at DESC, id DESC),
    KEY idx_admin_audit_actor (admin_user_id, created_at DESC),
    CONSTRAINT fk_admin_audit_admin FOREIGN KEY (admin_user_id) REFERENCES users (id),
    CONSTRAINT fk_admin_audit_report FOREIGN KEY (report_id) REFERENCES reports (id),
    CONSTRAINT fk_admin_audit_post FOREIGN KEY (post_id) REFERENCES posts (id),
    CONSTRAINT fk_admin_audit_comment FOREIGN KEY (comment_id) REFERENCES comments (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
