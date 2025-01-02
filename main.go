package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/mattn/go-sqlite3"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
)

const COMMANDER_NUMBER = "972506433349"
const GROUP_ID = "120363323028360735"

type Soldier struct {
	name    string
	jid     types.JID
	message string
}

func composeMessage(soldiers []*Soldier) string {
	output := ""
	for index, soldier := range soldiers {
		output += fmt.Sprintf("*%s*\n%s\n", soldier.name, soldier.message)
		if index != len(soldiers) {
			output += "--------------------\n"
		}
	}
	return output
}

func getSoldierAuthor(msg *events.Message, soldiers []*Soldier) *Soldier {
	for _, soldier := range soldiers {
		if soldier.jid.User == msg.Info.Sender.User {
			return soldier
		}
	}
	return nil
}

func allSoldiersAnswered(soldiers []*Soldier) bool {
	for _, soldier := range soldiers {
		if soldier.message == "" {
			return false
		}
	}
	return true
}

func reactWithLike(soldiers []*Soldier) func(*WhatsappService, *events.Message) error {
	return func(s *WhatsappService, msg *events.Message) error {
		if soldier := getSoldierAuthor(msg, soldiers); soldier != nil && soldier.message == "" {
			err := s.React(soldier.jid, msg.Info.Chat, msg.Info.ID, "👍")
			return err
		}
		return nil
	}
}

func registerMessage(soldiers []*Soldier) func(*WhatsappService, *events.Message) error {
	return func(s *WhatsappService, msg *events.Message) error {
		if soldier := getSoldierAuthor(msg, soldiers); soldier != nil && soldier.message == "" {
			var content string
			if msg.Message.ExtendedTextMessage == nil {
				content = msg.Message.GetConversation()
			} else {
				content = *msg.Message.GetExtendedTextMessage().Text
			}
			soldier.message = content
			fmt.Printf("%+v", soldier)
		}
		return nil
	}
}

func sendIfFinished(soldiers []*Soldier) func(*WhatsappService, *events.Message) error {
	return func(s *WhatsappService, msg *events.Message) error {
		if allSoldiersAnswered(soldiers) {
			s.SendMessage(composeMessage(soldiers), COMMANDER_NUMBER)
			os.Exit(0)
		}
		return nil
	}
}

func printMessage(_ *WhatsappService, msg *events.Message) error {
	fmt.Printf("--------------------\n%+v\n--------------------\n", msg)
	fmt.Println(msg.Message.ExtendedTextMessage)
	return nil
}

func main() {
	teamB := []*Soldier{
		{name: "גיא אביב", jid: types.NewJID("972586570151", WHATSAPP_SERVER), message: ""},
		{name: "רועי סביון", jid: types.NewJID("972533392950", WHATSAPP_SERVER), message: ""},
		{name: "מיכל ארץ קדושה", jid: types.NewJID("972547501467", WHATSAPP_SERVER), message: ""},
		{name: "מלאכי שוורץ", jid: types.NewJID("972533011490", WHATSAPP_SERVER), message: ""},
		{name: "יונתן אבידור", jid: types.NewJID("972542166594", WHATSAPP_SERVER), message: ""},
	}
	dbLog := waLog.Stdout("Database", "DEBUG", true)
	container, err := sqlstore.New("sqlite3", "file:examplestore.db?_foreign_keys=on&_journal_mode=WAL", dbLog)
	if err != nil {
		panic(err)
	}
	deviceStore, err := container.GetFirstDevice()
	if err != nil {
		panic(err)
	}
	clientLog := waLog.Stdout("Client", "INFO", true)
	client := whatsmeow.NewClient(deviceStore, clientLog)
	whatsapp := NewWhatsappService(client)
	whatsapp.
		OnMessage(reactWithLike(teamB)).
		OnMessage(registerMessage(teamB)).
		OnMessage(sendIfFinished(teamB)).
		OnMessage(printMessage).
		Init()
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
	client.Disconnect()
}
