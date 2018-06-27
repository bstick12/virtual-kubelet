package cf

import (
	"io"

	"github.com/BurntSushi/toml"
)

type providerConfig struct {
	CFAPI   string
	Org          string
	Space string
	AccessToken             string
}

func (p *CFProvider) loadConfig(r io.Reader) error {
	var config providerConfig
	if _, err := toml.DecodeReader(r, &config); err != nil {
		return err
	}
	p.providerConfig = config
	return nil
}
