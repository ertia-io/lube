package github

const githubRestApi = "https://api.github.com"

const (
	applicationJson = iota + 1
	applicationOctetStream
)

type Github struct {
	apiToken string
	debug    bool
	restApi  string
}

func New(apiToken string) *Github {
	return &Github{
		apiToken: apiToken,
		restApi:  githubRestApi,
	}
}
