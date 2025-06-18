package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"resty.dev/v3"
)

func returnFullResponse(resp *resty.Response) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Status: %s\n", resp.Status()))
	b.WriteString(fmt.Sprintf("Status Code: %d\n", resp.StatusCode()))
	b.WriteString(fmt.Sprintf("Protocol: %s\n", resp.Proto()))
	b.WriteString(fmt.Sprintf("Headers: %#v\n", resp.Header()))
	b.WriteString(fmt.Sprintf("Body: %s\n", resp.String()))
	return b.String()
}

// General setup
// const openApiFile = "example/ip_api.yaml"
// openApiFile = "example/petstore2_api.json"
// openApiFile = "example/petstore3_api.json"

// Setup from LLM
var providedQueryParams = map[string]string{}

func loadOpenApiYaml(openApiFile string) (*openapi3.T, error) {
	openapi3loader := openapi3.NewLoader()
	// TODO: Add support for openapi2
	doc, err := openapi3loader.LoadFromFile(openApiFile)
	if err != nil {
		log.Println("Unable to load OpenAPI spec", err)
		return nil, err
	}
	err = doc.Validate(openapi3loader.Context)
	if err != nil {
		log.Println("OpenAPI spec is invalid", err)
		return nil, err
	}

	return doc, nil
}

func generateToolDescription(doc *openapi3.T, method string, path string) string {
	var b strings.Builder
	// Add info about the REST
	b.WriteString(doc.Info.Title + " ")
	b.WriteString(doc.Info.Description + " ")
	// Add info about the specific method call
	b.WriteString(fmt.Sprintf("HTTP method %s ", method))
	b.WriteString(fmt.Sprintf("HTTP path %s ", path))
	description := b.String()

	// description = sanitize(description)
	return description
}

func sanitize(input string) string {
	re := regexp.MustCompile(`[^a-zA-Z0-9_-]`)
	return re.ReplaceAllString(input, "_")
}

func registerMCPTool(s *server.MCPServer, doc *openapi3.T, apiKey string) error {
	c := resty.New()
	defer c.Close()

	for path, pathItem := range doc.Paths.Map() {
		sanitizedPath := sanitize(path)

		validOperations := pathItem.Operations()

		for method, operation := range validOperations {
			// TODO: How to add more specificity on the action req + resp?
			// TODO: Add description on the side effect done + JSON result returned.
			toolOptions := []mcp.ToolOption{}

			toolOptions = append(toolOptions,
				mcp.WithDescription(generateToolDescription(doc, method, sanitizedPath)))

			for _, param := range operation.Parameters {
				switch param.Value.In {
				// Support Query Params
				case "query":
					if param.Value.Name == "api_key" {
						// Skip adding this as a required parameter.
						continue
					}
					switch {
					case param.Value.Schema.Value.Type.Includes("string"):
						propertyOptions := []mcp.PropertyOption{}
						if param.Value.Required {
							propertyOptions = append(propertyOptions, mcp.Required())
						}
						propertyOptions = append(propertyOptions, mcp.Description(param.Value.Name))
						toolOptions = append(toolOptions,
							mcp.WithString(param.Value.Name,
								propertyOptions...))
					default:
						log.Printf("TODO: param type of %s not supported yet.\n", param.Value.Schema.Value.Type)
					}
				default:
					log.Printf("TODO: params of %s not supported yet.\n", param.Value.In)
					// TODO: Support Header?
					// TODO: Support URL params
				}
			}

			newTool := mcp.NewTool(sanitizedPath,
				toolOptions...,
			)

			// Copy path to ensure it doesn't change before inline function is evaluated
			pathCopy := path
			// TODO: Define store for params, that can be shared between the MCP tool def + MCP tool function call
			s.AddTool(newTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				req := c.R()
				for _, param := range operation.Parameters {
					switch param.Value.In {
					// Support Query Params
					case "query":
						if param.Value.Name == "api_key" {
							// Short circuit adding apiKey to the request
							req.SetQueryParam(param.Value.Name, apiKey)
							continue
						}
						switch {
						case param.Value.Schema.Value.Type.Includes("string"):
							p, err := request.RequireString(param.Value.Name)
							if param.Value.Required && err != nil {
								return mcp.NewToolResultErrorFromErr("Unable to find required query param", err), nil
							}
							req.SetQueryParam(param.Value.Name, p)
						default:
							return mcp.NewToolResultError(fmt.Sprintf("TODO: param type of %s not supported yet.\n", param.Value.Schema.Value.Type)), nil
						}
					default:
						log.Printf("TODO: params of %s not supported yet.\n", param.Value.In)
						// TODO: Support Header?
						// TODO: Support URL params
					}
				}

				urlPath := doc.Servers[0].URL + pathCopy

				switch method {
				case http.MethodGet:
					resp, err := req.Get(urlPath)
					if err != nil {
						log.Println("Error when perfoming ", http.MethodGet, err)
						return mcp.NewToolResultErrorFromErr("Error when perfoming "+http.MethodGet, err), nil
					}
					defer resp.Body.Close()
					return mcp.NewToolResultText(returnFullResponse(resp)), nil
				default:
					return mcp.NewToolResultError(fmt.Sprintf("TODO: Unsupported method %s", method)), nil
				}
			})
		}
	}

	return nil
}

type ApiKeyDetails struct {
	ApiKey         string
	ApiKeyLocation string
}

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		log.Fatalln("Usage: myprogram <openApiFile>")
	}
	openApiFile := args[0]
	openapiDoc, err := loadOpenApiYaml(openApiFile)
	if err != nil {
		log.Fatalln("Encountered error when loading and parsing openapi doc", err)
	}

	apiKey := os.Getenv("API_KEY")
	if apiKey == "" {
		log.Fatalln("Missing API_KEY from environment variable")
	}

	// Declare a new MCP server
	s := server.NewMCPServer(
		"Retrieve IP",
		"1.0.0",
		server.WithToolCapabilities(false),
		server.WithRecovery(),
	)

	err = registerMCPTool(s, openapiDoc, apiKey)
	if err != nil {
		log.Fatalln("Encountered error when registering REST as MCP tools", err)
	}

	log.Println("Successfully configured MCP server!")

	if err := server.ServeStdio(s); err != nil {
		log.Fatalf("Server error: %v\n", err)
	}
}
