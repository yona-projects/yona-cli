// yona는 yuna 서버(REST API)를 감싸는 커맨드라인 도구다(gh CLI와 동일한 컨셉).
package main

import (
	"fmt"
	"os"

	"github.com/search5/yona-cli/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "오류:", err)
		os.Exit(1)
	}
}
