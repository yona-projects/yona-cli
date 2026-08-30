// Package config은 yona CLI의 로컬 설정 파일(~/.config/yona-cli/config.yml)을 읽고 쓴다.
// gh CLI의 ~/.config/gh/hosts.yml 패턴을 참고했다 — 서버(호스트) URL을 키로 삼아 여러 yona
// 서버에 대한 인증 정보를 동시에 보관할 수 있게 한다.
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Host는 한 yona 서버에 대해 저장된 인증 정보다.
type Host struct {
	Token string `yaml:"token"`
}

// Config는 config.yml 파일 전체 내용을 표현한다.
type Config struct {
	CurrentHost string          `yaml:"current_host,omitempty"`
	Hosts       map[string]Host `yaml:"hosts,omitempty"`
}

// ConfigDirEnvVar를 설정하면 기본 설정 디렉터리(~/.config/yona-cli) 대신 이 값을 사용한다.
// 테스트에서 실제 홈 디렉터리를 건드리지 않기 위한 용도다.
const ConfigDirEnvVar = "YONA_CLI_CONFIG_DIR"

// ServerEnvVar는 --server 플래그가 없을 때 사용할 기본 서버 URL을 지정하는 환경변수다.
const ServerEnvVar = "YONA_HOST"

// TokenEnvVar는 설정 파일을 거치지 않고 바로 토큰을 넘길 때 쓰는 환경변수다(CI 등에서 유용).
const TokenEnvVar = "YONA_TOKEN"

// Dir은 설정 파일이 위치한 디렉터리 경로를 반환한다.
func Dir() (string, error) {
	if dir := os.Getenv(ConfigDirEnvVar); dir != "" {
		return dir, nil
	}
	base, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("설정 디렉터리를 확인할 수 없습니다: %w", err)
	}
	return filepath.Join(base, "yona-cli"), nil
}

// Path는 config.yml 파일의 전체 경로를 반환한다.
func Path() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.yml"), nil
}

// Load는 설정 파일을 읽는다. 파일이 없으면 오류 없이 빈 Config를 반환한다(최초 실행 대응).
func Load() (*Config, error) {
	path, err := Path()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{Hosts: map[string]Host{}}, nil
		}
		return nil, fmt.Errorf("설정 파일을 읽을 수 없습니다(%s): %w", path, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("설정 파일 형식이 올바르지 않습니다(%s): %w", path, err)
	}
	if cfg.Hosts == nil {
		cfg.Hosts = map[string]Host{}
	}
	return &cfg, nil
}

// Save는 설정을 config.yml에 기록한다. 토큰이 담기므로 파일 권한을 0600으로 제한한다.
func Save(cfg *Config) error {
	dir, err := Dir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("설정 디렉터리를 만들 수 없습니다(%s): %w", dir, err)
	}

	path, err := Path()
	if err != nil {
		return err
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("설정을 직렬화할 수 없습니다: %w", err)
	}

	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("설정 파일을 쓸 수 없습니다(%s): %w", path, err)
	}
	return nil
}

// SetHost는 서버 URL에 대한 토큰을 저장하고, 이를 현재(current) 호스트로 지정한다.
func (c *Config) SetHost(server, token string) {
	if c.Hosts == nil {
		c.Hosts = map[string]Host{}
	}
	c.Hosts[server] = Host{Token: token}
	c.CurrentHost = server
}

// RemoveHost는 서버 URL에 대한 저장된 인증 정보를 삭제한다. 그 서버가 current host였다면
// current host도 함께 지운다(다른 host가 남아 있어도 자동으로 승격하지 않음 — 명시적 재로그인 유도).
func (c *Config) RemoveHost(server string) {
	delete(c.Hosts, server)
	if c.CurrentHost == server {
		c.CurrentHost = ""
	}
}

// ErrNoHost는 로그인된 서버가 하나도 없을 때 반환된다.
var ErrNoHost = fmt.Errorf("로그인된 yona 서버가 없습니다 — 먼저 'yona auth login'을 실행하세요")

// ResolveServer는 명령행 --server 플래그, YONA_HOST 환경변수, 설정 파일의 current_host 순으로
// 사용할 서버 URL을 결정한다.
func ResolveServer(flagServer string) (string, error) {
	if flagServer != "" {
		return flagServer, nil
	}
	if envServer := os.Getenv(ServerEnvVar); envServer != "" {
		return envServer, nil
	}
	cfg, err := Load()
	if err != nil {
		return "", err
	}
	if cfg.CurrentHost == "" {
		return "", ErrNoHost
	}
	return cfg.CurrentHost, nil
}

// ResolveToken은 --token 플래그, YONA_TOKEN 환경변수, 설정 파일에 저장된 서버별 토큰 순으로
// 사용할 토큰을 결정한다.
func ResolveToken(flagServer, flagToken string) (server string, token string, err error) {
	if flagToken != "" {
		server, err = ResolveServer(flagServer)
		if err != nil {
			return "", "", err
		}
		return server, flagToken, nil
	}
	if envToken := os.Getenv(TokenEnvVar); envToken != "" {
		server, err = ResolveServer(flagServer)
		if err != nil {
			return "", "", err
		}
		return server, envToken, nil
	}

	server, err = ResolveServer(flagServer)
	if err != nil {
		return "", "", err
	}
	cfg, err := Load()
	if err != nil {
		return "", "", err
	}
	host, ok := cfg.Hosts[server]
	if !ok || host.Token == "" {
		return "", "", ErrNoHost
	}
	return server, host.Token, nil
}
