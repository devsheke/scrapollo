package openvpn

import (
	"errors"
	"os"
	"path/filepath"
	"slices"

	"math/rand/v2"

	"github.com/go-cmd/cmd"
	"github.com/rs/zerolog/log"
	"github.com/shadowbizz/apollo-crawler/pkg/openvpn"
)

var ErrorConfigsNotFound = errors.New("no openvpn configs were found")

type OpenVPN struct {
	process         *cmd.Cmd
	status          <-chan cmd.Status
	args, auth, dir string
	configs         []string
	used            map[string]struct{}
}

func NewVPN(configs, auth, args string) (*OpenVPN, error) {
	_configs, err := loadConfigs(configs)
	if err != nil {
		return nil, err
	}
	o := &OpenVPN{
		args:    args,
		auth:    auth,
		configs: _configs,
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

func (o *OpenVPN) Start(config string) error {
	if !slices.Contains(o.configs, config) {
		return openvpn.ErrorConfigNotFound
	}
	config = filepath.Join(o.dir, config)

	var err error
	o.process, o.status, err = openvpn.Start(config, o.auth)

	return err
}

func (o *OpenVPN) Stop() error {
	return openvpn.Stop(o.process)
}

func (o *OpenVPN) Restart(config string) error {
	var err error
	o.process, o.status, err = openvpn.Restart(o.process, config, o.auth)

	return err
}

func (o *OpenVPN) filterUnused() []string {
	configs := make([]string, 0, len(o.configs))

	for _, config := range o.configs {
		if _, ok := o.used[config]; !ok {
			configs = append(configs, config)
		}
	}

	rand.Shuffle(len(configs), func(i, j int) {
		configs[i], configs[j] = configs[j], configs[i]
	})

	return configs
}

func (o *OpenVPN) Backup() (string, error) {
	log.Debug().Msg("fetching backup config since previous failed")

	unused := o.filterUnused()
	usedCache := make(map[int]int)

	for retries := 0; retries < 10; retries++ {
		r := rand.IntN(len(unused))
		if _, used := usedCache[r]; used {
			continue
		}

		if err := o.Start(unused[r]); err == nil {
			log.Debug().Str("config", unused[r]).Msg("got backup config")
			return unused[r], nil
		}
	}

	return "", openvpn.NewVPNTimeoutError("too many retries")
}
