FROM ubuntu:18.04

WORKDIR /go
VOLUME /opt/log
EXPOSE 8888
STOPSIGNAL SIGINT
COPY  ./ /go/

CMD ["/go/grabmail"]

# VANKE
# 电信：61.144.205.216
# 联通：58.251.18.42
# 移动：120.234.45.6

# docker run -d --name data_grabmail \
# -v /etc/localtime:/etc/localtime \
# -v /opt/log/data_grabmail/:/opt/log/ \
# -p 8888:8888 \
# -e ENV=prod \
# -e MySQL_Username="ts" \
# -e MySQL_Password="sdfe232t9ddde3d" \
# -e MySQL_Address="192.168.8.221" \
# -e MySQL_Port="3307" \
# --add-host mail.vanke.com:61.144.205.216 \
# hub.ifchange.com/data_group/data_grabmail
