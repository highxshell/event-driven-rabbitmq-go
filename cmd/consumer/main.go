package main

import (
	"context"
	"log"
	"time"

	"github.com/highxshell/eventdrivenrabbit/internal"
	"github.com/rabbitmq/amqp091-go"
	"golang.org/x/sync/errgroup"
)

func main() {
	conn, err := internal.ConnectRabbitMQ("artem", "artem", "localhost:5671", "customers", "C:/Users/highashell/go/src/event-driven-rmq/tls-gen/basic/result/ca_certificate.pem",
		"C:/Users/highashell/go/src/event-driven-rmq/tls-gen/basic/result/client_DESKTOP-AAQUR1B_certificate.pem",
		"C:/Users/highashell/go/src/event-driven-rmq/tls-gen/basic/result/client_DESKTOP-AAQUR1B_key.pem")
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	publishConn, err := internal.ConnectRabbitMQ("artem", "artem", "localhost:5671", "customers", "C:/Users/highashell/go/src/event-driven-rmq/tls-gen/basic/result/ca_certificate.pem",
		"C:/Users/highashell/go/src/event-driven-rmq/tls-gen/basic/result/client_DESKTOP-AAQUR1B_certificate.pem",
		"C:/Users/highashell/go/src/event-driven-rmq/tls-gen/basic/result/client_DESKTOP-AAQUR1B_key.pem")
	if err != nil {
		panic(err)
	}
	defer publishConn.Close()
	client, err := internal.NewRabbitMQClient(conn)
	if err != nil {
		panic(err)
	}
	defer client.Close()
	publishClient, err := internal.NewRabbitMQClient(publishConn)
	if err != nil {
		panic(err)
	}
	defer publishClient.Close()
	// Create Unnamed Queue which will generate a random name, set AutoDelete to True
	queue, err := client.CreateQueue("", true, true)
	if err != nil {
		panic(err)
	}
	// Create binding between the customer_events exchange and the new Random Queue
	// Can skip Binding key since fanout will skip that rule
	if err := client.CreateBinding(queue.Name, "", "customer_events"); err != nil {
		panic(err)
	}
	messageBus, err := client.Consume(queue.Name, "email-service", false)
	if err != nil {
		panic(err)
	}
	// Set a timeout for 15 secs
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	// Create an Errgroup to manage concurrecy
	g, ctx := errgroup.WithContext(ctx)
	// Apply Qos to limit amount of messages to consume
	if err := client.ApplyQos(10, 0, true); err != nil {
		panic(err)
	}
	// Set amount of concurrent tasks
	g.SetLimit(10)
	go func() {
		for message := range messageBus {
			// Spawn a worker
			msg := message
			g.Go(func() error {
				log.Printf("New MEssage: %v", msg)
				time.Sleep(10 * time.Second)
				if err := msg.Ack(false); err != nil {
					log.Println("Ack message failed")
					return err
				}
				// Use the msg.ReplyTo to send the message to the proper Queue
				if err := publishClient.Send(ctx, "customer_callbacks", msg.ReplyTo, amqp091.Publishing{
					ContentType:   "text/plain",
					DeliveryMode:  amqp091.Persistent,
					Body:          []byte("RPC COMPLETE"),
					CorrelationId: msg.CorrelationId,
				}); err != nil {
					panic(err)
				}
				log.Printf("Acknowledged message: %s\n", msg.MessageId)
				return nil
			})
		}
	}()
	log.Println("Cusuming, use CTRL+C to exit")
	select {}
}
