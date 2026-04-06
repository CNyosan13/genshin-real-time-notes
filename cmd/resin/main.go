package main

import (
	"fmt"
	"math/rand"
	"resin/cmd"
	"resin/embedded"
	"resin/pkg/config"
	"resin/pkg/hoyo"
	"resin/pkg/hoyo/genshin"
	"resin/pkg/logging"
	"resin/pkg/ui"
	"strconv"
	"time"

	"github.com/energye/systray"
)

var logFile string = ".\\resin.log"
var configFile string = ".\\hoyo_cookie.json"

type AllAssets struct {
	ResinFull    []byte `asset:"genshin/resin_full.ico"`
	ResinNotFull []byte `asset:"genshin/resin_not_full.ico"`
	ResinError   []byte `asset:"genshin/resin_error.ico"`
	Commission   []byte `asset:"genshin/commission.ico"`
	Expedition   []byte `asset:"genshin/expedition.ico"`
	Realm        []byte `asset:"genshin/realm.ico"`
	WeeklyBoss   []byte `asset:"genshin/weekly_boss.ico"`
	CheckIn      []byte `asset:"genshin/checkin.ico"`
}

var assets AllAssets

type Menu struct {
	Resin      *systray.MenuItem
	Commission *systray.MenuItem
	Expedition *systray.MenuItem
	Realm      *systray.MenuItem
	Domain     *systray.MenuItem
	CheckIn    *systray.MenuItem
}

func refreshData(cfg *config.Config, m *Menu) {
	uid := cfg.GetGenshinUID()
	if uid == "" {
		m.Resin.SetTitle("Resin: no UID set")
		return
	}
	server, ok := genshin.Servers[uid[0]]
	if !ok {
		m.Resin.SetTitle("Resin: unknown region")
		return
	}

	gr, err := hoyo.GetData[genshin.GenshinResponse](genshin.BaseURL, server, uid, cfg.Ltoken, cfg.Ltuid)
	if err != nil {
		logging.Fail("Genshin: failed getting data: %s", err)
		systray.SetIcon(assets.ResinError)
		return
	}
	if gr.Retcode != 0 {
		logging.Fail("Genshin: server responded (%d): %s", gr.Retcode, gr.Message)
		systray.SetIcon(assets.ResinError)
		return
	}

	var currentVal, maxVal int
	if gr.Data.MaxResin > 0 {
		currentVal = gr.Data.CurrentResin
		maxVal = gr.Data.MaxResin
	} else {
		logging.Fail("Genshin: data is empty or invalid")
		m.Resin.SetTitle("Resin: error")
		return
	}

	seconds, _ := strconv.Atoi(gr.Data.ResinRecoveryTime)
	recovery := ""
	if seconds != 0 {
		hours, minutes := hoyo.GetTime(seconds)
		recovery = fmt.Sprintf(" [%dh %dm]", hours, minutes)
	}

	if currentVal == maxVal {
		systray.SetIcon(assets.ResinFull)
	} else {
		systray.SetIcon(assets.ResinNotFull)
	}
	title := fmt.Sprintf("%d/%d%s", currentVal, maxVal, recovery)
	systray.SetTooltip(title)

	m.Resin.SetTitle(title)
	m.Commission.SetTitle(fmt.Sprintf("Commissions: %d/%d", gr.Data.FinishedTaskNum, gr.Data.TotalTaskNum))
	count := 0
	for _, exp := range gr.Data.Expeditions {
		if exp.Status == "Finished" {
			count++
		}
	}
	m.Expedition.SetTitle(fmt.Sprintf("Expeditions: %d/%d", count, gr.Data.MaxExpeditionNum))
	m.Realm.SetTitle(fmt.Sprintf("Realm: %d/%d", gr.Data.CurrentHomeCoin, gr.Data.MaxHomeCoin))
	m.Domain.SetTitle(fmt.Sprintf("Weekly Bosses: %d/%d", gr.Data.RemainResinDiscountNum, gr.Data.ResinDiscountNumLimit))
}

func onReady() {
	defer logging.CapturePanic()
	logging.SetFile(logFile)
	embedded.ReadAssets(&assets)

	m := &Menu{}
	m.Resin = ui.CreateMenuItem("Resin: ?/?", assets.ResinNotFull)
	m.Commission = ui.CreateMenuItem("Commissions: ?/?", assets.Commission)
	m.Expedition = ui.CreateMenuItem("Expeditions: ?/?", assets.Expedition)
	m.Realm = ui.CreateMenuItem("Realm: ?/?", assets.Realm)
	m.Domain = ui.CreateMenuItem("Weekly Bosses: ?/?", assets.WeeklyBoss)
	m.CheckIn = ui.CreateMenuItem("Check In", assets.CheckIn)

	rand.Seed(time.Now().UnixNano())

	mgr := ui.InitApp("Genshin Real-Time Notes", "?/?", assets.ResinNotFull, logFile, configFile, m, "genshin", refreshData)

	m.CheckIn.Click(func() {
		logging.Info("Clicked Genshin check-in")
		resp, err := hoyo.GetDailyData[genshin.GenshinDailyResponse](genshin.DailyURL, mgr.Get().Ltoken, mgr.Get().Ltuid, genshin.ActID, "genshin")
		if err != nil {
			logging.Fail("Genshin check-in failed: %s", err)
			return
		}
		logging.Info("Genshin check-in: %d %s", resp.Retcode, resp.Message)
		ui.Notify("Genshin Check-In", resp.Message, "genshin", assets.CheckIn)
	})
}

func main() {
	embedded.ExtractEmbeddedFiles()
	cmd.ReadArgs(configFile, ".\\daily_resin.log", func(cfg *config.Config) {
		hoyo.GetDailyData[genshin.GenshinDailyResponse](genshin.DailyURL, cfg.Ltoken, cfg.Ltuid, genshin.ActID, "genshin")
	})
	defer logging.CapturePanic()
	systray.Run(onReady, cmd.OnExit)
}
