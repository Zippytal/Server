package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"

	"github.com/loisBN/zippytal-desktop/back/manager"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func main() {
	lis, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatalln(err)
	}
	m, err := manager.NewManager()
	if err != nil {
		log.Fatal(err)
	}
	h := manager.NewWSHandler("app",m, []manager.WSMiddleware{manager.NewWSStateMiddleware()}, []manager.HTTPMiddleware{manager.NewSquadHTTPMiddleware(m), &manager.AuthHTTPMiddleware{}, manager.NewHostedSquadHTTPMiddleware(m), manager.NewPeerHTTPMiddleware(m),manager.NewNodeHTTPMiddleware(m),manager.NewZoneHTTPMiddleware(m)})
	h2 := manager.NewWSHandler("web",m, []manager.WSMiddleware{manager.NewWSStateMiddleware()}, []manager.HTTPMiddleware{manager.NewSquadHTTPMiddleware(m), &manager.AuthHTTPMiddleware{}, manager.NewHostedSquadHTTPMiddleware(m), manager.NewPeerHTTPMiddleware(m),manager.NewNodeHTTPMiddleware(m)})
	fmt.Println("server launch")
	certFile := "/etc/letsencrypt/live/app.zippytal.com/fullchain.pem"
	keyFile := "/etc/letsencrypt/live/app.zippytal.com/privkey.pem"
	certFileW := "/etc/letsencrypt/live/zippytal.com/fullchain.pem"
	keyFileW := "/etc/letsencrypt/live/zippytal.com/privkey.pem"
	certFileDev := "/etc/letsencrypt/live/dev.zippytal.com/fullchain.pem"
	keyFileDev := "/etc/letsencrypt/live/dev.zippytal.com/privkey.pem"
	go func() {
		tlsConfig := &tls.Config{}
		tlsConfig.Certificates = make([]tls.Certificate, 3)
		var err error
		tlsConfig.Certificates[0], err = tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			log.Fatalln(err)
		}
		tlsConfig.Certificates[1], err = tls.LoadX509KeyPair(certFileW, keyFileW)
		if err != nil {
			log.Fatalln(err)
		}
		tlsConfig.Certificates[2], err = tls.LoadX509KeyPair(certFileDev, keyFileDev)
		if err != nil {
			log.Fatalln(err)
		}
		http.Handle("app.zippytal.com/", h)
		http.Handle("https://app.zippytal.com/", h)
		http.Handle("dev.zippytal.com/", h2)
		http.Handle("https://dev.zippytal.com/", h2)
		http.HandleFunc("zippytal.com/", func(rw http.ResponseWriter, r *http.Request) {
			if _, err := os.Stat("./website/" + r.URL.Path); os.IsNotExist(err) {
				http.ServeFile(rw, r, "./website/index.html")
			} else {
				http.ServeFile(rw, r, "./website/"+r.URL.Path)
			}
		})
		s := &http.Server{
			TLSConfig: tlsConfig,
		}
		lis, err := tls.Listen("tcp", ":443", tlsConfig)
		if err != nil {
			log.Fatalln(err)
		}
		log.Fatalln(s.Serve(lis))
	}()
	creds,err := credentials.NewServerTLSFromFile(certFile,keyFile)
	if err != nil {
		log.Fatalln(err)
	}
	grpcServer := grpc.NewServer(grpc.Creds(creds),grpc.MaxConcurrentStreams(100000))
	manager.RegisterGrpcManagerServer(grpcServer, manager.NewGRPCManagerService(m))
	log.Fatalln(grpcServer.Serve(lis))
}
