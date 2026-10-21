package telegram

import (
	"fmt"
	"sync"
	"time"

	"github.com/MmadF14/vwireguard/store"
	"github.com/NicoNex/echotron/v3"
	"github.com/labstack/gommon/log"
)

type SendRequestedConfigsToTelegram func(db store.IStore, userid int64) []string

type TgBotInitDependencies struct {
	DB                             store.IStore
	SendRequestedConfigsToTelegram SendRequestedConfigsToTelegram
}

var (
	Token            string
	AllowConfRequest bool
	FloodWait        int
	LogLevel         log.Lvl

	Bot      *echotron.API
	BotMutex sync.RWMutex

	floodMutex       sync.RWMutex
	floodWait        = make(map[int64]int64)
	floodMessageSent = make(map[int64]struct{})
)

func Start(initDeps TgBotInitDependencies) (err error) {
	ticker := time.NewTicker(time.Minute)
	defer func() {
		ticker.Stop()
		if err != nil {
			BotMutex.Lock()
			Bot = nil
			BotMutex.Unlock()
		}
		if r := recover(); r != nil {
			err = fmt.Errorf("[PANIC] recovered from panic: %v", r)
		}
	}()

	token := Token
	if token == "" || len(token) < 30 {
		return fmt.Errorf("invalid telegram bot token")
	}

	bot := echotron.NewAPI(token)

	res, err := bot.GetMe()
	if !res.Ok || err != nil {
		log.Warnf("[Telegram] Unable to connect to bot.\n%v\n%v", res.Description, err)
		return fmt.Errorf("failed to connect to telegram bot: %v", err)
	}

	BotMutex.Lock()
	Bot = &bot
	BotMutex.Unlock()

	if LogLevel <= log.INFO {
		log.Infof("[Telegram] Authorized as %s", res.Result.Username)
	}

	go func() {
		for range ticker.C {
			updateFloodWait()
		}
	}()

	if !AllowConfRequest {
		return nil
	}

	updatesChan := echotron.PollingUpdatesOptions(token, false, echotron.UpdateOptions{AllowedUpdates: []echotron.UpdateType{echotron.MessageUpdate}})
	for update := range updatesChan {
		if update.Message != nil {
			userid := update.Message.Chat.ID

			floodMutex.RLock()
			_, wait := floodWait[userid]
			_, notified := floodMessageSent[userid]
			floodMutex.RUnlock()

			if wait {
				if notified {
					continue
				}
				floodMutex.Lock()
				floodMessageSent[userid] = struct{}{}
				floodMutex.Unlock()

				_, err := bot.SendMessage(
					fmt.Sprintf("You can only request your configs once per %d minutes", FloodWait),
					userid,
					&echotron.MessageOptions{
						ReplyToMessageID: update.Message.ID,
					})
				if err != nil {
					log.Errorf("Failed to send telegram message. Error %v", err)
				}
				continue
			}

			floodMutex.Lock()
			floodWait[userid] = time.Now().Unix()
			floodMutex.Unlock()

			failed := initDeps.SendRequestedConfigsToTelegram(initDeps.DB, userid)
			if len(failed) > 0 {
				messageText := "Failed to send configs:\n"
				for _, f := range failed {
					messageText += f + "\n"
				}
				_, err := bot.SendMessage(
					messageText,
					userid,
					&echotron.MessageOptions{
						ReplyToMessageID: update.Message.ID,
					})
				if err != nil {
					log.Errorf("Failed to send telegram message. Error %v", err)
				}
			}
		}
	}
	return err
}

func SendConfig(userid int64, clientName string, confData, qrData []byte, ignoreFloodWait bool) error {
	BotMutex.RLock()
	defer BotMutex.RUnlock()

	if Bot == nil {
		return fmt.Errorf("telegram bot is not configured or not available")
	}

	if !ignoreFloodWait {
		floodMutex.RLock()
		_, wait := floodWait[userid]
		floodMutex.RUnlock()

		if wait {
			return fmt.Errorf("this client already got their config less than %d minutes ago", FloodWait)
		}

		floodMutex.Lock()
		floodWait[userid] = time.Now().Unix()
		floodMutex.Unlock()
	}

	qrAttachment := echotron.NewInputFileBytes("qr.png", qrData)
	_, err := Bot.SendPhoto(qrAttachment, userid, &echotron.PhotoOptions{Caption: clientName})
	if err != nil {
		log.Errorf("Failed to send QR code: %v", err)
		return fmt.Errorf("unable to send qr picture: %v", err)
	}

	confAttachment := echotron.NewInputFileBytes(clientName+".conf", confData)
	_, err = Bot.SendDocument(confAttachment, userid, nil)
	if err != nil {
		log.Errorf("Failed to send config file: %v", err)
		return fmt.Errorf("unable to send conf file: %v", err)
	}
	return nil
}

func updateFloodWait() {
	floodMutex.Lock()
	defer floodMutex.Unlock()

	thresholdTS := time.Now().Unix() - 60*int64(FloodWait)
	for userid, ts := range floodWait {
		if ts < thresholdTS {
			delete(floodWait, userid)
			delete(floodMessageSent, userid)
		}
	}
}
