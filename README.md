## Tensor

Centralized infrastructure management REST API, based on ansible, provides role-based access control, job scheduling, inventory management.
Currently, the REST API supports the only ansible. Our expectation is to support other CI-CD automation tools like Chef,Puppet in the near future.

**Use Cases**

- Configuration Management
- Provisioning
- Code Deployments
- Continuous Integration & Continuous Delivery
- Security & Compliance
- Orchestration

updating....

### Installation instructions


```
docker-compose up
go get -u github.com/jteeuwen/go-bindata/...
go get github.com/mitchellh/gox
go get github.com/cespare/reflex
```

# Vagrant
vagrant up --provider=libvirt
export VAGRANT_DEFAULT_PROVIDER=libvirt

VAGRANT_VAGRANTFILE=Vagrantfile.centos VAGRANT_DOTFILE_PATH=.vagrant_centos vagrant up