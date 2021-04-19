package wireguardexporter

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/BurntSushi/toml"
	"github.com/naggie/dsnet"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

// file is the TOML mapping of public keys to peer names.
type file struct {
	Peers []struct {
		PublicKey string `toml:"public_key"`
		Name      string `toml:"name"`
	} `toml:"peer"`
}

// ParsePeers parses a TOML mapping of peer public keys to friendly names.
func ParsePeers(r io.Reader) (map[string]string, error) {
	var f file
	md, err := toml.DecodeReader(r, &f)
	if err != nil {
		return nil, err
	}
	if u := md.Undecoded(); len(u) > 0 {
		return nil, fmt.Errorf("unrecognized keys: %s", u)
	}

	peers := make(map[string]string)
	for _, p := range f.Peers {
		// Each peer must have a valid public key and a name set.
		if _, err := wgtypes.ParseKey(p.PublicKey); err != nil {
			return nil, fmt.Errorf("invalid public key %q: %v", p.PublicKey, err)
		}

		if p.Name == "" {
			return nil, fmt.Errorf("no name set for peer with public key %q", p.PublicKey)
		}

		peers[p.PublicKey] = p.Name
	}

	return peers, nil
}

// ParseDsnetConfig parses a dnset config file for friendly names.
func ParseDsnetConfig(r io.Reader) (map[string]string, error) {
	peers := make(map[string]string)

	f, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	var cfg dsnet.DsnetConfig
	if err := json.Unmarshal(f, &cfg); err != nil {
		return nil, err
	}

	for _, p := range cfg.Peers {
		if _, err := wgtypes.ParseKey(p.PublicKey.Key.String()); err != nil {
			return nil, fmt.Errorf("invalid public key %q: %v", p.PublicKey, err)
		}

		if p.Hostname == "" {
			return nil, fmt.Errorf("no name set for peer with public key %q", p.PublicKey)
		}

		peers[p.PublicKey.Key.String()] = p.Hostname
	}

	return peers, nil
}
