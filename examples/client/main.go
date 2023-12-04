package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
)

func main() {
	grpcSvcAddr := "localhost"
	proxyServer := "localhost:8080"
	resp, err := http.Get(fmt.Sprintf("http://%s/%s/helloworld/dynamic-proxy", proxyServer, grpcSvcAddr))
	if err != nil {
		log.Fatalf("failed to call serve: %v", err)
	}
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != 200 {
		log.Fatalf("Response failed with status code: %d and\nbody: %s\n", resp.StatusCode, body)
	}
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s", body)
}
