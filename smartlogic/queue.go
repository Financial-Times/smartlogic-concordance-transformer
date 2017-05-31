package smartlogic

import (
	"github.com/Financial-Times/message-queue-gonsumer/consumer"
)

type QueueService struct {
	consumerConfig  consumer.QueueConfig
	httpClient 	httpClient
}

func NewQueueService(consumerConfig  consumer.QueueConfig, httpClient httpClient) QueueService {
	return QueueService{
		consumerConfig: consumerConfig,
		httpClient:    httpClient,
	}
}
