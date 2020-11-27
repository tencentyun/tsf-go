FROM centos:7

RUN echo "ip_resolve=4" >> /etc/yum.conf
#RUN yum update -y && yum install -y ca-certificates

RUN /bin/cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime
RUN echo "Asia/Shanghai" > /etc/timezone
ENV workdir /app/

COPY provider ${workdir}
WORKDIR ${workdir}

CMD ["./provider"]