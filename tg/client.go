package tg

import (
	"encoding/json"
	"fmt"
	"log"
	"sync/atomic"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/rezam90/go-tdlib"
)

var (
	config = tdlib.Config{
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
		IgnoreFileNames:     false,
	}

	scopeNotificationSettings = tdlib.NewScopeNotificationSettings(int32((3 * 30 * 24 * time.Hour).Seconds()), "", false, true, true)
)

type Account struct {
	Client       *tdlib.Client
	db           *sqlx.DB
	floodWait    time.Duration
	lastFlood    time.Time
	stopCh       chan struct{}
	stopJoinerCh chan struct{}
	joinerActive int32
}

func NewAccount(config tdlib.Config) *Account {

	client := tdlib.NewClient(config)

	return &Account{
		Client:       client,
		lastFlood:    time.Time{},
		stopCh:       make(chan struct{}, 0),
		stopJoinerCh: make(chan struct{}, 0),
	}
}

func (c *Account) handleErr(err error) bool {
	var duration int64
	if n, _ := fmt.Sscanf(err.Error(), "FLOOD_WAIT_%d", &duration); n == 1 {
		log.Println("FLOOD_WAIT", duration, "sec")
		c.floodWait = time.Duration(duration) * time.Second
		c.lastFlood = time.Now()
		return true
	}

	return false
}

func (c *Account) Stop() {
	c.stopCh <- struct{}{}
}

func (c *Account) GetUpdates() {
	// rawUpdates gets all updates comming from tdlib
	rawUpdates := c.Client.GetRawUpdatesChannel(100)
	defer func() {
		close(rawUpdates)
		close(c.stopCh)
	}()

	for {
		select {
		case <-c.stopCh:
			return
		case update := <-rawUpdates:
			DeafultHashCollector.Collect(string(update.Raw))
		}
	}
}

func (c *Account) StopJoiner() {
	c.stopJoinerCh <- struct{}{}
}

func (c *Account) IsJoinng() bool {
	return atomic.LoadInt32(&c.joinerActive) == 1
}

func (c *Account) StartJoiner(interval time.Duration) {

	c.stopJoinerCh = make(chan struct{})
	atomic.StoreInt32(&c.joinerActive, 1)

	defer func() {
		atomic.StoreInt32(&c.joinerActive, 0)
		close(c.stopJoinerCh)
	}()

	joinTicker := time.NewTicker(interval)

	for {
		select {
		case <-c.stopJoinerCh:
			return
		case <-joinTicker.C:
			go func() {
				err := c.joinGroup()
				if err != nil {
					log.Println(err)
				}
			}()
		}
	}
}

func (c *Account) joinGroup() error {

	if time.Since(c.lastFlood) < c.floodWait {
		return fmt.Errorf("still flood waiting %s seconds", fmtDuration(c.floodWait-time.Since(c.lastFlood)))
	}

	tx, err := c.db.Beginx()
	if err != nil {
		log.Println("joinGroup: can't begin tx")
		return err
	}

	var id int64
	var hash string
	err = tx.QueryRowx("select id, hash from hashes where used = 0").Scan(&id, &hash)
	if err != nil {
		tx.Rollback()
		return err
	}

	_, err = tx.Exec(`update hashes set used = 1 where id = ?`, id)
	if err != nil {
		tx.Rollback()
		return err
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	// get chat info
	inviteLinkInfo, err := c.Client.CheckChatInviteLink(hash)
	if err != nil {
		if c.handleErr(err) {
			rollbackHash(c.db, id)
		}
		return err
	}

	// not group?
	if !(inviteLinkInfo.Type.GetChatTypeEnum() == tdlib.ChatTypeBasicGroupType ||
		inviteLinkInfo.Type.GetChatTypeEnum() == tdlib.ChatTypeSupergroupType) {
		return fmt.Errorf("is not a group hash: %s", hash)
	}

	// join groups with at least 100 members
	if inviteLinkInfo.MemberCount < 100 {
		return fmt.Errorf("low group member count: %d", inviteLinkInfo.MemberCount)
	}

	chat, err := c.Client.JoinChatByInviteLink(hash)
	if err != nil {
		if c.handleErr(err) {
			rollbackHash(c.db, id)
		}
		return err
	}
	log.Println("JOINED", chat.ID, chat.Title)

	return nil
}

func (c *Account) GetGroupLinks() error {
	messages, err := c.Client.SearchMessages("joinchat", 0, 0, 0, 100)
	if err != nil {
		c.handleErr(err)
		return err
	}

	for _, message := range messages.Messages {
		data, err := json.Marshal(message.Content)
		if err != nil {
			continue
		}
		DeafultHashCollector.Collect(string(data))
	}

	return nil
}
