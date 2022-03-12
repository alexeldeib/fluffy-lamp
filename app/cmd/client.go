package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/segmentio/ksuid"
	"github.com/streadway/amqp"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Model struct {
	ID        ksuid.KSUID    `gorm:"primarykey;type:char(27)" json:"id"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deletedAt"`
}

type User struct {
	Model
	FirstName string `json:"firstName,omitempty"`
	LastName  string `json:"lastName,omitempty"`
}

type Job struct {
	Model
	Status string `json:"status,omitempty"`
}

func (u *User) BeforeCreate(tx *gorm.DB) error {
	u.ID = ksuid.New()
	return nil
}

func (j *Job) BeforeCreate(tx *gorm.DB) error {
	j.ID = ksuid.New()
	j.Status = "New"
	return nil
}

func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
	}
}

func logError(err error, msg string) {
	if err != nil {
		log.Printf("%s: %s", msg, err)
	}
}

func main() {
	host, dbname, user, password := os.Getenv("PGHOST"), os.Getenv("PGDATABASE"), os.Getenv("PGUSER"), os.Getenv("PGPASSWORD")
	driverOptions := fmt.Sprintf("host=%s dbname=%s user=%s password=%s port=5432 sslmode=disable", host, dbname, user, password)
	db, err := gorm.Open(postgres.Open(driverOptions), &gorm.Config{
		// Logger: logger.Default.LogMode(logger.Info),
	})

	failOnError(err, "failed to connect database")

	failOnError(db.AutoMigrate(&User{}, &Job{}), "failed to connect database")

	pass := os.Getenv("COOKIE")
	conn, err := amqp.Dial(fmt.Sprintf("amqp://hellosvc:%s@rabbitmq-0.rabbitmq-headless.default.svc.cluster.local:5672/hellosvc", pass))
	failOnError(err, "Failed to connect to RabbitMQ")
	defer conn.Close()

	ch, err := conn.Channel()
	failOnError(err, "Failed to open a channel")
	defer ch.Close()

	err = ch.Confirm(false)
	failOnError(err, "Failed to set channel to confirm mode")

	err = ch.Qos(
		1,     // prefetch count
		0,     // prefetch size
		false, // global
	)

	failOnError(err, "Failed to set QoS")

	_, err = ch.QueueDeclare(
		"hello", // name
		true,    // durable
		false,   // delete when unused
		false,   // exclusive
		false,   // no-wait
		nil,     // arguments
	)
	failOnError(err, "Failed to declare a queue")

	msgs, err := ch.Consume(
		"hello", // queue
		"",      // consumer
		false,   // auto-ack
		false,   // exclusive
		false,   // no-local
		false,   // no-wait
		nil,     // args
	)
	failOnError(err, "Failed to register a consumer")

	///
	///
	///
	///
	// pass := os.Getenv("COOKIE")
	// conn, err := amqp.Dial(fmt.Sprintf("amqp://hellosvc:%s@rabbitmq-0.rabbitmq-headless.default.svc.cluster.local:5672/hellosvc", pass))
	// failOnError(err, "Failed to connect to RabbitMQ")
	// defer conn.Close()

	// ch, err := conn.Channel()
	// failOnError(err, "Failed to open a channel")
	// defer ch.Close()

	// q, err := ch.QueueDeclare(
	// 	"hello", // name
	// 	false,   // durable
	// 	false,   // delete when unused
	// 	false,   // exclusive
	// 	false,   // no-wait
	// 	nil,     // arguments
	// )
	// failOnError(err, "Failed to declare a queue")

	forever := make(chan bool)

	go func() {
		for d := range msgs {
			log.Printf("Received a message: %s", d.Body)

			var i Job
			if err := json.NewDecoder(bytes.NewReader(d.Body)).Decode(&i); err != nil {
				d.Nack(false, false)
				logError(err, "failed to decode message body")
				continue
			}

			var j Job
			result := db.Where("id = ?", i.ID).First(&j)
			if result.Error != nil {
				d.Nack(false, false)
				logError(result.Error, "failed to decode message body")
				continue
			}

			j.Status = "Done"

			result = db.Model(&j).Updates(j)
			if result.Error != nil {
				d.Nack(false, false)
				logError(result.Error, "failed to decode message body")
				continue
			}
			if result.RowsAffected < 1 {
				d.Nack(false, false)
				logError(fmt.Errorf("no rows affected: '%d' rows", result.RowsAffected), "failed to decode message body")
				continue
			}

			log.Printf("successfully processed message: %s", j.ID)

			d.Ack(false)
		}
	}()

	log.Printf(" [*] Waiting for messages. To exit press CTRL+C")
	<-forever
}
