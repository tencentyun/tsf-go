FROM centos:7

RUN echo "ip_resolve=4" >> /etc/yum.conf
#RUN yum update -y && yum install -y ca-certificates

RUN /bin/cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime
RUN echo "Asia/Shanghai" > /etc/timezone
ENV workdir /app/

COPY consumer ${workdir}
WORKDIR ${workdir}

# 如果加了${JAVA_OPTS},需要在TSF的容器部署组启动参数中删除默认的"-Xms128m xxx"参数,否则会启动失败
CMD ["sh", "-ec", "exec ${workdir}consumer ${JAVA_OPTS}"]