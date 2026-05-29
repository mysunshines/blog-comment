package config

import (
	"fmt"
	"os"

	commonconfig "github.com/mysunshines/gocommon/config"
	"gopkg.in/yaml.v3"
)

// 共享类型别名 — 直接复用 common/config，零映射开销
type (
	AppConfig       = commonconfig.AppConfig
	DatabaseConfig  = commonconfig.DatabaseConfig
	RedisConfig     = commonconfig.RedisConfig
	JWTConfig       = commonconfig.JWTConfig
	GRPCConfig      = commonconfig.GRPCConfig
	HTTPConfig      = commonconfig.HTTPConfig
	ConsulConfig    = commonconfig.ConsulConfig
	MetricsConfig   = commonconfig.MetricsConfig
	RateLimitConfig = commonconfig.RateLimitConfig
)

// Config 评论服务配置
type Config struct {
	App         AppConfig         `yaml:"app"`
	Database    DatabaseConfig    `yaml:"database"`
	Redis       RedisConfig       `yaml:"redis"`
	HTTP        HTTPConfig        `yaml:"http"`
	GRPC        GRPCConfig        `yaml:"grpc"`
	Consul      ConsulConfig      `yaml:"consul"`
	UserService UserServiceConfig `yaml:"user_service"`
	Metrics     MetricsConfig     `yaml:"metrics"`
	JWT         JWTConfig         `yaml:"jwt"`
	RateLimit   RateLimitConfig   `yaml:"rate_limit"`
}

// UserServiceConfig 用户服务配置（comment-service 独有）
type UserServiceConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

// Addr 返回用户服务地址
func (u *UserServiceConfig) Addr() string {
	return fmt.Sprintf("%s:%d", u.Host, u.Port)
}

var globalConfig *Config

// Load 加载配置文件
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var c Config
	if err := yaml.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	setDefaults(&c)
	globalConfig = &c
	return &c, nil
}

// Get 获取全局配置
func Get() *Config {
	return globalConfig
}

func setDefaults(c *Config) {
	if c.App.Env == "" {
		c.App.Env = "dev"
	}
	if c.App.Name == "" {
		c.App.Name = "comment-service"
	}
	if c.App.LogDir == "" {
		c.App.LogDir = "/var/log"
	}
	if c.App.LogLevel == "" {
		c.App.LogLevel = "info"
	}
	if c.Database.MaxOpenConns == 0 {
		c.Database.MaxOpenConns = 100
	}
	if c.Database.MaxIdleConns == 0 {
		c.Database.MaxIdleConns = 10
	}
	if c.Database.ConnMaxLifetime == 0 {
		c.Database.ConnMaxLifetime = 3600
	}
	if c.HTTP.Host == "" {
		c.HTTP.Host = "0.0.0.0"
	}
	if c.HTTP.Port == 0 {
		c.HTTP.Port = 8080
	}
	if c.GRPC.Host == "" {
		c.GRPC.Host = "0.0.0.0"
	}
	if c.GRPC.Port == 0 {
		c.GRPC.Port = 9000
	}
	if c.Metrics.Port == 0 {
		c.Metrics.Port = 9090
	}
	if c.Metrics.Path == "" {
		c.Metrics.Path = "/metrics"
	}
	if c.JWT.Secret == "" {
		c.JWT.Secret = "your-secret-key-change-in-production"
	}
	if c.JWT.ExpireTime == 0 {
		c.JWT.ExpireTime = 86400 // 24小时
	}
	if c.RateLimit.QPS == 0 {
		c.RateLimit.QPS = 100
	}
	if c.RateLimit.Burst == 0 {
		c.RateLimit.Burst = 200
	}
}
