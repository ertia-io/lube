package helm

import (
	"context"
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"github.com/teris-io/shortid"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"io"
	"io/ioutil"
	"strings"
	"time"

	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"os"
)

type HelmDeployer struct {

	KubeConfig string
	RestConfig *restclient.Config
}

func NewHelmDeployer(kubeCfgPath string) (*HelmDeployer, error){
	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", kubeCfgPath)
	if err != nil {
		panic(err.Error())
	}



	return &HelmDeployer{
		KubeConfig: kubeCfgPath,
		RestConfig: config,
	},nil
}

func (d *HelmDeployer) Name() (string) {
	return "HelmDeployer"
}

func (d *HelmDeployer) Deploy(ctx context.Context,namespace string, r io.Reader) (error) {

	actionConfig := new(action.Configuration)

	kubeCfg, err := ioutil.ReadFile(d.KubeConfig)
	if(err!=nil){
		return err
	}

	restClientGetter :=  NewRESTClientGetter(namespace, string(kubeCfg))

	if err := actionConfig.Init(restClientGetter, namespace, os.Getenv("HELM_DRIVER"), func(format string, v ...interface{}) {
		fmt.Sprintf(format, v)
	}); err != nil {
		return err
	}

	uid, err := shortid.Generate()
	if(err!=nil){
		return err
	}

	installer := action.NewInstall(actionConfig)
	installer.Namespace = namespace
	installer.ReleaseName = "lube-"+strings.ReplaceAll(strings.ReplaceAll(strings.ToLower(uid),"_","."), ".","")
	//installer.ClientOnly=true
	installer.IncludeCRDs=true
	//installer.Wait = true
	installer.Replace = true
	installer.WaitForJobs = false
	installer.Wait = false
	installer.Atomic = true
	installer.Timeout = time.Second * 300
	installer.DependencyUpdate = true

	chart, err := loader.LoadArchive(r)

	if(err!=nil){
		return err
	}
	err = chart.Validate()

	if(err!=nil){
		return err
	}

	release,err := installer.Run(chart, map[string]interface{}{})

	if(false) {
		spew.Dump(release)
	}

	if(err!=nil){
		return err
	}

	return nil
}


func (d *HelmDeployer) DeployPath(ctx context.Context,namespace string, path string) (error) {


	actionConfig := new(action.Configuration)

	kubeCfg, err := ioutil.ReadFile(d.KubeConfig)
	if(err!=nil){
		return err
	}

	restClientGetter :=  NewRESTClientGetter(namespace, string(kubeCfg))

	if err := actionConfig.Init(restClientGetter, namespace, os.Getenv("HELM_DRIVER"), func(format string, v ...interface{}) {
		fmt.Sprintf(format, v)
	}); err != nil {
		return err
	}

	uid, err := shortid.Generate()
	if(err!=nil){
		return err
	}

	installer := action.NewInstall(actionConfig)
	installer.Namespace = namespace
	installer.ReleaseName = "lube-"+strings.ReplaceAll(strings.ReplaceAll(strings.ToLower(uid),"_","."), ".","")
	//installer.ClientOnly=true
	installer.IncludeCRDs=true
	//installer.Wait = true
	installer.Replace = true
	installer.WaitForJobs = false
	installer.Wait = false
	installer.Atomic = true
	installer.Timeout = time.Second * 300
	installer.DependencyUpdate = true

	chart, err := loader.LoadDir(path)

	if(err!=nil){
		return err
	}
	err = chart.Validate()

	if(err!=nil){
		return err
	}

	release,err := installer.Run(chart, map[string]interface{}{})

	if(false) {
		spew.Dump(release)
	}

	if(err!=nil){
		return err
	}

	return nil
}
