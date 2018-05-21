package gitsplit

import (
	"fmt"
	"github.com/gosimple/slug"
	"github.com/jderusse/gitsplit/utils"
	"github.com/libgit2/git2go"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"
)

type GitRemoteCollection struct {
	repository      *git.Repository
	items           map[string]*GitRemote
	mutexRemoteList *sync.Mutex
}

func NewGitRemoteCollection(repository *git.Repository) *GitRemoteCollection {
	return &GitRemoteCollection{
		items:           make(map[string]*GitRemote),
		repository:      repository,
		mutexRemoteList: &sync.Mutex{},
	}
}

func (r *GitRemoteCollection) Add(alias string, url string, refs []string) *GitRemote {
	remote := NewGitRemote(r.repository, alias, url, refs)
	r.items[alias] = remote

	r.mutexRemoteList.Lock()
	defer r.mutexRemoteList.Unlock()
	remote.Init()

	return remote
}

func (r *GitRemoteCollection) Get(alias string) (*GitRemote, error) {
	if remote, ok := r.items[alias]; !ok {
		return nil, errors.New("The remote does not exists")
	} else {
		return remote, nil
	}

}

func (r *GitRemoteCollection) Clean() {
	knownRemotes := []string{}
	for _, remote := range r.items {
		knownRemotes = append(knownRemotes, remote.id)
	}

	r.mutexRemoteList.Lock()
	defer r.mutexRemoteList.Unlock()

	remotes, err := r.repository.Remotes.List()
	if err != nil {
		return
	}

	for _, remoteId := range remotes {
		if !utils.InArray(knownRemotes, remoteId) {
			log.WithFields(log.Fields{
			    "remote": remoteId,
			}).Info("Removing remote")
			r.repository.Remotes.Delete(remoteId)
		}
	}
}

func (r *GitRemoteCollection) Flush() error {
	for _, remote := range r.items {
		if err := remote.Flush(); err != nil {
			return err
		}
	}

	return nil
}

type GitRemote struct {
	repository      *git.Repository
	id              string
	alias           string
	refs            []string
	url             string
	fetched         bool
	pool            *utils.Pool
	cacheReferences []Reference
	mutexReferences *sync.Mutex
}

func NewGitRemote(repository *git.Repository, alias string, url string, refs []string) *GitRemote {
	id := slug.Make(alias)
	if id != alias {
		id = id + "-" + utils.Hash(alias)
	}

	return &GitRemote{
		repository:      repository,
		id:              id,
		alias:           alias,
		refs:            refs,
		url:             url,
		fetched:         false,
		pool:            utils.NewPool(10),
		mutexReferences: &sync.Mutex{},
	}
}

func (r *GitRemote) Init() error {
	remotes, err := r.repository.Remotes.List()
	if err != nil {
		return err
	}

	if !utils.InArray(remotes, r.id) {
		if _, err := r.repository.Remotes.Create(r.id, os.ExpandEnv(r.url)); err != nil {
			return errors.Wrapf(err, "failed to create remote %s", r.alias)
		}
	} else {
		if err := r.repository.Remotes.SetUrl(r.id, os.ExpandEnv(r.url)); err != nil {
			return errors.Wrapf(err, "failed to update remote %s", r.alias)
		}
	}

	return nil
}

func (r *GitRemote) GetReference(alias string) (*Reference, error) {
	references, err := r.GetReferences()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get reference")
	}

	for _, reference := range references {
		if reference.Alias == alias {
			return &reference, nil
		}
	}

	return nil, nil
}

func (r *GitRemote) AddReference(alias string, id *git.Oid) error {
	r.mutexReferences.Lock()
	defer r.mutexReferences.Unlock()

	r.cacheReferences = nil
	for _, ref := range r.refs {
		reference, err := r.repository.References.Create(fmt.Sprintf("refs/remotes/%s/%s/%s", r.id, ref, alias), id, true, "")
		if err != nil {
			return errors.Wrap(err, "failed to add reference")
		}
		defer reference.Free()
	}

	return nil
}

func (r *GitRemote) GetReferences() ([]Reference, error) {
	r.mutexReferences.Lock()
	defer r.mutexReferences.Unlock()

	if r.cacheReferences != nil {
		return r.cacheReferences, nil
	}

	var call func() ([]Reference, error)
	if r.fetched {
		call = r.getLocalReferences
	} else {
		call = r.getRemoteReferences
	}

	references, err := call()
	if err != nil {
		return nil, err
	}
	r.cacheReferences = references

	return references, nil
}

