FROM centos:centos7
MAINTAINER Hilbert Space Project <udaya.balagalla@pearson.com>

# add our user and group first to make sure their IDs get assigned consistently, regardless of whatever dependencies get added
RUN groupadd -r mongodb && useradd -r -g mongodb mongodb

ADD mongodb-org-3.2.repo /etc/yum.repos.d/mongodb-org-3.2.repo

RUN yum -y update;
RUN yum install -y mongodb-org; yum clean all

RUN mv /etc/mongod.conf /etc/mongod.conf.orig

RUN mkdir -p /data/db && chown -R mongodb:mongodb /data/db

# mongod config
ENV AUTH yes
ENV STORAGE_ENGINE wiredTiger
ENV JOURNALING yes
ENV REP_SET rs0
ENV MONGO_SECONDARY mongo2:27017
ENV MONGO_ARBITER mongo3:27017

# mongo root user (change me!)
ENV MONGO_ROOT_USER root
ENV MONGO_ROOT_PASSWORD root123

# mongo app user + database (change me!)
ENV MONGO_APP_USER hilbert
ENV MONGO_APP_PASSWORD hilbert
ENV MONGO_APP_DATABASE hilbertspace

RUN mkdir /opt/mongo
ADD mongodb-keyfile /opt/mongo/mongodb-keyfile
ADD mongo_setup_users.sh /opt/mongo/mongo_setup_users.sh
ADD mongo_setup_repset.sh /opt/mongo/mongo_setup_repset.sh
ADD check-keyfile.sh /opt/mongo/check-keyfile.sh

RUN chmod +x /opt/mongo/check-keyfile.sh
RUN /opt/mongo/check-keyfile.sh
ADD run.sh /run.sh
RUN chown -R mongodb:mongodb /opt/mongo
RUN chmod 600 /opt/mongo/mongodb-keyfile

VOLUME /data/db

EXPOSE 27017 28017

CMD ["/run.sh"]
