---
title: prompt image with claude 3 on amazon bedrock demo
author: haimtran
date: 05/04/2024
---

## Introduction

[![screencast thumbnail](./assets/video.png)](https://d2cvlmmg8c0xrp.cloudfront.net/demo/go-bedrock-demo.mp4)

This repo shows a basic example how to prompt image with Claude 3 in Amazon Bedrock using Go.

- a frontend page to capture image from camera
- a backend handler to invoke Claude 3 on Bedrock
- stream response to frontend

Project structure

```go
|--static
   |--index.html
   |--image.html
|--bedrock
   |--bedrock.go
|--main.go
|--go.mod
|--build.py
|--Dockerfile
```

The index.html is simple frontend to connect to laptop camera and capture images. bedrock.go implements a function to invoke Claude 3 on Bedrock, and main.go implements a http webserver. In addition, Dockerfile and build.py to dockerize this app to an image for deployment, for example on Amazon ECS.

## Run Application

Install Go

```bash
cd /home/ec2-user/
wget https://go.dev/dl/go1.21.5.linux-amd64.tar.gz
tar -xvf go1.21.5.linux-amd64.tar.gz
echo 'export PATH=/home/ec2-user/go/bin:$PATH' >> ~/.bashrc
```

Let clone the git repo and run

```
git clone https://github.com/cdk-entest/go-bedrock-image-demo.git
cd go-bedrock-image-demo
go run main.go
```

## Deploy

When you run the build.py, it will do three things

- build a Docker image
- create a ECR repository
- push the image to the ECR repository

```bash
python build.py
```

> [!IMPORTANT]  
> CORS has been allowed in the http webserver by using a cors library. So any frontend client can call the API for testing purpose, for example

```
API_ENDPOINT=https://alb-loadbalancer-example/claude-haiku-image
```

## Frontend

Use javascript to call a POST request to /claude-haiku-image, then process stream response from backend. More details on [Anthropic Claude 3 Prompt Messages API](https://docs.anthropic.com/claude/reference/messages_post)

```js
let messages = []
    messages.push({
      role: "user",
      content: [
        {
          type: "image",
          source: {
            type: "base64",
            media_type: "image/jpeg",
            data: imageBase64,
          },
        },
        { type: "text", text: "Your are an expert in image analyzing, espcially in human looking and fashion. Please describe this image in as details as possible in a very fun and positive way to make people happy" },
      ],
    });

    // call bedrock to describe the image
    desc.innerText = ""

    const response = await fetch(
      "claude-haiku-image",
      {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({ messages: messages })
      }
    );

    console.log(response)

    const reader = response.body.getReader();
    const decoder = new TextDecoder();

    while (true) {
      const { done, value } = await reader.read();
      if (done) {
        break;
      }
      try {
        const json = decoder.decode(value);
        desc.innerText += json;
        console.log(json);
      } catch (error) {
        console.log(error);
      }
    }
  })
```

## Backend

First we need to define some data struct for request and response

```go
type Message struct {
		Role    string        `json:"role"`
		Content []interface{} `json:"content"`
	}

	type Request struct {
		Messages []Message `json:"messages"`
	}

	type RequestBodyClaude3 struct {
		MaxTokensToSample int       `json:"max_tokens"`
		Temperature       float64   `json:"temperature,omitempty"`
		AnthropicVersion  string    `json:"anthropic_version"`
		Messages          []Message `json:"messages"`
	}

	// claude3 response data type
	type Delta struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}

	type ResponseClaude3 struct {
		Type  string `json:"type"`
		Index int    `json:"index"`
		Delta Delta  `json:"delta"`
	}
```

Then invoke Claude 3 on Bedrock and stream response to client

```go
  // parse request from client

	// payload for bedrock claude3 haikue
	messages := request.Messages

	payload := RequestBodyClaude3{
		MaxTokensToSample: 2048,
		AnthropicVersion:  "bedrock-2023-05-31",
		Temperature:       0.9,
		Messages:          messages,
	}

	// convert payload struct to bytes
	payloadBytes, error := json.Marshal(payload)

	if error != nil {
		fmt.Println(error)
		fmt.Fprintf(w, "ERROR")
		// return "", error
	}

	// fmt.Println("invoke bedrock ...")

	// invoke bedrock claude3 haiku
	output, error := BedrockClient.InvokeModelWithResponseStream(
		context.Background(),
		&bedrockruntime.InvokeModelWithResponseStreamInput{
			Body:        payloadBytes,
			ModelId:     aws.String("anthropic.claude-3-haiku-20240307-v1:0"),
			ContentType: aws.String("application/json"),
			Accept:      aws.String("application/json"),
		},
	)

	if error != nil {
		fmt.Println(error)
		fmt.Fprintf(w, "ERROR")
		// return "", error
	}

	// stream result to client
	for event := range output.GetStream().Events() {

		// fmt.Println(event)

		switch v := event.(type) {
		case *types.ResponseStreamMemberChunk:

			// fmt.Println("payload", string(v.Value.Bytes))

			var resp ResponseClaude3
			err := json.NewDecoder(bytes.NewReader(v.Value.Bytes)).Decode(&resp)
			if err != nil {
				fmt.Fprintf(w, "ERROR")
				// return "", err
			}

			// fmt.Println(resp.Delta.Text)

			fmt.Fprintf(w, resp.Delta.Text)
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			} else {
				fmt.Println("Damn, no flush")
			}

		case *types.UnknownUnionMember:
			fmt.Println("unknown tag:", v.Tag)

		default:
			fmt.Println("union is nil or unknown type")
		}
	}
```

Detail of the HandleHaikuImageAnalyzer

<details>
<summary>HandleHaikuImageAnalyzer.go</summary>

```go
package bedrock

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"
)

func HandleHaikuImageAnalyzer(w http.ResponseWriter, r *http.Request, BedrockClient *bedrockruntime.Client) {

	// data type request
	type Message struct {
		Role    string        `json:"role"`
		Content []interface{} `json:"content"`
	}

	type Request struct {
		Messages []Message `json:"messages"`
	}

	type RequestBodyClaude3 struct {
		MaxTokensToSample int       `json:"max_tokens"`
		Temperature       float64   `json:"temperature,omitempty"`
		AnthropicVersion  string    `json:"anthropic_version"`
		Messages          []Message `json:"messages"`
	}

	// claude3 response data type
	type Delta struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}

	type ResponseClaude3 struct {
		Type  string `json:"type"`
		Index int    `json:"index"`
		Delta Delta  `json:"delta"`
	}

	// parse request
	var request Request
	error := json.NewDecoder(r.Body).Decode(&request)

	if error != nil {
		panic(error)
	}

	// payload for bedrock claude3 haikue
	messages := request.Messages

	payload := RequestBodyClaude3{
		MaxTokensToSample: 2048,
		AnthropicVersion:  "bedrock-2023-05-31",
		Temperature:       0.9,
		Messages:          messages,
	}

	// convert payload struct to bytes
	payloadBytes, error := json.Marshal(payload)

	if error != nil {
		fmt.Println(error)
		fmt.Fprintf(w, "ERROR")
		// return "", error
	}

	// fmt.Println("invoke bedrock ...")

	// invoke bedrock claude3 haiku
	output, error := BedrockClient.InvokeModelWithResponseStream(
		context.Background(),
		&bedrockruntime.InvokeModelWithResponseStreamInput{
			Body:        payloadBytes,
			ModelId:     aws.String("anthropic.claude-3-haiku-20240307-v1:0"),
			ContentType: aws.String("application/json"),
			Accept:      aws.String("application/json"),
		},
	)

	if error != nil {
		fmt.Println(error)
		fmt.Fprintf(w, "ERROR")
		// return "", error
	}

	// stream result to client
	for event := range output.GetStream().Events() {

		// fmt.Println(event)

		switch v := event.(type) {
		case *types.ResponseStreamMemberChunk:

			// fmt.Println("payload", string(v.Value.Bytes))

			var resp ResponseClaude3
			err := json.NewDecoder(bytes.NewReader(v.Value.Bytes)).Decode(&resp)
			if err != nil {
				fmt.Fprintf(w, "ERROR")
				// return "", err
			}

			// fmt.Println(resp.Delta.Text)

			fmt.Fprintf(w, resp.Delta.Text)
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			} else {
				fmt.Println("Damn, no flush")
			}

		case *types.UnknownUnionMember:
			fmt.Println("unknown tag:", v.Tag)

		default:
			fmt.Println("union is nil or unknown type")
		}
	}
}

```

</details>

## Reference

- [anthropic claude 3 messages api](https://docs.anthropic.com/claude/reference/messages_post)

- [anthropic prompt cookbook](https://github.com/anthropics/anthropic-cookbook/blob/main/multimodal/getting_started_with_vision.ipynb)
