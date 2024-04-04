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
