package main

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/stianeikeland/go-rpio/v4"
	"github.com/tarm/serial"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"time"
)

const port = ":88"
const token = "[token]"
const chatId = -00000000000
const dvr = "rtsp://888888:888888@192.168.1.18:554"
const photo = "/var/www/web/images/call.jpeg"

func main() {
	err := rpio.Open()
	if err != nil {
		log.Fatal(err)
	}
	callPin := rpio.Pin(4)
	callPin.Input()
	bot, err := tgbotapi.NewBotAPI(token)

	go checkDoorCall(callPin, bot)

	if err != nil {
		log.Panic(err)
	}

	http.HandleFunc("/relay", switchHandler)
	http.HandleFunc("/image", getImageHandler)

	log.Println("HomePi v0.37a")
	log.Println("Listen port " + port)
	log.Println("Usage: /relay?r=7&s=1 - relay 7 on")
	log.Println("Authorized on account: " + bot.Self.UserName)
	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatal(err)
	}
	err = rpio.Close()
	if err != nil {
		log.Fatal(err)
	}
}

func checkDoorCall(p rpio.Pin, bot *tgbotapi.BotAPI) {
	for true {
		callStatus := p.Read()
		if callStatus == rpio.High {
			location, _ := time.LoadLocation("Europe/Moscow")
			dateNow := time.Now().In(location)
			log.Println("Call")
			relayHandler(true, "1")
			time.Sleep(time.Second)
			relayHandler(false, "1")
			msg := tgbotapi.NewMessage(chatId, "Звонят в дверь "+dateNow.Format(time.RFC1123))
			_, errMsg := bot.Send(msg)
			if errMsg != nil {
				log.Println("Can not send message")
			}
			getImage()
			photoByte, errByte := ioutil.ReadFile(photo)
			if errByte != nil {
				log.Println("Can not read image")
			}

			photoFileByte := tgbotapi.FileBytes{
				Name:  "Фотка",
				Bytes: photoByte,
			}
			_, errImg := bot.Send(tgbotapi.NewPhoto(chatId, photoFileByte))
			if errImg != nil {
				log.Println("Can not send image")
			}
		}
		time.Sleep(300 * time.Millisecond)
	}
}

func getImageHandler(_ http.ResponseWriter, _ *http.Request) {
	getImage()
}

func getImage() {
	cmd := exec.Command("/usr/bin/ffmpeg", "-y", "-i", dvr, "-vframes", "1", "-f", "image2", photo)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	errCmd := cmd.Run()
	if errCmd != nil {
		log.Println("Can not save image")
		log.Println(errCmd)
		log.Println(cmd)
		log.Println(cmd.Stderr)
		log.Println(cmd.Stdout)
	}
}

func switchHandler(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	relay := q["r"][0]
	status := q["s"][0] != "0"
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	_, _ = w.Write([]byte(relayHandler(status, relay)))
}

func relayHandler(st bool, no string) string {
	r := ""
	c := &serial.Config{Name: "/dev/ttyUSB0", Baud: 115200}
	s, err := serial.OpenPort(c)
	if err != nil {
		log.Println(err)
	}

	com := []byte("")
	if st {
		com = []byte("on" + no + "\n")
	} else {
		com = []byte("off" + no + "\n")
	}

	n, err := s.Write(com)
	if err != nil {
		log.Println(err)
	}

	//log.Printf("Command:" + string(com))
	buf := make([]byte, 128)
	n, err = s.Read(buf)
	if err != nil {
		log.Println(err)
	}
	log.Printf("%q", buf[:n])
	r = "OK" //TODO
	return r
}
