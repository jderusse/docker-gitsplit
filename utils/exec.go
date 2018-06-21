package utils

import (
	"bytes"
	"fmt"
	log "github.com/sirupsen/logrus"
	"os/exec"
	"strings"
	"syscall"
)

type ExecResut struct {
	ExitCode int
	Stdout   string
	Stderr   string
	Output   string
}

func Exec(name string, arg ...string) ExecResut {
	cmd := exec.Command(name, arg...)
	result := ExecResut{}

	var stdoutBuffer bytes.Buffer
	var stderrBuffer bytes.Buffer
	cmd.Stdout = &stdoutBuffer
	cmd.Stderr = &stderrBuffer
	err := cmd.Run()

	result.Stdout = stdoutBuffer.String()
	result.Stderr = stderrBuffer.String()
	result.Output = result.Stdout + result.Stderr
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			ws := exitError.Sys().(syscall.WaitStatus)
			result.ExitCode = ws.ExitStatus()
		} else {
			result.ExitCode = 128
			if result.Output == "" {
				result.Output = err.Error()
			}
		}
	} else {
		ws := cmd.ProcessState.Sys().(syscall.WaitStatus)
		result.ExitCode = ws.ExitStatus()
	}
	return result
}

func GitExec(repository string, command string, arg ...string) (ExecResut, error) {
	result := Exec("git", append([]string{"--git-dir", repository, command}, arg...)...)
	if result.ExitCode != 0 {
		return result, fmt.Errorf(result.Output)
	}

	log.Debug(strings.Join(append([]string{"git", command}, arg...), " "))
	log.Debug(result.Output)

	return result, nil
}
