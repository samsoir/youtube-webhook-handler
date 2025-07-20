module github.com/samsoir/youtube-webhook

go 1.23.0

toolchain go1.24.5

replace github.com/samsoir/youtube-webhook/function => ./function

require (
	github.com/GoogleCloudPlatform/functions-framework-go v1.9.2
	github.com/samsoir/youtube-webhook/function v0.0.0-00010101000000-000000000000
)

require (
	cloud.google.com/go/functions v1.19.3 // indirect
	github.com/cloudevents/sdk-go/v2 v2.15.2 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/json-iterator/go v1.1.10 // indirect
	github.com/modern-go/concurrent v0.0.0-20180228061459-e0a39a4cb421 // indirect
	github.com/modern-go/reflect2 v0.0.0-20180701023420-4b7aa43c6742 // indirect
	go.uber.org/atomic v1.4.0 // indirect
	go.uber.org/multierr v1.1.0 // indirect
	go.uber.org/zap v1.10.0 // indirect
)
