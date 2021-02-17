package env

import (
	"flag"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"

	"go.uber.org/zap"
)

var (
	logLevel          int
	logPath           string
	tracePath         string
	monitorPath       string
	consulAddressList string
	consulHost        string
	consulPort        int
	instanceId        string
	token             string
	localIP           string
	namespaceID       string
	applicationID     string
	groupID           string
	progVersion       string
	region            string
	zone              string
	serviceName       string
	port              int
	disableGrpcHttp   bool
	gopsPort          int
	pprofPort         int
	disableGops       bool
	disablePprof      bool

	sshUser    string
	sshHost    string
	sshPort    int
	sshPass    string
	sshKeyFile string
	remoteIP   string
)

func LogLevel() int {
	//DebugLevel Level = -1
	//InfoLevel Level = 0
	//WarnLevel Level = 1
	//ErrorLevel Level = 2
	return logLevel
}

func LogPath() string {
	if logPath == "" {
		if Token() == "" {
			return "stdout"
		}
		return "/data/logs/root.log"
	}
	return logPath
}

func TracePath() string {
	if tracePath == "" {
		if Token() == "" {
			// not run on tsf platform
			return "./trace/trace_log.log"
		}
		return "/data/tsf_apm/trace/logs/trace_log.log"
	}
	return tracePath
}

func MonitorPath() string {
	if monitorPath == "" {
		if Token() == "" {
			// not run on tsf platform
			return "./monitor/invocation_log.log"
		}
		return "/data/tsf_apm/monitor/logs/invocation_log.log"
	}
	return monitorPath
}

func ConsulHost() string {
	if consulHost == "" {
		return "127.0.0.1"
	}
	return consulHost
}

func ConsulPort() int {
	if consulPort == 0 {
		return 8500
	}
	return consulPort
}
func ConsulAddressList() []string {
	if consulAddressList == "" {
		return []string{fmt.Sprintf("%s:%d", ConsulHost(), ConsulPort())}
	}

	return strings.Split(consulAddressList, ",")
}

func InstanceId() string {
	if instanceId == "" {
		hostname, err := os.Hostname()
		if err == nil {
			return hostname + "-" + LocalIP()
		}
		return LocalIP()
	}
	return instanceId
}

func Token() string {
	return token
}

func LocalIP() string {
	if localIP == "" {
		return getIntranetIP()
	}
	return localIP
}

func NamespaceID() string {
	return namespaceID
}

func ApplicationID() string {
	return applicationID
}

func GroupID() string {
	return groupID
}

func ProgVersion() string {
	return progVersion
}

func Region() string {
	return region
}

func Zone() string {
	return zone
}

func ServiceName() string {
	if serviceName == "" {
		return "tsf-default-client-go"
	}
	return serviceName
}

func Port() int {
	if port == 0 {
		return 8080
	}
	return port
}

func SSHUser() string {
	return sshUser
}

func SSHPass() string {
	return sshPass
}

func SSHHost() string {
	return sshHost
}

func SSHPort() int {
	if sshPort == 0 {
		return 22
	}
	return sshPort
}

func SSHKey() string {
	if sshKeyFile == "" {
		return os.Getenv("HOME") + "/.ssh/id_rsa"
	}
	return sshKeyFile
}

func RemoteIP() string {
	return remoteIP
}

func DisableGrpcHttp() bool {
	return disableGrpcHttp
}

func DisableDisableGops() bool {
	return disableGops
}

func DisableDisablePprof() bool {
	return disablePprof
}

func PprofPort() int {
	if pprofPort == 0 {
		return 47077
	}
	return pprofPort
}

func GopsPort() int {
	if gopsPort == 0 {
		return 46066
	}
	return gopsPort
}

