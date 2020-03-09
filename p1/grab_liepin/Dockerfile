FROM ubuntu:18.04

WORKDIR /go
VOLUME /opt/log
EXPOSE 8892
STOPSIGNAL SIGINT
COPY  ./ /go/

CMD ["/go/grabliepin"]

# docker run -d --name data_grabliepin \
# -v /etc/localtime:/etc/localtime \
# -v /opt/log/data_grabliepin/:/opt/log/data_grabliepin/ \
# -p 8892:8892 \
# -e ENV=prod \
# -e MySQL_Username="ts" \
# -e MySQL_Password="sdfe232t9ddde3d" \
# -e MySQL_Address="192.168.8.221" \
# -e MySQL_Port="3307" \
# hub.ifchange.com/data_group/data_grabliepin
