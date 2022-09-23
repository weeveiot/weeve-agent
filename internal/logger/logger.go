package logger

import (
	"net"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"

	"github.com/shirou/logrusmqtt"
	"github.com/sirupsen/logrus"
)

var Log *logrus.Logger

func Initialise() {
	Log = logrus.New()
}

func AddHook(broker string, topic string, noTLS bool) {
	params := logrusmqtt.MQTTHookParams{Topic: topic}

	brokerUrl, err := url.Parse(broker)
	if err != nil {
		Log.Fatal("Error on parsing broker ", err)
	}

	host, port, err := net.SplitHostPort(brokerUrl.Host)
	if err != nil {
		Log.Fatal("Error on spliting host port ", err)
	}

	prt, err := strconv.Atoi(port)
	if err != nil {
		Log.Fatal("Error on converting port string into int ", err)
	}

	params.Hostname = host
	params.Port = prt

	if !noTLS {
		exePath, err := os.Getwd()
		if err != nil {
			Log.Fatal("Could not get the path to the ca.crt.")
		}
		dir := filepath.Dir(exePath)
		params.CAFilepath = path.Join(dir, "../ca.crt")
	}

	hook, err := logrusmqtt.NewMQTTHook(params, logrus.DebugLevel)
	if err != nil {
		Log.Fatal("Error on adding log hook ", err)
	}

	Log.Hooks.Add(hook)
}
