# Latest version of debian
FROM ubuntu:latest

# gcc for cgo and ansible dependencies
RUN export DEBIAN_FRONTEND=noninteractive && \
    apt-get -q update && \
    apt-get -q -y install software-properties-common && \
    add-apt-repository -y ppa:ansible/ansible && \
    apt-get -q update && \
    apt-get -y -q install ansible && \
    apt-get -y -q install --no-install-recommends \
    curl g++ gcc libc6-dev make git subversion mercurial ca-certificates openssl \
	build-essential python-pip python-dev python-yaml libffi-dev libssl-dev \
    libxml2-dev libxslt1-dev zlib1g-dev iputils-ping proot sshpass openssh-client && \
    pip install -U pip && \
    pip install --upgrade wheel setuptools && \
    pip install --upgrade pyyaml jinja2 pycrypto requests && \
    apt-get remove -y -q --auto-remove build-essential python-pip python-dev libffi-dev libssl-dev && \
    apt-get clean && rm -rf /var/lib/apt/lists/*

# GO lang
RUN curl -fsSL "https://golang.org/dl/go1.7.3.linux-amd64.tar.gz" -o golang.tar.gz && \
	echo "508028aac0654e993564b6e2014bf2d4a9751e3b286661b0b0040046cf18028e  golang.tar.gz" | sha256sum -c - && \
	tar -C /usr/local -xzf golang.tar.gz && \
	rm golang.tar.gz

# tensor
ENV GOPATH=/go PATH=$GOPATH/bin:/usr/local/go/bin:/opt/tensor/bin:$PATH

RUN mkdir -p "$GOPATH/src" "$GOPATH/bin" \
    "/etc/ansible" "/root/.ssh" \
    "/opt/tensor/bin/" "/opt/tensor/projects/" \
    "/opt/tensor/plugins/inventory/" && \
    chmod -R 777 "$GOPATH" && \
    echo 'localhost' > /etc/ansible/hosts

ADD bitbucket /root/.ssh/bitbucket
RUN chmod 600 /root/.ssh/bitbucket

# Add plugins and playbooks
ADD plugins/inventory/tensorrest.py /opt/tensor/plugins/inventory/tensorrest.py
RUN chmod +x /opt/tensor/plugins/inventory/tensorrest.py
ADD playbooks/project_update.yml /opt/tensor/system/projects/project_update.yml

WORKDIR $GOPATH/src/bitbucket.pearson.com/apseng/tensor

RUN echo "    IdentityFile ~/.ssh/bitbucket" >> /etc/ssh/ssh_config
RUN echo "StrictHostKeyChecking no " > /root/.ssh/config

ADD /run.sh /run.sh
RUN chmod +x /run.sh


ENV TENSOR_PORT=8010 PROJECT_PATH="/opt/tensor/projects" HOME_PATH="/opt/tensor/" \
    TENSOR_DB_USER=tensor TENSOR_DB_PASSWORD=tensor TENSOR_DB_NAME=tensordb \
    TENSOR_DB_REPLICA="" TENSOR_DB_HOSTS="mongo:27017"

CMD ["/run.sh"]
