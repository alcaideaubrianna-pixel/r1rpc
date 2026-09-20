CREATE DATABASE IF NOT EXISTS `r1rpc` DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
USE `r1rpc`;

CREATE TABLE IF NOT EXISTS users (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    username VARCHAR(64) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    role ENUM('admin', 'client') NOT NULL DEFAULT 'admin',
    enabled TINYINT(1) NOT NULL DEFAULT 1,
    notes VARCHAR(255) NOT NULL DEFAULT '',
    last_login_at DATETIME NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_users_role_enabled (role, enabled)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `groups` (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    name VARCHAR(128) NOT NULL UNIQUE,
    display_name VARCHAR(128) NOT NULL DEFAULT '',
    enabled TINYINT(1) NOT NULL DEFAULT 1,
    device_key VARCHAR(128) NOT NULL DEFAULT '',
    auth_mode VARCHAR(16) NOT NULL DEFAULT 'none',
    api_key VARCHAR(128) NOT NULL DEFAULT '',
    notes VARCHAR(255) NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_groups_enabled_name (enabled, name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS devices (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    client_id VARCHAR(128) NOT NULL UNIQUE,
    group_name VARCHAR(128) NOT NULL,
    platform VARCHAR(64) NOT NULL DEFAULT 'xposed',
    last_seen_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_ip VARCHAR(64) NOT NULL DEFAULT '',
    extra_json LONGTEXT NULL,
    actions_json LONGTEXT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_devices_group_last_seen (group_name, last_seen_at),
    INDEX idx_devices_last_seen (last_seen_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS storage_settings (
    id TINYINT PRIMARY KEY,
    backend VARCHAR(16) NOT NULL,
    local_path VARCHAR(1024) NOT NULL DEFAULT '',
    endpoint VARCHAR(1024) NOT NULL DEFAULT '',
    region VARCHAR(128) NOT NULL DEFAULT '',
    bucket VARCHAR(255) NOT NULL DEFAULT '',
    path_style TINYINT(1) NOT NULL DEFAULT 1,
    access_key_encrypted TEXT NULL,
    secret_key_encrypted TEXT NULL,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS data_sources (
    id CHAR(32) PRIMARY KEY,
    name VARCHAR(128) NOT NULL,
    base_url VARCHAR(512) NOT NULL,
    app_id VARCHAR(128) NOT NULL,
    access_key VARCHAR(256) NOT NULL,
    secret_key_encrypted TEXT NOT NULL,
    status VARCHAR(16) NOT NULL DEFAULT 'enabled',
    request_timeout_seconds INT NOT NULL DEFAULT 30,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uk_data_sources_name (name),
    INDEX idx_data_sources_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS channel_scan_tasks (
    id CHAR(32) PRIMARY KEY,
    data_source_id CHAR(32) NOT NULL,
    channel_id BIGINT NOT NULL,
    channel_title VARCHAR(255) NOT NULL DEFAULT '',
    mode VARCHAR(16) NOT NULL DEFAULT 'once',
    initial_limit INT NOT NULL DEFAULT 10,
    poll_interval_minutes INT NOT NULL DEFAULT 10,
    priority INT NOT NULL DEFAULT 0,
    search_config_id CHAR(32) NOT NULL DEFAULT '',
    status VARCHAR(16) NOT NULL DEFAULT 'created',
    cursor_value VARCHAR(512) NOT NULL DEFAULT '',
    watermark DATETIME NULL,
    next_run_at DATETIME NULL,
    last_success_at DATETIME NULL,
    last_error VARCHAR(1024) NOT NULL DEFAULT '',
    image_search_request_id CHAR(32) NOT NULL DEFAULT '',
    search_task_id CHAR(32) NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_channel_scan_tasks_due (status, next_run_at),
    INDEX idx_channel_scan_tasks_source_channel (data_source_id, channel_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS source_channels (
    id CHAR(32) PRIMARY KEY,
    data_source_id CHAR(32) NOT NULL,
    channel_id BIGINT NOT NULL,
    title VARCHAR(512) NOT NULL DEFAULT '',
    username VARCHAR(255) NOT NULL DEFAULT '',
    chat_type VARCHAR(64) NOT NULL DEFAULT '',
    raw_json LONGTEXT NULL,
    last_synced_at DATETIME NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uk_source_channels_source_channel (data_source_id, channel_id),
    INDEX idx_source_channels_source_title (data_source_id, title),
    INDEX idx_source_channels_synced (last_synced_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS source_notes (
    id CHAR(32) PRIMARY KEY,
    data_source_id CHAR(32) NOT NULL,
    scan_task_id CHAR(32) NOT NULL,
    channel_id BIGINT NOT NULL,
    external_note_id VARCHAR(128) NOT NULL,
    note_code VARCHAR(128) NOT NULL DEFAULT '',
    title VARCHAR(512) NOT NULL DEFAULT '',
    plain_text LONGTEXT NULL,
    attributes_json LONGTEXT NULL,
    raw_json LONGTEXT NULL,
    status VARCHAR(24) NOT NULL DEFAULT 'received',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uk_source_notes_source_note (data_source_id, channel_id, external_note_id),
    INDEX idx_source_notes_task_created (scan_task_id, created_at),
    INDEX idx_source_notes_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS source_note_images (
    id CHAR(32) PRIMARY KEY,
    note_id CHAR(32) NOT NULL,
    external_asset_id VARCHAR(128) NOT NULL DEFAULT '',
    asset_type VARCHAR(32) NOT NULL DEFAULT 'image',
    source_url VARCHAR(2048) NOT NULL,
    file_id CHAR(32) NOT NULL DEFAULT '',
    sha256 CHAR(64) NOT NULL DEFAULT '',
    phash VARCHAR(32) NOT NULL DEFAULT '',
    download_status VARCHAR(24) NOT NULL DEFAULT 'pending',
    preprocess_status VARCHAR(24) NOT NULL DEFAULT 'pending',
    ocr_text LONGTEXT NULL,
    filter_decision VARCHAR(24) NOT NULL DEFAULT 'pending',
    filter_reason VARCHAR(1024) NOT NULL DEFAULT '',
    raw_json LONGTEXT NULL,
    image_index INT NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uk_source_note_images_asset (note_id, external_asset_id),
    INDEX idx_source_note_images_queue (download_status, preprocess_status),
    INDEX idx_source_note_images_note (note_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS source_scan_task_notes (
    id CHAR(32) PRIMARY KEY,
    scan_task_id CHAR(32) NOT NULL,
    note_id CHAR(32) NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY uk_source_scan_task_notes (scan_task_id, note_id),
    INDEX idx_source_scan_task_notes_note (note_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS ocr_filter_rules (
    id CHAR(32) PRIMARY KEY,
    name VARCHAR(128) NOT NULL,
    mode VARCHAR(24) NOT NULL DEFAULT 'contains_any',
    pattern_json LONGTEXT NOT NULL,
    enabled TINYINT(1) NOT NULL DEFAULT 1,
    version INT NOT NULL DEFAULT 1,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uk_ocr_filter_rules_name (name),
    INDEX idx_ocr_filter_rules_enabled (enabled)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS search_configs (
    id CHAR(32) PRIMARY KEY,
    name VARCHAR(128) NOT NULL,
    ocr_enabled TINYINT(1) NOT NULL DEFAULT 0,
    ocr_keywords_json LONGTEXT NULL,
    ocr_match_mode VARCHAR(24) NOT NULL DEFAULT 'contains_any',
    score_threshold DECIMAL(6,5) NOT NULL DEFAULT 0.82000,
    max_phash_distance INT NOT NULL DEFAULT 12,
    max_dhash_distance INT NOT NULL DEFAULT 16,
    max_ahash_distance INT NOT NULL DEFAULT 16,
    max_candidates INT NOT NULL DEFAULT 20,
    pipeline_name VARCHAR(64) NOT NULL DEFAULT 'hash-v1',
    enabled TINYINT(1) NOT NULL DEFAULT 1,
    version INT NOT NULL DEFAULT 1,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uk_search_configs_name (name),
    INDEX idx_search_configs_enabled (enabled)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS search_tasks (
    id CHAR(32) PRIMARY KEY,
    source_type VARCHAR(32) NOT NULL,
    source_task_id CHAR(32) NOT NULL DEFAULT '',
    title VARCHAR(512) NOT NULL DEFAULT '',
    config_id CHAR(32) NOT NULL DEFAULT '',
    config_snapshot_json LONGTEXT NULL,
    status VARCHAR(24) NOT NULL DEFAULT 'created',
    total_count INT NOT NULL DEFAULT 0,
    filtered_count INT NOT NULL DEFAULT 0,
    search_count INT NOT NULL DEFAULT 0,
    matched_count INT NOT NULL DEFAULT 0,
    failed_count INT NOT NULL DEFAULT 0,
    error_message VARCHAR(1024) NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uk_search_tasks_source (source_type, source_task_id),
    INDEX idx_search_tasks_status_created (status, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS search_task_items (
    id CHAR(32) PRIMARY KEY,
    search_task_id CHAR(32) NOT NULL,
    source_note_id CHAR(32) NOT NULL,
    external_id VARCHAR(128) NOT NULL DEFAULT '',
    title VARCHAR(512) NOT NULL DEFAULT '',
    status VARCHAR(24) NOT NULL DEFAULT 'created',
    preprocess_status VARCHAR(24) NOT NULL DEFAULT 'pending',
    filter_status VARCHAR(24) NOT NULL DEFAULT 'pending',
    filter_reason VARCHAR(1024) NOT NULL DEFAULT '',
    image_search_request_id CHAR(32) NOT NULL DEFAULT '',
    matched TINYINT(1) NOT NULL DEFAULT 0,
    best_score DECIMAL(8,6) NULL,
    error_message VARCHAR(1024) NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uk_search_task_items_note (search_task_id, source_note_id),
    INDEX idx_search_task_items_status (search_task_id, status),
    INDEX idx_search_task_items_request (image_search_request_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS files (
    id CHAR(32) PRIMARY KEY,
    object_key VARCHAR(512) NOT NULL UNIQUE,
    original_name VARCHAR(512) NOT NULL,
    content_type VARCHAR(128) NOT NULL,
    size_bytes BIGINT NOT NULL,
    sha256 CHAR(64) NOT NULL,
    backend VARCHAR(16) NOT NULL,
    status VARCHAR(16) NOT NULL DEFAULT 'active',
    created_by BIGINT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at DATETIME NULL,
    deleted_at DATETIME NULL,
    INDEX idx_files_status_created (status, created_at),
    INDEX idx_files_created_by_created (created_by, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS image_assets (
    id CHAR(32) PRIMARY KEY,
    file_id CHAR(32) NOT NULL,
    sha256 CHAR(64) NOT NULL,
    mime_type VARCHAR(128) NOT NULL,
    size_bytes BIGINT NOT NULL,
    width INT NOT NULL DEFAULT 0,
    height INT NOT NULL DEFAULT 0,
    object_key VARCHAR(1024) NOT NULL DEFAULT '',
    storage_status VARCHAR(24) NOT NULL DEFAULT 'local_pending',
    phash VARCHAR(32) NOT NULL DEFAULT '',
    dhash VARCHAR(32) NOT NULL DEFAULT '',
    ahash VARCHAR(32) NOT NULL DEFAULT '',
    algorithm_version VARCHAR(64) NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at DATETIME NULL,
    UNIQUE KEY uk_image_assets_sha256 (sha256),
    INDEX idx_image_assets_file (file_id),
    INDEX idx_image_assets_storage_created (storage_status, created_at),
    INDEX idx_image_assets_phash (phash)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS image_batches (
    id CHAR(32) PRIMARY KEY,
    source VARCHAR(24) NOT NULL,
    external_id VARCHAR(128) NULL,
    status VARCHAR(24) NOT NULL DEFAULT 'created',
    priority INT NOT NULL DEFAULT 0,
    force_refresh TINYINT(1) NOT NULL DEFAULT 0,
    total_count INT NOT NULL DEFAULT 0,
    queued_count INT NOT NULL DEFAULT 0,
    running_count INT NOT NULL DEFAULT 0,
    completed_count INT NOT NULL DEFAULT 0,
    failed_count INT NOT NULL DEFAULT 0,
    cache_hit_count INT NOT NULL DEFAULT 0,
    requested_by_json LONGTEXT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    finished_at DATETIME NULL,
    deleted_at DATETIME NULL,
    UNIQUE KEY uk_image_batches_source_external (source, external_id),
    INDEX idx_image_batches_status_created (status, created_at),
    INDEX idx_image_batches_created (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS image_jobs (
    id CHAR(32) PRIMARY KEY,
    batch_id CHAR(32) NOT NULL DEFAULT '',
    source_asset_id CHAR(32) NOT NULL,
    source VARCHAR(24) NOT NULL,
    external_id VARCHAR(128) NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'created',
    stage VARCHAR(32) NOT NULL DEFAULT 'created',
    priority INT NOT NULL DEFAULT 0,
    force_refresh TINYINT(1) NOT NULL DEFAULT 0,
    attempt INT NOT NULL DEFAULT 0,
    assigned_client_id VARCHAR(128) NOT NULL DEFAULT '',
    assigned_session_incarnation VARCHAR(128) NOT NULL DEFAULT '',
    upload_request_id VARCHAR(64) NOT NULL DEFAULT '',
    search_request_id VARCHAR(64) NOT NULL DEFAULT '',
    upload_handle VARCHAR(255) NOT NULL DEFAULT '',
    cache_source_job_id CHAR(32) NOT NULL DEFAULT '',
    error_code VARCHAR(64) NOT NULL DEFAULT '',
    error_message VARCHAR(1024) NOT NULL DEFAULT '',
    input_json LONGTEXT NULL,
    normalized_response_json LONGTEXT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    started_at DATETIME NULL,
    finished_at DATETIME NULL,
    deleted_at DATETIME NULL,
    UNIQUE KEY uk_image_jobs_source_external (source, external_id),
    INDEX idx_image_jobs_batch_created (batch_id, created_at),
    INDEX idx_image_jobs_status_priority_created (status, priority, created_at),
    INDEX idx_image_jobs_asset_created (source_asset_id, created_at),
    INDEX idx_image_jobs_client_status (assigned_client_id, status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS image_search_requests (
    id CHAR(32) PRIMARY KEY,
    source VARCHAR(24) NOT NULL,
    external_id VARCHAR(128) NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'created',
    pipeline_name VARCHAR(64) NOT NULL DEFAULT 'xhs-image-match',
    pipeline_version VARCHAR(32) NOT NULL DEFAULT 'v1',
    priority INT NOT NULL DEFAULT 0,
    group_count INT NOT NULL DEFAULT 0,
    matched_count INT NOT NULL DEFAULT 0,
    not_matched_count INT NOT NULL DEFAULT 0,
    running_count INT NOT NULL DEFAULT 0,
    failed_count INT NOT NULL DEFAULT 0,
    requested_by_user_id BIGINT NULL,
    requested_by_subject VARCHAR(128) NOT NULL DEFAULT '',
    callback_url VARCHAR(1024) NOT NULL DEFAULT '',
    callback_status VARCHAR(24) NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    finished_at DATETIME NULL,
    deleted_at DATETIME NULL,
    UNIQUE KEY uk_image_search_requests_source_external (source, external_id),
    INDEX idx_image_search_requests_status_created (status, created_at),
    INDEX idx_image_search_requests_requester_created (requested_by_user_id, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS image_search_groups (
    id CHAR(32) PRIMARY KEY,
    request_id CHAR(32) NOT NULL,
    external_id VARCHAR(128) NULL,
    subject_user_id VARCHAR(128) NOT NULL DEFAULT '',
    status VARCHAR(32) NOT NULL DEFAULT 'created',
    match_policy_json LONGTEXT NULL,
    item_count INT NOT NULL DEFAULT 0,
    queued_count INT NOT NULL DEFAULT 0,
    running_count INT NOT NULL DEFAULT 0,
    completed_count INT NOT NULL DEFAULT 0,
    failed_count INT NOT NULL DEFAULT 0,
    cancelled_count INT NOT NULL DEFAULT 0,
    best_match_id CHAR(32) NOT NULL DEFAULT '',
    best_score DECIMAL(8,6) NULL,
    matched_at DATETIME NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    finished_at DATETIME NULL,
    deleted_at DATETIME NULL,
    UNIQUE KEY uk_image_search_groups_request_external (request_id, external_id),
    INDEX idx_image_search_groups_request_status (request_id, status),
    INDEX idx_image_search_groups_subject_created (subject_user_id, created_at),
    INDEX idx_image_search_groups_status_created (status, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS image_search_items (
    id CHAR(32) PRIMARY KEY,
    group_id CHAR(32) NOT NULL,
    source_asset_id CHAR(32) NOT NULL,
    ordinal INT NOT NULL DEFAULT 0,
    status VARCHAR(32) NOT NULL DEFAULT 'created',
    job_id CHAR(32) NOT NULL DEFAULT '',
    error_code VARCHAR(64) NOT NULL DEFAULT '',
    error_message VARCHAR(1024) NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    finished_at DATETIME NULL,
    deleted_at DATETIME NULL,
    UNIQUE KEY uk_image_search_items_group_asset (group_id, source_asset_id),
    UNIQUE KEY uk_image_search_items_group_ordinal (group_id, ordinal),
    INDEX idx_image_search_items_group_status (group_id, status),
    INDEX idx_image_search_items_job (job_id),
    INDEX idx_image_search_items_asset_created (source_asset_id, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS image_search_responses (
    id CHAR(32) PRIMARY KEY,
    job_id CHAR(32) NOT NULL,
    request_id VARCHAR(64) NOT NULL DEFAULT '',
    normalized_json LONGTEXT NOT NULL,
    raw_json LONGTEXT NULL,
    item_count INT NOT NULL DEFAULT 0,
    parse_status VARCHAR(24) NOT NULL DEFAULT 'pending',
    error_message VARCHAR(1024) NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uk_image_search_responses_job (job_id),
    INDEX idx_image_search_responses_status_created (parse_status, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS image_candidates (
    id CHAR(32) PRIMARY KEY,
    response_id CHAR(32) NOT NULL,
    job_id CHAR(32) NOT NULL,
    search_item_id CHAR(32) NOT NULL,
    rank_no INT NOT NULL,
    content_id VARCHAR(128) NOT NULL,
    title VARCHAR(512) NOT NULL DEFAULT '',
    author_id VARCHAR(128) NOT NULL DEFAULT '',
    author_name VARCHAR(255) NOT NULL DEFAULT '',
    cover_url VARCHAR(2048) NOT NULL DEFAULT '',
    image_file_id CHAR(32) NOT NULL DEFAULT '',
    raw_item_json LONGTEXT NULL,
    download_status VARCHAR(24) NOT NULL DEFAULT 'pending',
    image_sha256 CHAR(64) NOT NULL DEFAULT '',
    phash VARCHAR(32) NOT NULL DEFAULT '',
    dhash VARCHAR(32) NOT NULL DEFAULT '',
    ahash VARCHAR(32) NOT NULL DEFAULT '',
    algorithm_version VARCHAR(64) NOT NULL DEFAULT '',
    score DECIMAL(8,6) NULL,
    matched TINYINT(1) NOT NULL DEFAULT 0,
    error_message VARCHAR(1024) NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uk_image_candidates_job_rank (job_id, rank_no),
    INDEX idx_image_candidates_item_score (search_item_id, score),
    INDEX idx_image_candidates_content (content_id),
    INDEX idx_image_candidates_download_created (download_status, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS image_candidate_images (
    id CHAR(32) PRIMARY KEY,
    candidate_id CHAR(32) NOT NULL,
    image_index INT NOT NULL,
    source_url VARCHAR(2048) NOT NULL,
    image_file_id CHAR(32) NOT NULL DEFAULT '',
    download_status VARCHAR(24) NOT NULL DEFAULT 'pending',
    image_sha256 CHAR(64) NOT NULL DEFAULT '',
    phash VARCHAR(32) NOT NULL DEFAULT '',
    dhash VARCHAR(32) NOT NULL DEFAULT '',
    ahash VARCHAR(32) NOT NULL DEFAULT '',
    algorithm_version VARCHAR(64) NOT NULL DEFAULT '',
    score DECIMAL(8,6) NULL,
    phash_distance INT NULL,
    dhash_distance INT NULL,
    ahash_distance INT NULL,
    matched TINYINT(1) NOT NULL DEFAULT 0,
    error_message VARCHAR(1024) NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uk_image_candidate_images_candidate_index (candidate_id, image_index),
    INDEX idx_image_candidate_images_candidate_score (candidate_id, score),
    INDEX idx_image_candidate_images_sha256 (image_sha256)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS xhs_user_profiles (
    user_id VARCHAR(128) PRIMARY KEY,
    red_id VARCHAR(64) NOT NULL DEFAULT '',
    nickname VARCHAR(255) NOT NULL DEFAULT '',
    avatar_url VARCHAR(2048) NOT NULL DEFAULT '',
    avatar_file_id CHAR(32) NOT NULL DEFAULT '',
    description TEXT NULL,
    gender INT NOT NULL DEFAULT 0,
    ip_location VARCHAR(128) NOT NULL DEFAULT '',
    fans_count BIGINT NOT NULL DEFAULT 0,
    liked_count BIGINT NOT NULL DEFAULT 0,
    collected_count BIGINT NOT NULL DEFAULT 0,
    note_count BIGINT NOT NULL DEFAULT 0,
    share_link VARCHAR(2048) NOT NULL DEFAULT '',
    raw_json LONGTEXT NULL,
    fetched_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_xhs_user_profiles_red_id (red_id),
    INDEX idx_xhs_user_profiles_fetched (fetched_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS image_matches (
    id CHAR(32) PRIMARY KEY,
    search_item_id CHAR(32) NOT NULL,
    candidate_id CHAR(32) NOT NULL,
    algorithm VARCHAR(32) NOT NULL DEFAULT 'perceptual_hash',
    algorithm_version VARCHAR(64) NOT NULL,
    phash_distance INT NOT NULL DEFAULT 64,
    dhash_distance INT NOT NULL DEFAULT 64,
    ahash_distance INT NOT NULL DEFAULT 64,
    score DECIMAL(8,6) NOT NULL DEFAULT 0,
    decision VARCHAR(24) NOT NULL DEFAULT 'not_matched',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY uk_image_matches_item_candidate_algorithm (search_item_id, candidate_id, algorithm, algorithm_version),
    INDEX idx_image_matches_item_score (search_item_id, score),
    INDEX idx_image_matches_decision_created (decision, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS rpc_requests (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    request_id VARCHAR(64) NOT NULL UNIQUE,
    group_name VARCHAR(128) NOT NULL,
    action_name VARCHAR(128) NOT NULL,
    client_id VARCHAR(128) NOT NULL,
    requester_user_id BIGINT NULL,
    request_payload_json LONGTEXT NULL,
    response_payload_json LONGTEXT NULL,
    status ENUM('pending', 'success', 'error', 'timeout', 'no_client', 'rejected') NOT NULL DEFAULT 'pending',
    http_code INT NOT NULL DEFAULT 200,
    latency_ms BIGINT NOT NULL DEFAULT 0,
    error_message VARCHAR(255) NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    finished_at DATETIME NULL,
    INDEX idx_rpc_requests_lookup (group_name, action_name, client_id, created_at),
    INDEX idx_rpc_requests_group_client_created (group_name, client_id, created_at),
    INDEX idx_rpc_requests_client_created (client_id, created_at),
    INDEX idx_rpc_requests_action_created (action_name, created_at),
    INDEX idx_rpc_requests_created_group_action (created_at, group_name, action_name),
    INDEX idx_rpc_requests_created_at (created_at),
    INDEX idx_rpc_requests_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS daily_metrics (
    stat_date DATE NOT NULL,
    group_name VARCHAR(128) NOT NULL,
    action_name VARCHAR(128) NOT NULL DEFAULT '',
    client_id VARCHAR(128) NOT NULL DEFAULT '',
    total_requests BIGINT NOT NULL DEFAULT 0,
    success_requests BIGINT NOT NULL DEFAULT 0,
    failed_requests BIGINT NOT NULL DEFAULT 0,
    timeout_requests BIGINT NOT NULL DEFAULT 0,
    total_latency_ms BIGINT NOT NULL DEFAULT 0,
    max_latency_ms BIGINT NOT NULL DEFAULT 0,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (stat_date, group_name, action_name, client_id),
    INDEX idx_daily_metrics_group_date (group_name, stat_date),
    INDEX idx_daily_metrics_action_date (action_name, stat_date),
    INDEX idx_daily_metrics_client_date (client_id, stat_date)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS probe_buckets (
    group_name VARCHAR(128) NOT NULL,
    minute_ts BIGINT NOT NULL,
    online INT NOT NULL DEFAULT 0,
    healthy INT NOT NULL DEFAULT 0,
    total INT NOT NULL DEFAULT 0,
    max_lat_ms BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY (group_name, minute_ts),
    INDEX idx_probe_buckets_ts (minute_ts)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
