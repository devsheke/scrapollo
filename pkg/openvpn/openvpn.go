package openvpn

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/go-cmd/cmd"
	"github.com/rs/zerolog/log"
)

var (
	// ErrorConfigNotFound indicates that the specified OpenVPN config was not found.
	ErrorConfigNotFound = errors.New("specified openvpn config not found")
	// ErrorNoVPNProcess indicates that no OpenVPN process in running at the moment.
	ErrorNoVPNProcess = errors.New("no openvpn processes were found")
)

type (
	// ErrorVPNTimeout indicates that OpenVPN took too long.
	ErrorVPNTimeout struct{ msg string }
	// ErrorVPN indicates that the OpenVPN process failed to execute.
	ErrorVPN struct{ stdout, stderr string }
)

// NewVPNTimeoutError creates a new instance ErrorVPNTimeout.
func NewVPNTimeoutError(msg string) error {
	return ErrorVPNTimeout{msg}
}

func (e ErrorVPNTimeout) Error() string {
	return fmt.Sprintf("openvpn timed out: %s", e.msg)
}

func (e ErrorVPN) Error() string {
	return fmt.Sprintf("openvpn failed to run: stdout: %q; stderr: %q", e.stdout, e.stderr)
}

// Start starts a new OpenVPN process with the given OpenVPN credentials and configuration.
// This function also returns an instance of the go representation spawned process and its
// current status.
func Start(config, auth string) (*cmd.Cmd, <-chan cmd.Status, error) {
	log.Debug().Str("config", config).Msg("starting openvpn")

	process := cmd.NewCmdOptions(
		cmd.Options{Streaming: true},
		"sudo", "openvpn",
		"--config",
		config,
		"--auth-user-pass",
		auth,
	)

	status := process.Start()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	var stdout []string
	for {
		select {
		case <-ctx.Done():
			return nil, nil, ErrorVPNTimeout{msg: strings.Join(stdout, "\n")}

		case _stdout := <-process.Stdout:
			if strings.Contains(_stdout, "Initialization Sequence Completed") {
				return process, status, nil
			}
			if _stdout != "" {
				stdout = append(stdout, _stdout)
			}

		case stderr := <-process.Stderr:
			return nil, nil, ErrorVPN{stdout: strings.Join(stdout, "\n"), stderr: stderr}

		case status := <-status:
			if err := status.Error; err != nil {
				return nil, nil, err
			}

			stderr := status.Stderr
			if len(stderr) > 0 {
				return nil, nil, ErrorVPN{
					stdout: strings.Join(stdout, "\n"),
					stderr: stderr[len(stderr)-1],
				}
			}

			return nil, nil, ErrorVPN{stdout: strings.Join(stdout, "\n")}
		}
	}
}

// Stop kills the existing OpenVPN process.
func Stop(process *cmd.Cmd) error {
	log.Debug().Msg("stopping openvpn")

	defer func() {
		var err error
		if process != nil {
			err = exec.Command("kill", fmt.Sprintf("%d", process.Status().PID)).Run()
		} else {
			err = exec.Command("kill", "openvpn").Run()
		}
		if err != nil {
			log.Warn().Err(err).Msg("failed to kill openvpn")
		}
	}()

	if process == nil {
		return ErrorNoVPNProcess
	}

	if err := process.Stop(); err != nil {
		if errors.Is(err, cmd.ErrNotStarted) {
			return ErrorNoVPNProcess
		}
		return err
	}

	return nil
}

// Restart stops then starts a new OpenVPN process with the specified credentials and configuration.
func Restart(process *cmd.Cmd, config, auth string) (*cmd.Cmd, <-chan cmd.Status, error) {
	if err := Stop(process); err != nil && err != ErrorNoVPNProcess {
		return nil, nil, err
	}
	return Start(config, auth)
}
