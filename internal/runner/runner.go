package runner

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/awmiriiw/torsen/internal/logger"
)

type Runner struct {
	log *logger.Logger
}

func New(log *logger.Logger) *Runner {
	return &Runner{log: log}
}

func (r *Runner) Run(command ...string) error {
	if len(command) == 0 {
		return fmt.Errorf("empty command")
	}

	r.log.Debug("Executing: %s", strings.Join(command, " "))

	cmd := exec.Command(command[0], command[1:]...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		r.log.Error("Command failed: %s | Output: %s | Error: %v",
			strings.Join(command, " "),
			strings.TrimSpace(string(output)),
			err,
		)
		return fmt.Errorf("command '%s' failed: %w", strings.Join(command, " "), err)
	}

	if len(output) > 0 {
		r.log.Debug("Output: %s", strings.TrimSpace(string(output)))
	}

	return nil
}

func (r *Runner) RunSilent(command ...string) error {
	if len(command) == 0 {
		return fmt.Errorf("empty command")
	}

	cmd := exec.Command(command[0], command[1:]...)
	_, err := cmd.CombinedOutput()

	return err
}

func (r *Runner) RunWithOutput(command ...string) (string, error) {
	if len(command) == 0 {
		return "", fmt.Errorf("empty command")
	}

	r.log.Debug("Executing: %s", strings.Join(command, " "))

	cmd := exec.Command(command[0], command[1:]...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("command '%s' failed: %w",
			strings.Join(command, " "), err)
	}

	return strings.TrimSpace(string(output)), nil
}
