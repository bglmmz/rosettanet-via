package main

import (
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection"
	"io"
	"io/ioutil"
	"log"
	"math/big"
	"net"
	"via/conf"
	"via/proxy"

	"time"
	"via/test"
	"via/via"
)

var (
	address    string
	localVia   string
	destVia    string
	partner    string
	tlsFile    string
	tlsEnabled = false
	commands   map[string]Command
)

// A Command is the API for a sub-command
type Command func(*mathServer) error

const DefaultTaskId string = "testTaskId"
const DefaultPartyId string = "testPartyId"

func init() {
	flag.StringVar(&partner, "partner", "partner_1", "partner")
	flag.StringVar(&address, "address", ":10040", "Math server listen address")
	flag.StringVar(&localVia, "localVia", ":10031", "local VIA address")
	flag.StringVar(&destVia, "destVia", ":20031", "dest VIA address")
	flag.StringVar(&tlsFile, "tls", "", "TLS config file")
	flag.Parse()

	if len(tlsFile) > 0 {
		tlsEnabled = true
	}
	commands = map[string]Command{
		"unary":           unary,
		"serverStreaming": serverStreaming,
		"clientStreaming": clientStreaming,
		"bidi":            bidi,
	}
}

type mathServer struct {
	ctx    context.Context
	client test.MathServiceClient
}

func (s *mathServer) dialDestVIA() {
	ctx := context.Background()
	ctx = metadata.AppendToOutgoingContext(ctx, proxy.MetadataTaskIdKey, DefaultTaskId)
	ctx = metadata.AppendToOutgoingContext(ctx, proxy.MetadataPartyIdKey, DefaultPartyId)
	s.ctx = ctx

	if tlsEnabled {
		if conn, err := grpc.Dial(destVia, grpc.WithTransportCredentials(tlsCredentialsAsClient)); err != nil {
			log.Fatalf("did not connect to dest VIA server: %v", err)
		} else {
			log.Printf("Success to connect to dest VIA server with secure: %v", destVia)
			s.client = test.NewMathServiceClient(conn)
		}
	} else {
		if conn, err := grpc.Dial(destVia, grpc.WithInsecure()); err != nil {
			log.Fatalf("did not connect to dest VIA server: %v", err)
		} else {
			log.Printf("Success to connect to dest VIA server with insecure: %v", destVia)
			s.client = test.NewMathServiceClient(conn)
		}
	}
}

func (s *mathServer) Sum_Unary(ctx context.Context, metricList *test.MetricList) (*test.SumResponse, error) {
	log.Printf("??????(unary)?????????????????????%v", metricList.Metric)
	var sum int64
	for _, metric := range metricList.Metric {
		sum += metric
	}
	log.Printf("??????????????????(unary)?????????????????????%v sum=%d", metricList, sum)
	return &test.SumResponse{Count: int32(len(metricList.Metric)), Val: sum}, nil
}

func (s *mathServer) Sum_ServerStreaming(metricList *test.MetricList, stream test.MathService_Sum_ServerStreamingServer) error {
	log.Printf("??????(serverStreaming)?????????????????????%v", metricList.Metric)
	var sum int64
	for _, metric := range metricList.Metric {
		sum += metric
		log.Printf("??????(serverStreaming)??????????????????, temp sum???%d", sum)
		if err := stream.Send(&test.SumResponse{Count: int32(len(metricList.Metric)), Val: sum}); err != nil {
			return err
		}
	}
	log.Printf("??????????????????(serverStreaming)?????????????????????sum%v=%d", metricList, sum)
	return nil
}

func (s *mathServer) Sum_ClientStreaming(stream test.MathService_Sum_ClientStreamingServer) error {
	var metricList []int64
	var count int32
	var sum int64
	for {
		r, err := stream.Recv()
		if err == io.EOF {
			log.Printf("??????????????????(clientStreaming)?????????????????????sum%v=%d", metricList, sum)
			return stream.SendAndClose(&test.SumResponse{Count: count, Val: sum})
		}
		if err != nil {
			return err
		}
		count++
		sum += r.Metric
		metricList = append(metricList, r.Metric)
		log.Printf("??????(clientStreaming)??????????????????, ????????? %d ?????????: %d", count, r.Metric)
	}
}

