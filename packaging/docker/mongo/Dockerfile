FROM ubuntu:latest

# add our user and group first to make sure their IDs get assigned consistently, regardless of whatever dependencies get added
RUN groupadd -r mongodb && useradd -r -g mongodb mongodb

RUN apt-get update \
	&& apt-get install -y -q --no-install-recommends \
		numactl \
	&& rm -rf /var/lib/apt/lists/*

RUN apt-key adv --keyserver hkp://keyserver.ubuntu.com:80 --recv 42F3E95A2C4F08279C4960ADD68FA50FEA312927

RUN echo "deb http://repo.mongodb.org/apt/ubuntu xenial/mongodb-org/3.2 multiverse" > /etc/apt/sources.list.d/mongodb-org-3.2.list

RUN set -x \
	&& apt-get update \
	&& apt-get install -y -q mongodb-org \
	&& rm -rf /var/lib/apt/lists/* \
	&& rm -rf /var/lib/mongodb \
	&& mv /etc/mongod.conf /etc/mongod.conf.orig

RUN mkdir -p /data/db && chown -R mongodb:mongodb /data/db

# mongod config
ENV AUTH yes
ENV STORAGE_ENGINE wiredTiger
ENV JOURNALING yes

# mongo root user (change me!)
ENV MONGO_ROOT_USER root
ENV MONGO_ROOT_PASSWORD root123

# mongo app user + database (change me!)
ENV MONGO_APP_USER tensor
ENV MONGO_APP_PASSWORD tensor
ENV MONGO_APP_DATABASE tensordb

RUN mkdir /opt/mongo
ADD mongo_setup_users.sh /opt/mongo/mongo_setup_users.sh
RUN chmod +x /opt/mongo/mongo_setup_users.sh

ADD run.sh /run.sh
RUN chmod +x /run.sh
RUN chown -R mongodb:mongodb /opt/mongo

VOLUME /data/db

EXPOSE 27017

CMD ["/run.sh"]
