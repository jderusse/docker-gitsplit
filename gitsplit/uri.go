package gitsplit

import (
	"github.com/jderusse/gitsplit/utils"
	"os"
	"strings"
)

type GitUrl struct {
	scheme string
	url    string
}

func (u *GitUrl) IsLocal() bool {
	return u.scheme == "file"
}

func (u *GitUrl) Url() string {
	if u.scheme == "" {
		return u.SchemelessUrl()
	}

	return u.scheme + "://" + u.SchemelessUrl()
}

func (u *GitUrl) SchemelessUrl() string {
	if u.IsLocal() {
		return utils.ResolvePath(u.url)
	}

	return os.ExpandEnv(u.url)
}

func ParseUrl(url string) *GitUrl {
	parts := strings.SplitN(url, "://", 2)
	if len(parts) == 2 {
		return &GitUrl{
			scheme: parts[0],
			url:    parts[1],
		}
	}

	parts = strings.SplitN(url, "/", 2)
	if strings.Index(parts[0], ":") > 0 {
		return &GitUrl{
			scheme: "",
			url:    url,
		}
	}

	return &GitUrl{
		scheme: "file",
		url:    url,
	}
}
