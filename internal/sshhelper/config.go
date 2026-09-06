// Package sshhelper는 yona 서버(yona-wiki P3-03 Step4)의
// /internal/ssh/authenticate·authorize를 호출하는 "yona internal ssh-auth"/"internal ssh-shell"
// 서브커맨드가 공유하는 설정 로딩과 HTTP 클라이언트를 담는다.
//
// 이 두 서브커맨드는 시스템 OpenSSH의 AuthorizedKeysCommand 훅으로 실행되므로(설정 예시는
// docs/yona-wiki/plans/p3-03-ssh-gpg.md 참고), sshd가 넘겨주는 환경변수가 거의 없다(보안을 위해
// 대부분 제거됨) — 그래서 --server/--token 플래그나 ~/.config/yona-cli/config.yml이 아니라
// 고정 경로의 별도 설정 파일(기본 /etc/yona/ssh-helper.yml)에서 서버 URL과 공유 시크릿을 읽는다.
package sshhelper

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// ConfigPathEnvVar를 설정하면 기본 경로(/etc/yona/ssh-helper.yml) 대신 이 값을 읽는다 —
// 테스트에서 실제 /etc를 건드리지 않기 위한 용도다.
const ConfigPathEnvVar = "YONA_SSH_HELPER_CONFIG"

const defaultConfigPath = "/etc/yona/ssh-helper.yml"

// Config는 ssh-helper.yml 파일 하나의 내용이다. yona 서버 하나에 대응한다(현재 이 훅은 서버
// 하나에서만 동작한다고 가정 — yona-cli의 다중 서버 지원(config.Hosts)과는 별개 관심사).
type Config struct {
	Server string `yaml:"server"`
	Secret string `yaml:"secret"`
}

func configPath() string {
	if p := os.Getenv(ConfigPathEnvVar); p != "" {
		return p
	}
	return defaultConfigPath
}

// LoadConfig는 설정 파일을 읽는다. Server/Secret이 비어 있으면(파일이 없거나 필드 누락) 오류를
// 반환한다 — 이 두 값 없이는 어떤 요청도 보낼 수 없으므로 조용히 진행하는 것보다 즉시 실패가 낫다.
func LoadConfig() (*Config, error) {
	path := configPath()
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("ssh-helper 설정 파일을 읽을 수 없습니다(%s): %w", path, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("ssh-helper 설정 파일이 올바른 YAML이 아닙니다(%s): %w", path, err)
	}
	if cfg.Server == "" {
		return nil, fmt.Errorf("ssh-helper 설정 파일(%s)에 server가 없습니다", path)
	}
	if cfg.Secret == "" {
		return nil, fmt.Errorf("ssh-helper 설정 파일(%s)에 secret이 없습니다", path)
	}
	return &cfg, nil
}
