package main

import (
	"context"
	"encoding/json"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	peer2 "google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
	"net"
	"strings"
	"sync"
	"time"
)

type methodCall struct {
	method   string
	consumer string
}

type Microservice struct {
	UnimplementedAdminServer
	UnimplementedBizServer

	host string

	acl map[string][]string

	methodCalls chan methodCall

	methodCallSubsMu *sync.Mutex
	//map[consumer]chan for receiving registered method calls
	methodCallsSubs map[string]chan<- methodCall
}

func (m *Microservice) Logging(_ *Nothing, ss Admin_LoggingServer) error {
	consumer, err := getConsumerFromContext(ss.Context())
	if err != nil {
		return err
	}
	peer, ok := peer2.FromContext(ss.Context())
	if !ok {
		return status.Errorf(codes.PermissionDenied, "peer is not provided")
	}

	methodCallChan := make(chan methodCall)
	m.methodCallSubsMu.Lock()
	m.methodCallsSubs[consumer] = methodCallChan
	m.methodCallSubsMu.Unlock()
	defer func() {
		m.methodCallSubsMu.Lock()
		delete(m.methodCallsSubs, consumer)
		m.methodCallSubsMu.Unlock()
		close(methodCallChan)
	}()

	for {
		select {
		case <-ss.Context().Done():
			return nil
		case call := <-methodCallChan:
			//skip call if it is from itself
			if call.consumer == consumer {
				continue
			}
			event := &Event{
				Timestamp: time.Now().Unix(),
				Consumer:  call.consumer,
				Method:    call.method,
				Host:      peer.Addr.String(),
			}
			err := ss.Send(event)
			if err != nil {
				return err
			}
			//log.Printf("[%s] sent [Event]:%v", consumer, event)
		}
	}
}
func (m *Microservice) Statistics(statsConfig *StatInterval, ss Admin_StatisticsServer) error {
	consumer, err := getConsumerFromContext(ss.Context())
	if err != nil {
		return err
	}

	methodCallChan := make(chan methodCall)
	m.methodCallSubsMu.Lock()
	m.methodCallsSubs[consumer] = methodCallChan
	m.methodCallSubsMu.Unlock()
	defer func() {
		m.methodCallSubsMu.Lock()
		delete(m.methodCallsSubs, consumer)
		m.methodCallSubsMu.Unlock()
		close(methodCallChan)
	}()

	byMethod := make(map[string]uint64)
	byConsumer := make(map[string]uint64)

	interval := statsConfig.GetIntervalSeconds()
	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ss.Context().Done():
			return nil
		case <-ticker.C:
			//releasing data

			err := ss.Send(&Stat{
				Timestamp:  time.Now().Unix(),
				ByMethod:   byMethod,
				ByConsumer: byConsumer,
			})
			if err != nil {
				return err
			}
			//log.Printf("[%s] send [byMethod]:%v [byConsumer]:%v", consumer, byMethod, byConsumer)

			byMethod = make(map[string]uint64)
			byConsumer = make(map[string]uint64)
		case call := <-methodCallChan:
			//accumulating data

			//skip call if it is from itself
			if call.consumer == consumer {
				continue
			}
			byMethod[call.method]++
			byConsumer[call.consumer]++
			//log.Printf("[%s] accumulated call %v to [byMethod]:%v [byConsumer]:%v", consumer, call, byMethod, byConsumer)
		}
	}
}

func (m *Microservice) Check(context.Context, *Nothing) (*Nothing, error) {
	return &Nothing{}, nil
}
func (m *Microservice) Add(context.Context, *Nothing) (*Nothing, error) {
	return &Nothing{}, nil
}
func (m *Microservice) Test(context.Context, *Nothing) (*Nothing, error) {
	return &Nothing{}, nil
}

func NewMicroservice(host, ACLData string) (*Microservice, error) {
	acl := make(map[string][]string)
	err := json.Unmarshal([]byte(ACLData), &acl)
	if err != nil {
		return nil, err
	}
	return &Microservice{
		host:             host,
		acl:              acl,
		methodCalls:      make(chan methodCall),
		methodCallSubsMu: &sync.Mutex{},
		methodCallsSubs:  make(map[string]chan<- methodCall),
	}, nil
}

