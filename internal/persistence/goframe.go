package persistence

import (
	"context"
	"strconv"
	"time"

	"r1rpc/internal/config"

	_ "github.com/gogf/gf/contrib/drivers/mysql/v2"
	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
)

const defaultGroup = "default"

// ConfigureGoFrameDB 复用现有配置初始化 GoFrame ORM。
// 基础迁移阶段仍保留 database/sql Store，新图片领域通过生成 DAO 使用同一 MySQL 实例。
func ConfigureGoFrameDB(ctx context.Context, cfg config.Config) error {
	node := gdb.ConfigNode{
		Type:             "mysql",
		Host:             cfg.MySQL.Host,
		Port:             strconv.Itoa(cfg.MySQL.Port),
		User:             cfg.MySQL.User,
		Pass:             cfg.MySQL.Password,
		Name:             cfg.MySQL.DB,
		Extra:            cfg.MySQL.Params,
		Charset:          "utf8mb4",
		Timezone:         cfg.TimeZone,
		MaxOpenConnCount: cfg.MySQL.MaxOpenConns,
		MaxIdleConnCount: cfg.MySQL.MaxIdleConns,
		MaxConnLifeTime:  time.Duration(cfg.MySQL.ConnMaxLifetimeMinutes) * time.Minute,
		CreatedAt:        "created_at",
		UpdatedAt:        "updated_at",
		DeletedAt:        "deleted_at",
	}
	if err := gdb.SetConfig(gdb.Config{defaultGroup: gdb.ConfigGroup{node}}); err != nil {
		return err
	}
	_, err := g.DB(defaultGroup).GetAll(ctx, "SELECT 1")
	return err
}
