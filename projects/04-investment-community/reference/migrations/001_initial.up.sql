CREATE TABLE IF NOT EXISTS users (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    email VARCHAR(254) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    display_name VARCHAR(80) NOT NULL,
    role ENUM('user', 'admin') NOT NULL DEFAULT 'user',
    status ENUM('active', 'disabled') NOT NULL DEFAULT 'active',
    created_at DATETIME(6) NOT NULL DEFAULT (UTC_TIMESTAMP(6)),
    updated_at DATETIME(6) NOT NULL DEFAULT (UTC_TIMESTAMP(6)),
    UNIQUE KEY uq_users_email (email)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS circles (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    slug VARCHAR(64) NOT NULL,
    name VARCHAR(80) NOT NULL,
    description VARCHAR(500) NOT NULL DEFAULT '',
    status ENUM('active', 'archived') NOT NULL DEFAULT 'active',
    created_at DATETIME(6) NOT NULL DEFAULT (UTC_TIMESTAMP(6)),
    updated_at DATETIME(6) NOT NULL DEFAULT (UTC_TIMESTAMP(6)),
    UNIQUE KEY uq_circles_slug (slug),
    UNIQUE KEY uq_circles_name (name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS circle_memberships (
    circle_id BIGINT UNSIGNED NOT NULL,
    user_id BIGINT UNSIGNED NOT NULL,
    joined_at DATETIME(6) NOT NULL DEFAULT (UTC_TIMESTAMP(6)),
    PRIMARY KEY (circle_id, user_id),
    KEY idx_circle_memberships_user (user_id, joined_at),
    CONSTRAINT fk_circle_memberships_circle FOREIGN KEY (circle_id) REFERENCES circles (id),
    CONSTRAINT fk_circle_memberships_user FOREIGN KEY (user_id) REFERENCES users (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS securities (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    market VARCHAR(16) NOT NULL,
    code VARCHAR(12) NOT NULL,
    name VARCHAR(80) NOT NULL,
    status ENUM('active', 'inactive') NOT NULL DEFAULT 'active',
    created_at DATETIME(6) NOT NULL DEFAULT (UTC_TIMESTAMP(6)),
    UNIQUE KEY uq_securities_market_code (market, code),
    KEY idx_securities_name (name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS posts (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    circle_id BIGINT UNSIGNED NOT NULL,
    author_id BIGINT UNSIGNED NOT NULL,
    title VARCHAR(120) NOT NULL,
    body TEXT NOT NULL,
    visibility ENUM('visible', 'hidden') NOT NULL DEFAULT 'visible',
    version BIGINT UNSIGNED NOT NULL DEFAULT 1,
    idempotency_key VARCHAR(128) NULL,
    request_hash CHAR(64) NULL,
    created_at DATETIME(6) NOT NULL DEFAULT (UTC_TIMESTAMP(6)),
    updated_at DATETIME(6) NOT NULL DEFAULT (UTC_TIMESTAMP(6)),
    deleted_at DATETIME(6) NULL,
    UNIQUE KEY uq_posts_author_idempotency (author_id, idempotency_key),
    KEY idx_posts_feed (circle_id, visibility, deleted_at, created_at DESC, id DESC),
    CONSTRAINT fk_posts_circle FOREIGN KEY (circle_id) REFERENCES circles (id),
    CONSTRAINT fk_posts_author FOREIGN KEY (author_id) REFERENCES users (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS post_securities (
    post_id BIGINT UNSIGNED NOT NULL,
    security_id BIGINT UNSIGNED NOT NULL,
    created_at DATETIME(6) NOT NULL DEFAULT (UTC_TIMESTAMP(6)),
    PRIMARY KEY (post_id, security_id),
    KEY idx_post_securities_security (security_id, post_id),
    CONSTRAINT fk_post_securities_post FOREIGN KEY (post_id) REFERENCES posts (id),
    CONSTRAINT fk_post_securities_security FOREIGN KEY (security_id) REFERENCES securities (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS comments (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    post_id BIGINT UNSIGNED NOT NULL,
    author_id BIGINT UNSIGNED NOT NULL,
    parent_id BIGINT UNSIGNED NULL,
    body VARCHAR(2000) NOT NULL,
    visibility ENUM('visible', 'hidden') NOT NULL DEFAULT 'visible',
    idempotency_key VARCHAR(128) NULL,
    request_hash CHAR(64) NULL,
    created_at DATETIME(6) NOT NULL DEFAULT (UTC_TIMESTAMP(6)),
    deleted_at DATETIME(6) NULL,
    UNIQUE KEY uq_comments_author_idempotency (author_id, idempotency_key),
    KEY idx_comments_thread (post_id, parent_id, visibility, deleted_at, created_at, id),
    CONSTRAINT fk_comments_post FOREIGN KEY (post_id) REFERENCES posts (id),
    CONSTRAINT fk_comments_author FOREIGN KEY (author_id) REFERENCES users (id),
    CONSTRAINT fk_comments_parent FOREIGN KEY (parent_id) REFERENCES comments (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS reports (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    reporter_id BIGINT UNSIGNED NOT NULL,
    post_id BIGINT UNSIGNED NULL,
    comment_id BIGINT UNSIGNED NULL,
    reason_code ENUM('spam', 'harassment', 'misleading', 'illegal', 'other') NOT NULL,
    details VARCHAR(1000) NOT NULL DEFAULT '',
    status ENUM('pending', 'dismissed', 'resolved') NOT NULL DEFAULT 'pending',
    resolution_action ENUM('dismiss', 'hide', 'author_deleted') NULL,
    handled_by BIGINT UNSIGNED NULL,
    handled_at DATETIME(6) NULL,
    resolution_note VARCHAR(1000) NULL,
    created_at DATETIME(6) NOT NULL DEFAULT (UTC_TIMESTAMP(6)),
    updated_at DATETIME(6) NOT NULL DEFAULT (UTC_TIMESTAMP(6)),
    CONSTRAINT chk_reports_one_target CHECK ((post_id IS NOT NULL) <> (comment_id IS NOT NULL)),
    UNIQUE KEY uq_reports_reporter_post (reporter_id, post_id),
    UNIQUE KEY uq_reports_reporter_comment (reporter_id, comment_id),
    KEY idx_reports_queue (status, created_at, id),
    CONSTRAINT fk_reports_reporter FOREIGN KEY (reporter_id) REFERENCES users (id),
    CONSTRAINT fk_reports_post FOREIGN KEY (post_id) REFERENCES posts (id),
    CONSTRAINT fk_reports_comment FOREIGN KEY (comment_id) REFERENCES comments (id),
    CONSTRAINT fk_reports_handler FOREIGN KEY (handled_by) REFERENCES users (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS notifications (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    user_id BIGINT UNSIGNED NOT NULL,
    type ENUM('comment', 'reply', 'content_hidden', 'content_restored') NOT NULL,
    actor_user_id BIGINT UNSIGNED NULL,
    post_id BIGINT UNSIGNED NULL,
    comment_id BIGINT UNSIGNED NULL,
    read_at DATETIME(6) NULL,
    created_at DATETIME(6) NOT NULL DEFAULT (UTC_TIMESTAMP(6)),
    KEY idx_notifications_inbox (user_id, read_at, created_at DESC, id DESC),
    CONSTRAINT fk_notifications_user FOREIGN KEY (user_id) REFERENCES users (id),
    CONSTRAINT fk_notifications_actor FOREIGN KEY (actor_user_id) REFERENCES users (id),
    CONSTRAINT fk_notifications_post FOREIGN KEY (post_id) REFERENCES posts (id),
    CONSTRAINT fk_notifications_comment FOREIGN KEY (comment_id) REFERENCES comments (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS admin_audit_logs (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    admin_user_id BIGINT UNSIGNED NOT NULL,
    action ENUM('report_dismissed', 'content_hidden', 'content_restored') NOT NULL,
    report_id BIGINT UNSIGNED NULL,
    post_id BIGINT UNSIGNED NULL,
    comment_id BIGINT UNSIGNED NULL,
    before_status VARCHAR(32) NOT NULL,
    after_status VARCHAR(32) NOT NULL,
    reason VARCHAR(1000) NOT NULL,
    request_id VARCHAR(64) NOT NULL,
    created_at DATETIME(6) NOT NULL DEFAULT (UTC_TIMESTAMP(6)),
    KEY idx_admin_audit_created (created_at DESC, id DESC),
    KEY idx_admin_audit_actor (admin_user_id, created_at DESC),
    CONSTRAINT fk_admin_audit_admin FOREIGN KEY (admin_user_id) REFERENCES users (id),
    CONSTRAINT fk_admin_audit_report FOREIGN KEY (report_id) REFERENCES reports (id),
    CONSTRAINT fk_admin_audit_post FOREIGN KEY (post_id) REFERENCES posts (id),
    CONSTRAINT fk_admin_audit_comment FOREIGN KEY (comment_id) REFERENCES comments (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
