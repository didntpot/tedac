package main

import (
	"github.com/hugolgst/rich-go/client"
	"time"
)

// discordID ...
const discordID = "710885082100924416"

// startRPC ...
func (t *Tedac) startRPC() {
	err := client.Login(discordID)
	if err != nil {
		return
	}

	start := time.Now()
	tick := time.NewTicker(time.Second)
	defer tick.Stop()
	for {
		select {
		case <-tick.C:
			err = client.SetActivity(client.Activity{
				State:      t.remoteAddress,
				Details:    "Playing Minecraft: Bedrock Edition on 1.12",
				LargeImage: "tedac",
				LargeText:  "TedacMC",
				SmallImage: "mc",
				SmallText:  "Minecraft 1.12 Support",
				Timestamps: &client.Timestamps{
					Start: &start,
				},
			})
			if err != nil {
				return
			}
		case <-t.c:
			return
		}
	}
}
