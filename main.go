package main

import (
	"git.restartfu.com/restart/gophig.git"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
)

func init() {
	slog.SetLogLoggerLevel(slog.LevelInfo)
}

func main() {
	log := slog.Default()
	goph := gophig.NewGophig[ProxyInfo]("./config.toml", gophig.TOMLMarshaler{}, os.ModePerm)
	conf, err := goph.LoadConf()
	if os.IsNotExist(err) {
		_ = goph.SaveConf(ProxyInfo{
			RemoteAddress: "127.0.0.1:19133",
			LocalAddress:  "127.0.0.1:19132",
		})
	} else if err != nil {
		log.Error("failed to connect to remote server: " + err.Error())
		return
	}

	t := NewTedac(conf.LocalAddress)

	log.Info("starting tedac...")
	err = t.Connect(conf.RemoteAddress)
	if err != nil {
		log.Error("failed to connect to remote server: " + err.Error())
		return
	}

	info, err := t.ProxyingInfo()
	if err != nil {
		log.Error("failed to retrieve proxy info: " + err.Error())
		t.Terminate()
		return
	}
	log.Info("started tedac", "local", info.LocalAddress, "remote", info.RemoteAddress)

	c := make(chan os.Signal, 2)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	<-c

	log.Info("terminating tedac...")
	t.Terminate()
	log.Info("tedac is offline.")
}
