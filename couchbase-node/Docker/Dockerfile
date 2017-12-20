FROM debian:jessie

RUN apt-get update && apt-get install -y \
    ca-certificates wget python-httplib2 curl
# RUN apt-get install libssl-dev -y
RUN wget -N -P /tmp https://packages.couchbase.com/releases/5.0.1/couchbase-server-community_5.0.1-debian8_amd64.deb
RUN dpkg -i /tmp/couchbase-server-community_5.0.1-debian8_amd64.deb
RUN rm /tmp/couchbase-server-community_5.0.1-debian8_amd64.deb
RUN apt-get install -y curl
ADD run.sh /usr/bin/couchbase-node
RUN chmod +x /usr/bin/couchbase-node
ADD couchbase-node-announce.linux /usr/bin/couchbase-node-announce
RUN chmod +x /usr/bin/couchbase-node-announce

ENTRYPOINT ["/usr/bin/couchbase-node"]

EXPOSE 8091 8092 11207 11210 11211 18091 18092
