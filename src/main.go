package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"os"
	"runtime"
	"ssl-checker/database"
	"ssl-checker/helper"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
)

type Domain struct {
	Host string
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("START")

	log.Println("runtime.GOMAXPROCS:", runtime.GOMAXPROCS(0))

	if err := godotenv.Load("../.env"); err != nil {
		log.Println(".env not found, using ENV: ", err)
	}

	log.Println("ENV:", os.Getenv("ENV"))

	chatID := helper.StrToInt64(os.Getenv("TELEGRAM_PUSH_CHAT_ID"))
	warningDays := helper.StrToInt(os.Getenv("SSL_WARNING_DAYS"))
	telegramToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	mysqlURL := os.Getenv("MYSQL_URL")
	mysqlDomainsQuery := os.Getenv("MYSQL_GET_HOSTS_QUERY")
	if os.Getenv("MYSQL_URL_FILE") != "" {
		mysqlURL_, err := os.ReadFile(os.Getenv("MYSQL_URL_FILE"))
		if err != nil {
			log.Fatal(err)
		}
		mysqlURL = strings.TrimSpace(string(mysqlURL_))
	}

	dbService, err := database.NewService(mysqlURL)
	if err != nil {
		log.Fatal(err)
	} else {
		log.Println("dbService OK")
	}

	bot, err := tgbotapi.NewBotAPI(telegramToken)
	if err != nil {
		log.Fatal("Telegram error:", err)
	}

	var domains []*Domain
	for {
		err = dbService.DB.Raw(mysqlDomainsQuery).Find(&domains).Error
		if err != nil {
			log.Println("Ошибка при получении доменов:", err)
			sendMessage(bot, chatID, fmt.Sprintf("❌ Ошибка при получении доменов: %v", err))
		} else {
			for _, domain := range domains {
				go checkDomain(domain.Host, bot, chatID, warningDays)
			}
		}
		time.Sleep(time.Hour)
	}
}

func checkDomain(domain string, bot *tgbotapi.BotAPI, chatID int64, warningDays int) {
	address := domain + ":443"
	conn, err := tls.Dial("tcp", address, &tls.Config{
		InsecureSkipVerify: true,
	})
	if err != nil {
		sendMessage(bot, chatID, fmt.Sprintf("❌ Ошибка подключения к %s: %v", domain, err))
		return
	}
	defer conn.Close()

	state := conn.ConnectionState()
	if len(state.PeerCertificates) == 0 {
		sendMessage(bot, chatID, fmt.Sprintf("❌ Нет сертификатов у %s", domain))
		return
	}
	cert := state.PeerCertificates[0]

	now := time.Now()
	daysLeft := int(cert.NotAfter.Sub(now).Hours() / 24)

	// Проверка доверия
	roots, _ := x509.SystemCertPool()
	if _, err := cert.Verify(x509.VerifyOptions{
		DNSName: domain,
		Roots:   roots,
	}); err != nil {
		sendMessage(bot, chatID, fmt.Sprintf("⚠️ %s: недоверенный сертификат: %v", domain, err))
		return
	}

	if daysLeft < warningDays {
		log.Printf("⚠️ %s: сертификат истекает через %d дней (%s)", domain, daysLeft, cert.NotAfter.Format("02 Jan 2006"))
		sendMessage(bot, chatID, fmt.Sprintf("⚠️ %s: сертификат истекает через %d дней (%s)", domain, daysLeft, cert.NotAfter.Format("02 Jan 2006")))
	} else {
		log.Printf("✅ %s: сертификат действителен, %d дней осталось\n", domain, daysLeft)
	}
}

func sendMessage(bot *tgbotapi.BotAPI, chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	bot.Send(msg)
}
