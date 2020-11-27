# TSF部署
### 虚拟机部署
1. 编写start.sh：
```
#!/bin/bash

already_run=`ps -ef|grep "./provider"|grep -v grep|wc -l`
if [ ${already_run} -ne 0 ];then
	echo "provider already Running!!!! Stop it first"
	exit -1
fi

nohup ./provider >stdout.log 2>&1 &
```
替换其中的provider为实际的可执行二进制文件名

2. 编写stop.sh:
```
#!/bin/bash

pid=`ps -ef|grep "./provider"|grep -v grep|awk '{print $2}'`
kill -SIGTERM $pid
echo "process ${pid} killed"
```
替换其中的provider为实际的可执行二进制文件名

3. 编写cmdline:
`./provider`
替换其中的provider为实际的可执行二进制文件名

4. 将可执行二进制文件拷贝进当前目录
   
5. 将当前目录打包成tar.gz: tar –czf provider.tar.gz *
   
6. 上传provider.tar.gz并部署
   
### 容器部署
1. 编写Dockerfile:
```
FROM centos:7

RUN echo "ip_resolve=4" >> /etc/yum.conf
#RUN yum update -y && yum install -y ca-certificates

# 设置时区。这对于日志、调用链等功能能否在 TSF 控制台被检索到非常重要。
RUN /bin/cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime
RUN echo "Asia/Shanghai" > /etc/timezone
ENV workdir /app/

COPY provider ${workdir}
WORKDIR ${workdir}

# tsf-consul-template-docker 用于文件配置功能，如不需要可注释掉该行
#ADD tsf-consul-template-docker.tar.gz /root/

# JAVA_OPTS 环境变量的值为部署组的 JVM 启动参数，在运行时 bash 替换。使用 exec 以使 Java 程序可以接收 SIGTERM 信号。
CMD ["sh", "-ec", "exec ${workdir}provider ${JAVA_OPTS}"]
```
替换其中的provider为实际的可执行二进制文件名
2. 将编译出的二进制文件放在Dockfile同一目录下
   
3. 打包镜像docker build . -t ccr.ccs.tencentyun.com/tsf_xxx/provider:1.0
   
4. docker push ccr.ccs.tencentyun.com/tsf_xxx/provider:1.0
   
5. 在tsf上部署镜像