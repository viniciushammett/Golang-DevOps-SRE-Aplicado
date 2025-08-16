package config

import (
	"errors"
	"os"

	"gopkg.in/yaml.v3"
)

type ServerCfg struct {
	Addr        string   `yaml:"addr"`
	CORSOrigins []string `yaml:"cors_origins"`
}

type SecurityCfg struct {
	MasterKeyB64 string `yaml:"master_key_b64"`
	JWTSecretB64 string `yaml:"jwt_secret_b64"`
}

type AuditCfg struct { File string `yaml:"file"` }

type StorageCfg struct { BoltPath string `yaml:"bolt_path"` }

type User struct {
	Username       string   `yaml:"username"`
	PasswordBcrypt string   `yaml:"password_bcrypt"`
	Roles          []string `yaml:"roles"`
}

type TransitCfg struct {
	Enabled bool   `yaml:"enabled"`
	BaseURL string `yaml:"base_url"`
	Token   string `yaml:"token"`
}

type Config struct {
	Server   ServerCfg   `yaml:"server"`
	Security SecurityCfg `yaml:"security"`
	Audit    AuditCfg    `yaml:"audit"`
	Storage  StorageCfg  `yaml:"storage"`
	Users    []User      `yaml:"users"`
	Transit  TransitCfg  `yaml:"transit"`
}

func Load(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if err != nil { return nil, err }
	var c Config
	if err := yaml.Unmarshal(b, &c); err != nil { return nil, err }
	// Env overrides
	if v := os.Getenv("GSV_MASTER_KEY"); v != "" { c.Security.MasterKeyB64 = v }
	if v := os.Getenv("GSV_JWT_SECRET"); v != "" { c.Security.JWTSecretB64 = v }
	if c.Security.MasterKeyB64 == "" || c.Security.JWTSecretB64 == "" {
		return nil, errors.New("missing master/jwt secrets; set in config or env")
	}
	if c.Server.Addr == "" { c.Server.Addr = ":8080" }
	if c.Storage.BoltPath == "" { c.Storage.BoltPath = "./data/vault.db" }
	if c.Audit.File == "" { c.Audit.File = "./data/audit.log" }
	return &c, nil
}