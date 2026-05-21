package main

import (
	"log"
	"net/http"

	"pg_provision_dbuser_codex/internal/app"
)

func main() {
	addr := "127.0.0.1:8080"
	server, err := app.New()
	if err != nil {
		log.Fatalf("初始化应用失败: %v", err)
	}

	log.Printf("PostgreSQL 用户开通工具启动: http://%s", addr)
	if err := http.ListenAndServe(addr, server.Routes()); err != nil {
		log.Fatalf("HTTP 服务退出: %v", err)
	}
}
