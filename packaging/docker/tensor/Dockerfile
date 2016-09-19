FROM buildpack-deps:jessie-scm

# gcc for cgo
RUN apt-get update && apt-get install -y --no-install-recommends \
		g++ \
		gcc \
		libc6-dev \
		make \
		git \
	&& rm -rf /var/lib/apt/lists/*

ENV GOLANG_VERSION 1.7.1
ENV GOLANG_DOWNLOAD_URL https://golang.org/dl/go$GOLANG_VERSION.linux-amd64.tar.gz
ENV GOLANG_DOWNLOAD_SHA256 43ad621c9b014cde8db17393dc108378d37bc853aa351a6c74bf6432c1bbd182

RUN curl -fsSL "$GOLANG_DOWNLOAD_URL" -o golang.tar.gz \
	&& echo "$GOLANG_DOWNLOAD_SHA256  golang.tar.gz" | sha256sum -c - \
	&& tar -C /usr/local -xzf golang.tar.gz \
	&& rm golang.tar.gz

ENV GOPATH /go
ENV PATH $GOPATH/bin:/usr/local/go/bin:$PATH

ADD tensor.conf /etc/tensor.conf
RUN mkdir -p /root/.ssh
ADD bitbucket /root/.ssh/bitbucket
RUN chmod 600 /root/.ssh/bitbucket

RUN mkdir -p "$GOPATH/src" "$GOPATH/bin" && chmod -R 777 "$GOPATH"

WORKDIR $GOPATH/src/bitbucket.pearson.com/apseng/tensor

RUN echo "    IdentityFile ~/.ssh/bitbucket" >> /etc/ssh/ssh_config
RUN echo "StrictHostKeyChecking no " > /root/.ssh/config

ADD /run.sh /run.sh
RUN chmod +x /run.sh

COPY go-wrapper /usr/local/bin/

CMD ["/run.sh"]
