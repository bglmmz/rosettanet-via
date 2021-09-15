package main

import (
	"crypto/rand"
	"flag"
	"fmt"
	"github.com/bglmmz/grpc"
	"github.com/bglmmz/grpc/metadata"
	"golang.org/x/net/context"
	"math/big"
	"time"
	"via/conf"
	"via/creds"
	"via/helloworld"
	"via/proxy"
	"via/test"
)

var (
	address    string
	localVia   string
	destVia    string
	partner    string
	sslFile    string
	sslEnabled = false
)

const DefaultTaskId string = "testTaskId"
const DefaultPartyId string = "testPartyId"

func init() {
	flag.StringVar(&partner, "partner", "partner_1", "partner")
	flag.StringVar(&address, "address", "192.168.112.33:50051", "Math server listen address")
	//flag.StringVar(&address, "address", "127.0.0.1:10040", "Math server listen address")
	flag.StringVar(&localVia, "localVia", ":10031", "local VIA address")
	flag.StringVar(&destVia, "destVia", ":20031", "dest VIA address")
	flag.StringVar(&sslFile, "ssl", "conf/ssl-dahui.yml", "SSL config file")
	flag.Parse()

	if len(sslFile) > 0 {
		sslEnabled = true
	}
}

func main2() {
	tlsCredentialsAsClient, err := creds.NewClientGMTLSTwoWay(
		sslConfig.Conf.GMSSL.CaCertFile,
		sslConfig.Conf.GMSSL.ViaSignCertFile, sslConfig.Conf.GMSSL.ViaSignKeyFile,
		sslConfig.Conf.GMSSL.ViaEncryptCertFile, sslConfig.Conf.GMSSL.ViaEncryptKeyFile,
	)
	if err != nil {
		panic(err)
	}

	grpcOptions := []grpc.DialOption{grpc.WithTransportCredentials(tlsCredentialsAsClient)}

	ctx, _ := context.WithTimeout(context.Background(), time.Minute*10)
	ctx = metadata.AppendToOutgoingContext(ctx, proxy.MetadataTaskIdKey, DefaultTaskId)
	ctx = metadata.AppendToOutgoingContext(ctx, proxy.MetadataPartyIdKey, DefaultPartyId)

	conn, err := grpc.DialContext(ctx, address, grpcOptions...)
	if err != nil {
		fmt.Println("dial error:", err)
		return
	}

	cli := test.NewMathServiceClient(conn)
	ret, err := cli.Sum_Unary(ctx, &test.MetricList{Metric: randMetricList()})

	if err != nil {
		fmt.Println("helle response error:", err)
		return
	}
	fmt.Println("helle response:", ret.Val)
}

func randMetricList() []int64 {
	countBigInt, _ := rand.Int(rand.Reader, big.NewInt(20))
	count := int(countBigInt.Int64())
	metricList := make([]int64, count)
	for i := 0; i < count; i++ {
		metric, _ := rand.Int(rand.Reader, big.NewInt(1000))
		metricList[i] = metric.Int64()
	}
	return metricList
}

func main() {

	tlsCredentialsAsClient, err := creds.NewClientGMTLSTwoWay(
		sslConfig.Conf.GMSSL.CaCertFile,
		sslConfig.Conf.GMSSL.IoSignCertFile, sslConfig.Conf.GMSSL.IoSignKeyFile,
		sslConfig.Conf.GMSSL.IoEncryptCertFile, sslConfig.Conf.GMSSL.IoEncryptKeyFile,
	)
	if err != nil {
		panic(err)
	}

	grpcOptions := []grpc.DialOption{grpc.WithTransportCredentials(tlsCredentialsAsClient)}

	ctx, _ := context.WithTimeout(context.Background(), time.Minute*10)
	ctx = metadata.AppendToOutgoingContext(ctx, proxy.MetadataTaskIdKey, DefaultTaskId)
	ctx = metadata.AppendToOutgoingContext(ctx, proxy.MetadataPartyIdKey, DefaultPartyId)

	conn, err := grpc.DialContext(ctx, address, grpcOptions...)
	if err != nil {
		fmt.Println("dial error:", err)
		return
	}

	cli := helloworld.NewGreeterClient(conn)
	ret, err := cli.SayHello(ctx, &helloworld.HelloRequest{Name: "LVXIAOYI"})

	if err != nil {
		fmt.Println("helle response error:", err)
		return
	}
	fmt.Println("helle response:", ret.Message)
}

var sslConfig *conf.Config

func init() {
	if !sslEnabled {
		return
	}
	sslConfig = conf.LoadSSLConfig(sslFile)

}
