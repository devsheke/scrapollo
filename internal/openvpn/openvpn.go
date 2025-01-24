package openvpn

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"time"

	"math/rand/v2"

	openvpn "github.com/devsheke/scrapollo/pkg/openvpn-go"
	"github.com/go-cmd/cmd"
	"github.com/rs/zerolog/log"
)

var (
	// ErrorConfigsNotFound indicates that no openvpn configuration files were found in the specified path.
	ErrorConfigsNotFound = errors.New("no valid openvpn configuration files were found")

	// ErrorNoUnusedConfigs indicates that all given configurations have been used previously.
	ErrorNoUnusedConfigs = errors.New("no unused openvpn configuration files were found")
)

// Manager is a type that is used to configure and control a instances of OpenVPN.
type Manager struct {
	args, auth, dir string
	configs         []string
	process         *cmd.Cmd
	status          <-chan cmd.Status
	timeout         time.Duration
	used            map[string]struct{}
}

// NewManager returns a configured instance of [*Manager].
func NewManager(configsDir, auth, args string) (*Manager, error) {
	configs, err := loadConfigs(configsDir)
	if err != nil {
		return nil, err
	}
	v := &Manager{
		args:    args,
		auth:    auth,
		configs: configs,
		dir:     configsDir,
		used:    make(map[string]struct{}),
	}

	return v, nil
}

func loadConfigs(dir string) ([]string, error) {
	_dir, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	configs := make([]string, 0, len(_dir))
	for _, entry := range _dir {
		configs = append(configs, entry.Name())
	}

	if len(configs) == 0 {
		return nil, ErrorConfigsNotFound
	}
	return configs, nil
}

// Start spawns a new instance of OpenVPN with the provided config.
func (v *Manager) Start(config string) error {
	if !slices.Contains(v.configs, config) {
		return openvpn.ErrorConfigNotFound
	}
	config = filepath.Join(v.dir, config)

	var err error
	v.process, v.status, err = openvpn.Start(config, v.auth, v.args, v.timeout)
	if err != nil {
		return err
	}

	if _, used := v.used[config]; !used {
		v.UseConfig(config)
	}

	return nil
}

// Stop attemps to stop the currently running instance of OpenVPN.
func (v *Manager) Stop() error {
	return openvpn.Stop(v.process)
}

// Restart restarts the currently running instance of OpenVPN with the provided config.
func (v *Manager) Restart(config string) error {
	var err error
	v.process, v.status, err = openvpn.Restart(
		v.process,
		filepath.Join(v.dir, config),
		v.auth,
		v.args,
		v.timeout,
	)

	return err
}

// UseConfig adds the provided config to a cache of previously used config files.
func (v *Manager) UseConfig(config string) {
	v.used[config] = struct{}{}
}

// IsConfigUsed returns true if the provided config has been used before.
func (v *Manager) IsConfigUsed(config string) bool {
	_, used := v.used[config]
	return used
}

func (v *Manager) filterUnused() []string {
	configs := make([]string, 0, len(v.configs))

	for _, config := range v.configs {
		if !v.IsConfigUsed(config) {
			configs = append(configs, config)
		}
	}

	rand.Shuffle(len(configs), func(i, j int) {
		configs[i], configs[j] = configs[j], configs[i]
	})

	return configs
}

// Backup is meant to be used after starting an OpenVPN instance fails. This function
// attempts to start an instance of OpenVPN and returns the config used to spawn
// the instance if successful in trying to do so.
func (v *Manager) Backup() (string, error) {
	log.Debug().Msg("fetching backup config since previous failed")

	unused := v.filterUnused()
	if len(unused) < 1 {
		return "", ErrorNoUnusedConfigs
	}

	usedCache := make(map[int]int)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()
	for {
		select {
		case <-ctx.Done():
			return "", openvpn.ErrorVpnTimedOut{Msg: "too many retries"}
		default:
		}
		r := rand.IntN(len(unused))
		if _, used := usedCache[r]; used {
			continue
		}

		if err := v.Start(unused[r]); err == nil {
			log.Debug().Str("config", unused[r]).Msg("got backup config")
			return unused[r], nil
		}
	}
}
