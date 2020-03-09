FROM ubuntu:18.04

WORKDIR /go
VOLUME /opt/log
EXPOSE 8889
STOPSIGNAL SIGINT
COPY  ./ /go/

# Enable MADV_DONTNEED for Go-runtime.
# It will free memory to OS more active.
# More informations: https://github.com/golang/go/issues/28466
ENV GODEBUG madvdontneed=1

CMD ["/go/grablagou"]

# docker run -d --name data_grablagou \
# -v /etc/localtime:/etc/localtime \
# -v /opt/log/data_grablagou/:/opt/log/ \
# -p 8889:8889 \
# -e ENV=prod \
# -e MySQL_Username="ts" \
# -e MySQL_Password="sdfe232t9ddde3d" \
# -e MySQL_Address="192.168.8.221" \
# -e MySQL_Port="3307" \
# hub.ifchange.com/data_group/data_grablagou
