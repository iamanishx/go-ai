module github.com/iamanishx/go-ai/examples/mcp

go 1.23.0

require (
	github.com/iamanishx/go-ai/agent v0.1.0
	github.com/iamanishx/go-ai/mcp v0.1.0
	github.com/iamanishx/go-ai/provider v0.1.0
	github.com/iamanishx/go-ai/provider/bedrock v0.1.0
)

require (
	github.com/aws/aws-sdk-go-v2 v1.32.4 // indirect
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.6.6 // indirect
	github.com/aws/aws-sdk-go-v2/config v1.25.0 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.17.0 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.15.0 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.3.23 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.6.23 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.7.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/bedrockruntime v1.20.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.11.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.11.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.19.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.22.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.27.0 // indirect
	github.com/aws/smithy-go v1.22.0 // indirect
	github.com/google/jsonschema-go v0.3.0 // indirect
	github.com/iamanishx/go-ai/stream v0.1.0 // indirect
	github.com/modelcontextprotocol/go-sdk v1.0.0 // indirect
	github.com/yosida95/uritemplate/v3 v3.0.2 // indirect
)

replace github.com/iamanishx/go-ai/agent => ../../agent

replace github.com/iamanishx/go-ai/mcp => ../../mcp

replace github.com/iamanishx/go-ai/provider => ../../provider

replace github.com/iamanishx/go-ai/provider/bedrock => ../../provider/bedrock

replace github.com/iamanishx/go-ai/stream => ../../stream
