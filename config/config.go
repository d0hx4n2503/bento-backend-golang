package config

import (
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config contains the configuration shared by the API process.
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Postgres PostgresConfig `mapstructure:"postgres"`
	Redis    RedisConfig    `mapstructure:"redis"`
	MongoDB  MongoDBConfig  `mapstructure:"mongodb"`
	Cookie   CookieConfig   `mapstructure:"cookie"`
	Store    StoreConfig    `mapstructure:"store"`
	Session  SessionConfig  `mapstructure:"session"`
	Metrics  MetricsConfig  `mapstructure:"metrics"`
	Tracing  TracingConfig  `mapstructure:"tracing"`
	Logger   LoggerConfig   `mapstructure:"logger"`
	AWS      AWSConfig      `mapstructure:"aws"`
}

type ServerConfig struct {
	AppVersion     string        `mapstructure:"app_version"`
	Port           string        `mapstructure:"port"`
	PprofPort      string        `mapstructure:"pprof_port"`
	Mode           string        `mapstructure:"mode"`
	JWTSecretKey   string        `mapstructure:"jwt_secret_key"`
	CSRFSecretKey  string        `mapstructure:"csrf_secret_key"`
	CookieName     string        `mapstructure:"cookie_name"`
	ReadTimeout    time.Duration `mapstructure:"read_timeout"`
	WriteTimeout   time.Duration `mapstructure:"write_timeout"`
	SSL            bool          `mapstructure:"ssl"`
	ContextTimeout time.Duration `mapstructure:"context_timeout"`
	CSRF           bool          `mapstructure:"csrf"`
	Debug          bool          `mapstructure:"debug"`
}

type LoggerConfig struct {
	Development       bool   `mapstructure:"development"`
	DisableCaller     bool   `mapstructure:"disable_caller"`
	DisableStacktrace bool   `mapstructure:"disable_stacktrace"`
	Encoding          string `mapstructure:"encoding"`
	Level             string `mapstructure:"level"`
}

type PostgresConfig struct {
	Host     string `mapstructure:"host"`
	Port     string `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	Database string `mapstructure:"database"`
	SSLMode  string `mapstructure:"ssl_mode"`
	Driver   string `mapstructure:"driver"`
}

type RedisConfig struct {
	Address      string        `mapstructure:"address"`
	Password     string        `mapstructure:"password"`
	DB           int           `mapstructure:"db"`
	MinIdleConns int           `mapstructure:"min_idle_conns"`
	PoolSize     int           `mapstructure:"pool_size"`
	PoolTimeout  time.Duration `mapstructure:"pool_timeout"`
}

type MongoDBConfig struct {
	URI string `mapstructure:"uri"`
}

type CookieConfig struct {
	Name     string `mapstructure:"name"`
	MaxAge   int    `mapstructure:"max_age"`
	Secure   bool   `mapstructure:"secure"`
	HTTPOnly bool   `mapstructure:"http_only"`
}

type SessionConfig struct {
	Prefix string        `mapstructure:"prefix"`
	Name   string        `mapstructure:"name"`
	Expire time.Duration `mapstructure:"expire"`
}

type MetricsConfig struct {
	Address     string `mapstructure:"address"`
	ServiceName string `mapstructure:"service_name"`
}

type TracingConfig struct {
	Enabled       bool    `mapstructure:"enabled"`
	Endpoint      string  `mapstructure:"endpoint"`
	ServiceName   string  `mapstructure:"service_name"`
	ServerAddress string  `mapstructure:"server_address"`
	Insecure      bool    `mapstructure:"insecure"`
	SamplingRatio float64 `mapstructure:"sampling_ratio"`
}

type StoreConfig struct {
	ImagesFolder string `mapstructure:"images_folder"`
}

type AWSConfig struct {
	Endpoint      string `mapstructure:"endpoint"`
	AccessKey     string `mapstructure:"access_key"`
	SecretKey     string `mapstructure:"secret_key"`
	UseSSL        bool   `mapstructure:"use_ssl"`
	MinioEndpoint string `mapstructure:"minio_endpoint"`
}

// LoadConfig reads a config file. filename can be a path or a file name such
// as "config-local.yml"; file names are searched for in the current directory
// and the config/ directory. Environment variables use the APP_ prefix and
// uppercase underscore-separated keys, for example APP_SERVER_PORT.
func LoadConfig(filename string) (*viper.Viper, error) {
	if strings.TrimSpace(filename) == "" {
		return nil, fmt.Errorf("config filename is required")
	}

	v := viper.New()
	v.SetDefault("tracing.service_name", "bento-backend-golang")
	v.SetDefault("tracing.sampling_ratio", 0.1)
	if filepath.IsAbs(filename) || strings.ContainsAny(filename, `/\\`) {
		v.SetConfigFile(filename)
	} else {
		extension := filepath.Ext(filename)
		v.SetConfigName(strings.TrimSuffix(filename, extension))
		if extension != "" {
			v.SetConfigType(strings.TrimPrefix(extension, "."))
		}
		v.AddConfigPath(".")
		v.AddConfigPath("config")
	}

	v.SetEnvPrefix("APP")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()
	if err := bindEnvironment(v, "", reflect.TypeOf(Config{})); err != nil {
		return nil, fmt.Errorf("bind config environment: %w", err)
	}

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	return v, nil
}

// ParseConfig decodes Viper values into Config.
func ParseConfig(v *viper.Viper) (*Config, error) {
	if v == nil {
		return nil, fmt.Errorf("viper config is nil")
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("decode config: %w", err)
	}
	return &cfg, nil
}

func bindEnvironment(v *viper.Viper, prefix string, typ reflect.Type) error {
	durationType := reflect.TypeOf(time.Duration(0))
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if field.PkgPath != "" {
			continue
		}

		name := strings.SplitN(field.Tag.Get("mapstructure"), ",", 2)[0]
		if name == "" || name == "-" {
			continue
		}

		key := name
		if prefix != "" {
			key = prefix + "." + name
		}
		if field.Type.Kind() == reflect.Struct && field.Type != durationType {
			if err := bindEnvironment(v, key, field.Type); err != nil {
				return err
			}
			continue
		}
		if err := v.BindEnv(key); err != nil {
			return fmt.Errorf("bind %s: %w", key, err)
		}
	}
	return nil
}
