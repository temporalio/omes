package lambda

import (
	"context"
	"crypto/tls"
	"fmt"
	"os"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/nexus-rpc/sdk-go/nexus"
	"github.com/temporalio/omes/clioptions"
	"github.com/temporalio/omes/workers/go/harness"
	"github.com/temporalio/omes/workers/go/workerlib/ebbandflow"
	"github.com/temporalio/omes/workers/go/workerlib/kitchensink"
	"github.com/temporalio/omes/workers/go/workerlib/schedulerstress"
	"go.temporal.io/sdk/activity"
	sdkclient "go.temporal.io/sdk/client"
	"go.temporal.io/sdk/contrib/aws/lambdaworker"
	"go.temporal.io/sdk/workflow"
)

const defaultTaskQueueName = "omes"

var App = harness.App{
	LambdaWorker: configureLambdaWorker,
}

func getEnvDefault(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

// Main is the Lambda entry point. All configuration is supplied via environment variables:
//
// Connection:
//
//	TEMPORAL_ADDRESS          - server address (via envconfig, e.g. my-server:7233)
//	TEMPORAL_NAMESPACE        - namespace (via envconfig)
//	TEMPORAL_TASK_QUEUE       - task queue (default: "omes")
//
// TLS / credentials (fetched from AWS Secrets Manager):
//
//	ENABLE_TLS                - set to any non-empty value to enable TLS
//	TLS_CERT                  - Secrets Manager secret ID for the client certificate (PEM)
//	TLS_KEY                   - Secrets Manager secret ID for the client private key (PEM)
//	API_KEY                   - Secrets Manager secret ID for the Temporal API key
//
// Worker deployment versioning:
//
//	TEMPORAL_OMES_DEPLOYMENT_NAME - deployment name (required)
//	TEMPORAL_OMES_BUILD_ID        - build ID
func Main() {
	if err := harness.Run(App); err != nil {
		clioptions.BackupLogger.Fatal(err)
	}
}

func configureLambdaWorker(opts *lambdaworker.Options) error {
	opts.TaskQueue = getEnvDefault("TEMPORAL_TASK_QUEUE", defaultTaskQueueName)

	enableTLS := os.Getenv("ENABLE_TLS")
	tlsKeyID := os.Getenv("TLS_KEY")
	tlsCertID := os.Getenv("TLS_CERT")
	apiKeyID := os.Getenv("API_KEY")

	ctx := context.Background()

	var tlsConfig *tls.Config
	var credentials sdkclient.Credentials

	if enableTLS != "" {
		tlsConfig = &tls.Config{InsecureSkipVerify: true}

		cfg, err := awsconfig.LoadDefaultConfig(ctx)
		if err != nil {
			return fmt.Errorf("failed to load AWS config: %w", err)
		}
		svc := secretsmanager.NewFromConfig(cfg)

		if tlsCertID != "" && tlsKeyID != "" {
			clientCert, err := svc.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{SecretId: &tlsCertID})
			if err != nil {
				return fmt.Errorf("failed to fetch TLS cert secret: %w", err)
			}

			clientKey, err := svc.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{SecretId: &tlsKeyID})
			if err != nil {
				return fmt.Errorf("failed to fetch TLS key secret: %w", err)
			}

			cert, err := tls.X509KeyPair([]byte(*clientCert.SecretString), []byte(*clientKey.SecretString))
			if err != nil {
				return fmt.Errorf("failed to parse TLS key pair: %w", err)
			}
			tlsConfig.Certificates = append(tlsConfig.Certificates, cert)
		}

		if apiKeyID != "" {
			apiKeyValue, err := svc.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{SecretId: &apiKeyID})
			if err != nil {
				return fmt.Errorf("failed to fetch API key secret: %w", err)
			}
			credentials = sdkclient.NewAPIKeyStaticCredentials(*apiKeyValue.SecretString)
		}
	}

	opts.ClientOptions.ConnectionOptions = sdkclient.ConnectionOptions{
		TLS: tlsConfig,
	}
	opts.ClientOptions.Credentials = credentials
	opts.WorkerOptions.DeploymentOptions.DefaultVersioningBehavior = workflow.VersioningBehaviorPinned

	ebbFlowActivities := ebbandflow.Activities{}

	service := nexus.NewService(kitchensink.KitchenSinkServiceName)
	for _, op := range []nexus.RegisterableOperation{kitchensink.EchoSyncOperation, kitchensink.EchoAsyncOperation} {
		if err := service.Register(op); err != nil {
			return fmt.Errorf("failed to register nexus operation: %w", err)
		}
	}

	opts.RegisterWorkflowWithOptions(kitchensink.KitchenSinkWorkflow, workflow.RegisterOptions{Name: "kitchenSink"})
	opts.RegisterWorkflow(kitchensink.NexusHandlerWorkflow)
	opts.RegisterWorkflowWithOptions(ebbandflow.EbbAndFlowTrackWorkflow, workflow.RegisterOptions{Name: "ebbAndFlowTrack"})
	opts.RegisterWorkflowWithOptions(schedulerstress.NoopScheduledWorkflow, workflow.RegisterOptions{Name: "NoopScheduledWorkflow"})
	opts.RegisterWorkflowWithOptions(schedulerstress.SleepScheduledWorkflow, workflow.RegisterOptions{Name: "SleepScheduledWorkflow"})
	opts.RegisterActivityWithOptions(kitchensink.Noop, activity.RegisterOptions{Name: "noop"})
	opts.RegisterActivityWithOptions(kitchensink.Delay, activity.RegisterOptions{Name: "delay"})
	opts.RegisterActivityWithOptions(kitchensink.Payload, activity.RegisterOptions{Name: "payload"})
	opts.RegisterActivityWithOptions(kitchensink.RetryableError, activity.RegisterOptions{Name: "retryable_error"})
	opts.RegisterActivityWithOptions(kitchensink.Timeout, activity.RegisterOptions{Name: "timeout"})
	opts.RegisterActivityWithOptions(kitchensink.Heartbeat, activity.RegisterOptions{Name: "heartbeat"})
	opts.RegisterActivity(&ebbFlowActivities)

	opts.RegisterNexusService(service)

	return nil
}
