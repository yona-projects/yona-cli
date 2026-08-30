package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/search5/yona-cli/internal/config"
)

// isolateConfigDir는 실제 사용자의 ~/.config/yona-cli를 절대 건드리지 않도록 각 테스트마다
// 독립된 임시 설정 디렉터리를 쓰게 한다.
func isolateConfigDir(t *testing.T) {
	t.Helper()
	t.Setenv(config.ConfigDirEnvVar, t.TempDir())
	t.Setenv(config.ServerEnvVar, "")
	t.Setenv(config.TokenEnvVar, "")
}

// runCLI는 NewRootCmd()를 새로 만들어(플래그 상태 격리) args로 실행하고, stdin/stdout을
// 캡처해 돌려준다.
func runCLI(t *testing.T, stdin string, args ...string) (stdout string, err error) {
	t.Helper()
	root := NewRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetIn(strings.NewReader(stdin))
	root.SetArgs(args)
	err = root.Execute()
	return out.String(), err
}
