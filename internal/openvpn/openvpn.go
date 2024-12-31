package openvpn

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"time"

	"math/rand/v2"

	"github.com/go-cmd/cmd"
	"github.com/rs/zerolog/log"
	"github.com/shadowbizz/apollo-crawler/pkg/openvpn"
)

var (
	// ErrorConfigsNotFound indicates that no openvpn configuration files were found in the specified path.
	ErrorConfigsNotFound = errors.New("no openvpn configs were found")
	// ErrorNoUnusedConfigs indicates that all given configurations have been used previously.
	ErrorNoUnusedConfigs = errors.New("no unused openvpn configs were found")
)

// OpenVPN represents an instance of an OpenVPN process.
type OpenVPN struct {
	process         *cmd.Cmd
	status          <-chan cmd.Status
	args, auth, dir string
	Configs         []string
	used            map[string]struct{}
}

// NewVPN creates a new instane of OpenVPN.
func NewVPN(configs, auth, args string) (*OpenVPN, error) {
	_configs, err := loadConfigs(configs)
	if err != nil {
		return nil, err
	}
	o := &OpenVPN{
		args:    args,
		auth:    auth,
		Configs: _configs,
		dir:     configs,
		used:    make(map[string]struct{}),
	}

	return o, nil
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

func (o *OpenVPN) SetUsed(configs []string) {
	for _, config := range configs {
		o.used[config] = struct{}{}
	}
}

// Start starts a new OpenVPN process.
func (o *OpenVPN) Start(config string) error {
	if !slices.Contains(o.Configs, config) {
		return openvpn.ErrorConfigNotFound
	}
	config = filepath.Join(o.dir, config)

	var err error
	o.process, o.status, err = openvpn.Start(config, o.auth)

	return err
}

// Stop stops the OpenVPN process if it is running.
func (o *OpenVPN) Stop() error {
	return openvpn.Stop(o.process)
}

// Restart restarts the OpenVPN process with the specified configuration file.
func (o *OpenVPN) Restart(config string) error {
	var err error
	o.process, o.status, err = openvpn.Restart(o.process, filepath.Join(o.dir, config), o.auth)

	return err
}

// UpdateUsed marks the given OpenVPN configuration file as used.
func (o *OpenVPN) UpdateUsed(config string) {
	o.used[config] = struct{}{}
}

// ConfigIsUsed returns true if the given OpenVPN configuration file has been used.
func (o *OpenVPN) ConfigIsUsed(config string) bool {
	_, used := o.used[config]
	return used
}

func (o *OpenVPN) filterUnused() []string {
	configs := make([]string, 0, len(o.Configs))

	for _, config := range o.Configs {
		if !o.ConfigIsUsed(config) {
			configs = append(configs, config)
		}
	}

	rand.Shuffle(len(configs), func(i, j int) {
		configs[i], configs[j] = configs[j], configs[i]
	})

	return configs
}

// Backup starts a new OpenVPN process with a random unused OpenVPN configuration file and returns
// the newly used configuration if successful. It returns ErrorNoUnusedConfigs if all given
// configurations have been used.
//
// # NOTE: This method should be used when the Start method fails.
func (o *OpenVPN) Backup() (string, error) {
	log.Debug().Msg("fetching backup config since previous failed")

	unused := o.filterUnused()
	if len(unused) < 1 {
		return "", ErrorNoUnusedConfigs
	}

	usedCache := make(map[int]int)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()
	for {
		select {
		case <-ctx.Done():
			return "", openvpn.NewVPNTimeoutError("too many retries")
		default:
		}
		r := rand.IntN(len(unused))
		if _, used := usedCache[r]; used {
			continue
		}

		if err := o.Start(unused[r]); err == nil {
			log.Debug().Str("config", unused[r]).Msg("got backup config")
			return unused[r], nil
		}
	}
}
