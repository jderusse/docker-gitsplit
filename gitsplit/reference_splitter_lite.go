package gitsplit

import (
	"github.com/libgit2/git2go"
	lite "github.com/splitsh/lite/splitter"
	"strings"
)

func NewReferenceSplitterLite(repository *git.Repository) *ReferenceSplitterLite {
	return &ReferenceSplitterLite{
		repository: repository,
	}
}

type ReferenceSplitterLite struct {
	repository *git.Repository
}

func formatLitePrefixes(prefixes []string) []*lite.Prefix {
	litePrefixes := []*lite.Prefix{}
	for _, prefix := range prefixes {
		parts := strings.Split(prefix, ":")
		from := parts[0]
		to := ""
		if len(parts) > 1 {
			to = parts[1]
		}
		litePrefixes = append(litePrefixes, &lite.Prefix{From: from, To: to})
	}

	return litePrefixes
}

func (r *ReferenceSplitterLite) Split(reference string, prefixes []string) (*git.Oid, error) {
	config := &lite.Config{
		Path:       r.repository.Path(),
		Origin:     reference,
		Prefixes:   formatLitePrefixes(prefixes),
		Target:     "",
		Commit:     "",
		Debug:      false,
		Scratch:    false,
		GitVersion: "latest",
	}

	result := &lite.Result{}
	if err := lite.Split(config, result); err != nil {
		return nil, err
	}

	return result.Head(), nil
}