func (r *GitRemote) getRemoteReferences() ([]Reference, error) {
	result, err := utils.GitExec(r.repository.Path(), "ls-remote", r.id)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to fetch references of %s", r.alias)
	}

	references := []Reference{}
	cleanShortNameRegexp := regexp.MustCompile(fmt.Sprintf("^refs/"))
	cleanAliasRegexp := regexp.MustCompile(fmt.Sprintf("^refs/(%s)/", strings.Join(r.refs, "|")))
	cleanNameRegexp := regexp.MustCompile(fmt.Sprintf("^refs/"))
	filterRegexp := regexp.MustCompile(fmt.Sprintf("^refs/(%s)/", strings.Join(r.refs, "|")))

	for _, line := range strings.Split(result.Stdout, "\n") {
		if len(line) == 0 {
			continue
		}
		columns := strings.Split(line, "\t")
		if len(columns) != 2 {
			return nil, fmt.Errorf("failed to parse reference %s: 2 columns expected", line)
		}
		referenceId := columns[0]
		referenceName := columns[1]

		if !filterRegexp.MatchString(referenceName) {
			continue
		}

		oid, err := git.NewOid(referenceId)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse reference %s", line)
		}

		references = append(references, Reference{
			Alias:     cleanAliasRegexp.ReplaceAllString(referenceName, ""),
			ShortName: cleanShortNameRegexp.ReplaceAllString(referenceName, ""),
			Name:      cleanNameRegexp.ReplaceAllString(referenceName, fmt.Sprintf("refs/remotes/%s/", r.id)),
			Id:        oid,
		})
	}

	return references, nil
}

func (r *GitRemote) getLocalReferences() ([]Reference, error) {
	iterator, err := r.repository.NewReferenceIteratorGlob(fmt.Sprintf("refs/remotes/%s/*", r.id))
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch references")
	}

	defer iterator.Free()
	references := []Reference{}

	reference, err := iterator.Next()
	cleanShortNameRegexp := regexp.MustCompile(fmt.Sprintf("^refs/remotes/%s/", r.id))
	cleanAliasRegexp := regexp.MustCompile(fmt.Sprintf("^refs/remotes/%s/(%s)/", r.id, strings.Join(r.refs, "|")))
	filterRegexp := regexp.MustCompile(fmt.Sprintf("^refs/remotes/%s/(%s)/", r.id, strings.Join(r.refs, "|")))
	for err == nil {
		if filterRegexp.MatchString(reference.Name()) {
			references = append(references, Reference{
				Alias:     cleanAliasRegexp.ReplaceAllString(reference.Name(), ""),
				ShortName: cleanShortNameRegexp.ReplaceAllString(reference.Name(), ""),
				Name:      reference.Name(),
				Id:        reference.Target(),
			})
		}
		reference, err = iterator.Next()
	}

	r.cacheReferences = references
	return references, nil
}

func (r *GitRemote) Fetch() {
	r.pool.Push(func() (interface{}, error) {
		log.WithFields(log.Fields{
		    "remote": r.alias,
		    "refs": r.refs,
		}).Warn("Fetching from remote")
		for _, ref := range r.refs {
			if _, err := utils.GitExec(r.repository.Path(), "fetch", "--force", "--prune", r.id, fmt.Sprintf("refs/%s/*:refs/remotes/%s/%s/*", ref, r.id, ref)); err != nil {
				return nil, errors.Wrapf(err, "failed to update cache of %s", r.alias)
			}
		}

		r.fetched = true

		return nil, nil
	})
}

func (r *GitRemote) PushRef(refs string) {
	r.pool.Push(func() (interface{}, error) {
		log.WithFields(log.Fields{
		    "remote": r.alias,
		    "refs": refs,
		}).Warn("Pushing to remote")
		if _, err := utils.GitExec(r.repository.Path(), "push", "--force", r.id, refs); err != nil {
			return nil, errors.Wrapf(err, "failed to push reference %s", refs)
		}

		return nil, nil
	})
}

func (r *GitRemote) PushAll() {
	for _, ref := range r.refs {
		r.PushRef(fmt.Sprintf("refs/remotes/%s/%s/*:refs/%s/*", r.id, ref, ref))
	}
}

