package config

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	AppAddr          string
	AppLoginUser     string
	AppLoginKey      string
	AppSessionSecret string
	Postgres         PostgresConfig
}

type PostgresConfig struct {
	Host          string
	Port          string
	AdminDB       string
	SuperUser     string
	SuperPassword string
	SSLMode       string
}

func Load(path string) (Config, error) {
	values, err := parseEnvFile(path)
	if err != nil {
		return Config{}, err
	}
	return fromValues(values)
}

func fromValues(values map[string]string) (Config, error) {
	cfg := Config{
		AppAddr: getDefault(values, "APP_ADDR", "127.0.0.1:8080"),
		Postgres: PostgresConfig{
			Port:    getDefault(values, "PG_PORT", "5432"),
			AdminDB: getDefault(values, "PG_ADMIN_DB", "postgres"),
			SSLMode: getDefault(values, "PG_SSLMODE", "disable"),
		},
	}

	cfg.AppLoginUser = strings.TrimSpace(values["APP_LOGIN_USER"])
	cfg.AppLoginKey = values["APP_LOGIN_KEY"]
	cfg.AppSessionSecret = values["APP_SESSION_SECRET"]
	cfg.Postgres.Host = strings.TrimSpace(values["PG_HOST"])
	cfg.Postgres.SuperUser = strings.TrimSpace(values["PG_SUPERUSER"])
	cfg.Postgres.SuperPassword = values["PG_SUPER_PASSWORD"]

	if err := cfg.validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (cfg Config) validate() error {
	required := map[string]string{
		"APP_LOGIN_USER":     cfg.AppLoginUser,
		"APP_LOGIN_KEY":      cfg.AppLoginKey,
		"APP_SESSION_SECRET": cfg.AppSessionSecret,
		"PG_HOST":            cfg.Postgres.Host,
		"PG_SUPERUSER":       cfg.Postgres.SuperUser,
		"PG_SUPER_PASSWORD":  cfg.Postgres.SuperPassword,
	}

	var missing []string
	for key, value := range required {
		if value == "" {
			missing = append(missing, key)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("缺少必要配置: %s", strings.Join(missing, ", "))
	}
	if len(cfg.AppSessionSecret) < 16 {
		return errors.New("APP_SESSION_SECRET 长度至少需要 16 字符")
	}
	if _, err := strconv.Atoi(cfg.Postgres.Port); err != nil {
		return fmt.Errorf("PG_PORT 必须是数字: %w", err)
	}
	return nil
}

func (cfg Config) TargetSummary() string {
	return fmt.Sprintf("%s:%s/%s", cfg.Postgres.Host, cfg.Postgres.Port, cfg.Postgres.AdminDB)
}

func getDefault(values map[string]string, key, fallback string) string {
	value := strings.TrimSpace(values[key])
	if value == "" {
		return fallback
	}
	return value
}

func parseEnvFile(path string) (map[string]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("读取 .env 失败: %w", err)
	}
	defer file.Close()

	values := make(map[string]string)
	scanner := bufio.NewScanner(file)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			return nil, fmt.Errorf(".env 第 %d 行缺少等号", lineNumber)
		}
		key = strings.TrimSpace(key)
		if key == "" {
			return nil, fmt.Errorf(".env 第 %d 行缺少配置名", lineNumber)
		}

		values[key] = trimQuotes(strings.TrimSpace(value))
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("读取 .env 失败: %w", err)
	}
	return values, nil
}

func trimQuotes(value string) string {
	if len(value) < 2 {
		return value
	}
	if (value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'') {
		return value[1 : len(value)-1]
	}
	return value
}
