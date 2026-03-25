package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"

	"github.com/fengwk/wslgit-for-windows/internal/bridge"
)

func main() {
	config := bridge.LoadConfigFromEnv()
	logger := bridge.NewDebugLogger(config)
	defer logger.Close()

	if err := run(config, logger); err != nil {
		var exitErr bridge.ExitCodeError
		if errors.As(err, &exitErr) {
			logger.Printf("command exit=%d", exitErr.Code)
			os.Exit(exitErr.Code)
		}

		logger.Printf("fatal error: %v", err)
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(config bridge.Config, logger *bridge.DebugLogger) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("读取当前工作目录失败: %w", err)
	}

	logger.Printf("original cwd=%q raw args=%q", cwd, os.Args[1:])

	wslCWD, err := bridge.WindowsPathToWSL(cwd)
	if err != nil {
		return fmt.Errorf("当前目录不受支持，请将仓库放在本机盘符路径下: %w", err)
	}

	rewrittenArgs := bridge.RewriteArgs(os.Args[1:], cwd, bridge.OSPathChecker{})
	useExec := bridge.SupportsExecMode(config, logger)
	invocation := bridge.BuildInvocation(config, wslCWD, rewrittenArgs, useExec)

	logger.Printf("rewritten cwd=%q rewritten args=%q useExec=%t command=%q", wslCWD, rewrittenArgs, useExec, append([]string{invocation.Program}, invocation.Args...))

	cmd := exec.Command(invocation.Program, invocation.Args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()

	err = cmd.Run()
	if err == nil {
		logger.Printf("command exit=0")
		return nil
	}

	var exitError *exec.ExitError
	if errors.As(err, &exitError) {
		return bridge.ExitCodeError{Code: exitError.ExitCode()}
	}

	return fmt.Errorf("启动 %s 失败: %w，请确认 %s 可用", invocation.Program, err, strconv.Quote(invocation.Program))
}
