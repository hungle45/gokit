package check

import (
	"context"
	"fmt"

	"github.com/IBM/sarama"
)

type kafkaChecker struct {
	cli sarama.Client
}

func NewKafkaChecker(cli sarama.Client) Checker {
	return &kafkaChecker{
		cli: cli,
	}
}

func (k *kafkaChecker) Name() string {
	return "kafka"
}

func (k *kafkaChecker) Check(ctx context.Context) ServiceStatus {
	resultChan := make(chan ServiceStatus)
	go func() {
		broker, err := k.cli.RefreshController()
		if err != nil {
			resultChan <- ServiceStatus{
				Status: StatusDown,
				Details: map[string]interface{}{
					"error": fmt.Sprintf("failed to refresh controller: %v", err),
				},
			}
			return
		}

		connected, err := broker.Connected()
		if err != nil {
			resultChan <- ServiceStatus{
				Status: StatusDown,
				Details: map[string]interface{}{
					"error": fmt.Sprintf("failed to check broker connection: %v", err),
				},
			}
			return
		}
		if !connected {
			resultChan <- ServiceStatus{
				Status: StatusDown,
				Details: map[string]interface{}{
					"error": "broker not connected",
				},
			}
			return
		}

		brokerAddresses := make([]string, 0)
		for _, b := range k.cli.Brokers() {
			brokerAddresses = append(brokerAddresses, b.Addr())
		}

		resultChan <- ServiceStatus{
			Status: StatusUp,
			Details: map[string]interface{}{
				"broker_addresses": brokerAddresses,
				"client_version":   k.cli.Config().Version.String(),
			},
		}
	}()

	select {
	case result := <-resultChan:
		return result
	case <-ctx.Done():
		return ServiceStatus{
			Status: StatusDown,
			Details: map[string]interface{}{
				"error":       "Check aborted due to context cancellation or timeout",
				"context_err": ctx.Err().Error(),
			},
		}
	}
}