func init() {
	flag.IntVar(&logLevel, "tsf_log_level", parseInt(os.Getenv("tsf_log_level")), "-tsf_log_level 0")
	flag.StringVar(&logPath, "tsf_log_path", os.Getenv("tsf_log_path"), "-tsf_log_path stdout")
	flag.StringVar(&tracePath, "tsf_trace_path", os.Getenv("tsf_trace_path"), "-tsf_trace_path ./trace")
	flag.StringVar(&monitorPath, "tsf_monitor_path", os.Getenv("tsf_monitor_path"), "-tsf_monitor_path ./monitor")
	flag.StringVar(&consulHost, "tsf_consul_ip", os.Getenv("tsf_consul_ip"), "-tsf_consul_ip 127.0.0.1")
	flag.StringVar(&consulAddressList, "tsf_consul_list", os.Getenv("tsf_consul_list"), "-tsf_consul_list 127.0.0.1:8080")
	flag.IntVar(&consulPort, "tsf_consul_port", parseInt(os.Getenv("tsf_consul_port")), "-tsf_consul_port 85000")
	flag.StringVar(&instanceId, "tsf_instance_id", os.Getenv("tsf_instance_id"), "-tsf_instance_id xxx")
	flag.StringVar(&token, "tsf_token", os.Getenv("tsf_token"), "-tsf_token xxx")
	flag.StringVar(&localIP, "tsf_local_ip", os.Getenv("tsf_local_ip"), "-tsf_local_ip 127.0.0.1")
	flag.StringVar(&namespaceID, "tsf_namespace_id", os.Getenv("tsf_namespace_id"), "-tsf_namespace_id xxx")
	flag.StringVar(&applicationID, "tsf_application_id", os.Getenv("tsf_application_id"), "-tsf_application_id xxx")
	flag.StringVar(&groupID, "tsf_group_id", os.Getenv("tsf_group_id"), "-tsf_group_id xxx")
	flag.StringVar(&progVersion, "tsf_prog_version", os.Getenv("tsf_prog_version"), "-tsf_prog_version 1.0.0")
	flag.StringVar(&zone, "tsf_zone", os.Getenv("tsf_zone"), "-tsf_zone 100004")
	flag.StringVar(&region, "tsf_region", os.Getenv("tsf_region"), "-tsf_region ap-guangzhou")
	flag.StringVar(&region, "tsf_service_name", os.Getenv("tsf_service_name"), "-service_name tsf-default-client-grpc")
	flag.IntVar(&port, "tsf_service_port", parseInt(os.Getenv("tsf_service_port")), "-service_port 8080")
	flag.BoolVar(&disableGrpcHttp, "tsf_disable_grpc_http", parseBool(os.Getenv("tsf_disable_grpc_http")), "-tsf_disable_grpc_http false")
	flag.BoolVar(&disableGops, "tsf_disable_gops", parseBool(os.Getenv("tsf_disable_gops")), "-tsf_disable_gops false")
	flag.BoolVar(&disablePprof, "tsf_disable_pprof", parseBool(os.Getenv("tsf_disable_pprof")), "-tsf_disable_pprof false")
	flag.IntVar(&pprofPort, "tsf_pprof_port", parseInt(os.Getenv("tsf_pprof_port")), "-tsf_pprof_port 47077")
	flag.IntVar(&gopsPort, "tsf_gops_port", parseInt(os.Getenv("tsf_gops_port")), "-tsf_gops_port 46066")

	flag.StringVar(&sshUser, "ssh_user", os.Getenv("ssh_user"), "-ssh_user root")
	flag.StringVar(&sshHost, "ssh_host", os.Getenv("ssh_host"), "-ssh_host 127.0.0.1")
	flag.IntVar(&sshPort, "ssh_port", parseInt(os.Getenv("ssh_port")), "-ssh_port 22")
	flag.StringVar(&sshPass, "ssh_pass", os.Getenv("ssh_pass"), "-ssh_pass 123456")
	flag.StringVar(&sshKeyFile, "ssh_key", os.Getenv("ssh_key"), "-ssh_key ~/.ssh/id_rsa")
	flag.StringVar(&remoteIP, "cvm_remote_ip", os.Getenv("cvm_remote_ip"), "-cvm_remote_ip 172.168.1.1")

	flag.String("Xms128m", "", "-Xms128m")
	flag.String("Xmx512m", "", "-Xmx512m")
	flag.String("XX:MetaspaceSize", "", "-XX:MetaspaceSize")
	flag.String("XX:MaxMetaspaceSize", "", "XX:MaxMetaspaceSize")

}

func parseInt(i string) int {
	res, _ := strconv.ParseInt(i, 10, 64)
	return int(res)
}

func parseBool(b string) bool {
	ok, _ := strconv.ParseBool(b)
	return ok
}

func getIntranetIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		fmt.Println("net.InterfaceAddrs get ip address failed!", zap.Error(err))
		return ""
	}

	for _, address := range addrs {
		// 检查ip地址判断是否回环地址
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return ""
}