func (s *mathServer) Sum_BidiStreaming(stream test.MathService_Sum_BidiStreamingServer) error {
	var metricList []int64
	var count int32
	var sum int64
	for {
		r, err := stream.Recv()
		if err == io.EOF {
			log.Printf("??????????????????(bidi)?????????????????????sum(%v)=%d", metricList, sum)
			return nil
		}
		if err != nil {
			return err
		}

		count++
		sum += r.Metric
		metricList = append(metricList, r.Metric)
		log.Printf("??????(bidi)??????????????????, ????????? %d ?????????:%d, sum???%d", count, r.Metric, sum)
		err = stream.Send(&test.SumResponse{Count: count, Val: sum})
		if err != nil {
			log.Fatalf("?????????????????????????????????, err: %v", err)
			return err
		}
	}
}

func signupTask() error {
	log.Printf("dial to local VIA server on %v", localVia)

	var conn *grpc.ClientConn
	var err error

	if tlsCredentialsAsClient == nil {
		conn, err = grpc.Dial(localVia, grpc.WithInsecure())
	} else {
		conn, err = grpc.Dial(localVia, grpc.WithTransportCredentials(tlsCredentialsAsClient))
	}

	if err != nil {
		log.Fatalf("did not connect to local VIA server: %v", err)
	}
	defer conn.Close()

	log.Printf("signup task to local VIA server %v", localVia)

	c := via.NewVIAServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	r, err := c.Signup(ctx, &via.SignupReq{TaskId: DefaultTaskId, PartyId: DefaultPartyId, Address: address})
	if err != nil {
		return err
	}

	log.Printf("Signup task result: %v", r.Result)
	return nil
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

func unary(s *mathServer) error {
	metricList := randMetricList()
	log.Printf("?????????????????????????????????(unary)???%v", metricList)
	resp, err := s.client.Sum_Unary(s.ctx, &test.MetricList{Metric: metricList})
	if err != nil {
		return err
	}
	log.Printf("unary resp, count: %d, sum: %d", resp.Count, resp.Val)
	return nil
}

func serverStreaming(s *mathServer) error {
	metricList := randMetricList()
	log.Printf("?????????????????????????????????(serverStreaming)???%v", metricList)
	//??????100?????????
	stream, err := s.client.Sum_ServerStreaming(s.ctx, &test.MetricList{Metric: metricList})
	if err != nil {
		return err
	}
	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		log.Printf("serverStreaming resp, count: %d, sum: %d", resp.Count, resp.Val)
	}
	return nil
}

func clientStreaming(s *mathServer) error {
	stream, err := s.client.Sum_ClientStreaming(s.ctx)
	if err != nil {
		return err
	}
	metricList := randMetricList()
	log.Printf("?????????????????????????????????(clientStreaming)???%v", metricList)
	for i := 0; i < len(metricList); i++ {
		metric := metricList[i]
		log.Printf("?????????????????? %d ????????????%d", i+1, metric)
		err := stream.Send(&test.Metric{Metric: metric})
		if err != nil {
			return err
		}
	}
	resp, err := stream.CloseAndRecv()
	if err != nil {
		return err
	}

	log.Printf("clientStreaming resp, count: %d, sum: %d", resp.Count, resp.Val)
	return nil
}

func bidi(s *mathServer) error {
	stream, err := s.client.Sum_BidiStreaming(s.ctx)
	if err != nil {
		return err
	}

	metricList := randMetricList()
	log.Printf("?????????????????????????????????(bidi)???%v", metricList)

	for i := 0; i < len(metricList); i++ {
		metric := metricList[i]
		log.Printf("?????????????????? %d ????????????%d", i+1, metric)
		err = stream.Send(&test.Metric{Metric: metric})
		if err != nil {
			log.Fatalf("???????????????????????????, %v", err)
			return err
		}

		resp, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("?????????????????????????????????????????????, %v", err)
			return err
		}
		log.Printf("??????????????????????????????????????????count: %d, average: %d", resp.Count, resp.Val)
		time.Sleep(time.Duration(100) * time.Millisecond)
	}
	if err := stream.CloseSend(); err != nil {
		log.Fatalf("?????????????????????????????????, %v", err)
	}
	return nil
}

