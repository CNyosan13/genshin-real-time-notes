package main

import (
	"fmt"
	"math/rand"
	"resin/cmd"
	"resin/embedded"
	"resin/pkg/config"
	"resin/pkg/hoyo"
	"resin/pkg/hoyo/hsr"
	"resin/pkg/logging"
	"resin/pkg/ui"
	"time"

	"github.com/energye/systray"
)

var logFile string = ".\\stamina.log"
var configFile string = ".\\hoyo_cookie.json"

type AllAssets struct {
	StaminaFull    []byte `asset:"hsr/stamina_full.ico"`
	StaminaNotFull []byte `asset:"hsr/stamina_not_full.ico"`
	StaminaError   []byte `asset:"hsr/stamina_error.ico"`
	Training       []byte `asset:"hsr/training.ico"`
	Expedition     []byte `asset:"hsr/expedition.ico"`
	EchoOfWar      []byte `asset:"hsr/echo_of_war.ico"`
	CheckIn        []byte `asset:"hsr/checkin.ico"`
}

var assets AllAssets

type Menu struct {
	Stamina    *systray.MenuItem
	Training   *systray.MenuItem
	Expedition *systray.MenuItem
	Reserve    *systray.MenuItem
	EchoOfWar  *systray.MenuItem
	CheckIn    *systray.MenuItem
}

func refreshData(cfg *config.Config, m *Menu) {
	uid := cfg.GetHsrUID()
	if uid == "" {
		m.Stamina.SetTitle("Stamina: no UID set")
		return
	}
	server, ok := hsr.Servers[uid[0]]
	if !ok {
		m.Stamina.SetTitle("Stamina: unknown region")
		return
	}

	hr, err := hoyo.GetData[hsr.HsrResponse](hsr.BaseURL, server, uid, cfg.Ltoken, cfg.Ltuid)
	if err != nil {
		logging.Fail("HSR: failed getting data from %s: %s", hsr.BaseURL, err)
		systray.SetIcon(assets.StaminaError)
		return
	}
	if hr.Retcode != 0 {
		logging.Fail("Server responded with (%d): %s", hr.Retcode, hr.Message)
		systray.SetIcon(assets.StaminaError)
		return
	}

	var currentVal, maxVal int
	if hr.Data.MaxStamina > 0 {
		currentVal = hr.Data.CurrentStamina
		maxVal = hr.Data.MaxStamina
	} else {
		logging.Fail("HSR: data is empty or invalid")
		m.Stamina.SetTitle("Stamina: error")
		return
	}

	seconds := hr.Data.StaminaRecoveryTime
	recovery := ""
	if seconds != 0 {
		hours, minutes := hoyo.GetTime(seconds)
		recovery = fmt.Sprintf(" [%dh %dm]", hours, minutes)
	}

	if currentVal == maxVal {
		systray.SetIcon(assets.StaminaFull)
	} else {
		systray.SetIcon(assets.StaminaNotFull)
	}
	title := fmt.Sprintf("%d/%d%s", currentVal, maxVal, recovery)
	systray.SetTooltip(title)

	m.Stamina.SetTitle(title)
	m.Training.SetTitle(fmt.Sprintf("Training: %d/%d", hr.Data.CurrentTrainScore, hr.Data.MaxTrainScore))
	count := 0
	for _, exp := range hr.Data.Expeditions {
		if exp.Status == "Finished" {
			count++
		}
	}
	m.Expedition.SetTitle(fmt.Sprintf("Expeditions: %d/%d", count, hr.Data.TotalExpeditionNum))
	m.Reserve.SetTitle(fmt.Sprintf("Reserve: %d/2400", hr.Data.CurrentReserveStamina))
	m.EchoOfWar.SetTitle(fmt.Sprintf("Echo of War: %d/%d", hr.Data.WeeklyCocoonCnt, hr.Data.WeeklyCocoonLimit))
}

func onReady() {
	defer logging.CapturePanic()
	logging.SetFile(logFile)
	embedded.ReadAssets(&assets)

	m := &Menu{}
	m.Stamina = ui.CreateMenuItem("Stamina: ?/?", assets.StaminaNotFull)
	m.Training = ui.CreateMenuItem("Training: ?/?", assets.Training)
	m.Expedition = ui.CreateMenuItem("Expeditions: ?/?", assets.Expedition)
	m.Reserve = ui.CreateMenuItem("Reserve: ?/?", assets.StaminaFull)
	m.EchoOfWar = ui.CreateMenuItem("Echo of War: ?/?", assets.EchoOfWar)
	m.CheckIn = ui.CreateMenuItem("Check In", assets.CheckIn)

	rand.Seed(time.Now().UnixNano())

	mgr := ui.InitApp("Honkai Star Rail Real-Time Notes", "?/?", assets.StaminaNotFull, logFile, configFile, m, "hsr", refreshData)

	m.CheckIn.Click(func() {
		logging.Info("Clicked check in")
		resp, err := hoyo.GetDailyData[hsr.HsrDailyResponse](hsr.DailyURL, mgr.Get().Ltoken, mgr.Get().Ltuid, hsr.ActID, "hsr")
		if err != nil {
			logging.Fail("HSR check-in failed: %s", err)
			return
		}
		logging.Info("HSR check-in: %d %s", resp.Retcode, resp.Message)
		ui.Notify("HSR Check-In", resp.Message, "hsr", assets.CheckIn)
	})
}

func main() {
	embedded.ExtractEmbeddedFiles()
	cmd.ReadArgs(configFile, ".\\daily_hsr.log", func(cfg *config.Config) {
		hoyo.GetDailyData[hsr.HsrDailyResponse](hsr.DailyURL, cfg.Ltoken, cfg.Ltuid, hsr.ActID, "hsr")
	})
	defer logging.CapturePanic()
	systray.Run(onReady, cmd.OnExit)
}
