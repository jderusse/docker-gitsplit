package gitsplit

import (
	"fmt"
	"github.com/jderusse/gitsplit/utils"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"strings"
)

type StringCollection []string
type PrefixCollection StringCollection

type Split struct {
	Prefixes PrefixCollection `yaml:"prefix"`
	Targets  StringCollection `yaml:"target"`
}

type Config struct {
	CacheUrl   *GitUrl  `yaml:"cache_url"`
	ProjectUrl *GitUrl  `yaml:"project_url"`
	Splits     []Split  `yaml:"splits"`
	Origins    []string `yaml:"origins"`
}

func (s *PrefixCollection) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var raw StringCollection
	if err := unmarshal(&raw); err != nil {
		return err
	}

	if len(raw) > 1 {
		seen := []string{}
		for _, prefix := range raw {
			parts := strings.Split(prefix, ":")
			if len(parts) != 2 {
				return fmt.Errorf("Using several prefixes requires to use the syntax `source:target`. Got %s", prefix)
			}
			if utils.InArray(seen, parts[1]) {
				return fmt.Errorf("Cannot have two prefix splits under the same directory. Got twice %s", parts[1])
			}
			seen = append(seen, parts[1])
		}
	}

	*s = PrefixCollection(raw)

	return nil
}

func (s *GitUrl) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var raw string
	if err := unmarshal(&raw); err != nil {
		return err
	}

	*s = *ParseUrl(raw)

	return nil
}

func (s *StringCollection) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var rawString string
	if err := unmarshal(&rawString); err == nil {
		*s = []string{rawString}

		return nil
	}

	var rawArray []string
	if err := unmarshal(&rawArray); err == nil {
		*s = rawArray

		return nil
	}

	return fmt.Errorf("expects a string or n array of strings")
}

func (s *Config) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var raw struct {
		CacheDir   *GitUrl  `yaml:"cache_dir"`
		CacheUrl   *GitUrl  `yaml:"cache_url"`
		ProjectDir *GitUrl  `yaml:"project_dir"`
		ProjectUrl *GitUrl  `yaml:"project_url"`
		Splits     []Split  `yaml:"splits"`
		Origins    []string `yaml:"origins"`
	}

	if err := unmarshal(&raw); err != nil {
		return err
	}

	if raw.CacheDir != nil {
		log.Error(`The config parameter "cache_dir" is deprecated. Use "cache_url" instead`)
	}

	if raw.ProjectDir != nil {
		log.Error(`The config parameter "project_dir" is deprecated. Use "project_url" instead`)
	}

	if raw.CacheUrl == nil {
		raw.CacheUrl = raw.CacheDir
	}
	if raw.ProjectUrl == nil {
		raw.ProjectUrl = raw.ProjectDir
	}

	if raw.ProjectUrl == nil {
		raw.ProjectUrl = ParseUrl(".")
	}
	if len(raw.Origins) == 0 {
		raw.Origins = []string{".*"}
	}

	*s = Config{
		CacheUrl:   raw.CacheUrl,
		ProjectUrl: raw.ProjectUrl,
		Splits:     raw.Splits,
		Origins:    raw.Origins,
	}

	return nil
}

func NewConfigFromFile(filePath string) (*Config, error) {
	config := &Config{}

	yamlFile, err := ioutil.ReadFile(utils.ResolvePath(filePath))
	if err != nil {
		return nil, errors.Wrap(err, "Fail to read config file")
	}

	if err = yaml.Unmarshal(yamlFile, &config); err != nil {
		return nil, errors.Wrap(err, "Fail to load config file")
	}

	return config, nil
}
