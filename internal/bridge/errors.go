package bridge

import "fmt"

type ExitCodeError struct {
	Code int
}

func (e ExitCodeError) Error() string {
	return fmt.Sprintf("命令执行失败，退出码=%d", e.Code)
}
