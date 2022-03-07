package lube

import (
	"archive/tar"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/ertia-io/lube/deployers/helm"
	"github.com/ertia-io/lube/deployers/yaml"
	"github.com/ertia-io/lube/pkg/github"
	"github.com/rs/zerolog/log"
)

type Deployer interface {
	Deploy(context.Context, string, io.Reader) error
	DeployPath(context.Context, string, string) error
	Name() string
}

type LubeDeployer struct {
	KubeConfig string
	Namespace  string
}

func NewLubeDeployer(kubeCfgPath string, namespace string) *LubeDeployer {
	return &LubeDeployer{
		KubeConfig: kubeCfgPath,
		Namespace:  namespace,
	}
}

func (d *LubeDeployer) WithLoggingContext(ctx context.Context) context.Context {
	logger := log.With().Str("module", "lube").Logger()
	return logger.WithContext(ctx)
}

//Deploy with URL to tar.gz archive (like github repo)
func (d *LubeDeployer) DeployArchiveUrl(ctx context.Context, owner, repo, tag, token string) error {
	ertiaDir, archiveFile, err := downloadArchive(owner, repo, tag, token)
	if err != nil {
		return err
	}

	log.Ctx(ctx).Info().Msgf("\nDeploying Archive: %s, %s, %s\n", owner, repo, tag)

	err = untarDeployment(ertiaDir, archiveFile)
	if err != nil {
		return err
	}

	err = os.Remove(archiveFile)
	if err != nil {
		return err
	}

	//err = d.DeployDirectoryRecursive(ctx, ertiaDir)

	return nil
}

//Deploy with path to directory, recursive
func (d *LubeDeployer) DeployDirectoryRecursive(ctx context.Context, dir string) error {
	fi, err := ioutil.ReadDir(filepath.Join(dir))
	if err != nil {
		return err
	}

	log.Ctx(ctx).Info().Msgf("\nFound %d deployments\n", len(fi))

	if len(fi) > 0 {

		//Create namepsace..
		kubeDeployer, err := yaml.NewYamlDeployer(d.KubeConfig)
		if err != nil {
			return err
		}
		err = kubeDeployer.CreateNamespace(context.Background(), d.Namespace)
		if err != nil {
			err = nil //Already existed, ignore..
		}
	}

	for _, collection := range fi {
		var deployer Deployer

		if strings.Contains(strings.ToLower(collection.Name()), "_helm_") {
			deployer, err = helm.NewHelmDeployer(d.KubeConfig)
		} else if strings.Contains(strings.ToLower(collection.Name()), "_yaml_") {
			deployer, err = yaml.NewYamlDeployer(d.KubeConfig)
		} else if collection.IsDir() {
			log.Ctx(ctx).Info().Msgf("\n Checking %s for deployments \n", filepath.Join(dir, collection.Name()))
			err = d.DeployDirectoryRecursive(ctx, filepath.Join(dir, collection.Name()))
			if err != nil {
				return err
			}
			continue
		} else {
			continue
		}

		if err != nil {
			return err
		}

		log.Ctx(ctx).Info().Msgf("\nDeploying %s with Deployer: %s\n", collection.Name(), deployer.Name())
		err = deployer.DeployPath(context.Background(), d.Namespace, filepath.Join(dir, collection.Name()))
		if err != nil {
			log.Ctx(ctx).Err(err).Msgf("\nCould not deploy %s with Deployer %s\n", collection.Name(), deployer.Name())
			return err
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
