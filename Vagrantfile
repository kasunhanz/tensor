module OS
  def OS.windows?
    (/cygwin|mswin|mingw|bccwin|wince|emx/ =~ RUBY_PLATFORM) != nil
  end

  def OS.mac?
    (/darwin/ =~ RUBY_PLATFORM) != nil
  end

  def OS.unix?
    !OS.windows?
  end

  def OS.linux?
    OS.unix? and not OS.mac?
  end
end

# -*- mode: ruby -*-
# vi: set ft=ruby :

# All Vagrant configuration is done below. The "2" in Vagrant.configure
# configures the configuration version (we support older styles for
# backwards compatibility). Please don't change it unless you know what
# you're doing.
Vagrant.configure("2") do |config|
  # The most common configuration options are documented and commented below.
  # For a complete reference, please see the online documentation at
  # https://docs.vagrantup.com.

  # Every Vagrant development environment requires a box. You can search for
  # boxes at https://atlas.hashicorp.com/search.
  config.vm.define "debian", primary: true do |debian|
    debian.vm.box = "debian/jessie64"
    # Create a forwarded port mapping which allows access to a specific port
    # within the machine from a port on the host machine.
    debian.vm.network "forwarded_port", guest: 8010, host: 8010
    debian.vm.network "forwarded_port", guest: 6379, host: 6379 # redis
    debian.vm.network "forwarded_port", guest: 27017, host: 27017 # mongodb
    # Create a private network, which allows host-only access to the machine
    # using a specific IP.
    debian.vm.network "private_network", ip: "10.10.10.1"
  end

  config.vm.define "centos" do |centos|
    centos.vm.box = "centos/7"
    # Create a forwarded port mapping which allows access to a specific port
    # within the machine from a port on the host machine.
    centos.vm.network "forwarded_port", guest: 8010, host: 8011
    centos.vm.network "forwarded_port", guest: 6379, host: 6380 # redis
    centos.vm.network "forwarded_port", guest: 27017, host: 27018 # mongodb
    # Create a private network, which allows host-only access to the machine
    # using a specific IP.
    centos.vm.network "private_network", ip: "10.10.10.2"
  end

  config.vm.define "ubuntu" do |ubuntu|
    ubuntu.vm.box = "ubuntu/xenial64"
    # Create a forwarded port mapping which allows access to a specific port
    # within the machine from a port on the host machine.
    ubuntu.vm.network "forwarded_port", guest: 8010, host: 8012
    ubuntu.vm.network "forwarded_port", guest: 6379, host: 6381 # redis
    ubuntu.vm.network "forwarded_port", guest: 27017, host: 27019 # mongodb
    # Create a private network, which allows host-only access to the machine
    # using a specific IP.
    ubuntu.vm.network "private_network", ip: "10.10.10.3"
  end

  # Create a public network, which generally matched to bridged network.
  # Bridged networks make the machine appear as another physical device on
  # your network.
  # config.vm.network "public_network"

  # Provider-specific configuration so you can fine-tune various
  # backing providers for Vagrant. These expose provider-specific options.
  config.vm.provider "virtualbox" do |vb|
    vb.memory = "2048"
  end

  config.vm.provider "libvirt" do |lv|
    lv.memory = "2048"
  end

  config.vm.provider "vmware_workstation" do |vw|
    vw.memory = "2048"
  end


  # Share an additional folder to the guest VM. The first argument is
  # the path on the host to the actual folder. The second argument is
  # the path on the guest to mount the folder. And the optional third
  # argument is a set of non-required options.
  config.vm.synced_folder '.', '/vagrant', disabled: true # doesn't work with golang
  config.vm.synced_folder '.', '/go/src/github.com/pearsonappeng/tensor/'

  # Use local provision so that vagrant will work with windows
  config.vm.provision "ansible_local" do |ansible|
    ansible.provisioning_path = "/go/src/github.com/pearsonappeng/tensor/packaging/vagrant/"
    ansible.playbook = "playbook.yml"
    ansible.version = "latest"
    ansible.install_mode = "pip" # since debian doesn't support getting a latest version
  end
end