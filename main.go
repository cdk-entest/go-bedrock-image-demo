package main

import (
	"context"
	"fmt"
	gobedrock "haimtran/gobedrock/bedrock"
	"log"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/rs/cors"
)

const REGION = "us-west-2"

// bedrock client
var BedrockClient *bedrockruntime.Client

// create an init function to initializing opensearch client
func init() {

	//
	fmt.Println("init and create an opensearch client")

	// load aws credentials from profile demo using config
	awsCfg1, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion(REGION),
	)

	if err != nil {
		log.Fatal(err)
	}

	// create bedorck runtime client
	BedrockClient = bedrockruntime.NewFromConfig(awsCfg1)

}

func main() {

	// create handler multiplexer
	mux := http.NewServeMux()

	// frontend camera  
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// w.Write([]byte("Hello"))
		http.ServeFile(w, r, "./static/index.html")
	})

	// bedrock frontend for image analyzer
	mux.HandleFunc("/image", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./static/image.html")
	})

	// bedrock backend to analyze image
	mux.HandleFunc("/claude-haiku-image", func(w http.ResponseWriter, r *http.Request) {
		gobedrock.HandleHaikuImageAnalyzer(w, r, BedrockClient)
	})

	// allow cors
	handler := cors.AllowAll().Handler(mux)

	// create a http server using http
	server := http.Server{
		Addr:           ":3000",
		Handler:        handler,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	server.ListenAndServe()

}
