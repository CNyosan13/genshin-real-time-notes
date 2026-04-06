package main

import (
	"fmt"
	"math/rand"
	"resin/cmd"
	"resin/embedded"
	"resin/pkg/config"
	"resin/pkg/hoyo"
	"resin/pkg/hoyo/zzz"
	"resin/pkg/logging"
	"resin/pkg/ui"
	"time"

	"github.com/energye/systray"
)

var logFile string = ".\\charge.log"
var configFile string = ".\\hoyo_cookie.json"

type AllAssets struct {
	ChargeFull     []byte `asset:"zzz/charge_full.ico"`
	ChargeNotFull  []byte `asset:"zzz/charge_not_full.ico"`
	ChargeError    []byte `asset:"zzz/charge_error.ico"`
	Engagement     []byte `asset:"zzz/engagement.ico"`
	EngagementDone []byte `asset:"zzz/engagement_done.ico"`
	CheckIn        []byte `asset:"zzz/checkin.ico"`
	Ticket         []byte `asset:"zzz/ticket.ico"`
	Tape           []byte `asset:"zzz/tape.ico"`
}

var assets AllAssets

type Menu struct {
	Charge      *systray.MenuItem
	Engagement  *systray.MenuItem
	ScratchCard *systray.MenuItem
	VideoStore  *systray.MenuItem
	CheckIn     *systray.MenuItem
}

var SaleStates = map[string]string{
	"SaleStateDoing": "Open",
	"SaleStateNo":    "Closed",
	"SaleStateDone":  "Done",
}

var CardSigns = map[string]string{
	"CardSignNo":   "Incomplete",
	"CardSignDone": "Done",
}

func refreshData(cfg *config.Config, m *Menu) {
	uid := cfg.GetZzzUID()
	if uid == "" {
		m.Charge.SetTitle("Charge: no UID set")
		return
	}
	server, ok := zzz.Servers[uid[1]]
	if !ok {
		m.Charge.SetTitle("Charge: unknown region")
		return
	}

	zr, err := hoyo.GetData[zzz.ZzzResponse](zzz.BaseURL, server, uid, cfg.Ltoken, cfg.Ltuid)
	if err != nil {
		logging.Fail("ZZZ: failed getting data from %s: %s", zzz.BaseURL, err)
		systray.SetIcon(assets.ChargeError)
		return
	}
	if zr.Retcode != 0 {
		logging.Fail("ZZZ: server responded with (%d): %s", zr.Retcode, zr.Message)
		systray.SetIcon(assets.ChargeError)
		return
	}

	var currentVal, maxVal int
	if zr.Data.Energy.Progress.Max > 0 {
		currentVal = zr.Data.Energy.Progress.Current
		maxVal = zr.Data.Energy.Progress.Max
	} else {
		logging.Fail("ZZZ: data is empty or invalid")
		m.Charge.SetTitle("Charge: error")
		return
	}

	secs := zr.Data.Energy.Restore
	recovery := ""
	if secs != 0 {
		hours, minutes := hoyo.GetTime(secs)
		recovery = fmt.Sprintf(" [%dh %dm]", hours, minutes)
	}

	if currentVal == maxVal {
		systray.SetIcon(assets.ChargeFull)
	} else {
		systray.SetIcon(assets.ChargeNotFull)
	}
	title := fmt.Sprintf("%d/%d%s", currentVal, maxVal, recovery)
	systray.SetTooltip(title)

	m.Charge.SetTitle(title)
	dailyCur := zr.Data.Vitality.Current
	dailyMax := zr.Data.Vitality.Max
	m.Engagement.SetTitle(fmt.Sprintf("Engagement: %d/%d", dailyCur, dailyMax))

	if saleState, ok := SaleStates[zr.Data.VhsSale.SaleState]; ok {
		m.VideoStore.SetTitle(fmt.Sprintf("Video Store: %s", saleState))
	} else {
		m.VideoStore.SetTitle("Video Store: ERROR")
	}

	if cardSign, ok := CardSigns[zr.Data.CardSign]; ok {
		m.ScratchCard.SetTitle(fmt.Sprintf("Scratch Card: %s", cardSign))
	} else {
		m.ScratchCard.SetTitle("Scratch Card: ERROR")
	}
}

func onReady() {
	defer logging.CapturePanic()
	logging.SetFile(logFile)
	embedded.ReadAssets(&assets)

	m := &Menu{}
	m.Charge = ui.CreateMenuItem("Charge: ?/?", assets.ChargeNotFull)
	m.Engagement = ui.CreateMenuItem("Engagement: ?/?", assets.Engagement)
	m.ScratchCard = ui.CreateMenuItem("Scratch Card: ???", assets.Ticket)
	m.VideoStore = ui.CreateMenuItem("Video Store: ???", assets.Tape)
	m.CheckIn = ui.CreateMenuItem("Check In", assets.CheckIn)

	rand.Seed(time.Now().UnixNano())

	mgr := ui.InitApp("Zenless Zone Zero Real-Time Notes", "?/?", assets.ChargeNotFull, logFile, configFile, m, "zzz", refreshData)

	m.CheckIn.Click(func() {
		logging.Info("Clicked ZZZ check-in")
		resp, err := hoyo.GetDailyData[zzz.ZzzDailyResponse](zzz.DailyURL, mgr.Get().Ltoken, mgr.Get().Ltuid, zzz.ActID, "zzz")
		if err != nil {
			logging.Fail("ZZZ check-in failed: %s", err)
			return
		}
		logging.Info("ZZZ check-in: %d %s", resp.Retcode, resp.Message)
		ui.Notify("ZZZ Check-In", resp.Message, "zzz", assets.CheckIn)
	})
}

func main() {
	embedded.ExtractEmbeddedFiles()
	cmd.ReadArgs(configFile, ".\\daily_charge.log", func(cfg *config.Config) {
		hoyo.GetDailyData[zzz.ZzzDailyResponse](zzz.DailyURL, cfg.Ltoken, cfg.Ltuid, zzz.ActID, "zzz")
	})
	defer logging.CapturePanic()
	systray.Run(onReady, cmd.OnExit)
}
