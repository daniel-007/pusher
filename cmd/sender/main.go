package main

import (
	"flag"
	"github.com/Lupino/go-periodic"
	"github.com/Lupino/pusher"
	"github.com/Lupino/pusher/senders"
	"github.com/sendgrid/sendgrid-go"
	"log"
)

var (
	periodicPort string
	sgUser       string
	sgKey        string
	from         string
	fromName     string
)

func init() {
	flag.StringVar(&periodicPort, "periodic_port", "unix:///tmp/periodic.sock", "the periodic server port.")
	flag.StringVar(&sgUser, "sendgrid_user", "", "The SendGrid username.")
	flag.StringVar(&sgKey, "sendgrid_key", "", "The SendGrid password.")
	flag.StringVar(&from, "from", "", "The sendmail from address.")
	flag.StringVar(&fromName, "from_name", "", "The sendmail from name.")
	flag.Parse()
}

func main() {
	pw := periodic.NewWorker()
	if err := pw.Connect(periodicPort); err != nil {
		log.Fatal(err)
	}
	var sg = sendgrid.NewSendGridClient(sgUser, sgKey)
	var mailSender = senders.NewMailSender(sg, from, fromName)
	pusher.RunWorker(pw, mailSender)
}
