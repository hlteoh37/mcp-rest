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
	parts := []string{}
	parts = append(parts, fmt.Sprintf("Status: %s", resp.Status()))
	parts = append(parts, fmt.Sprintf("Status Code: %d", resp.StatusCode()))
	parts = append(parts, fmt.Sprintf("Protocol: %s", resp.Proto()))
	parts = append(parts, fmt.Sprintf("Headers: %#v", resp.Header()))
	parts = append(parts, fmt.Sprintf("Body: %s", resp.String()))
	return strings.Join(parts, "\n")
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
	descriptionParts := []string{}
	// Add info about the REST
	descriptionParts = append(descriptionParts, doc.Info.Title)
	descriptionParts = append(descriptionParts, doc.Info.Description)
	// Add info about the specific method call
	descriptionParts = append(descriptionParts, fmt.Sprintf("HTTP method %s", method))
	descriptionParts = append(descriptionParts, fmt.Sprintf("HTTP path %s", path))
	description := strings.Join(descriptionParts, "_")
	description = sanitize(description)
	return description
}

func sanitize(input string) string {
	re := regexp.MustCompile(`[^a-zA-Z0-9_-]`)
	return re.ReplaceAllString(input, "_")
}

type ToolStore struct {
	ParsedTools map[string]*Tools
}

type Tools map[string]*Tool

type Tool struct {
	Path      string
	Method    string
	Operation *openapi3.Operation
}

func registerMCPTool(s *server.MCPServer, doc *openapi3.T) error {
	c := resty.New()
	defer c.Close()

	toolStore := ToolStore{
		ParsedTools: make(map[string]*Tools),
	}

	for path, pathItem := range doc.Paths.Map() {
		sanitizedPath := sanitize(path)

		validOperations := pathItem.Operations()

		// Populate the internal data structure
		tools := Tools{}
		toolStore.ParsedTools[sanitizedPath] = &tools
		for method, operation := range validOperations {
			tools[method] = &Tool{
				Path:      sanitizedPath,
				Method:    method,
				Operation: operation,
			}

			// TODO: How to add more specificity on the action req + resp?
			// TODO: Add description on the side effect done + JSON result returned.
			toolOptions := []mcp.ToolOption{}

			toolOptions = append(toolOptions,
				mcp.WithDescription(generateToolDescription(doc, method, sanitizedPath)))

			for _, param := range operation.Parameters {
				switch param.Value.In {
				// Support Query Params
				case "query":
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
					// if value, exists := providedQueryParams[param.Value.Name]; exists {
					// 	req.SetQueryParam(param.Value.Name, value)
					// } else {
					// 	if param.Value.Required {
					// 		errMsg := fmt.Sprintf("Required param %s but not provided.", param.Value.Name)
					// 		log.Println(errMsg)
					// 		return errors.New(errMsg)
					// 	}
					// }
				default:
					log.Printf("TODO: params of %s not supported yet.\n", param.Value.In)
					// TODO: Support Header?
					// TODO: Support URL params
				}
			}

			newTool := mcp.NewTool(sanitizedPath,
				toolOptions...,
			)

			// TODO: Define store for params, that can be shared between the MCP tool def + MCP tool function call
			s.AddTool(newTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				req := c.R()
				for _, param := range operation.Parameters {

					switch param.Value.In {
					// Support Query Params
					case "query":
						if value, exists := providedQueryParams[param.Value.Name]; exists {
							req.SetQueryParam(param.Value.Name, value)
						} else {
							if param.Value.Required {
								errMsg := fmt.Sprintf("Required param %s but not provided.", param.Value.Name)
								log.Println(errMsg)
								return mcp.NewToolResultError(errMsg), nil
							}
						}
					default:
						log.Printf("TODO: params of %s not supported yet.\n", param.Value.In)
						// TODO: Support Header?
						// TODO: Support URL params
					}
				}
				urlPath := doc.Servers[0].URL + path

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

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		log.Fatalln("Usage: myprogram <openApiFile>")
	}
	openApiFile := args[0]
	openapiDoc, err := loadOpenApiYaml(openApiFile)
	if err != nil {
		log.Println("Encountered error when loading and parsing openapi doc", err)
	}

	// Declare a new MCP server
	s := server.NewMCPServer(
		"Retrieve IP",
		"1.0.0",
		server.WithToolCapabilities(false),
		server.WithRecovery(),
	)

	err = registerMCPTool(s, openapiDoc)
	if err != nil {
		log.Println("Encountered error when registering REST as MCP tools", err)
	}

	log.Println("Successfully configured MCP server!")

	if err := server.ServeStdio(s); err != nil {
		log.Printf("Server error: %v\n", err)
	}

}
