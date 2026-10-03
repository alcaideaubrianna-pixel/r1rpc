package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

var configSearchPaths = []string{
	"config.yaml",
	"r1rpc.yaml",
}

// yamlToEnvKey 把分组式 YAML 的 (section, key) 映射到内部使用的环境变量风格键名，
// 这样 getString/getInt 的「环境变量 > 配置文件 > 默认值」取值逻辑可以原样复用。
var yamlToEnvKey = []struct {
	Section string
	Key     string
	Env     string
}{
	{"server", "app_name", "APP_NAME"},
	{"server", "server_id", "SERVER_ID"},
	{"server", "http_addr", "HTTP_ADDR"},
	{"server", "jwt_secret", "JWT_SECRET"},
	{"server", "time_zone", "TIME_ZONE"},
	{"logging", "dir", "LOG_DIR"},
	{"logging", "max_dir_mb", "LOG_MAX_DIR_MB"},
	{"admin", "username", "BOOTSTRAP_ADMIN_USERNAME"},
	{"admin", "password", "BOOTSTRAP_ADMIN_PASSWORD"},
	{"mysql", "host", "MYSQL_HOST"},
	{"mysql", "port", "MYSQL_PORT"},
	{"mysql", "user", "MYSQL_USER"},
	{"mysql", "password", "MYSQL_PASSWORD"},
	{"mysql", "db", "MYSQL_DB"},
	{"mysql", "params", "MYSQL_PARAMS"},
	{"mysql", "max_open_conns", "MYSQL_MAX_OPEN_CONNS"},
	{"mysql", "max_idle_conns", "MYSQL_MAX_IDLE_CONNS"},
	{"mysql", "conn_max_lifetime_minutes", "MYSQL_CONN_MAX_LIFETIME_MINUTES"},
	{"redis", "addr", "REDIS_ADDR"},
	{"redis", "password", "REDIS_PASSWORD"},
	{"redis", "db", "REDIS_DB"},
	{"redis", "worker_concurrency", "REDIS_WORKER_CONCURRENCY"},
	{"redis", "api_worker_concurrency", "REDIS_API_WORKER_CONCURRENCY"},
	{"redis", "download_worker_concurrency", "REDIS_DOWNLOAD_WORKER_CONCURRENCY"},
	{"redis", "scan_interval_seconds", "QUEUE_SCAN_INTERVAL_SECONDS"},
	{"redis", "download_interval_seconds", "DOWNLOAD_SCAN_INTERVAL_SECONDS"},
	{"storage", "backend", "STORAGE_BACKEND"},
	{"storage", "local_path", "STORAGE_LOCAL_PATH"},
	{"storage", "endpoint", "STORAGE_S3_ENDPOINT"},
	{"storage", "region", "STORAGE_S3_REGION"},
	{"storage", "bucket", "STORAGE_S3_BUCKET"},
	{"storage", "path_style", "STORAGE_S3_PATH_STYLE"},
	{"storage", "access_key", "STORAGE_S3_ACCESS_KEY"},
	{"storage", "secret_key", "STORAGE_S3_SECRET_KEY"},
	{"limits", "request_timeout_seconds", "REQUEST_TIMEOUT_SECONDS"},
	{"limits", "raw_retention_days", "RAW_RETENTION_DAYS"},
	{"limits", "raw_request_keep_latest_per_scope", "RAW_REQUEST_KEEP_LATEST_PER_SCOPE"},
	{"limits", "aggregate_retention_days", "AGGREGATE_RETENTION_DAYS"},
	{"limits", "device_offline_seconds", "DEVICE_OFFLINE_SECONDS"},
	{"limits", "device_offline_minutes", "DEVICE_OFFLINE_MINUTES"},
	{"limits", "heartbeat_interval_seconds", "HEARTBEAT_INTERVAL_SECONDS"},
	{"limits", "presence_flush_seconds", "PRESENCE_FLUSH_SECONDS"},
	{"limits", "persist_queue_size", "PERSIST_QUEUE_SIZE"},
	{"limits", "persist_workers", "PERSIST_WORKERS"},
	{"limits", "client_queue_size", "CLIENT_QUEUE_SIZE"},
	{"limits", "client_queue_idle_ttl_seconds", "CLIENT_QUEUE_IDLE_TTL_SECONDS"},
	{"limits", "client_max_in_flight", "CLIENT_MAX_IN_FLIGHT"},
	{"limits", "queue_lease_duration_seconds", "QUEUE_LEASE_DURATION_SECONDS"},
	{"limits", "queue_lease_reaper_interval_seconds", "QUEUE_LEASE_REAPER_INTERVAL_SECONDS"},
	{"limits", "execution_grace_seconds", "EXECUTION_GRACE_SECONDS"},
	{"limits", "action_default_max_in_flight", "ACTION_DEFAULT_MAX_IN_FLIGHT"},
}

