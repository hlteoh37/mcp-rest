package main

import (
	"fmt"
	"net/http"

	"github.com/getkin/kin-openapi/openapi3"
	"resty.dev/v3"
)

func printFullResponse(resp *resty.Response) {
	fmt.Printf(`
Status: %s
Status Code: %d
Protocol: %s
Headers: %#v
Body: %s
`, resp.Status(), resp.StatusCode(), resp.Proto(), resp.Header(), resp.String())
	fmt.Printf("Status: %s\n", resp.Status())
	fmt.Println("Status Code:")
}

func loadOpenApiYaml() {
	// General setup
	openApiFile := "./example/ip_api.yaml"
	// openApiFile = "./example/petstore2_api.json"
	// openApiFile = "./example/petstore3_api.json"

	// Setup from LLM
	var providedQueryParams = map[string]string{}


	openapi3loader := openapi3.NewLoader()
	// TODO: Add support for openapi2
	doc, err := openapi3loader.LoadFromFile(openApiFile)
	if err != nil {
		fmt.Println("Unable to load OpenAPI spec", err)
		return
	}
	err = doc.Validate(openapi3loader.Context)
	if err != nil {
		fmt.Println("OpenAPI spec is invalid", err)
		return
	}

	c := resty.New()
	defer c.Close()

	fmt.Printf("%#v\n", doc.Paths)
	fmt.Println("")
	for path, pathItem := range doc.Paths.Map() {
		// fmt.Printf("%#v\t: %#v\n", path, pathItem)

		validOperations := pathItem.Operations()
		// fmt.Printf("%#v\n", validOperations)

		for method, operation := range validOperations {
			req := c.R()
			// req = http.NewRequest(method, operation.Servers.BasePath(), nil)

			for _, param := range operation.Parameters {

				switch param.Value.In {
				// Support Query Params
				case "query":
					if value, exists := providedQueryParams[param.Value.Name]; exists {
						req.SetQueryParam(param.Value.Name, value)
					} else {
						if param.Value.Required {
							fmt.Printf("Required param %s but not provided.\n", param.Value.Name)
							return
						}
					}
				default:
					fmt.Printf("TODO: params of %s not supported yet.\n", param.Value.In)
					// Support Header?
					// Support URL params
				}
			}

			urlPath := doc.Servers[0].URL + path

			switch method {
			case http.MethodGet:
				fmt.Printf("Performing %s on %s\n", method, urlPath)
				resp, err := req.Get(urlPath)
				if err != nil {
					fmt.Println("Error when perfoming ", http.MethodGet, err)
					return
				}
				defer resp.Body.Close()
				// printFullResponse(resp)
			default:
				fmt.Printf("TODO: Unsupported method %s", method)
			}
			// operation.Parameters = append(operation.Parameters, )

		}

	}

	// router, err := gorillamux.NewRouter(doc)
	// if err != nil {
	// 	fmt.Println("Unable to initiate gorillamux router", err)
	// 	return
	// }

	// httpReq, _ := http.NewRequest(http.MethodGet, "https://ipgeolocation.abstractapi.com/v1/", nil)
	// route, pathParams, err := router.FindRoute(httpReq)
	// if err != nil {
	// 	fmt.Println("Unable to find requested route", err)
	// 	return
	// }

	// fmt.Printf("%#v\n%#v\n", route.Spec, pathParams)

	// resp, err := http.DefaultClient.Do(httpReq)
	// if err != nil {
	// 	fmt.Println("Failed to call http request", err)
	// }
	// defer resp.Body.Close()

	// body, _ := io.ReadAll(resp.Body)
	// fmt.Println(string(body))

}

func main() {
	loadOpenApiYaml()

	fmt.Println("Successfully run main function!")
}
