# MCP REST

A Model Context Protocol (MCP) server that converts OpenAPI specifications into MCP tools, enabling seamless integration of REST APIs with AI assistants like Claude Desktop.

## Overview

This project automatically generates MCP tools from OpenAPI 3.0 specifications, allowing AI assistants to interact with REST APIs through standardized tool interfaces. It supports dynamic discovery and execution of API endpoints based on their OpenAPI definitions.

## Features

- **OpenAPI 3.0 Support**: Load and validate OpenAPI specifications from JSON/YAML files
- **Automatic Tool Generation**: Convert API endpoints into MCP tools with proper parameter handling
- **Query Parameter Support**: Handle required and optional query parameters with type validation
- **Comprehensive Response Handling**: Return detailed HTTP response information including status, headers, and body
- **Claude Desktop Integration**: Pre-configured for seamless integration with Claude Desktop

## Installation

```bash
go mod download
go build -o mcp-rest main.go
```

## Usage

Run the MCP server with an OpenAPI specification file:

```bash
./mcp-rest <path-to-openapi-file>
```

### Example

```bash
./mcp-rest example/ip_api.yaml
```

## Claude Desktop Configuration

Add the following configuration to your Claude Desktop config file:

```json
{
  "mcpServers": {
    "rest-api": {
      "command": "/path/to/mcp-rest",
      "args": ["/path/to/your/openapi-spec.yaml"]
    }
  }
}
```

## Examples

The repository includes several example OpenAPI specifications:

- `example/ip_api.yaml` - IP geolocation API
- `example/coin_market_cap_api.yaml` - CoinMarketCap API
- `example/petstore2_api.json` - Pet Store API v2
- `example/petstore3_api.json` - Pet Store API v3

## Supported Features

### Current Support
- ✅ OpenAPI 3.0 specifications
- ✅ GET HTTP methods
- ✅ Query parameters (string type)
- ✅ Required/optional parameter validation
- ✅ Comprehensive response formatting

### Planned Features
- 🔄 Additional HTTP methods (POST, PUT, DELETE, PATCH)
- 🔄 Header parameters
- 🔄 Path parameters
- 🔄 Request body support
- 🔄 Additional parameter types (integer, boolean, array)
- 🔄 OpenAPI 2.0 support

## Dependencies

- [mcp-go](https://github.com/mark3labs/mcp-go) - MCP server implementation
- [kin-openapi](https://github.com/getkin/kin-openapi) - OpenAPI 3.0 parser and validator
- [resty](https://resty.dev) - HTTP client library

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests for new functionality
5. Submit a pull request

## License

This project is open source. Please check the license file for details.