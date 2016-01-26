FROM debian:latest

RUN apt-get update && apt-get install -y \
    ca-certificates wget python-httplib2 curl

RUN wget -N -P /tmp http://packages.couchbase.com/releases/4.0.0/couchbase-server-community_4.0.0-debian7_amd64.deb
RUN dpkg -i /tmp/couchbase-server-community_4.0.0-debian7_amd64.deb
RUN rm /tmp/couchbase-server-community_4.0.0-debian7_amd64.deb
RUN apt-get install -y curl
ADD run.sh /usr/bin/couchbase-node
RUN chmod +x /usr/bin/couchbase-node
ADD couchbase-node-announce.linux /usr/bin/couchbase-node-announce
RUN chmod +x /usr/bin/couchbase-node-announce

CMD ["/usr/bin/couchbase-node"]

EXPOSE 8091 8092 11207 11210 11211 18091 18092
