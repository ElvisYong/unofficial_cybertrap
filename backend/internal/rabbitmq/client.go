package rabbitmq

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"time"

	"github.com/rabbitmq/amqp091-go"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ScanMessage defines the structure of the message received from RabbitMQ
// We just check the primitive.ObjectID at the API layer
type ScanMessage struct {
	MultiScanId   primitive.ObjectID   `json:"multi_scan_id"`
	ScanId        primitive.ObjectID   `json:"scan_id"`
	TemplateIds   []primitive.ObjectID `json:"template_ids"`
	DomainId      primitive.ObjectID   `json:"domain_id"`
	Domain        string               `json:"domain"`
	ScanAllNuclei bool                 `json:"scan_all_nuclei"`
}

type RabbitMQClient struct {
	conn    *amqp091.Connection
	channel *amqp091.Channel
	url     string
}

const (
	maxRetries    = 5
	retryInterval = 2 * time.Second
)

func NewRabbitMQClient(amqpURL string) (*RabbitMQClient, error) {
	// Parse the AMQP URL
	// Set up TLS configuration
	uri, err := amqp091.ParseURI(amqpURL)
	if err != nil {
		log.Logger.Error().Err(err).Msg("Failed to parse AMQP URL")
		return nil, err
	}
	serverName := uri.Host

	tlsConfig := &tls.Config{
		ServerName: serverName,
		ClientAuth: tls.RequireAndVerifyClientCert,
	}

	// Dial with TLS
	conn, err := amqp091.DialTLS(amqpURL, tlsConfig)
	if err != nil {
		log.Logger.Error().Err(err).Msg("Failed to connect to RabbitMQ")
		return nil, err
	}

	channel, err := conn.Channel()
	if err != nil {
		log.Logger.Error().Err(err).Msg("Failed to open a channel")
		conn.Close()
		return nil, err
	}

	return &RabbitMQClient{
		conn:    conn,
		channel: channel,
		url:     amqpURL,
	}, nil
}

// This will define the exchange and queue for the mq that we will use
// in the nuclei scanner and domains_api
func (r *RabbitMQClient) DeclareExchangeAndQueue() error {
	err := r.channel.ExchangeDeclare(
		"nuclei_scans", // name
		"direct",       // type
		true,           // durable
		false,          // auto-deleted
		false,          // internal
		false,          // no-wait
		nil,            // arguments
	)
	if err != nil {
		log.Logger.Error().Err(err).Msg("Failed to declare exchange")
		return err
	}

	_, err = r.channel.QueueDeclare(
		"nuclei_scan_queue", // name
		true,                // durable
		false,               // delete when unused
		false,               // exclusive
		false,               // no-wait
		nil,                 // arguments
	)
	if err != nil {
		log.Logger.Error().Err(err).Msg("Failed to declare queue")
		return err
	}

	err = r.channel.QueueBind(
		"nuclei_scan_queue", // queue name
		"",                  // routing key
		"nuclei_scans",      // exchange
		false,
		nil,
	)
	if err != nil {
		log.Logger.Error().Err(err).Msg("Failed to bind queue")
	}
	return err
}

func (r *RabbitMQClient) Publish(message interface{}) error {
	body, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("error marshaling message: %w", err)
	}

	return r.channel.Publish(
		"nuclei_scans", // exchange
		"",             // routing key
		false,          // mandatory
		false,          // immediate
		amqp091.Publishing{
			ContentType: "application/json",
			Body:        body,
		},
	)
	if err != nil {
		log.Logger.Error().Err(err).Msg("Failed to publish message")
	}
	return err
}

func (r *RabbitMQClient) Consume() (<-chan amqp091.Delivery, error) {
	msgs, err := r.channel.Consume(
		"nuclei_scan_queue", // queue
		"",                  // consumer
		false,               // auto-ack
		false,               // exclusive
		false,               // no-local
		false,               // no-wait
		nil,                 // args
	)
	if err != nil {
		log.Logger.Error().Err(err).Msg("Failed to register a consumer")
		return nil, err
	}
	return msgs, nil
}

func (r *RabbitMQClient) Get() (*amqp091.Delivery, bool, error) {
	if r.channel == nil || r.channel.IsClosed() {
		if err := r.reconnect(); err != nil {
			return nil, false, fmt.Errorf("failed to reconnect: %w", err)
		}
	}

	log.Info().Msg("Getting message from RabbitMQ")
	msg, ok, err := r.channel.Get(
		"nuclei_scan_queue",
		false, // auto-ack
	)
	// If there's no message, return immediately without error
	if !ok {
		log.Info().Msg("No message available")
		return nil, false, nil
	}

	log.Info().Msgf("Received message: %s", string(msg.Body))
	// Only return error if there actually was one
	if err != nil {
		return nil, false, err
	}

	return &msg, true, nil
}

func (r *RabbitMQClient) Close() {
	if err := r.channel.Close(); err != nil {
		log.Logger.Error().Err(err).Msg("Failed to close channel")
	}
	if err := r.conn.Close(); err != nil {
		log.Logger.Error().Err(err).Msg("Failed to close connection")
	}
}

func (r *RabbitMQClient) reconnect() error {
	for i := 0; i < maxRetries; i++ {
		// Parse the AMQP URL and set up TLS config (reuse from NewRabbitMQClient)
		uri, err := amqp091.ParseURI(r.url)
		if err != nil {
			continue
		}

		tlsConfig := &tls.Config{
			ServerName: uri.Host,
			ClientAuth: tls.RequireAndVerifyClientCert,
		}

		r.conn, err = amqp091.DialTLS(r.url, tlsConfig)
		if err != nil {
			log.Warn().Err(err).Msgf("Failed to reconnect to RabbitMQ (attempt %d/%d)", i+1, maxRetries)
			time.Sleep(retryInterval)
			continue
		}

		r.channel, err = r.conn.Channel()
		if err != nil {
			log.Warn().Err(err).Msgf("Failed to create channel (attempt %d/%d)", i+1, maxRetries)
			r.conn.Close()
			time.Sleep(retryInterval)
			continue
		}

		// Redeclare exchange and queue
		err = r.DeclareExchangeAndQueue()
		if err != nil {
			log.Warn().Err(err).Msgf("Failed to declare exchange and queue (attempt %d/%d)", i+1, maxRetries)
			r.Close()
			time.Sleep(retryInterval)
			continue
		}

		return nil
	}
	return fmt.Errorf("failed to reconnect after %d attempts", maxRetries)
}
