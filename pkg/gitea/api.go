package gitea

import (
	"code.gitea.io/sdk/gitea"
	_ "code.gitea.io/sdk/gitea"
	"encoding/base64"
	"fmt"
	"github.com/rs/zerolog/log"
)

func getErtiaIdentity() gitea.Identity {
	return gitea.Identity{
		Name:  "ertia",
		Email: "support@ertia.io",
	}
}

func (g *Gitea) GetFile(owner, repo, filePath, branch string) ([]byte, string, error) {

	cr, r, err := g.client.GetContents(owner, repo, branch, filePath)

	if err != nil {

		if err.Error() == "404 Not Found" {
			return nil, "", nil
		}

		errStr := fmt.Sprintf("Failed to get ertia config to %s/repos/%s/%s/contents/%s", g.host, owner, repo, filePath)
		log.Ctx(g.ctx).Error().Err(err).Msgf(errStr)
		return nil, "", fmt.Errorf(errStr)
	}

	if r.StatusCode > 400 {
		errStr := fmt.Sprintf("Failed to get ertia config file to repo %s, got response code: %d", g.host, r.StatusCode)
		log.Ctx(g.ctx).Error().Err(err).Msgf(errStr)
		return nil, "", fmt.Errorf(errStr)
	}

	log.Ctx(g.ctx).Info().Msgf("Get config file with sha %d", cr.SHA)

	data, err := base64.StdEncoding.DecodeString(*cr.Content)
	if err != nil {
		return nil, "", err
	}

	return data, cr.SHA, nil
}

func (g *Gitea) UpdateFile(owner, repo, filePath, branch string, content []byte, existingSHA string) error {

	updateFileOptions := gitea.UpdateFileOptions{
		FileOptions: gitea.FileOptions{
			Message:    "Upload ertia config",
			BranchName: branch,
			Author:     getErtiaIdentity(),
			Committer:  getErtiaIdentity(),
		},
		Content: base64.StdEncoding.EncodeToString(content),
		SHA:     existingSHA,
	}

	fr, r, err := g.client.UpdateFile(owner, repo, filePath, updateFileOptions)

	if err != nil {
		errStr := fmt.Sprintf("Failed to update ertia config to %s/repos/%s/%s/contents/%s", g.host, owner, repo, filePath)
		log.Ctx(g.ctx).Error().Err(err).Msgf(errStr)
		return fmt.Errorf(errStr)
	}

	if r.StatusCode > 400 {
		errStr := fmt.Sprintf("Failed to push ertia config file to repo %s, got response code: %d", g.host, r.StatusCode)
		log.Ctx(g.ctx).Error().Err(err).Msgf(errStr)
		return fmt.Errorf(errStr)
	}

	log.Ctx(g.ctx).Info().Msgf("Uploaded config file with hash %s", fr.Content.SHA)

	return nil

}

func (g *Gitea) CreateFile(owner, repo, filePath, branch string, content []byte) error {

	createFileOptions := gitea.CreateFileOptions{
		FileOptions: gitea.FileOptions{
			Message:    "Upload ertia config",
			BranchName: branch,
			Author:     getErtiaIdentity(),
			Committer:  getErtiaIdentity(),
		},
		Content: base64.StdEncoding.EncodeToString(content),
	}

	fr, r, err := g.client.CreateFile(owner, repo, filePath, createFileOptions)

	if err != nil {
		errStr := fmt.Sprintf("Failed to update ertia config to %s/repos/%s/%s/contents/%s", g.host, owner, repo, filePath)
		log.Ctx(g.ctx).Error().Err(err).Msgf(errStr)
		return fmt.Errorf(errStr)
	}

	if r.StatusCode > 400 {
		errStr := fmt.Sprintf("Failed to push ertia config file to repo %s, got response code: %d", g.host, r.StatusCode)
		log.Ctx(g.ctx).Error().Err(err).Msgf(errStr)
		return fmt.Errorf(errStr)
	}

	log.Ctx(g.ctx).Info().Msgf("Uploaded config file with hash %s", fr.Content.SHA)

	return nil

}
