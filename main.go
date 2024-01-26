package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"

	"Parallels/simple-reverse-proxy/config"
)

func main() {
	cfg, err := config.NewConfig()
	if err != nil {
		fmt.Println("Error reading config file:", err)
		os.Exit(1)
	}

	for _, host := range cfg.Hosts {
		go startServer(host)
	}

	select {}
}

func NewReverseProxy(target string) *httputil.ReverseProxy {
	url, _ := url.Parse(target)

	return httputil.NewSingleHostReverseProxy(url)
}

func startServer(host config.Host) {
	if host.Tcp != nil {
		listener, err := net.Listen("tcp", fmt.Sprintf(":%s", host.Port))
		if err != nil {
			log.Fatal(err)
		}
		defer listener.Close()
		log.Printf("Listening on %s\n", host.Port)
		for {
			conn, err := listener.Accept()
			if err != nil {
				log.Fatal(err)
			}
			go handle(conn, host.Host, host.Tcp.Target)
		}
	} else {
		target := host.Host
		if host.Port != "" {
			target = fmt.Sprintf("%s:%s", host.Host, host.Port)
		}
		mux := http.NewServeMux()
		proxy := NewReverseProxy(target)
		proxy.ModifyResponse = func(resp *http.Response) error {
			resp.Header.Set("Access-Control-Allow-Origin", "*")
			resp.Header.Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			resp.Header.Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			return nil
		}

		proxy.Director = func(req *http.Request) {
			target := host.Host
			if host.Port != "" {
				target = fmt.Sprintf("%s:%s", host.Host, host.Port)
			}
			if strings.EqualFold(target, req.Host) {
				for _, route := range host.Routes {
					if route.Pattern.MatchString(req.URL.Path) {
						forwardTo := route.Target
						if route.TargetPort != "" {
							forwardTo = fmt.Sprintf("%s:%s", route.Target, route.TargetPort)
						}
						if strings.HasPrefix(forwardTo, "http") {
							forwardTo = strings.TrimPrefix(forwardTo, "http://")
							forwardTo = strings.TrimPrefix(forwardTo, "https://")
						}
						log.Printf("Forwarding http traffic from host %s%s to proxy on %s", target, req.URL.Path, forwardTo)
						req.Host = forwardTo
						req.URL.Scheme = "http"
						req.URL.Host = forwardTo
						req.Header.Add("X-Forwarded-By", "simple-reverse-proxy")
						req.Header.Add("X-Forwarded-Host", forwardTo)
						req.Header.Add("X-Forwarded-Proto", req.URL.Scheme)

						// req.URL.Path = route.Pattern.ReplaceAllString(req.URL.Path, "")
						if req.URL.Path == "" {
							req.URL.Path = "/"
						}
						break
					}
				}
			}
		}
		if host.Port == "" {
			host.Port = "80"
		}

		mux.Handle("/", proxy)
		log.Printf("Listening to %s on port %s...\n", host.Host, host.Port)
		log.Fatal(http.ListenAndServe(":"+host.Port, mux))
	}
}

func handle(src net.Conn, host string, target string) {
	log.Printf("Forwarding tcp traffic from host %s to proxy on %s", host, target)
	dst, err := net.Dial("tcp", target)
	if err != nil {
		log.Fatalf("Unable to connect to target: %s", err)
	}
	defer dst.Close()

	go func() {
		// forward traffic from source to destination
		if _, err := io.Copy(dst, src); err != nil {
			log.Println(err.Error())
		}
	}()

	// forward traffic from destination to source
	if _, err := io.Copy(src, dst); err != nil {
		log.Println(err.Error())
	}
}
