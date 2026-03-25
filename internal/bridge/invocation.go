package bridge

import (
	"os/exec"
	"strings"
)

type Invocation struct {
	Program string
	Args    []string
}

func BuildInvocation(config Config, wslCWD string, gitArgs []string, useExec bool) Invocation {
	baseArgs := make([]string, 0, len(gitArgs)+8)
	if config.Distro != "" {
		baseArgs = append(baseArgs, "-d", config.Distro)
	}

	if useExec {
		baseArgs = append(baseArgs, "--cd", wslCWD, "--exec", config.GitBinary)
		baseArgs = append(baseArgs, gitArgs...)
		return Invocation{Program: config.WSLPath, Args: baseArgs}
	}

	script := `cd "$1" && shift && exec "$@"`
	baseArgs = append(baseArgs, "sh", "-lc", script, "sh", wslCWD, config.GitBinary)
	baseArgs = append(baseArgs, gitArgs...)
	return Invocation{Program: config.WSLPath, Args: baseArgs}
}

func SupportsExecMode(config Config, logger *DebugLogger) bool {
	if config.ForceShell {
		logger.Printf("force shell mode enabled")
		return false
	}

	cmd := exec.Command(config.WSLPath, "--help")
	output, err := cmd.CombinedOutput()
	if err != nil {
		logger.Printf("detect exec mode failed: %v", err)
		return true
	}

	help := string(output)
	supportsExec := strings.Contains(help, "--exec") && strings.Contains(help, "--cd")
	logger.Printf("detect exec mode supportsExec=%t", supportsExec)
	return supportsExec
}
