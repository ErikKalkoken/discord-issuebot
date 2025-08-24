package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/goccy/go-yaml"
)

var errInvalid = errors.New("invalid")

type configGithub struct {
	Repo  string `yaml:"repo"`
	Owner string `yaml:"owner"`
	Token string `yaml:"token"`
}

type config struct {
	Discord struct {
		AppID    string `yaml:"appID"`
		BotToken string `yaml:"botToken"`
	}
	Github []configGithub
}

func (c config) validate() error {
	if c.Discord.AppID == "" {
		return fmt.Errorf("missing app ID: %w", errInvalid)
	}
	if c.Discord.BotToken == "" {
		return fmt.Errorf("missing bot token: %w", errInvalid)
	}
	if len(c.Github) == 0 {
		return fmt.Errorf("no github repos: %w", errInvalid)
	}
	for i, g := range c.Github {
		if g.Repo == "" {
			return fmt.Errorf("missing github name #%d: %w", i+1, errInvalid)
		}
		if g.Token == "" {
			return fmt.Errorf("missing github token #%d: %w", i+1, errInvalid)
		}
	}
	return nil
}

func readConfig(path string) (config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return config{}, err
	}
	var c config
	if err := yaml.Unmarshal(b, &c); err != nil {
		return config{}, err
	}
	if err := c.validate(); err != nil {
		return config{}, err
	}
	return c, nil
}
