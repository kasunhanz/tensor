# Latest version of centos
FROM centos:centos7

# ansible
RUN yum clean all && \
    yum -y install epel-release && yum -y update && \
    yum -y install PyYAML python-jinja2 python-httplib2 python-keyczar python-paramiko python-setuptools git python-pip python-requests wget

RUN yum -y install ansible
# Adding hosts
RUN mkdir -p /etc/ansible && echo 'localhost' > /etc/ansible/hosts

# install proot
RUN wget http://download.opensuse.org/repositories/home:cedric-vincent/CentOS_CentOS-5/home:cedric-vincent.repo  -O /etc/yum.repos.d/cedric-vincent.repo
RUN yum -y update && yum -y install proot

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