func (r *GitRemote) Push(reference Reference, splitId *git.Oid) error {
	references, err := r.GetReferences()
	if err != nil {
		return errors.Wrapf(err, "failed to get references for remote %s", r.alias)
	}

	for _, remoteReference := range references {
		if remoteReference.Alias == reference.Alias {
			if remoteReference.Id.Equal(splitId) {
				log.WithFields(log.Fields{
				    "remote": r.alias,
				}).Info("Already pushed " + reference.Alias)
				return nil
			}
			log.WithFields(log.Fields{
			    "remote": r.alias,
			}).Warn("Out of date " + reference.Alias)
			break
		}
	}

	r.PushRef(splitId.String() + ":refs/" + reference.ShortName)

	return nil
}

func (r *GitRemote) FetchFile(referenceName string, fileName string, filePath string) error {
	reference, err := r.GetReference("splitsh")
	if err != nil {
		return errors.Wrapf(err, "failed to fetch file reference %s", referenceName)
	}
	if reference == nil {
		return nil
	}
	commit, err := r.repository.LookupCommit(reference.Id)
	if err != nil {
		return errors.Wrapf(err, "failed to find commit %s", reference.Id)
	}
	defer commit.Free()
	tree, err := commit.Tree()
	if err != nil {
		return errors.Wrapf(err, "failed to fetch commit tree")
	}
	defer tree.Free()
	entry, err := tree.EntryByPath(fileName)
	if err != nil {
		return errors.Wrapf(err, "failed to fetch file from tree")
	}

	odb, err := r.repository.Odb()
	if err != nil {
		return errors.Wrap(err, "failed to open odb")
	}
	defer odb.Free()
	object, err := odb.Read(entry.Id)
	if err != nil {
		return errors.Wrap(err, "failed to read from odb")
	}
	defer object.Free()
	if err := ioutil.WriteFile(filePath, object.Data(), os.FileMode(entry.Filemode)); err != nil {
		return errors.Wrapf(err, "failed to write file %s", filePath)
	}

	return nil
}

func (r *GitRemote) PushFile(fileName string, filePath string, message string, referenceName string) error {
	treeBuilder, err := r.repository.TreeBuilder()
	if err != nil {
		return errors.Wrap(err, "failed to create treeBuilder")
	}
	defer treeBuilder.Free()

	file, err := os.Open(filePath)
	if err != nil {
		return errors.Wrap(err, "failed to open file")
	}
	defer file.Close()
	content, err := ioutil.ReadAll(file)
	if err != nil {
		return errors.Wrap(err, "failed to read file")
	}
	odb, err := r.repository.Odb()
	if err != nil {
		return errors.Wrap(err, "failed to open odb")
	}
	blobId, err := odb.Write(content, git.ObjectBlob)
	if err != nil {
		return errors.Wrap(err, "failed to write in odb")
	}
	if err = treeBuilder.Insert(fileName, blobId, git.FilemodeBlob); err != nil {
		return errors.Wrap(err, "failed to insert tree")
	}

	treeID, err := treeBuilder.Write()
	if err != nil {
		return errors.Wrap(err, "failed to write tree")
	}

	tree, err := r.repository.LookupTree(treeID)
	if err != nil {
		return errors.Wrap(err, "failed to find tree")
	}
	defer tree.Free()

	reference, err := r.GetReference(referenceName)
	if err == nil && reference != nil {
		return r.replaceFile(reference, message, tree)
	}

	return r.insertFile(referenceName, message, tree)
}

func (r *GitRemote) GetSignature() *git.Signature {
	return &git.Signature{
		Name:  "gitsplit",
		Email: "jeremy+gitsplit@derusse.com",
		When:  time.Now(),
	}
}

func (r *GitRemote) replaceFile(reference *Reference, message string, tree *git.Tree) error {
	r.mutexReferences.Lock()
	defer r.mutexReferences.Unlock()

	r.cacheReferences = nil

	commit, err := r.repository.LookupCommit(reference.Id)
	if err != nil {
		return errors.Wrapf(err, "failed to find commit %s", reference.Id)
	}

	sig := r.GetSignature()
	if _, err := commit.Amend(reference.Name, sig, sig, message, tree); err != nil {
		return err
	}

	return nil
}

func (r *GitRemote) insertFile(referenceName string, message string, tree *git.Tree) error {
	r.mutexReferences.Lock()
	defer r.mutexReferences.Unlock()

	r.cacheReferences = nil

	sig := r.GetSignature()
	if _, err := r.repository.CreateCommit(fmt.Sprintf("refs/remotes/%s/%s/%s", r.id, r.refs[0], referenceName), sig, sig, message, tree); err != nil {
		return err
	}

	return nil
}

func (r *GitRemote) Flush() error {
	results := r.pool.Wait()
	if err := results.FirstError(); err != nil {
		return err
	}

	return nil
}
