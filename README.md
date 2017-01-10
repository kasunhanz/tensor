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
VAGRANT_VAGRANTFILE=Vagrantfile.fedora VAGRANT_DOTFILE_PATH=.vagrant_fedora vagrant up


Vagrant cannot forward the specified ports on this VM, since they
would collide with some other application that is already listening
on these ports. The forwarded port to 80 is already in use
on the host machine.

To fix this, modify your current project's Vagrantfile to use another
port. Example, where '1234' would be replaced by a unique host port:

  config.vm.network :forwarded_port, guest: 80, host: 1234

Sometimes, Vagrant will attempt to auto-correct this for you. In this
case, Vagrant was unable to. This is usually because the guest machine
is in a state which doesn't allow modifying port forwarding. You could
try 'vagrant reload' (equivalent of running a halt followed by an up)
so vagrant can attempt to auto-correct this upon booting. Be warned
that any unsaved work might be lost.