type Config struct {
	AppName                  string
	ServerID                 string
	HTTPAddr                 string
	JWTSecret                string
	RequestTimeout           time.Duration
	RawRetentionDays         int
	RawRequestKeepLatest     int
	AggregateRetentionDays   int
	DeviceOfflineSeconds     int
	DeviceOfflineMinutes     int
	HeartbeatIntervalSeconds int
	PresenceFlushSeconds     int
	PersistQueueSize         int
	PersistWorkers           int
	ClientQueueSize          int
	ClientQueueIdleTTL       time.Duration
	ClientMaxInFlight        int
	QueueLeaseDuration       time.Duration
	QueueLeaseReaperInterval time.Duration
	ExecutionGrace           time.Duration
	ActionDefaultMaxInFlight int
	TimeZone                 string
	BootstrapAdminUser       string
	BootstrapAdminPass       string
	MySQL                    MySQLConfig
	Redis                    RedisConfig
	Storage                  StorageConfig
	Logging                  LoggingConfig
}

type RedisConfig struct {
	Addr                      string
	Password                  string
	DB                        int
	WorkerConcurrency         int
	APIWorkerConcurrency      int
	DownloadWorkerConcurrency int
	ScanInterval              time.Duration
	DownloadInterval          time.Duration
}

type MySQLConfig struct {
	Host                   string
	Port                   int
	User                   string
	Password               string
	DB                     string
	Params                 string
	MaxOpenConns           int
	MaxIdleConns           int
	ConnMaxLifetimeMinutes int
}

type StorageConfig struct {
	Backend   string
	LocalPath string
	Endpoint  string
	Region    string
	Bucket    string
	PathStyle bool
	AccessKey string
	SecretKey string
}

type LoggingConfig struct {
	Dir      string
	MaxDirMB int
}

