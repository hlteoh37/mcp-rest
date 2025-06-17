package client_test

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"testing"

	"github.com/hlteoh37/mcp-rest-go"
)

const apiKey = "1dc93610ddc840e594e363cbed2d7976"

func TestClient_canCall(t *testing.T) {
	// custom HTTP client
	hc := http.Client{}

	c, err := client.NewClientWithResponses("https://ipgeolocation.abstractapi.com", client.WithHTTPClient(&hc))
	if err != nil {
		log.Fatal(err)
	}

	resp, err := c.GetV1WithResponse(context.TODO(), &client.GetV1Params{
		ApiKey: apiKey,
	})
	if err != nil {
		log.Fatal(err)
	}
	if resp.StatusCode() != http.StatusOK {
		log.Fatalf("Expected HTTP 200 but received %d", resp.StatusCode())
	}

	fmt.Printf("resp.JSON200: %v\n", string(resp.Body))

}
