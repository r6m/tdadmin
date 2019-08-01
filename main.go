package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"tdadmin/tg"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/rezam90/go-tdlib"
)

var (
	APIID                  = GetEnv("API_ID", "21724")
	APIHash                = GetEnv("API_HASH", "3e0cb5efcd52300aec5994fdfc5bdc16")
	SystemLanguageCode     = GetEnv("SYSTEM_LANGUAGE_CODE", "en")
	DeviceModel            = GetEnv("DEVICE_MODEL", "Server")
	SystemVersion          = GetEnv("SYSTEM_VERSION", "1.0.0")
	ApplicationVersion     = GetEnv("APPLICATION_VERSION", "1.0.0")
	UseMessageDatabase, _  = strconv.ParseBool(GetEnv("USE_MESSAGE_DATABASE", "true"))
	UseFileDatabase, _     = strconv.ParseBool(GetEnv("USE_FILE_DATABASE", "true"))
	UseChatInfoDatabase, _ = strconv.ParseBool(GetEnv("USE_CHAT_INFO_DATABASE", "true"))
	UseTestDataCenter, _   = strconv.ParseBool(GetEnv("USE_TEST_DATA_CENTER", "false"))
	DatabaseDirectory      = GetEnv("DATABASE_DIRECTORY", "./tdlib-db")
	FileDirectory          = GetEnv("FILE_DIRECTORY", "./tdlib-files")
	IgnoreFileNames, _     = strconv.ParseBool(GetEnv("IGNORE_FILE_NAMES", "false"))
)

func init() {
	log.SetFlags(log.Lshortfile)
}

func main() {
	tdlib.SetFilePath("./errors.txt")
	tdlib.SetLogVerbosityLevel(1)

	log.Println("init db")
	db := initDB()
	log.Println("init hash collector")
	tg.NewHashCollector(db)

	// Create new instance of client
	account := tg.NewAccount(tdlib.Config{
		APIID:               "21724",
		APIHash:             "3e0cb5efcd52300aec5994fdfc5bdc16",
		SystemLanguageCode:  "en",
		DeviceModel:         "Server",
		SystemVersion:       "1.0.0",
		ApplicationVersion:  "1.0.0",
		UseMessageDatabase:  true,
		UseFileDatabase:     true,
		UseChatInfoDatabase: true,
		UseTestDataCenter:   false,
		DatabaseDirectory:   "./tdlib-db",
		FileDirectory:       "./tdlib-files",
		IgnoreFileNames:     true,
	})

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	defer func() {
		<-quit
		log.Println("quit signal")
		account.StopJoiner()
		account.Stop()
		account.Client.DestroyInstance()
		os.Exit(1)
	}()

	for {
		currentState, _ := account.Client.Authorize()
		if currentState.GetAuthorizationStateEnum() == tdlib.AuthorizationStateWaitPhoneNumberType {
			fmt.Print("Enter phone: ")
			var number string
			fmt.Scanln(&number)
			_, err := account.Client.SendPhoneNumber(number)
			if err != nil {
				fmt.Printf("Error sending phone number: %v\n", err)
			}
		} else if currentState.GetAuthorizationStateEnum() == tdlib.AuthorizationStateWaitCodeType {
			fmt.Print("Enter code: ")
			var code string
			fmt.Scanln(&code)
			_, err := account.Client.SendAuthCode(code)
			if err != nil {
				fmt.Printf("Error sending auth code : %v\n", err)
			}
		} else if currentState.GetAuthorizationStateEnum() == tdlib.AuthorizationStateWaitPasswordType {
			fmt.Print("Enter Password: ")
			var password string
			fmt.Scanln(&password)
			_, err := account.Client.SendAuthPassword(password)
			if err != nil {
				fmt.Printf("Error sending auth password: %v\n", err)
			}
		} else if currentState.GetAuthorizationStateEnum() == tdlib.AuthorizationStateReadyType {
			fmt.Println("Authorization Ready! Let's rock")
			break
		}
	}

	fmt.Println("getting updates")

	go account.GetUpdates()

	time.AfterFunc(2*time.Second, func() {
		account.GetGroupLinks()
	})

	go account.StartJoiner(5 * time.Second)
}

func initDB() *sqlx.DB {
	if os.Getenv("DB_URL") == "" {
		log.Fatal("DB_URL empty")
	}
	db, err := sqlx.Open("mysql", os.Getenv("DB_URL"))
	if err != nil {
		log.Fatalln("db open", err)
	}
	err = db.Ping()
	if err != nil {
		log.Fatalln("db.Ping", err)
	}

	return db
}

func GetEnv(key, deafultValue string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return deafultValue
}
