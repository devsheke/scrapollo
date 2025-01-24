// Copyright 2025 Abhisheke Acharya
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
	// ErrorConfigNotFound is returned when a specified OpenVPN file is not found.
	ErrorConfigNotFound = errors.New("specified openvpn config not found")

	// ErrorNoVpnProcess is returned when a user attempts to stop an OpenVPN instance that
	// does not exist.
	ErrorNoVpnProcess = errors.New("no openvpn processes were found")
)

type (
	// ErrorVpnTimedOut is returned when the OpenVPN proccess doesn't launch within the
	// specified timeout.
	ErrorVpnTimedOut struct{ Msg string }

	// ErrorVpnFailure is returned when the OpenVPN process fails to start-up.
	ErrorVpnFailure struct{ stdout, stderr string }
)

func (e ErrorVpnTimedOut) Error() string {
	return fmt.Sprintf("openvpn timed out: %s", e.Msg)
}

func (e ErrorVpnFailure) Error() string {
	return fmt.Sprintf("openvpn failed to run: stdout: %q; stderr: %q", e.stdout, e.stderr)
}

// Start spawns an OpenVPN process with the provided configuration, credentials and arguments and returns
// [*cmd.Cmd] and [<-chan cmd.Status] for controlling and monitoring the spawned process.
func Start(config, auth, args string, timeout time.Duration) (*cmd.Cmd, <-chan cmd.Status, error) {
	log.Debug().Str("config", config).Msg("starting openvpn")

	process := cmd.NewCmdOptions(
		cmd.Options{Streaming: true},
		"openvpn",
		"--config",
		config,
		"--auth-user-pass",
		auth,
		args,
	)

	status := process.Start()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var stdoutStack []string
	for {
		select {
		case <-ctx.Done():
			return nil, nil, ErrorVpnTimedOut{Msg: strings.Join(stdoutStack, "\n")}

		case stdout := <-process.Stdout:
			if strings.Contains(stdout, "Initialization Sequence Completed") {
				return process, status, nil
			}
			if stdout != "" {
				stdoutStack = append(stdoutStack, stdout)
			}

		case stderr := <-process.Stderr:
			return nil, nil, ErrorVpnFailure{
				stdout: strings.Join(stdoutStack, "\n"),
				stderr: stderr,
			}

		case status := <-status:
			if err := status.Error; err != nil {
				return nil, nil, err
			}

			stderr := status.Stderr
			if len(stderr) > 0 {
				return nil, nil, ErrorVpnFailure{
					stdout: strings.Join(stdoutStack, "\n"),
					stderr: strings.Join(stderr, "\n"),
				}
			}

			return nil, nil, ErrorVpnFailure{stdout: strings.Join(stdoutStack, "\n")}
		}
	}
}

// Stop is a function which attempts to stop the provided OpenVPN process. ErrorNoVpnProcess is
// returned if there is no process found matching the details of the provided process.
func Stop(process *cmd.Cmd) (err error) {
	log.Debug().Msg("stopping openvpn")

	defer func() {
		if process != nil {
			err = errors.Join(
				err,
				exec.Command("kill", fmt.Sprintf("%d", process.Status().PID)).Run(),
			)
		}
	}()

	if process == nil {
		return ErrorNoVpnProcess
	}

	if err := process.Stop(); err != nil {
		if errors.Is(err, cmd.ErrNotStarted) {
			return ErrorNoVpnProcess
		}
		return err
	}

	return
}

// Restart is a function which attempts to restart the OpenVPN process with the provided configuration,
// credentials and arguments.
func Restart(
	process *cmd.Cmd,
	config, auth, args string,
	timeout time.Duration,
) (*cmd.Cmd, <-chan cmd.Status, error) {
	if err := Stop(process); err != nil && err != ErrorNoVpnProcess {
		return nil, nil, err
	}
	return Start(config, auth, args, timeout)
}
