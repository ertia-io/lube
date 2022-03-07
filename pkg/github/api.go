package github

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
)

const stateUploaded = "uploaded"

type asset struct {
	Url   string `json:"url"`
	Name  string `json:"name"`
	State string `json:"state"`
}

type release struct {
	TagName    string  `json:"tag:name"`
	Draft      bool    `json:"draft"`
	Prerelease bool    `json:"prerelease"`
	Assets     []asset `json:"assets"`
}

// GetReleaseAssetByTag gets a published Ertia release asset with the specified tag.
func (g *Github) GetReleaseAssetByTag(owner, repo, tag string) ([]byte, error) {
	// GET /repos/{owner}/{repo}/releases/tags/{tag}
	url := fmt.Sprintf("%s/repos/%s/%s/releases/tags/%s",
		g.restApi, url.PathEscape(owner), url.PathEscape(repo), url.PathEscape(tag))

	sc, res, err := g.request(http.MethodGet, url, nil, applicationJson)
	if err != nil {
		return nil, err
	}

	if err := statusToErr(sc); err != nil {
		return nil, err
	}

	var rel release
	if err := json.Unmarshal(res, &rel); err != nil {
		return nil, err
	}

	var (
		releaseName = fmt.Sprintf("release-%s.tar", tag)
		releaseUrl  string
	)

	for _, a := range rel.Assets {
		if a.Name != releaseName {
			continue
		}

		if a.State != stateUploaded {
			return nil, errors.New(fmt.Sprintf("asset: %s - not uploaded yet", a.Name))
		}

		releaseUrl = a.Url
	}

	sc, res, err = g.request(http.MethodGet, releaseUrl, nil, applicationOctetStream)
	if err != nil {
		return nil, err
	}

	if err := statusToErr(sc); err != nil {
		return nil, err
	}

	return res, nil
}