func Load() (Config, error) {
	values, path, err := loadConfigFromSearchPaths()
	if err != nil {
		return Config{}, err
	}

	cfg := Config{
		AppName:                  getString(values, "APP_NAME", "r1rpc-demo"),
		ServerID:                 getString(values, "SERVER_ID", "r1rpc-node-1"),
		HTTPAddr:                 getString(values, "HTTP_ADDR", ":8080"),
		JWTSecret:                getString(values, "JWT_SECRET", ""),
		RequestTimeout:           time.Duration(getInt(values, "REQUEST_TIMEOUT_SECONDS", 25)) * time.Second,
		RawRetentionDays:         getInt(values, "RAW_RETENTION_DAYS", 3),
		RawRequestKeepLatest:     getInt(values, "RAW_REQUEST_KEEP_LATEST_PER_SCOPE", 100),
		AggregateRetentionDays:   getInt(values, "AGGREGATE_RETENTION_DAYS", 30),
		DeviceOfflineSeconds:     getInt(values, "DEVICE_OFFLINE_SECONDS", 0),
		DeviceOfflineMinutes:     getInt(values, "DEVICE_OFFLINE_MINUTES", 0),
		HeartbeatIntervalSeconds: getInt(values, "HEARTBEAT_INTERVAL_SECONDS", 5),
		PresenceFlushSeconds:     getInt(values, "PRESENCE_FLUSH_SECONDS", 5),
		PersistQueueSize:         getInt(values, "PERSIST_QUEUE_SIZE", 4096),
		PersistWorkers:           getInt(values, "PERSIST_WORKERS", 2),
		ClientQueueSize:          getInt(values, "CLIENT_QUEUE_SIZE", 256),
		ClientQueueIdleTTL:       time.Duration(getInt(values, "CLIENT_QUEUE_IDLE_TTL_SECONDS", 600)) * time.Second,
		ClientMaxInFlight:        getInt(values, "CLIENT_MAX_IN_FLIGHT", 1),
		QueueLeaseDuration:       time.Duration(getInt(values, "QUEUE_LEASE_DURATION_SECONDS", 10)) * time.Second,
		QueueLeaseReaperInterval: time.Duration(getInt(values, "QUEUE_LEASE_REAPER_INTERVAL_SECONDS", 1)) * time.Second,
		ExecutionGrace:           time.Duration(getInt(values, "EXECUTION_GRACE_SECONDS", 30)) * time.Second,
		ActionDefaultMaxInFlight: getInt(values, "ACTION_DEFAULT_MAX_IN_FLIGHT", 1),
		TimeZone:                 getString(values, "TIME_ZONE", "Asia/Shanghai"),
		BootstrapAdminUser:       getString(values, "BOOTSTRAP_ADMIN_USERNAME", "admin"),
		BootstrapAdminPass:       getString(values, "BOOTSTRAP_ADMIN_PASSWORD", ""),
		MySQL: MySQLConfig{
			Host:                   getString(values, "MYSQL_HOST", "mysql"),
			Port:                   getInt(values, "MYSQL_PORT", 3306),
			User:                   getString(values, "MYSQL_USER", "root"),
			Password:               getString(values, "MYSQL_PASSWORD", ""),
			DB:                     getString(values, "MYSQL_DB", "r1rpc"),
			Params:                 getString(values, "MYSQL_PARAMS", "charset=utf8mb4&parseTime=true&loc=Asia%2FShanghai&timeout=5s&readTimeout=30s&writeTimeout=30s&clientFoundRows=true"),
			MaxOpenConns:           getInt(values, "MYSQL_MAX_OPEN_CONNS", 32),
			MaxIdleConns:           getInt(values, "MYSQL_MAX_IDLE_CONNS", 8),
			ConnMaxLifetimeMinutes: getInt(values, "MYSQL_CONN_MAX_LIFETIME_MINUTES", 10),
		},
		Redis: RedisConfig{
			Addr:                      getString(values, "REDIS_ADDR", "127.0.0.1:6379"),
			Password:                  getString(values, "REDIS_PASSWORD", ""),
			DB:                        getInt(values, "REDIS_DB", 0),
			WorkerConcurrency:         getInt(values, "REDIS_WORKER_CONCURRENCY", 4),
			APIWorkerConcurrency:      getInt(values, "REDIS_API_WORKER_CONCURRENCY", 2),
			DownloadWorkerConcurrency: getInt(values, "REDIS_DOWNLOAD_WORKER_CONCURRENCY", 6),
			ScanInterval:              time.Duration(getInt(values, "QUEUE_SCAN_INTERVAL_SECONDS", 5)) * time.Second,
			DownloadInterval:          time.Duration(getInt(values, "DOWNLOAD_SCAN_INTERVAL_SECONDS", 2)) * time.Second,
		},
		Storage: StorageConfig{
			Backend:   strings.ToLower(getString(values, "STORAGE_BACKEND", "file")),
			LocalPath: getString(values, "STORAGE_LOCAL_PATH", "./data/uploads"),
			Endpoint:  getString(values, "STORAGE_S3_ENDPOINT", ""),
			Region:    getString(values, "STORAGE_S3_REGION", "us-east-1"),
			Bucket:    getString(values, "STORAGE_S3_BUCKET", ""),
			PathStyle: getBool(values, "STORAGE_S3_PATH_STYLE", true),
			AccessKey: getString(values, "STORAGE_S3_ACCESS_KEY", ""),
			SecretKey: getString(values, "STORAGE_S3_SECRET_KEY", ""),
		},
		Logging: LoggingConfig{
			Dir:      getString(values, "LOG_DIR", "./data/logs"),
			MaxDirMB: getInt(values, "LOG_MAX_DIR_MB", 200),
		},
	}

	// 对外 RPC 调用的鉴权（none/apikey）已下沉到「分组」级别，不再走全局配置。

	if strings.TrimSpace(cfg.JWTSecret) == "" {
		return Config{}, fmt.Errorf("%s 中必须配置 JWT_SECRET", path)
	}
	if strings.TrimSpace(cfg.BootstrapAdminPass) == "" {
		return Config{}, fmt.Errorf("%s 中必须配置 BOOTSTRAP_ADMIN_PASSWORD", path)
	}
	if strings.TrimSpace(cfg.MySQL.Host) == "" {
		return Config{}, fmt.Errorf("%s 中必须配置 MYSQL_HOST", path)
	}
	if strings.TrimSpace(cfg.MySQL.User) == "" {
		return Config{}, fmt.Errorf("%s 中必须配置 MYSQL_USER", path)
	}
	if strings.TrimSpace(cfg.MySQL.DB) == "" {
		return Config{}, fmt.Errorf("%s 中必须配置 MYSQL_DB", path)
	}
	if strings.TrimSpace(cfg.Logging.Dir) == "" {
		return Config{}, fmt.Errorf("%s 中 LOG_DIR 不能为空", path)
	}
	if cfg.Logging.MaxDirMB < 1 {
		return Config{}, fmt.Errorf("%s 中 LOG_MAX_DIR_MB 必须大于 0", path)
	}
	if cfg.RawRetentionDays <= 0 {
		cfg.RawRetentionDays = 3
	}
	if cfg.RawRequestKeepLatest <= 0 {
		cfg.RawRequestKeepLatest = 100
	}
	if cfg.AggregateRetentionDays <= 0 {
		cfg.AggregateRetentionDays = 30
	}
	if cfg.DeviceOfflineSeconds <= 0 {
		if cfg.DeviceOfflineMinutes > 0 {
			cfg.DeviceOfflineSeconds = cfg.DeviceOfflineMinutes * 60
		} else {
			cfg.DeviceOfflineSeconds = 20
		}
	}
	cfg.DeviceOfflineMinutes = cfg.DeviceOfflineSeconds / 60
	if cfg.HeartbeatIntervalSeconds <= 0 {
		cfg.HeartbeatIntervalSeconds = 5
	}
	if cfg.HeartbeatIntervalSeconds >= cfg.DeviceOfflineSeconds {
		cfg.HeartbeatIntervalSeconds = maxInt(1, cfg.DeviceOfflineSeconds/2)
	}
	if cfg.PresenceFlushSeconds <= 0 {
		cfg.PresenceFlushSeconds = minInt(5, cfg.DeviceOfflineSeconds)
	}
	if cfg.PresenceFlushSeconds >= cfg.DeviceOfflineSeconds {
		cfg.PresenceFlushSeconds = maxInt(1, cfg.DeviceOfflineSeconds/2)
	}
	if cfg.PersistQueueSize < 256 {
		cfg.PersistQueueSize = 256
	}
	if cfg.PersistWorkers < 1 {
		cfg.PersistWorkers = 1
	}
	if cfg.ClientQueueSize < 64 {
		cfg.ClientQueueSize = 64
	}
	if cfg.ClientQueueIdleTTL <= 0 {
		cfg.ClientQueueIdleTTL = 10 * time.Minute
	}
	if cfg.ClientMaxInFlight < 1 {
		cfg.ClientMaxInFlight = 1
	}
	if cfg.QueueLeaseDuration <= 0 {
		cfg.QueueLeaseDuration = 10 * time.Second
	}
	if cfg.QueueLeaseReaperInterval <= 0 {
		cfg.QueueLeaseReaperInterval = time.Second
	}
	if cfg.ExecutionGrace <= 0 {
		cfg.ExecutionGrace = 30 * time.Second
	}
	if cfg.ActionDefaultMaxInFlight < 1 {
		cfg.ActionDefaultMaxInFlight = 1
	}
	if cfg.MySQL.MaxOpenConns < 4 {
		cfg.MySQL.MaxOpenConns = 4
	}
	if cfg.MySQL.MaxIdleConns < 2 {
		cfg.MySQL.MaxIdleConns = 2
	}
	if cfg.MySQL.ConnMaxLifetimeMinutes <= 0 {
		cfg.MySQL.ConnMaxLifetimeMinutes = 10
	}
	if cfg.TimeZone == "" {
		cfg.TimeZone = "Asia/Shanghai"
	}
	if cfg.Storage.Backend != "file" && cfg.Storage.Backend != "s3" {
		return Config{}, fmt.Errorf("%s 中 STORAGE_BACKEND 仅支持 file 或 s3", path)
	}
	if cfg.Storage.Backend == "file" && strings.TrimSpace(cfg.Storage.LocalPath) == "" {
		cfg.Storage.LocalPath = "./data/uploads"
	}
	if cfg.Storage.Backend == "s3" && strings.TrimSpace(cfg.Storage.Bucket) == "" {
		return Config{}, fmt.Errorf("%s 中 S3 存储必须配置 STORAGE_S3_BUCKET", path)
	}
	if err := cfg.ValidateExecutionTiming(); err != nil {
		return Config{}, fmt.Errorf("%s 中执行时序配置无效: %w", path, err)
	}
	return cfg, nil
}

