package lube

import (
	"archive/tar"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/ertia-io/lube/deployers/helm"
	"github.com/ertia-io/lube/deployers/yaml"
	"github.com/ertia-io/lube/pkg/github"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const (
	deployInfoFile = "deploy_info.json"
	deployTypeYaml = "yaml"
	deployTypeHelm = "chart"
	notApplicable  = "n/a"
	emptyString    = ""
)

type Deployer interface {
	DeployPath(context.Context, string, string) error
	Name() string
}

type deploy struct {
	ID        int    `json:"id"`
	Type      string `json:"type"`
	File      string `json:"file"`
	Namespace string `json:"namespace"`
}

type deployInfo struct {
	Deploys []deploy `json:"deploys"`
}

type LubeDeployer struct {
	Domain     string
	KubeConfig string
}

func NewLubeDeployer(kubeCfgPath, domain string) *LubeDeployer {
	return &LubeDeployer{
		Domain:     domain,
		KubeConfig: kubeCfgPath,
	}
}

func WithLoggingContext(ctx context.Context) context.Context {
	logger := zerolog.New(os.Stdout).With().Str("module", "lube").Logger()
	return logger.WithContext(ctx)
}

//Deploy tar archive
func (d *LubeDeployer) DeployArchive(ctx context.Context, owner, repo, tag, token string) error {
	ertiaDir, archiveFile, err := downloadArchive(owner, repo, tag, token)
	if err != nil {
		return err
	}

	log.Ctx(ctx).Info().Msgf("deploying archive: %s, %s, %s", owner, repo, tag)

	if err := untarDeployment(ertiaDir, archiveFile); err != nil {
		return err
	}

	if err := os.Remove(archiveFile); err != nil {
		return err
	}

	if err := d.deployRelease(ctx, ertiaDir); err != nil {
		return err
	}

	return nil
}

//Deploy
func (ld *LubeDeployer) deployRelease(ctx context.Context, dir string) error {
	diBuf, err := ioutil.ReadFile(filepath.Join(dir, deployInfoFile))
	if err != nil {
		return err
	}

	var di deployInfo

	if err := json.Unmarshal(diBuf, &di); err != nil {
		return err
	}

	log.Ctx(ctx).Info().Msgf("found %d deployments", len(di.Deploys))

	if len(di.Deploys) > 0 {
		for k, d := range di.Deploys {
			if k+1 != d.ID {
				return fmt.Errorf("deploy: %s with id %d, is out of deploy order", d.File, d.ID)
			}

			if d.Namespace != notApplicable && d.Namespace != emptyString {
				//Create namepsace..
				kubeDeployer, err := yaml.NewYamlDeployer(ld.KubeConfig)
				if err != nil {
					return err
				}
				err = kubeDeployer.CreateNamespace(context.Background(), d.Namespace)
				if err != nil {
					err = nil //Already existed, ignore..
				}
			} else {
				d.Namespace = emptyString
			}

			var deployer Deployer

			if d.Type == deployTypeYaml {
				deployer, err = yaml.NewYamlDeployer(ld.KubeConfig)
				if err != nil {
					return err
				}
			} else if d.Type == deployTypeHelm {
				deployer, err = helm.NewHelmDeployer(ld.KubeConfig, ld.Domain)
				if err != nil {
					return err
				}
			} else {
				return fmt.Errorf("unknown deploy type: %s", d.Type)
			}

			log.Ctx(ctx).Info().Msgf("deploying %s with deployer: %s", d.File, deployer.Name())

			err = deployer.DeployPath(
				context.Background(), d.Namespace, filepath.Join(dir, d.File),
			)
			if err != nil {
				log.Ctx(ctx).Err(err).
					Msgf("could not deploy %s with deployer %s", d.File, deployer.Name())
				return err
			}
		}
	}

	return nil
}

func downloadArchive(owner, repo, tag, token string) (string, string, error) {
	g := github.New(token)
	release, err := g.GetReleaseAssetByTag(owner, repo, tag)
	if err != nil {
		return "", "", err
	}

	tmpDir, err := ioutil.TempDir("", "ertia.deployments") //TODO: Add checksum? would work as cache...?
	if err != nil {
		return "", "", err
	}

	outFile, err := os.Create(filepath.Join(tmpDir, "deployments.tar"))
	if err != nil {
		return "", "", err
	}

	if _, err := outFile.Write(release); err != nil {
		return "", "", err
	}

	return tmpDir, outFile.Name(), nil
}

func untarDeployment(dir string, filePath string) error {
	f, err := os.Open(filePath)
	if err != nil {
		return err
	}

	tarReader := tar.NewReader(f)

	if err != nil {
		return err
	}

	for {
		header, err := tarReader.Next()

		if err == io.EOF {
			break
		}

		if err != nil {
			return err
		}

		switch header.Typeflag {
		case tar.TypeDir:
			//Do nothing, only accepting yaml or helm Archives
			continue
		case tar.TypeReg:
			if err := sanitizeExtractPath(dir, header.Name); err != nil {
				return err
			}

			err = os.MkdirAll(filepath.Join(dir, filepath.Dir(header.Name)), 0777)
			if err != nil {
				return err
			}

			outFile, err := os.Create(filepath.Join(dir, header.Name))
			if err != nil {
				return err
			}
			defer outFile.Close()

			if _, err := io.Copy(outFile, tarReader); err != nil {
				return err
			}
		default:
			continue
		}
	}

	return nil
}

// zip-slip prevention
func sanitizeExtractPath(dir, filePath string) error {
	destpath := filepath.Join(dir, filePath)
	if !strings.HasPrefix(destpath, filepath.Clean(dir)+string(os.PathSeparator)) {
		return fmt.Errorf("%s: illegal file path", filePath)
	}
	return nil
}
