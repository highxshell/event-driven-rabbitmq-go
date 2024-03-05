package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/highxshell/eventdrivenrabbit/internal"
	"github.com/rabbitmq/amqp091-go"
)

func main() {
	conn, err := internal.ConnectRabbitMQ("artem", "artem", "localhost:5671", "customers",
		"C:/Users/highashell/go/src/event-driven-rmq/tls-gen/basic/result/ca_certificate.pem",
		"C:/Users/highashell/go/src/event-driven-rmq/tls-gen/basic/result/client_DESKTOP-AAQUR1B_certificate.pem",
		"C:/Users/highashell/go/src/event-driven-rmq/tls-gen/basic/result/client_DESKTOP-AAQUR1B_key.pem",
	)
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	// Never use the same Connection for Consume and Publish
	consumeConn, err := internal.ConnectRabbitMQ("artem", "artem", "localhost:5671", "customers",
		"C:/Users/highashell/go/src/event-driven-rmq/tls-gen/basic/result/ca_certificate.pem",
		"C:/Users/highashell/go/src/event-driven-rmq/tls-gen/basic/result/client_DESKTOP-AAQUR1B_certificate.pem",
		"C:/Users/highashell/go/src/event-driven-rmq/tls-gen/basic/result/client_DESKTOP-AAQUR1B_key.pem")
	if err != nil {
		panic(err)
	}
	defer consumeConn.Close()
	client, err := internal.NewRabbitMQClient(conn)
	if err != nil {
		panic(err)
	}
	defer client.Close()
	consumeClient, err := internal.NewRabbitMQClient(consumeConn)
	if err != nil {
		panic(err)
	}
	defer consumeClient.Close()
	queue, err := consumeClient.CreateQueue("", true, true)
	if err != nil {
		panic(err)
	}
	if err := consumeClient.CreateBinding(queue.Name, queue.Name, "customer_callbacks"); err != nil {
		panic(err)
	}
	messageBus, err := consumeClient.Consume(queue.Name, "customer-api", true)
	if err != nil {
		panic(err)
	}
	go func() {
		for message := range messageBus {
			log.Printf("Message Callback %s\n", message.CorrelationId)
		}
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for i := 0; i < 10; i++ {
		if err := client.Send(ctx, "customer_events", "customers.created.kz", amqp091.Publishing{
			ContentType:   "text/plain",
			DeliveryMode:  amqp091.Persistent, // This tells rabbitMQ that this message should be Saved if no resources accepts it before a restart (durable)
			ReplyTo:       queue.Name,
			CorrelationId: fmt.Sprintf("customer_created_%d", i),
			Body:          []byte("A cool message between services"),
		}); err != nil {
			panic(err)
		}
	}
	select {}
}
