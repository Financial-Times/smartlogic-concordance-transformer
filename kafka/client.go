package kafka

import (
	"regexp"
	"strings"
	"time"

	"github.com/Shopify/sarama"
	log "github.com/Sirupsen/logrus"
	"github.com/wvanbergen/kafka/consumergroup"
	"github.com/wvanbergen/kazoo-go"
)

type Clienter interface {
	StartListening(messageHandler func(message FTMessage) error)
	Shutdown()
}

type Client struct {
	topics         []string
	consumerGroup  string
	zookeeperNodes []string
	consumer       *consumergroup.ConsumerGroup
}

type FTMessage struct {
	Headers map[string]string
	Body    string
}

func NewKafkaClient(brokerConnectionString string, consumerGroup string, topics []string) (Clienter, error) {

	var zookeeperNodes []string

	config := consumergroup.NewConfig()
	config.Offsets.Initial = sarama.OffsetNewest
	config.Offsets.ProcessingTimeout = 10 * time.Second

	zookeeperNodes, config.Zookeeper.Chroot = kazoo.ParseConnectionString(brokerConnectionString)

	consumer, err := consumergroup.JoinConsumerGroup(consumerGroup, topics, zookeeperNodes, config)
	if err != nil {
		log.WithError(err).Error("Error creating Kafka consumer")
		return &Client{}, err
	}

	return &Client{
		topics:         topics,
		consumerGroup:  consumerGroup,
		zookeeperNodes: zookeeperNodes,
		consumer:       consumer,
	}, nil
}

func (c *Client) StartListening(messageHandler func(message FTMessage) error) {
	go func() {
		for err := range c.consumer.Errors() {
			log.WithError(err).Error("Error proccessing message")
		}
	}()

	go func() {
		for message := range c.consumer.Messages() {
			ftMsg, err := rawToFTMessage(message.Value)
			if err != nil {
				log.WithError(err).Error("Error converting Kafka message body to FTMessage")
			}
			err = messageHandler(ftMsg)
			if err != nil {
				log.WithError(err).WithField("messageKey", message.Key).Error("Error processing message")
			}
			c.consumer.CommitUpto(message)
		}
	}()
}

func (c *Client) Shutdown() {
	if err := c.consumer.Close(); err != nil {
		log.WithError(err).Error("Error closing the consumer")
	}
}

func rawToFTMessage(msg []byte) (FTMessage, error) {
	var err error
	ftMsg := FTMessage{}
	raw := string(msg)

	doubleNewLineStartIndex := getHeaderSectionEndingIndex(string(raw[:]))
	if ftMsg.Headers, err = parseHeaders(string(raw[:doubleNewLineStartIndex])); err != nil {
		return ftMsg, err
	}
	ftMsg.Body = strings.TrimSpace(string(raw[doubleNewLineStartIndex:]))
	return ftMsg, nil
}

var re = regexp.MustCompile("[\\w-]*:[\\w\\-:/. ]*")
var kre = regexp.MustCompile("[\\w-]*:")
var vre = regexp.MustCompile(":[\\w-:/. ]*")

func getHeaderSectionEndingIndex(msg string) int {
	//FT msg format uses CRLF for line endings
	i := strings.Index(msg, "\r\n\r\n")
	if i != -1 {
		return i
	}
	//fallback to UNIX line endings
	i = strings.Index(msg, "\n\n")
	if i != -1 {
		return i
	}
	log.Printf("WARN  - message with no message body: [%s]", msg)
	return len(msg)
}

func parseHeaders(msg string) (map[string]string, error) {
	headerLines := re.FindAllString(msg, -1)

	headers := make(map[string]string)
	for _, line := range headerLines {
		key, value := parseHeader(line)
		headers[key] = value
	}
	return headers, nil
}
func parseHeader(header string) (string, string) {
	key := kre.FindString(header)
	value := vre.FindString(header)
	return key[:len(key)-1], strings.TrimSpace(value[1:])
}

//func saramaToFTMessage(msg *sarama.ConsumerMessage) (m queueConsumer.Message, err error) {
//
//	raw := string(msg.Value)
//
//	decoded, err := base64.StdEncoding.DecodeString(raw)
//	if err != nil {
//		log.Printf("ERROR - failure in decoding base64 value: %s", err.Error())
//		return
//	}
//	doubleNewLineStartIndex := getHeaderSectionEndingIndex(string(decoded[:]))
//	if m.Headers, err = parseHeaders(string(decoded[:doubleNewLineStartIndex])); err != nil {
//		return
//	}
//	m.Body = strings.TrimSpace(string(decoded[doubleNewLineStartIndex:]))
//	return
//}