func main() {
	lis, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	var grpcServer *grpc.Server
	grpcServer = grpc.NewServer()
	if tlsEnabled {
		log.Print("running math server with secure!")
		grpcServer = grpc.NewServer(grpc.Creds(tlsCredentialsAsServer))
	} else {
		log.Print("running math server with insecure!")
		grpcServer = grpc.NewServer()
	}

	mathServ := &mathServer{}

	mathServ.dialDestVIA()

	// Register a non-ssl server for local VIA
	test.RegisterMathServiceServer(grpcServer, mathServ)
	reflection.Register(grpcServer)

	log.Printf("Listening on %v", address)
	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()

	// Signup TASK
	if err := signupTask(); err != nil {
		log.Fatalf("failed to signup task to local VIA server: %v", err)
	}

	//????????????????????????????????????
	time.Sleep(time.Duration(5) * time.Second)

	var cmdLine string

	for {
		fmt.Println("Please input command???unary|serverStreaming|clientStreaming|bidi|quit")
		fmt.Scanln(&cmdLine)
		if cmdLine == "quit" {
			break
		} else {
			// Execute command
			if cmd, ok := commands[cmdLine]; ok {
				if err := cmd(mathServ); err != nil {
					log.Fatal(err)
				}
			} else {
				log.Printf("invalid command: %s", cmdLine)
			}
		}
	}
}

var tlsCredentialsAsClient credentials.TransportCredentials
var tlsCredentialsAsServer credentials.TransportCredentials
var tlsConfig *conf.TlsConfig

func init() {

	if !tlsEnabled {
		return
	}
	tlsConfig = conf.LoadTlsConfig(tlsFile)

	log.Printf("??????????????????tlsConfig.Tls.Mode=%s", tlsConfig.Tls.Mode)

	// Load io's certificate and private key
	ioCert, err := tls.LoadX509KeyPair(tlsConfig.Tls.IoCertFile, tlsConfig.Tls.IoKeyFile)
	if err != nil {
		log.Fatalf("failed to load VIA certificate and private key. %v", err)
	}

	//??????SSL?????????VIA?????????????????????ca?????????
	caPool := loadCaPool()

	if tlsConfig.Tls.Mode == "one_way" {
		// VIA??????ssl???VIA????????????ssl????????????node??????node????????????ssl?????????????????????node?????????ssl??????
		// ??????io????????????????????????ca?????????????????????VIA??????ssl???tls.config?????????
		log.Printf("VIA??????SSL")
		serverSSLConfig := &tls.Config{
			Certificates: []tls.Certificate{ioCert},
			ClientAuth:   tls.NoClientCert,
		}
		tlsCredentialsAsServer = credentials.NewTLS(serverSSLConfig)

		clientSSLConfig := &tls.Config{
			RootCAs: caPool,
		}
		tlsCredentialsAsClient = credentials.NewTLS(clientSSLConfig)

	} else if tlsConfig.Tls.Mode == "two_way" {
		// VIA??????ssl
		// ??????io????????????????????????ca?????????
		log.Printf("VIA??????SSL")
		serverSSLConfig := &tls.Config{
			Certificates: []tls.Certificate{ioCert},
			ClientAuth:   tls.RequireAndVerifyClientCert,
			ClientCAs:    caPool,
		}
		tlsCredentialsAsServer = credentials.NewTLS(serverSSLConfig)

		clientSSLConfig := &tls.Config{
			Certificates: []tls.Certificate{ioCert},
			RootCAs:      caPool,
		}
		tlsCredentialsAsClient = credentials.NewTLS(clientSSLConfig)
	} else {
		log.Fatalf("Tls.Mode value error: %s", tlsConfig.Tls.Mode)
	}
}

func loadCaPool() *x509.CertPool {
	// Load certificate of the CA who signed server's certificate
	pemServerCA, err := ioutil.ReadFile("cert/ca.crt")
	if err != nil {
		log.Fatalf("failed to read CA cert file. %v", err)
	}

	caPool := x509.NewCertPool()
	if !caPool.AppendCertsFromPEM(pemServerCA) {
		log.Fatalf("failed to add CA cert to cert pool. %v", err)
	}

	return caPool
}
