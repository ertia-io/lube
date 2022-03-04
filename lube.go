package lube

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"github.com/ertia-io/lube/deployers/helm"
	"github.com/ertia-io/lube/deployers/yaml"
	"github.com/rs/zerolog/log"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
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
func (d *LubeDeployer) DeployArchiveUrl(ctx context.Context, url string, token string) error {
	ertiaDir, archiveFile, err := downloadArchive(url, token)
	if err != nil {
		return err
	}

	log.Ctx(ctx).Info().Msgf("\nDeploying Archive: %s\n", url)

	err = decompressArchive(ertiaDir, archiveFile)
	if err != nil {
		return err
	}

	err = os.Remove(archiveFile)
	if err != nil {
		return err
	}

	err = d.DeployDirectoryRecursive(ctx, ertiaDir)

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

		if strings.Contains(strings.ToLower(collection.Name()), "_helm") {
			deployer, err = helm.NewHelmDeployer(d.KubeConfig)
		} else if strings.Contains(strings.ToLower(collection.Name()), "yaml") {
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

func downloadArchive(url string, token string) (string, string, error) {

	httpClient := http.DefaultClient
	httpClient.Timeout = time.Second * 60

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", "", err
	}

	if token != "" {
		req.Header.Add("Authorization", "token "+token)
	}

	resp, err := httpClient.Do(req)

	if err != nil {
		return "", "", err
	}

	defer resp.Body.Close()

	dir, err := ioutil.TempDir("", "ertia.deployments") //TODO: Add checksum? would work as cache...?

	if err != nil {
		return "", "", err
	}

	outFile, err := os.Create(filepath.Join(dir, "deployments.tgz"))
	if err != nil {
		return "", "", err
	}

	if _, err := io.Copy(outFile, resp.Body); err != nil {
		return "", "", err
	}
	return dir, outFile.Name(), err
}

func decompressArchive(dir string, filePath string) error {
	r, err := os.Open(filePath)
	if err != nil {
		return err
	}

	uncompressedStream, err := gzip.NewReader(r)
	if err != nil {
		return err
	}

	tarReader := tar.NewReader(uncompressedStream)

	if err != nil {
		return err
	}

	for true {
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
