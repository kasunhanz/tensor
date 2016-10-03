FROM ubuntu:latest

# gcc for cgo and ansible dependencies
RUN apt-get update && apt-get install -y -q --no-install-recommends \
        curl  g++  gcc  libc6-dev  make  git subversion mercurial ca-certificates openssl \
		build-essential python-pip python-dev python-yaml libffi-dev libssl-dev \
        libxml2-dev libxslt1-dev zlib1g-dev iputils-ping proot && \
        pip install -U pip && \
        pip install --upgrade wheel setuptools && \
        pip install --upgrade pyyaml jinja2 pycrypto

# GO lang
ENV GOLANG_VERSION 1.7.1
ENV GOLANG_DOWNLOAD_URL https://golang.org/dl/go$GOLANG_VERSION.linux-amd64.tar.gz
ENV GOLANG_DOWNLOAD_SHA256 43ad621c9b014cde8db17393dc108378d37bc853aa351a6c74bf6432c1bbd182

RUN curl -fsSL "$GOLANG_DOWNLOAD_URL" -o golang.tar.gz \
	&& echo "$GOLANG_DOWNLOAD_SHA256  golang.tar.gz" | sha256sum -c - \
	&& tar -C /usr/local -xzf golang.tar.gz \
	&& rm golang.tar.gz

ENV GOPATH /go
ENV PATH $GOPATH/bin:/usr/local/go/bin:$PATH

RUN mkdir -p "$GOPATH/src" "$GOPATH/bin" && chmod -R 777 "$GOPATH"

# ansible

# Download ansible's source tree
RUN git clone git://github.com/ansible/ansible.git --recursive

# Compile ansible
RUN cd ansible && bash -c 'source ./hacking/env-setup'  && \
    mkdir -p /opt/ansible                && \
    mv /ansible/bin   /opt/ansible/bin   && \
    mv /ansible/lib   /opt/ansible/lib   && \
    mv /ansible/docs  /opt/ansible/docs  && \
    rm -rf /ansible

# Optinal tools
RUN apt-get install -y sshpass openssh-client

# Clean up
RUN apt-get remove -y --auto-remove build-essential python-pip python-dev libffi-dev libssl-dev  && \
    apt-get clean && rm -rf /var/lib/apt/lists/*

# Adding hosts
RUN mkdir -p /etc/ansible && echo 'localhost' > /etc/ansible/hosts



ENV PATH        /opt/ansible/bin:$PATH
ENV PYTHONPATH  /opt/ansible/lib:$PYTHONPATH
ENV MANPATH /opt/ansible/docs/man:$MANPATH

# tensor
RUN mkdir -p /opt/tensor/bin/
ENV PATH    /opt/tensor/bin:$PATH

ADD tensor.conf /etc/tensor.conf
RUN mkdir -p /root/.ssh
ADD bitbucket /root/.ssh/bitbucket
RUN chmod 600 /root/.ssh/bitbucket

RUN mkdir -p /opt/tensor/plugins/inventory/
# create projects path
RUN mkdir -p /opt/tensor/projects/

# Add plugins and playbooks
ADD plugins/inventory/tensorrest.py /opt/tensor/plugins/inventory/tensorrest.py
RUN chmod +x /opt/tensor/plugins/inventory/tensorrest.py
ADD playbooks/project_update.yml /opt/tensor/system/projects/project_update.yml

WORKDIR $GOPATH/src/bitbucket.pearson.com/apseng/tensor

RUN echo "    IdentityFile ~/.ssh/bitbucket" >> /etc/ssh/ssh_config
RUN echo "StrictHostKeyChecking no " > /root/.ssh/config

ADD /run.sh /run.sh
RUN chmod +x /run.sh

COPY go-wrapper /usr/local/bin/

CMD ["/run.sh"]
