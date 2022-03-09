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
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
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

func (u *User) BeforeCreate(tx *gorm.DB) error {
	u.ID = ksuid.New()
	return nil
}

func main() {
	fmt.Println("hello, world!")
	host, dbname, user, password := os.Getenv("PGHOST"), os.Getenv("PGDATABASE"), os.Getenv("PGUSER"), os.Getenv("PGPASSWORD")
	driverOptions := fmt.Sprintf("host=%s dbname=%s user=%s password=%s port=5432 sslmode=disable", host, dbname, user, password)
	db, err := gorm.Open(postgres.Open(driverOptions), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		log.Fatalln(err)
	}

	if err := db.AutoMigrate(&User{}); err != nil {
		fmt.Println("failed to auto migrate")
		log.Fatalln(err)
	}

	store := &server{db: &dbstore{db: db}}
	r := mux.NewRouter()
	r.HandleFunc("/create", store.Create).Methods("PUT")
	r.HandleFunc("/read", store.Read).Methods("POST")
	r.HandleFunc("/update", store.Update).Methods("PUT")
	r.HandleFunc("/delete", store.Delete).Methods("POST")
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
	db store
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
	CreateUser(u *User) (*User, error)
	GetUser(id ksuid.KSUID) (*User, error)
	UpdateUser(u *User) (*User, error)
	DeleteUser(u *User) (*User, error)
}

type dbstore struct {
	db *gorm.DB
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
