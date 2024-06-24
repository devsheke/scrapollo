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
	ErrorConfigNotFound = errors.New("specified openvpn config not found")
	ErrorNoVPNProcess   = errors.New("no openvpn processes were found")
)

type (
	ErrorVPNTimeout struct{ msg string }
	ErrorVPN        struct{ msg string }
)

func NewVPNTimeoutError(msg string) error {
	return ErrorVPNTimeout{msg}
}

func (e ErrorVPNTimeout) Error() string {
	return fmt.Sprintf("openvpn timed out: %s", e.msg)
}

func (e ErrorVPN) Error() string {
	return fmt.Sprintf("openvpn failed to run: %s", e.msg)
}

func Start(config, auth string) (*cmd.Cmd, <-chan cmd.Status, error) {
	log.Debug().Str("config", config).Msg("starting openvpn")

	process := cmd.NewCmdOptions(
		cmd.Options{Streaming: true},
		"openvpn",
		"--config",
		config,
		"--auth-user-pass",
		auth,
	)

	status := process.Start()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	var prevStdout string
	for {
		select {
		case <-ctx.Done():
			return nil, nil, ErrorVPNTimeout{msg: prevStdout}

		case stdout := <-process.Stdout:
			if strings.Contains(stdout, "Initialization Sequence Completed") {
				return process, status, nil
			}
			prevStdout = stdout

		case stderr := <-process.Stderr:
			return nil, nil, ErrorVPN{msg: stderr}

		case status := <-status:
			if err := status.Error; err != nil {
				return nil, nil, err
			}

			stderr := status.Stderr
			if len(stderr) > 0 {
				return nil, nil, ErrorVPN{msg: stderr[len(stderr)-1]}
			}

			return nil, nil, ErrorVPN{msg: prevStdout}
		}
	}
}

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

func Restart(process *cmd.Cmd, config, auth string) (*cmd.Cmd, <-chan cmd.Status, error) {
	if err := Stop(process); err != nil && err != ErrorNoVPNProcess {
		return nil, nil, err
	}
	return Start(config, auth)
}
