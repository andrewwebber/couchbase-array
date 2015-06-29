FROM alpine

ADD cbfs-client.linux /usr/bin/cbfs-client.linux
RUN chmod +x /usr/bin/cbfs-client.linux
ADD ./monitor /opt/monitor

EXPOSE 8484

CMD []
ENTRYPOINT ["cbfs-client.linux"]
