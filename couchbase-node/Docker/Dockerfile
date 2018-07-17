FROM couchbase:enterprise-5.1.1

ADD run.sh /usr/bin/couchbase-node
RUN chmod +x /usr/bin/couchbase-node
ADD couchbase-node-announce.linux /usr/bin/couchbase-node-announce
RUN chmod +x /usr/bin/couchbase-node-announce

ENTRYPOINT ["/usr/bin/couchbase-node"]
