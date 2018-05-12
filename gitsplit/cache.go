package gitsplit

import (
	"fmt"
	"github.com/jderusse/gitsplit/utils"
	"github.com/libgit2/git2go"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"path/filepath"
	"strings"
)

type CachePoolInterface interface {
	SaveItem(item *CacheItem) error
	GetItem(referenceName string, split Split) (*CacheItem, error)
	Load() error
	Dump() error
	Push()
}

type NullCachePool struct {
}

func (c *NullCachePool) SaveItem(item *CacheItem) error {
	return nil
}

func (c *NullCachePool) GetItem(referenceName string, split Split) (*CacheItem, error) {
	return &CacheItem{
		flagName: getFlagName(referenceName, split),
	}, nil
}

func (c *NullCachePool) Load() error {
	return nil
}

func (c *NullCachePool) Dump() error {
	return nil
}

func (c *NullCachePool) Push() {
}

type CachePool struct {
	workingSpacePath string
	remote           *GitRemote
}

type CacheItem struct {
	flagName string
	sourceId *git.Oid
	targetId *git.Oid
}

func NewCachePool(workingSpacePath string, remote *GitRemote) *CachePool {
	return &CachePool{
		workingSpacePath,
		remote,
	}
}

func getFlagName(referenceName string, split Split) string {
	return fmt.Sprintf("%s-%s", utils.Hash(referenceName), utils.Hash(strings.Join(split.Prefixes, "-")))
}

func (c *CachePool) Load() error {
	if err := c.remote.FetchFile("splitsh", "splitsh.db", filepath.Join(c.workingSpacePath, "splitsh.db")); err != nil {
		return errors.Wrap(err, "Fail to fetch cache")
	}
	log.Info("Cache loaded")

	return nil
}

func (c *CachePool) Dump() error {
	if !utils.FileExists(filepath.Join(c.workingSpacePath, "splitsh.db")) {
		return nil
	}

	if err := c.remote.PushFile("splitsh.db", filepath.Join(c.workingSpacePath, "splitsh.db"), "Update splitsh cache", "splitsh"); err != nil {
		return errors.Wrap(err, "Fail to save cache")
	}
	log.Info("Cache dumped")

	return nil
}

func (c *CachePool) Push() {
	c.remote.PushAll()
}

func (c *CachePool) SaveItem(item *CacheItem) error {
	if item.SourceId() != nil {
		if err := c.remote.AddReference("source-"+item.flagName, item.SourceId()); err != nil {
			return errors.Wrapf(err, "Unable to create source reference %s targeting %s", item.flagName, item.SourceId())
		}
	}
	if item.TargetId() != nil {
		if err := c.remote.AddReference("target-"+item.flagName, item.TargetId()); err != nil {
			return errors.Wrapf(err, "Unable to create target reference %s targeting %s", item.flagName, item.TargetId())
		}
	}

	return nil
}

func (c *CachePool) GetItem(referenceName string, split Split) (*CacheItem, error) {
	flagName := getFlagName(referenceName, split)
	sourceReference, err := c.remote.GetReference("source-" + flagName)
	if err != nil {
		return nil, err
	}

	if sourceReference == nil {
		return &CacheItem{
			flagName: flagName,
		}, nil
	}

	targetReference, err := c.remote.GetReference("target-" + flagName)
	if err != nil {
		return nil, err
	}

	if targetReference == nil {
		return &CacheItem{
			flagName: flagName,
			sourceId: sourceReference.Id,
			targetId: nil,
		}, nil
	}

	return &CacheItem{
		flagName: flagName,
		sourceId: sourceReference.Id,
		targetId: targetReference.Id,
	}, nil
}

func (c *CacheItem) IsFresh(reference Reference) bool {
	if c.sourceId == nil {
		return false
	}

	return c.sourceId.Equal(reference.Id)
}

func (c *CacheItem) SourceId() *git.Oid {
	return c.sourceId
}
func (c *CacheItem) TargetId() *git.Oid {
	return c.targetId
}
func (c *CacheItem) Set(sourceId *git.Oid, targetId *git.Oid) {
	c.sourceId = sourceId
	c.targetId = targetId
}