// submitMethodCallsLoop receives registered method calls and
// send it to subscribers such as loggers & statisticians
func (m *Microservice) submitMethodCallsLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case call := <-m.methodCalls:
			m.methodCallSubsMu.Lock()
			for _, subChan := range m.methodCallsSubs {
				subChan <- call
			}
			m.methodCallSubsMu.Unlock()
		}
	}
}

func (m *Microservice) checkAccess(consumer, method string) bool {
	allowedMethods := m.acl[consumer]
	if len(allowedMethods) == 0 {
		return false
	}

	for _, allowedMethod := range allowedMethods {
		if method == allowedMethod {
			return true
		}

		if strings.Contains(allowedMethod, "*") {
			sectionsAllowed := strings.Split(allowedMethod, "/")
			sectionsGiven := strings.Split(method, "/")
			if len(sectionsAllowed) != len(sectionsGiven) {
				return false
			}

			for idx, sectionAllowed := range sectionsAllowed {
				if sectionAllowed == "*" {
					continue
				}

				if sectionsAllowed[idx] != sectionsGiven[idx] {
					return false
				}
			}

			return true
		}
	}
	return false
}

func (m *Microservice) unaryAuthCheckInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	consumer, err := getConsumerFromContext(ctx)
	if err != nil {
		return nil, err
	}

	authorized := m.checkAccess(consumer, info.FullMethod)
	if !authorized {
		return nil, status.Errorf(codes.Unauthenticated, "consumer %s has no access to %s", consumer, info.FullMethod)
	}

	return handler(ctx, req)
}

func (m *Microservice) streamAuthCheckInterceptor(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	consumer, err := getConsumerFromContext(ss.Context())
	if err != nil {
		return err
	}

	authorized := m.checkAccess(consumer, info.FullMethod)
	if !authorized {
		return status.Errorf(codes.Unauthenticated, "consumer %s has no access to %s", consumer, info.FullMethod)
	}

	return handler(srv, ss)
}

func (m *Microservice) unaryMethodCallsRegistrator(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	consumer, err := getConsumerFromContext(ctx)
	if err != nil {
		return nil, err
	}
	m.registerMethodCall(consumer, info.FullMethod)
	result, resultErr := handler(ctx, req)
	return result, resultErr
}

func (m *Microservice) streamMethodCallsRegistrator(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	consumer, err := getConsumerFromContext(ss.Context())
	if err != nil {
		return err
	}
	m.registerMethodCall(consumer, info.FullMethod)
	resultErr := handler(srv, ss)
	return resultErr
}

func (m *Microservice) registerMethodCall(consumer, method string) {
	m.methodCalls <- methodCall{
		method:   method,
		consumer: consumer,
	}
}

func getConsumerFromContext(ctx context.Context) (string, error) {
	md, mdExists := metadata.FromIncomingContext(ctx)
	if !mdExists {
		return "", status.Error(codes.InvalidArgument, "metadata is not provided")
	}

	consumers := md.Get("consumer")
	if len(consumers) != 1 {
		return "", status.Error(codes.Unauthenticated, "expected 1 consumer provided")
	}

	return consumers[0], nil
}

func StartMyMicroservice(ctx context.Context, listenAddr string, ACLData string) error {
	microservice, err := NewMicroservice(listenAddr, ACLData)
	if err != nil {
		return err
	}

	lis, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return err
	}

	server := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			microservice.unaryAuthCheckInterceptor,
			microservice.unaryMethodCallsRegistrator,
		),

		grpc.ChainStreamInterceptor(
			microservice.streamAuthCheckInterceptor,
			microservice.streamMethodCallsRegistrator,
		),
	)

	RegisterAdminServer(server, microservice)
	RegisterBizServer(server, microservice)

	go microservice.submitMethodCallsLoop(ctx)

	go func() {
		server.Serve(lis)
	}()

	go func() {
		for {
			select {
			case <-ctx.Done():
				server.GracefulStop()
				return
			}
		}
	}()

	return nil
}