func (c Config) ValidateExecutionTiming() error {
	if c.RequestTimeout <= 0 {
		return fmt.Errorf("request timeout must be positive")
	}
	if c.QueueLeaseDuration <= 0 {
		return fmt.Errorf("queue lease duration must be positive")
	}
	if c.QueueLeaseReaperInterval <= 0 || c.QueueLeaseReaperInterval >= c.QueueLeaseDuration {
		return fmt.Errorf("queue lease reaper interval must be positive and below queue lease duration")
	}
	if c.QueueLeaseDuration > c.RequestTimeout {
		return fmt.Errorf("queue lease duration must not exceed request timeout")
	}
	if c.ExecutionGrace < c.QueueLeaseReaperInterval {
		return fmt.Errorf("execution grace must be at least the queue lease reaper interval")
	}
	return nil
}

func (c Config) MySQLDSN(withoutDB bool) string {
	dbName := c.MySQL.DB
	if withoutDB {
		dbName = ""
	}
	if dbName != "" {
		return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?%s", c.MySQL.User, c.MySQL.Password, c.MySQL.Host, c.MySQL.Port, dbName, c.MySQL.Params)
	}
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/?%s", c.MySQL.User, c.MySQL.Password, c.MySQL.Host, c.MySQL.Port, c.MySQL.Params)
}

