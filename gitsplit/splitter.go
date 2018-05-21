package gitsplit

import (
	"github.com/jderusse/gitsplit/utils"
	"github.com/libgit2/git2go"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"regexp"
	"strings"
)

type Reference struct {
	Alias     string
	ShortName string
	Name      string
	Id        *git.Oid
}

type Splitter struct {
	config            Config
	referenceSplitter *ReferenceSplitterLite
	workingSpace      *WorkingSpace
	cachePool         CachePoolInterface
}

func NewSplitter(config Config, workingSpace *WorkingSpace, cachePool CachePoolInterface) *Splitter {
	return &Splitter{
		config:            config,
		workingSpace:      workingSpace,
		referenceSplitter: NewReferenceSplitterLite(workingSpace.Repository()),
		cachePool:         cachePool,
	}
}

func (s *Splitter) Split(whitelistReferences []string) error {
	remote, err := s.workingSpace.Remotes().Get("origin")
	if err != nil {
		return err
	}

	references, err := remote.GetReferences()
	if err != nil {
		return errors.Wrap(err, "failed to split references")
	}
	for _, reference := range references {
		for _, referencePattern := range s.config.Origins {
			referenceRegexp := regexp.MustCompile(referencePattern)
			if !referenceRegexp.MatchString(reference.Alias) {
				continue
			}
			if len(whitelistReferences) > 0 && !utils.InArray(whitelistReferences, reference.Alias) {
				continue
			}

			for _, split := range s.config.Splits {
				if err := s.splitReference(reference, split); err != nil {
					return errors.Wrap(err, "failed to split references")
				}
			}
		}
	}

	if err := s.workingSpace.Remotes().Flush(); err != nil {
		return errors.Wrap(err, "failed to flush references")
	}
	return nil
}

func (s *Splitter) splitReference(reference Reference, split Split) error {
	flagTemp := "refs/split-temp/" + utils.Hash(reference.Name) + "-" + utils.Hash(strings.Join(split.Prefixes, "-"))

	previousReference, err := s.cachePool.GetItem(reference.Name, split)
	if err != nil {
		return errors.Wrap(err, "failed to fetch previous state")
	}

	contextualLog := log.WithFields(log.Fields{
	    "reference": reference.Alias,
	    "splits": split.Prefixes,
	})

	if previousReference.IsFresh(reference) {
		contextualLog.Info("Already splitted")
	} else {
		contextualLog.Warn("Splitting")
		tempReference, err := s.workingSpace.Repository().References.Create(flagTemp, reference.Id, true, "Temporary reference")
		if err != nil {
			return errors.Wrapf(err, "failed to create temporary reference %s", flagTemp)
		}
		defer tempReference.Free()

		splitId, err := s.referenceSplitter.Split(flagTemp, split.Prefixes)
		if err != nil {
			return errors.Wrap(err, "failed to split reference")
		}

		err = tempReference.Delete()
		if err != nil {
			return errors.Wrapf(err, "failed to delete temporary reference %s", flagTemp)
		}

		previousReference.Set(reference.Id, splitId)
		if err := s.cachePool.SaveItem(previousReference); err != nil {
			return errors.Wrapf(err, "failed to cache reference %s", flagTemp)
		}
	}

	// Reference does not exists
	if previousReference.TargetId() == nil {
		return nil
	}

	for _, target := range split.Targets {
		remote, err := s.workingSpace.Remotes().Get(target)
		if err != nil {
			return err
		}
		if err := remote.Push(reference, previousReference.TargetId()); err != nil {
			return err
		}
	}

	return nil
}

func (s *Splitter) getLocalReference(referenceName string) (*git.Oid, error) {
	reference, err := s.workingSpace.Repository().References.Dwim(referenceName)
	if err != nil {
		return nil, nil
	}

	return reference.Target(), nil
}
