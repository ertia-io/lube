# lube
## Kubernetes deployments without the friction


## Deployments
### Helm
Put your deployments in a directory with a name containing _helm

### Yaml
Put your yaml files in a directory, it will deploy any yaml as kube yaml that are not in a directory with a name containing _helm


### As CLI tool
go build -o lube cmd/main.go

#### Deploy Archive URL (url to any tar.gz, example a git repo)
`./lube deploy <url>`
  
#### Deploy Directory

`./lube deploy <dirpath>`


### As library

`./lube deploy <dirpath>


go ```
package main
import(
  "github.com/ertia-io/lube"
)

func main(){

  dp := lube.NewLubeDeployer("~/.kube/config", "lube-namespace" )

  //Deploy archive url (any url to tar.gz)
  err = ld.DeployArchiveUrl(context.Background(),https://github.com/ertia-io/deployments.tar.gz)
  
  //Deploy directory any dir containing helm charts / kube yaml
  err = ld.DeployDirectoryRecursive(context.Background(),"/opt/ertia/mydeploys/")
}


```