func (c Config) LoadLocation() (*time.Location, error) {
	return time.LoadLocation(c.TimeZone)
}

func (c Config) ApplyTimeZone() error {
	loc, err := c.LoadLocation()
	if err != nil {
		return err
	}
	time.Local = loc
	return nil
}

func (c Config) TimeZoneOffsetString() string {
	loc, err := c.LoadLocation()
	if err != nil {
		loc = time.FixedZone("UTC+8", 8*3600)
	}
	_, offsetSeconds := time.Now().In(loc).Zone()
	sign := "+"
	if offsetSeconds < 0 {
		sign = "-"
		offsetSeconds = -offsetSeconds
	}
	hours := offsetSeconds / 3600
	minutes := (offsetSeconds % 3600) / 60
	return fmt.Sprintf("%s%02d:%02d", sign, hours, minutes)
}

// loadConfigFromSearchPaths 读取配置文件（可选）。
// 找不到文件不再报错——允许纯环境变量部署；缺失的必填项由 Load 的校验兜底。
func loadConfigFromSearchPaths() (map[string]string, string, error) {
	for _, candidate := range configSearchPaths {
		if _, err := os.Stat(candidate); err == nil {
			values, parsedPath, parseErr := parseConfigFile(candidate)
			if parseErr != nil {
				return nil, parsedPath, parseErr
			}
			return values, parsedPath, nil
		}
	}
	return map[string]string{}, "environment", nil
}

// parseConfigFile 读取分组式 YAML 配置，拍平成环境变量风格的键值 map。
// 缺省的字段不写入 map，由 Load 的默认值兜底。
func parseConfigFile(name string) (map[string]string, string, error) {
	path, err := filepath.Abs(name)
	if err != nil {
		path = name
	}
	raw, err := os.ReadFile(name)
	if err != nil {
		return nil, path, fmt.Errorf("打开配置文件 %s 失败: %w", path, err)
	}
	doc := map[string]map[string]any{}
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return nil, path, fmt.Errorf("解析 YAML 配置 %s 失败: %w", path, err)
	}
	values := map[string]string{}
	for _, m := range yamlToEnvKey {
		section, ok := doc[m.Section]
		if !ok {
			continue
		}
		v, ok := section[m.Key]
		if !ok || v == nil {
			continue
		}
		values[m.Env] = strings.TrimSpace(fmt.Sprintf("%v", v))
	}
	return values, path, nil
}

// getString 取值优先级：环境变量 > 配置文件 > 默认值。
// 这样 docker --env-file / compose environment 注入的变量能直接生效。
func getString(values map[string]string, key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	if value := strings.TrimSpace(values[key]); value != "" {
		return value
	}
	return fallback
}

// getInt 取值优先级同 getString：环境变量 > 配置文件 > 默认值。
func getInt(values map[string]string, key string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		raw = strings.TrimSpace(values[key])
	}
	if raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			return parsed
		}
	}
	return fallback
}

func getBool(values map[string]string, key string, fallback bool) bool {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		raw = strings.TrimSpace(values[key])
	}
	if raw == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(raw)
	if err != nil {
		return fallback
	}
	return parsed
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
