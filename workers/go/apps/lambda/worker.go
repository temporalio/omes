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
//	TLS_COMBINED              - Secrets Manager secret ID for tls-combined.pem
//	TLS_CERT, TLS_KEY         - the same credential as two secret IDs; the older
//	                            form, still set by deployed workers
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
	tlsCombinedID := os.Getenv("TLS_COMBINED")
	tlsCertID := os.Getenv("TLS_CERT")
	tlsKeyID := os.Getenv("TLS_KEY")
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

		cert, err := loadClientCert(ctx, svc, tlsCombinedID, tlsCertID, tlsKeyID)
		if err != nil {
			return err
		}
		if cert != nil {
			tlsConfig.Certificates = append(tlsConfig.Certificates, *cert)
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
	if err := service.Register(kitchensink.KitchenSinkNexusOperation); err != nil {
		return fmt.Errorf("failed to register nexus operation: %w", err)
	}

	opts.RegisterWorkflowWithOptions(kitchensink.KitchenSinkWorkflow, workflow.RegisterOptions{Name: "kitchenSink"})
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

// loadClientCert reads the mTLS keypair from Secrets Manager, either as one
// combined PEM or as the older cert/key pair. Returns nil when none is set.
func loadClientCert(
	ctx context.Context, svc *secretsmanager.Client, combinedID, certID, keyID string,
) (*tls.Certificate, error) {
	switch {
	case combinedID != "" && (certID != "" || keyID != ""):
		return nil, fmt.Errorf("set TLS_COMBINED or TLS_CERT/TLS_KEY, not both")
	case combinedID != "":
		combined, err := fetchSecretString(ctx, svc, combinedID)
		if err != nil {
			return nil, err
		}
		cert, err := clioptions.X509KeyPairFromCombinedPEM([]byte(combined))
		if err != nil {
			return nil, err
		}
		return &cert, nil
	case certID != "" && keyID != "":
		clientCert, err := fetchSecretString(ctx, svc, certID)
		if err != nil {
			return nil, err
		}
		clientKey, err := fetchSecretString(ctx, svc, keyID)
		if err != nil {
			return nil, err
		}
		cert, err := tls.X509KeyPair([]byte(clientCert), []byte(clientKey))
		if err != nil {
			return nil, fmt.Errorf("failed to parse TLS key pair: %w", err)
		}
		return &cert, nil
	case certID != "" || keyID != "":
		return nil, fmt.Errorf("TLS_CERT and TLS_KEY must be set together")
	default:
		return nil, nil
	}
}

func fetchSecretString(ctx context.Context, svc *secretsmanager.Client, id string) (string, error) {
	out, err := svc.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{SecretId: &id})
	if err != nil {
		return "", fmt.Errorf("failed to fetch secret %s: %w", id, err)
	}
	if out.SecretString == nil {
		return "", fmt.Errorf("secret %s has no string value; it must be pushed as a string, not binary", id)
	}
	return *out.SecretString, nil
}
