package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
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

func main() {
	fmt.Println("hello, world!")
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

	confirm := make(chan amqp.Confirmation, 1)

	ch.NotifyPublish(confirm)

	q, err := ch.QueueDeclare(
		"hello", // name
		true,    // durable
		false,   // delete when unused
		false,   // exclusive
		false,   // no-wait
		nil,     // arguments
	)
	failOnError(err, "Failed to declare a queue")

	store := &server{
		db:      &dbstore{db: db},
		q:       &q,
		ch:      ch,
		confirm: confirm,
	}

	r := mux.NewRouter()
	r.HandleFunc("/create", store.Create).Methods("PUT")
	r.HandleFunc("/read", store.Read).Methods("POST")
	r.HandleFunc("/update", store.Update).Methods("PUT")
	r.HandleFunc("/delete", store.Delete).Methods("POST")
	r.HandleFunc("/start", store.Start).Methods("POST")
	r.HandleFunc("/status", store.Status).Methods("GET")

	s := newHttpServer()
	s.Handler = r

	// Signal handler
	int := make(chan os.Signal, 2)
	signal.Notify(int, os.Interrupt, syscall.SIGTERM)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-int
		cancel()
		if err := s.Shutdown(ctx); err != nil {
			fmt.Printf("shutdown err: %s\n", err)
		}
	}()

	if err := s.ListenAndServe(); err != nil {
		log.Fatalln(err)
	}
}

func newHttpServer() *http.Server {
	return &http.Server{
		Addr:              ":8080",
		ReadHeaderTimeout: 20 * time.Second,
		ReadTimeout:       1 * time.Minute,
		WriteTimeout:      2 * time.Minute,
	}
}

type server struct {
	db      *dbstore
	q       *amqp.Queue
	ch      *amqp.Channel
	confirm chan amqp.Confirmation
}

func (s *server) Start(w http.ResponseWriter, r *http.Request) {
	var j Job
	job, err := s.db.CreateJob(&j)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	body, err := json.Marshal(job)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = s.ch.Publish(
		"",       // exchange
		s.q.Name, // routing key
		false,    // mandatory
		false,    // immediate
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte(body),
		})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	select {
	case confirm := <-s.confirm:
		if !confirm.Ack {
			http.Error(w, "failed to add job to queue", http.StatusInternalServerError)
			return
		}
	case <-time.After(time.Second * 10):
		http.Error(w, "failed to add job to queue", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(job); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *server) Create(w http.ResponseWriter, r *http.Request) {
	var u User
	if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	user, err := s.db.CreateUser(&u)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(user); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *server) Read(w http.ResponseWriter, r *http.Request) {
	var u User
	if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	user, err := s.db.GetUser(u.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(user); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *server) Status(w http.ResponseWriter, r *http.Request) {
	var j Job
	if err := json.NewDecoder(r.Body).Decode(&j); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	job, err := s.db.GetJob(j.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(job); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *server) Update(w http.ResponseWriter, r *http.Request) {
	var u User
	if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	user, err := s.db.UpdateUser(&u)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(user); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *server) Delete(w http.ResponseWriter, r *http.Request) {
	var u User
	if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	user, err := s.db.DeleteUser(&u)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(user); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

type store interface {
	CreateJob(j *Job) (*Job, error)
	CreateUser(u *User) (*User, error)
	GetUser(id ksuid.KSUID) (*User, error)
	GetJob(id ksuid.KSUID) (*Job, error)
	UpdateUser(u *User) (*User, error)
	DeleteUser(u *User) (*User, error)
}

type dbstore struct {
	db *gorm.DB
}

func (s *dbstore) CreateJob(j *Job) (*Job, error) {
	result := s.db.Create(&j)
	if result.Error != nil {
		return nil, result.Error
	}
	return j, nil
}

func (s *dbstore) CreateUser(u *User) (*User, error) {
	result := s.db.Create(&u)
	if result.Error != nil {
		return nil, result.Error
	}
	return u, nil
}

func (s *dbstore) GetUser(id ksuid.KSUID) (*User, error) {
	var user User
	result := s.db.Where("id = ?", id).First(&user)
	if result.Error != nil {
		return nil, result.Error
	}
	return &user, nil
}

func (s *dbstore) GetJob(id ksuid.KSUID) (*Job, error) {
	var j Job
	result := s.db.Where("id = ?", id).First(&j)
	if result.Error != nil {
		return nil, result.Error
	}
	return &j, nil
}

func (s *dbstore) UpdateUser(u *User) (*User, error) {
	result := s.db.Model(&u).Updates(u)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected < 1 {
		return nil, fmt.Errorf("no rows affected: '%d' rows", result.RowsAffected)
	}
	return u, nil
}

func (s *dbstore) DeleteUser(u *User) (*User, error) {
	result := s.db.Delete(&u)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected < 1 {
		return nil, fmt.Errorf("no rows affected: '%d' rows", result.RowsAffected)
	}
	return u, nil
}
