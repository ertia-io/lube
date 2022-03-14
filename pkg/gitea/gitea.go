package gitea

import (
	"code.gitea.io/sdk/gitea"
	"context"
	"fmt"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"os"
)

type Gitea struct {
	apiToken     string
	client       *gitea.Client
	ctx          context.Context
	debug        bool
	host         string
	organization string
}

func (d *Gitea) WithLoggingContext(ctx context.Context) context.Context {
	logger := zerolog.New(os.Stdout).With().Str("module", "lube").Logger()
	return logger.WithContext(ctx)
}

func New(ctx context.Context, apiToken string, host string, organization string, debug bool) (*Gitea, error) {

	var (
		git *gitea.Client
		err error
	)

	if debug {
		git, err = gitea.NewClient(host, gitea.SetToken(apiToken), gitea.SetDebugMode())
	} else {
		git, err = gitea.NewClient(host, gitea.SetToken(apiToken))
	}

	if err != nil {
		errStr := fmt.Sprintf("failed to init gitea client for organization %s", organization)
		log.Ctx(ctx).Error().Err(err).Msgf(errStr)
		return nil, fmt.Errorf(errStr)
	}

	return &Gitea{
		client:       git,
		debug:        debug,
		apiToken:     apiToken,
		host:         host,
		organization: organization,
		ctx:          ctx,
	}, nil
}
