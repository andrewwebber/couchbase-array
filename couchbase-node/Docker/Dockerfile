FROM debian:latest

RUN apt-get update && apt-get install -y \
    ca-certificates wget librtmp0 python-httplib2

RUN wget -N -P /tmp http://packages.couchbase.com/releases/3.0.1/couchbase-server-community_3.0.1-debian7_amd64.deb
RUN dpkg -i /tmp/couchbase-server-community_3.0.1-debian7_amd64.deb
RUN rm /tmp/couchbase-server-community_3.0.1-debian7_amd64.deb
RUN apt-get install -y curl
ADD run.sh /usr/bin/couchbase-node
RUN chmod +x /usr/bin/couchbase-node
ADD couchbase-node-announce.linux /usr/bin/couchbase-node-announce
RUN chmod +x /usr/bin/couchbase-node-announce

CMD ["/usr/bin/couchbase-node"]
