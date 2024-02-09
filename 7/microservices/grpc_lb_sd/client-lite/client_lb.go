package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"strconv"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/resolver"
	"google.golang.org/grpc/resolver/manual"

	"gws/7/microservices/grpc/session"

	consulapi "github.com/hashicorp/consul/api"
)

var (
	consulAddr = flag.String("addr", "127.0.0.1:8500", "consul addr (8500 in original consul)")
)

var (
	consul *consulapi.Client
)

func main() {
	flag.Parse()

	var err error
	config := consulapi.DefaultConfig()
	config.Address = *consulAddr
	consul, err = consulapi.NewClient(config)
	if err != nil {
		log.Fatalf("cant connect to consul")
	}

	health, _, err := consul.Health().Service("session-api", "", false, nil)
	if err != nil {
		log.Fatalf("cant get alive services")
	}

	servers := make([]resolver.Address, 0, len(health))
	currAddrs := []string{}
	for _, item := range health {
		addr := item.Service.Address + ":" + strconv.Itoa(item.Service.Port)
		currAddrs = append(currAddrs, addr)
		servers = append(servers, resolver.Address{Addr: addr})
	}

	nameResolver := manual.NewBuilderWithScheme("session-api")
	nameResolver.InitialState(resolver.State{
		Addresses: servers,
	})

	grcpConn, err := grpc.Dial(
		nameResolver.Scheme()+":///",
		grpc.WithDefaultServiceConfig(`{"loadBalancingConfig": [{"round_robin":{}}]}`),
		grpc.WithInsecure(),
		grpc.WithBlock(),
		grpc.WithResolvers(nameResolver),
	)
	if err != nil {
		log.Fatalf("cant connect to grpc")
	}
	defer grcpConn.Close()

	sessManager := session.NewAuthCheckerClient(grcpConn)

	// тут мы будем периодически опрашивать консул на предмет изменений
	go runOnlineServiceDiscovery(nameResolver, currAddrs)

	ctx := context.Background()
	step := 1
	for {
		// проверяем несуществуюущую сессию
		// потому что сейчас между сервисами нет общения
		// получаем загшулку
		sess, err := sessManager.Check(ctx,
			&session.SessionID{
				ID: "not_exist_" + strconv.Itoa(step),
			})
		fmt.Println("get sess", step, sess, err)

		time.Sleep(1500 * time.Millisecond)
		step++
	}
}

func runOnlineServiceDiscovery(nameResolver *manual.Resolver, servers []string) {
	currAddrs := make(map[string]struct{}, len(servers))
	for _, addr := range servers {
		currAddrs[addr] = struct{}{}
	}
	ticker := time.Tick(5 * time.Second)
	for _ = range ticker {
		health, _, err := consul.Health().Service("session-api", "", false, nil)
		if err != nil {
			log.Fatalf("cant get alive services")
		}

		newAddrs := make(map[string]struct{}, len(health))
		servers := make([]resolver.Address, 0, len(health))
		for _, item := range health {
			addr := item.Service.Address + ":" + strconv.Itoa(item.Service.Port)
			newAddrs[addr] = struct{}{}
			servers = append(servers, resolver.Address{Addr: addr})
		}

		for _, item := range health {
			addr := item.Service.Address + ":" + strconv.Itoa(item.Service.Port)
			newAddrs[addr] = struct{}{}
		}

		updates := 0
		// проверяем что удалилось
		for addr := range currAddrs {
			if _, exist := newAddrs[addr]; !exist {
				updates++
				delete(currAddrs, addr)
				fmt.Println("remove", addr)
			}
		}
		// проверяем что добавилось
		for addr := range newAddrs {
			if _, exist := currAddrs[addr]; !exist {
				updates++
				currAddrs[addr] = struct{}{}
				fmt.Println("add", addr)
			}
		}
		if updates > 0 {
			nameResolver.CC.NewAddress(servers)
		}
	}
}
