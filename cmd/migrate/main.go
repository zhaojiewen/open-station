package main

import (
	"flag"
	"fmt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	
	"github.com/zhaojiewen/open-station/internal/domain/entity"
)

var configPath = flag.String("config", "", "config file path (not used)")

func main() {
	flag.Parse()
	
	dsn := "host=localhost port=5432 user=postgres password=postgres dbname=ai_gateway sslmode=disable"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		panic(fmt.Sprintf("failed to connect database: %v", err))
	}
	
	// 先修复api_keys表的quota_id问题 - 添加默认值
	result := db.Exec("ALTER TABLE api_keys ALTER COLUMN quota_id SET DEFAULT '00000000-0000-0000-0000-000000000000'")
	if result.Error != nil {
		fmt.Println("Warning: could not set default for quota_id:", result.Error)
	}
	
	// 然后迁移所有auth相关的表
	err = db.AutoMigrate(
		&entity.UserTenant{},
		&entity.LoginAudit{},
		&entity.PasswordHistory{},
		&entity.RefreshToken{},
	)
	if err != nil {
		fmt.Println("Migration error:", err)
	} else {
		fmt.Println("Auth tables created successfully!")
	}
	
	// 验证表是否存在
	var tables []string
	db.Raw("SELECT tablename FROM pg_tables WHERE schemaname = 'public'").Scan(&tables)
	fmt.Println("Existing tables:", tables)
}
