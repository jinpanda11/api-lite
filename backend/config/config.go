package config

import (
	"log"

	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	JWT      JWTConfig      `mapstructure:"jwt"`
	SMTP     SMTPConfig     `mapstructure:"smtp"`
	Admin    AdminConfig    `mapstructure:"admin"`
}

type ServerConfig struct {
	Port  int  `mapstructure:"port"`
	Debug bool `mapstructure:"debug"`
}

type DatabaseConfig struct {
	Driver string `mapstructure:"driver"`
	DSN    string `mapstructure:"dsn"`
}

type JWTConfig struct {
	Secret      string `mapstructure:"secret"`
	ExpireHours int    `mapstructure:"expire_hours"`
}

type SMTPConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	From     string `mapstructure:"from"`
	SSL      bool   `mapstructure:"ssl"`
}

type AdminConfig struct {
	Username string `mapstructure:"username"`
	Email    string `mapstructure:"email"`
	Password string `mapstructure:"password"`
}

var C Config

func Load(path string) {
	viper.SetConfigFile(path)
	viper.SetConfigType("yaml")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("failed to read config: %v", err)
	}
	if err := viper.Unmarshal(&C); err != nil {
		log.Fatalf("failed to unmarshal config: %v", err)
	}
}
