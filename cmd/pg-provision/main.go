package main

import (
	"log"
	"net/http"

	"pg_provision_dbuser_codex/internal/app"
	"pg_provision_dbuser_codex/internal/config"
)

func main() {
	cfg, err := config.Load(".env")
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	server, err := app.New(cfg)
	if err != nil {
		log.Fatalf("初始化应用失败: %v", err)
	}

	log.Printf("PostgreSQL 用户开通工具启动: http://%s", cfg.AppAddr)
	if err := http.ListenAndServe(cfg.AppAddr, server.Routes()); err != nil {
		log.Fatalf("HTTP 服务退出: %v", err)
	}
}
