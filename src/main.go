package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/laytan/go-fff-notifications-bot/database"

	"gorm.io/gorm"
)

// User model
type User struct {
	gorm.Model
	Name     string
	Username string
	chatID   uint
	Notis    []Noti
}

// Noti model
type Noti struct {
	gorm.Model
	UserID uint
	User   User
	start  uint
	end    uint
}

func main() {
	fmt.Println("Hello, World")

	db := database.New("database.db", &User, &Noti)

	var wg sync.WaitGroup
	wg.Add(10)

	for i := 0; i < 10; i++ {
		go func() {
			defer wg.Done()
			defer db.Mutex.Unlock()
			db.Mutex.Lock()
			user := User{
				Name:     "Laytan Laats",
				Username: "laytanl",
				chatID:   1,
				Notis:    make([]Noti, 0),
			}
			db.Conn.Create(&user)

			fmt.Println("Created user id: ", user.ID)
		}()
	}

	wg.Wait()
	var count int64
	db.Mutex.Lock()
	db.Conn.Model(&User{}).Count(&count)
	db.Mutex.Unlock()
	fmt.Println("counted ", count)

	ticker := time.NewTicker(10 * time.Second)
	stop := make(chan bool)

	go func() {
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				fmt.Println("tick")
			}
		}
	}()

	time.Sleep(70 * time.Second)
	ticker.Stop()
	stop <- true
}
