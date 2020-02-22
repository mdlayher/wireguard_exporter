package main

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"os"
	"strings"
)

func parsePeerNamesString(peerNames map[string]string, wgPeerNames string) error {
	for _, kvs := range strings.Split(wgPeerNames, ",") {
		kv := strings.Split(kvs, ":")
		if err := checkPubKey(kv[0]); len(kv) != 2 || err != nil {
			return fmt.Errorf("failed to parse %q as a valid public key and peer name", kv)
		}
		peerNames[kv[0]] = kv[1]
	}
	return nil
}

func parsePeerNamesFile(peerNames map[string]string, filename string) error {
	f, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("wireguard.peer-names-file: %v", err)
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	for i := 1; s.Scan(); i++ {
		// strip comments
		line := strings.SplitN(s.Text(), "#", 2)[0]
		parts := strings.SplitN(line, " ", 2)
		if len(parts) == 1 && parts[0] == "" {
			continue
		} else if len(parts) != 2 {
			return fmt.Errorf(`line %d: invalid syntax, expected "<PUBKEY> description"`, i)
		}
		key := parts[0]
		name := parts[1]
		if err := checkPubKey(key); err != nil {
			return fmt.Errorf("line %d: %w", i, err)
		}
		peerNames[key] = name
	}
	return nil
}

func checkPubKey(key string) error {
	if len(key) != 44 {
		return fmt.Errorf("could not decode public key: length %d, should be 45", len(key))
	}
	_, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		return fmt.Errorf("could not decode public key: %w", err)
	}
	return nil
}
