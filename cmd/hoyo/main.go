package main

import (
	"fmt"
	"math/rand"
	"resin/cmd"
	"resin/embedded"
	"resin/pkg/config"
	"resin/pkg/hoyo"
	"resin/pkg/hoyo/genshin"
	"resin/pkg/hoyo/hsr"
	"resin/pkg/hoyo/zzz"
	"resin/pkg/logging"
	"resin/pkg/ui"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/energye/systray"
	"github.com/skratchdot/open-golang/open"
	"golang.org/x/sys/windows"
)

// ─── File paths ───────────────────────────────────────────────────────────────

var logFile string = ".\\hoyo.log"
var configFile string = ".\\hoyo_cookie.json"

// ─── Embedded assets ──────────────────────────────────────────────────────────

type AllAssets struct {
	// Genshin
	ResinFull    []byte `asset:"genshin/resin_full.ico"`
	ResinNotFull []byte `asset:"genshin/resin_not_full.ico"`
	ResinError   []byte `asset:"genshin/resin_error.ico"`
	Commission   []byte `asset:"genshin/commission.ico"`
	Expedition   []byte `asset:"genshin/expedition.ico"`
	Realm        []byte `asset:"genshin/realm.ico"`
	WeeklyBoss   []byte `asset:"genshin/weekly_boss.ico"`
	GenshinCI    []byte `asset:"genshin/checkin.ico"`

	// HSR
	StaminaFull    []byte `asset:"hsr/stamina_full.ico"`
	StaminaNotFull []byte `asset:"hsr/stamina_not_full.ico"`
	StaminaError   []byte `asset:"hsr/stamina_error.ico"`
	Training       []byte `asset:"hsr/training.ico"`
	HsrExpedition  []byte `asset:"hsr/expedition.ico"`
	EchoOfWar      []byte `asset:"hsr/echo_of_war.ico"`
	HsrCI          []byte `asset:"hsr/checkin.ico"`

	// ZZZ
	ChargeFull     []byte `asset:"zzz/charge_full.ico"`
	ChargeNotFull  []byte `asset:"zzz/charge_not_full.ico"`
	ChargeError    []byte `asset:"zzz/charge_error.ico"`
	Engagement     []byte `asset:"zzz/engagement.ico"`
	EngagementDone []byte `asset:"zzz/engagement_done.ico"`
	ZzzCI          []byte `asset:"zzz/checkin.ico"`
	Ticket         []byte `asset:"zzz/ticket.ico"`
	Tape           []byte `asset:"zzz/tape.ico"`
}

var assets AllAssets

// ─── Menu items ───────────────────────────────────────────────────────────────

type Menu struct {
	// Genshin
	Resin      *systray.MenuItem
	Commission *systray.MenuItem
	Expedition *systray.MenuItem
	Realm      *systray.MenuItem
	Domain     *systray.MenuItem
	GenshinCI  *systray.MenuItem
	GenshinWeb *systray.MenuItem

	// HSR
	Stamina   *systray.MenuItem
	Training  *systray.MenuItem
	HsrExp    *systray.MenuItem
	Reserve   *systray.MenuItem
	EchoOfWar *systray.MenuItem
	HsrCI     *systray.MenuItem
	HsrWeb    *systray.MenuItem

	// ZZZ
	Charge      *systray.MenuItem
	Engagement  *systray.MenuItem
	ScratchCard *systray.MenuItem
	VideoStore  *systray.MenuItem
	ZzzCI       *systray.MenuItem
	ZzzWeb      *systray.MenuItem
	SettingsNoti *systray.MenuItem
	TestNoti    *systray.MenuItem
}

// ─── ZZZ state maps ───────────────────────────────────────────────────────────

var SaleStates = map[string]string{
	"SaleStateDoing": "Open",
	"SaleStateNo":    "Closed",
	"SaleStateDone":  "Done",
}

var CardSigns = map[string]string{
	"CardSignNo":   "Incomplete",
	"CardSignDone": "Done",
}

// ─── Notification state ───────────────────────────────────────────────────────

var notifiedFull = struct {
	sync.Mutex
	states map[string]bool
}{
	states: make(map[string]bool),
}

func setNotified(game string, notified bool) {
	notifiedFull.Lock()
	defer notifiedFull.Unlock()
	notifiedFull.states[game] = notified
}

func isNotified(game string) bool {
	notifiedFull.Lock()
	defer notifiedFull.Unlock()
	return notifiedFull.states[game]
}

// ─── Auto Check-in state ─────────────────────────────────────────────────────

var activeCheckIns = struct {
	sync.Mutex
	lastDone map[string]time.Time
}{
	lastDone: make(map[string]time.Time),
}

var checkInGlobalLock sync.Mutex // Sequential execution for all check-ins

func tryAutoCheckIn(game string, ltoken string, ltuid string, url string, actID string) {
	activeCheckIns.Lock()
	last, exists := activeCheckIns.lastDone[game]
	// If checked in today (since 4 PM UTC), skip
	now := time.Now().UTC()
	resetTime := time.Date(now.Year(), now.Month(), now.Day(), 16, 0, 0, 0, time.UTC)
	if now.Before(resetTime) {
		resetTime = resetTime.AddDate(0, 0, -1)
	}

	if exists && last.After(resetTime) {
		activeCheckIns.Unlock()
		return
	}
	activeCheckIns.Unlock()

	// Perform check-in sequentially
	checkInGlobalLock.Lock()
	defer checkInGlobalLock.Unlock()

	// Double check status after getting lock
	activeCheckIns.Lock()
	if last, exists = activeCheckIns.lastDone[game]; exists && last.After(resetTime) {
		activeCheckIns.Unlock()
		return
	}
	activeCheckIns.Unlock()

	// 10 seconds random delay as requested
	jitter := rand.Intn(10) + 1
	logging.Info("Auto Check-In (%s): scheduled in %d seconds", game, jitter)
	time.Sleep(time.Duration(jitter) * time.Second)

	resp, err := hoyo.GetDailyData[genshin.GenshinDailyResponse](url, ltoken, ltuid, actID, game)
	if err != nil {
		logging.Fail("Auto Check-In (%s): failed: %s", game, err)
		return
	}
	logging.Info("Auto Check-In (%s): message=%s", game, resp.Message)

	activeCheckIns.Lock()
	activeCheckIns.lastDone[game] = time.Now().UTC()
	activeCheckIns.Unlock()
}

// ─── Refresh logic ────────────────────────────────────────────────────────────

// refreshAll fetches all three games in parallel and updates tray menu items.
func refreshAll(cfg *config.Config, m *Menu) {
	if cfg == nil || m == nil {
		return
	}

	var wg sync.WaitGroup
	wg.Add(3)

	go func() {
		defer wg.Done()
		refreshGenshin(cfg, m)
	}()
	go func() {
		defer wg.Done()
		refreshHsr(cfg, m)
	}()
	go func() {
		defer wg.Done()
		refreshZzz(cfg, m)
	}()

	wg.Wait()
}

// refreshGenshin updates Genshin Impact tray items.
func refreshGenshin(cfg *config.Config, m *Menu) {
	uid := cfg.GetGenshinUID()
	if uid == "" {
		m.Resin.SetTitle("Resin: no UID set")
		return
	}
	server, ok := genshin.Servers[uid[0]]
	if !ok {
		logging.Fail("Unknown Genshin UID prefix '%c' (UID=%s)", uid[0], uid)
		m.Resin.SetTitle("Resin: unknown region")
		return
	}

	gr, err := hoyo.GetData[genshin.GenshinResponse](genshin.BaseURL, server, uid, cfg.Ltoken, cfg.Ltuid)
	if err != nil {
		logging.Fail("Genshin: failed getting data: %s", err)
		m.Resin.SetTitle("Resin: error")
		return
	}
	if gr.Retcode != 0 {
		logging.Fail("Genshin: server responded (%d): %s", gr.Retcode, gr.Message)
		m.Resin.SetTitle(fmt.Sprintf("Resin: server error %d", gr.Retcode))
		return
	}

	var currentVal, maxVal int
	if gr.Data.MaxResin > 0 {
		currentVal = gr.Data.CurrentResin
		maxVal = gr.Data.MaxResin

		if cfg.MaxResin != maxVal {
			cfg.MaxResin = maxVal
			config.WriteConfig(cfg, configFile)
		}
	} else {
		logging.Fail("Genshin: data is empty or invalid")
		return
	}

	seconds, err := strconv.Atoi(gr.Data.ResinRecoveryTime)
	var recovery string
	if err != nil {
		recovery = " [?]"
	} else if seconds == 0 {
		recovery = ""
	} else {
		hours, minutes := hoyo.GetTime(seconds)
		recovery = fmt.Sprintf(" [%dh %dm]", hours, minutes)
	}

	title := fmt.Sprintf("Resin: %d/%d%s", currentVal, maxVal, recovery)
	threshold := cfg.ResinNotifyThreshold
	if threshold == 0 {
		threshold = maxVal
	}

	// Dashboard UI Logic
	isFull := currentVal >= maxVal
	isThresholdMet := currentVal >= threshold
	indicator := ""

	if isFull {
		m.Resin.SetIcon(assets.ResinFull)
		indicator = "\t‼️"
		setNotified("genshin_threshold", true) // Ensure threshold alert is suppressed
		if !isNotified("genshin_max") {
			ui.Notify("Genshin Impact", fmt.Sprintf("Your Resin is FULL (%d/%d)!", currentVal, maxVal), "genshin", assets.ResinFull)
			setNotified("genshin_max", true)
		}
	} else if isThresholdMet {
		m.Resin.SetIcon(assets.ResinNotFull)
		indicator = "\t❗"
		setNotified("genshin_max", false)
		if !isNotified("genshin_threshold") {
			ui.Notify("Genshin Impact", fmt.Sprintf("Your Resin has reached your threshold: %d/%d!", currentVal, maxVal), "genshin", assets.ResinFull)
			setNotified("genshin_threshold", true)
		}
	} else {
		m.Resin.SetIcon(assets.ResinNotFull)
		setNotified("genshin_threshold", false)
		setNotified("genshin_max", false)
	}
	m.Resin.SetTitle(title + indicator)

	commCur := gr.Data.FinishedTaskNum
	commMax := gr.Data.TotalTaskNum
	commTitle := fmt.Sprintf("Commissions: %d/%d", commCur, commMax)
	indicator = ""
	if commCur == commMax {
		m.Commission.Disable()
	} else {
		m.Commission.Enable()
		indicator = "\t❗"
	}
	m.Commission.SetTitle(commTitle + indicator)

	count := 0
	for _, exp := range gr.Data.Expeditions {
		if exp.Status == "Finished" {
			count++
		}
	}
	expTitle := fmt.Sprintf("Expeditions: %d/%d", count, gr.Data.MaxExpeditionNum)
	indicator = ""
	if count > 0 {
		indicator = "\t❗"
		if !isNotified("genshin_exp_done") {
			ui.Notify("Genshin Impact", "Expeditions are ready to claim!", "genshin", assets.Expedition)
			setNotified("genshin_exp_done", true)
		}
	} else {
		setNotified("genshin_exp_done", false)
	}
	m.Expedition.SetTitle(expTitle + indicator)

	realmCur := gr.Data.CurrentHomeCoin
	realmMax := gr.Data.MaxHomeCoin
	realmTitle := fmt.Sprintf("Realm: %d/%d", realmCur, realmMax)
	indicator = ""
	if realmCur >= realmMax && realmMax > 0 {
		indicator = "\t❗"
		if !isNotified("genshin_realm_full") {
			ui.Notify("Genshin Impact", "Realm Currency is FULL!", "genshin", assets.Realm)
			setNotified("genshin_realm_full", true)
		}
	} else {
		setNotified("genshin_realm_full", false)
	}
	m.Realm.SetTitle(realmTitle + indicator)
	m.Domain.SetTitle(fmt.Sprintf("Weekly Bosses: %d/%d", gr.Data.RemainResinDiscountNum, gr.Data.ResinDiscountNumLimit))

	// Check-in logic
	ci, err := hoyo.GetCheckInStatus[genshin.GenshinCheckInInfoResponse](genshin.InfoURL, cfg.Ltoken, cfg.Ltuid, genshin.ActID, "genshin")
	if err == nil && ci.Retcode == 0 {
		status := "Not Done"
		indicator := "\t❗"
		if ci.Data.IsSign {
			status = "Done"
			indicator = ""
		} else {
			// Auto check-in if not signed
			go tryAutoCheckIn("genshin", cfg.Ltoken, cfg.Ltuid, genshin.DailyURL, genshin.ActID)
		}
		m.GenshinCI.SetTitle(fmt.Sprintf("Check In (Genshin) : %s%s", status, indicator))
	} else {
		m.GenshinCI.SetTitle("Check In (Genshin) : Error")
	}
}

// refreshHsr updates Honkai: Star Rail tray items.
func refreshHsr(cfg *config.Config, m *Menu) {
	uid := cfg.GetHsrUID()
	if uid == "" {
		m.Stamina.SetTitle("Stamina: no UID set")
		return
	}
	server, ok := hsr.Servers[uid[0]]
	if !ok {
		logging.Fail("Unknown HSR UID prefix '%c' (UID=%s)", uid[0], uid)
		m.Stamina.SetTitle("Stamina: unknown region")
		return
	}

	hr, err := hoyo.GetData[hsr.HsrResponse](hsr.BaseURL, server, uid, cfg.Ltoken, cfg.Ltuid)
	if err != nil {
		logging.Fail("HSR: failed getting data: %s", err)
		m.Stamina.SetTitle("Stamina: error")
		return
	}
	if hr.Retcode != 0 {
		logging.Fail("HSR: server responded (%d): %s", hr.Retcode, hr.Message)
		m.Stamina.SetTitle(fmt.Sprintf("Stamina: server error %d", hr.Retcode))
		return
	}

	var currentVal, maxVal int
	if hr.Data.MaxStamina > 0 {
		currentVal = hr.Data.CurrentStamina
		maxVal = hr.Data.MaxStamina

		if cfg.MaxStamina != maxVal {
			cfg.MaxStamina = maxVal
			config.WriteConfig(cfg, configFile)
		}
	} else {
		logging.Fail("HSR: data is empty or invalid")
		return
	}
	secs := hr.Data.StaminaRecoveryTime
	recovery := ""
	if secs != 0 {
		hours, minutes := hoyo.GetTime(secs)
		recovery = fmt.Sprintf(" [%dh %dm]", hours, minutes)
	}

	title := fmt.Sprintf("Stamina: %d/%d%s", currentVal, maxVal, recovery)
	threshold := cfg.StaminaNotifyThreshold
	if threshold == 0 {
		threshold = maxVal
	}

	// Dashboard UI Logic
	isFull := currentVal >= maxVal
	isThresholdMet := currentVal >= threshold
	indicator := ""

	if isFull {
		m.Stamina.SetIcon(assets.StaminaFull)
		indicator = "\t‼️"
		setNotified("hsr_threshold", true)
		if !isNotified("hsr_max") {
			ui.Notify("Honkai: Star Rail", fmt.Sprintf("Your Stamina is FULL (%d/%d)!", currentVal, maxVal), "hsr", assets.StaminaFull)
			setNotified("hsr_max", true)
		}
	} else if isThresholdMet {
		m.Stamina.SetIcon(assets.StaminaNotFull)
		indicator = "\t❗"
		setNotified("hsr_max", false)
		if !isNotified("hsr_threshold") {
			ui.Notify("Honkai: Star Rail", fmt.Sprintf("Your Stamina has reached your threshold: %d/%d!", currentVal, maxVal), "hsr", assets.StaminaFull)
			setNotified("hsr_threshold", true)
		}
	} else {
		m.Stamina.SetIcon(assets.StaminaNotFull)
		setNotified("hsr_threshold", false)
		setNotified("hsr_max", false)
	}
	m.Stamina.SetTitle(title + indicator)

	trainCur := hr.Data.CurrentTrainScore
	trainMax := hr.Data.MaxTrainScore
	trainTitle := fmt.Sprintf("Training: %d/%d", trainCur, trainMax)
	indicator = ""
	if trainCur == trainMax {
		m.Training.Disable()
	} else {
		m.Training.Enable()
		indicator = "\t❗"
	}
	m.Training.SetTitle(trainTitle + indicator)

	count := 0
	for _, exp := range hr.Data.Expeditions {
		if exp.Status == "Finished" {
			count++
		}
	}
	expTitle := fmt.Sprintf("Expeditions: %d/%d", count, hr.Data.TotalExpeditionNum)
	indicator = ""
	if count > 0 {
		indicator = "\t❗"
		if !isNotified("hsr_exp_done") {
			ui.Notify("Honkai: Star Rail", "Expeditions are ready to claim!", "hsr", assets.HsrExpedition)
			setNotified("hsr_exp_done", true)
		}
	} else {
		setNotified("hsr_exp_done", false)
	}
	m.HsrExp.SetTitle(expTitle + indicator)
	m.Reserve.SetTitle(fmt.Sprintf("Reserve: %d/2400", hr.Data.CurrentReserveStamina))
	m.EchoOfWar.SetTitle(fmt.Sprintf("Echo of War: %d/%d", hr.Data.WeeklyCocoonCnt, hr.Data.WeeklyCocoonLimit))

	// Check-in logic
	ci, err := hoyo.GetCheckInStatus[hsr.HsrCheckInInfoResponse](hsr.InfoURL, cfg.Ltoken, cfg.Ltuid, hsr.ActID, "hsr")
	if err == nil && ci.Retcode == 0 {
		status := "Not Done"
		indicator := "\t❗"
		if ci.Data.IsSign {
			status = "Done"
			indicator = ""
		} else {
			// Auto check-in if not signed
			go tryAutoCheckIn("hsr", cfg.Ltoken, cfg.Ltuid, hsr.DailyURL, hsr.ActID)
		}
		m.HsrCI.SetTitle(fmt.Sprintf("Check In (HSR) : %s%s", status, indicator))
	} else {
		m.HsrCI.SetTitle("Check In (HSR) : Error")
	}
}

// refreshZzz updates Zenless Zone Zero tray items.
func refreshZzz(cfg *config.Config, m *Menu) {
	uid := cfg.GetZzzUID()
	if uid == "" {
		m.Charge.SetTitle("Charge: no UID set")
		return
	}
	if len(uid) < 2 {
		logging.Fail("ZZZ UID too short (UID=%s)", uid)
		m.Charge.SetTitle("Charge: invalid UID")
		return
	}
	server, ok := zzz.Servers[uid[1]]
	if !ok {
		logging.Fail("Unknown ZZZ UID region '%c' (UID=%s)", uid[1], uid)
		m.Charge.SetTitle("Charge: unknown region")
		return
	}

	zr, err := hoyo.GetData[zzz.ZzzResponse](zzz.BaseURL, server, uid, cfg.Ltoken, cfg.Ltuid)
	if err != nil {
		logging.Fail("ZZZ: failed getting data: %s", err)
		m.Charge.SetTitle("Charge: error")
		return
	}
	if zr.Retcode != 0 {
		logging.Fail("ZZZ: server responded (%d): %s", zr.Retcode, zr.Message)
		m.Charge.SetTitle(fmt.Sprintf("Charge: server error %d", zr.Retcode))
		return
	}

	var currentVal, maxVal int
	if zr.Data.Energy.Progress.Max > 0 {
		currentVal = zr.Data.Energy.Progress.Current
		maxVal = zr.Data.Energy.Progress.Max

		if cfg.MaxCharge != maxVal {
			cfg.MaxCharge = maxVal
			config.WriteConfig(cfg, configFile)
		}
	} else {
		logging.Fail("ZZZ: data is empty or invalid")
		return
	}
	secs := zr.Data.Energy.Restore
	recovery := ""
	if secs != 0 {
		hours, minutes := hoyo.GetTime(secs)
		recovery = fmt.Sprintf(" [%dh %dm]", hours, minutes)
	}

	title := fmt.Sprintf("Charge: %d/%d%s", currentVal, maxVal, recovery)
	threshold := cfg.ChargeNotifyThreshold
	if threshold == 0 {
		threshold = maxVal
	}

	isFull := currentVal >= maxVal
	isThresholdMet := currentVal >= threshold
	indicator := ""

	if isFull {
		m.Charge.SetIcon(assets.ChargeFull)
		indicator = "\t‼️"
		setNotified("zzz_threshold", true)
		if !isNotified("zzz_max") {
			ui.Notify("Zenless Zone Zero", fmt.Sprintf("Your Battery Charge is FULL (%d/%d)!", currentVal, maxVal), "zzz", assets.ChargeFull)
			setNotified("zzz_max", true)
		}
	} else if isThresholdMet {
		m.Charge.SetIcon(assets.ChargeNotFull)
		indicator = "\t❗"
		setNotified("zzz_max", false)
		if !isNotified("zzz_threshold") {
			ui.Notify("Zenless Zone Zero", fmt.Sprintf("Your Battery Charge has reached your threshold: %d/%d!", currentVal, maxVal), "zzz", assets.ChargeFull)
			setNotified("zzz_threshold", true)
		}
	} else {
		m.Charge.SetIcon(assets.ChargeNotFull)
		setNotified("zzz_threshold", false)
		setNotified("zzz_max", false)
	}
	m.Charge.SetTitle(title + indicator)

	dailyCur := zr.Data.Vitality.Current
	dailyMax := zr.Data.Vitality.Max
	dailyTitle := fmt.Sprintf("Engagement: %d/%d", dailyCur, dailyMax)
	indicator = ""
	if dailyCur == dailyMax {
		m.Engagement.Disable()
		m.Engagement.SetIcon(assets.EngagementDone)
	} else {
		m.Engagement.Enable()
		m.Engagement.SetIcon(assets.Engagement)
		indicator = "\t❗"
	}
	m.Engagement.SetTitle(dailyTitle + indicator)

	if saleState, ok := SaleStates[zr.Data.VhsSale.SaleState]; ok {
		indicator := ""
		if zr.Data.VhsSale.SaleState != "SaleStateDoing" {
			indicator = "\t❗"
		}
		m.VideoStore.SetTitle(fmt.Sprintf("Video Store: %s%s", saleState, indicator))
	} else {
		m.VideoStore.SetTitle("Video Store: ERROR")
	}

	if cardSign, ok := CardSigns[zr.Data.CardSign]; ok {
		indicator := ""
		if zr.Data.CardSign == "CardSignNo" {
			indicator = "\t❗"
		}
		m.ScratchCard.SetTitle(fmt.Sprintf("Scratch Card: %s%s", cardSign, indicator))
	} else {
		m.ScratchCard.SetTitle("Scratch Card: ERROR")
	}

	// Check-in logic
	ci, err := hoyo.GetCheckInStatus[zzz.ZzzCheckInInfoResponse](zzz.InfoURL, cfg.Ltoken, cfg.Ltuid, zzz.ActID, "zzz")
	if err == nil && ci.Retcode == 0 {
		status := "Not Done"
		indicator := "\t❗"
		if ci.Data.IsSign {
			status = "Done"
			indicator = ""
		} else {
			// Auto check-in if not signed
			go tryAutoCheckIn("zzz", cfg.Ltoken, cfg.Ltuid, zzz.DailyURL, zzz.ActID)
		}
		m.ZzzCI.SetTitle(fmt.Sprintf("Check In (ZZZ) : %s%s", status, indicator))
	} else {
		m.ZzzCI.SetTitle("Check In (ZZZ) : Error")
	}
}

// ─── Daily check-in ──────────────────────────────────────────────────────────

func checkIn(cfg *config.Config) {
	var wg sync.WaitGroup
	wg.Add(3)

	go func() {
		defer wg.Done()
		resp, err := hoyo.GetDailyData[genshin.GenshinDailyResponse](genshin.DailyURL, cfg.Ltoken, cfg.Ltuid, genshin.ActID, "genshin")
		if err != nil {
			logging.Fail("Genshin check-in failed: %s", err)
			return
		}
		logging.Info("Genshin check-in: %d %s", resp.Retcode, resp.Message)
	}()

	go func() {
		defer wg.Done()
		resp, err := hoyo.GetDailyData[hsr.HsrDailyResponse](hsr.DailyURL, cfg.Ltoken, cfg.Ltuid, hsr.ActID, "hsr")
		if err != nil {
			logging.Fail("HSR check-in failed: %s", err)
			return
		}
		logging.Info("HSR check-in: %d %s", resp.Retcode, resp.Message)
	}()

	go func() {
		defer wg.Done()
		resp, err := hoyo.GetDailyData[zzz.ZzzDailyResponse](zzz.DailyURL, cfg.Ltoken, cfg.Ltuid, zzz.ActID, "zzz")
		if err != nil {
			logging.Fail("ZZZ check-in failed: %s", err)
			return
		}
		logging.Info("ZZZ check-in: %d %s", resp.Retcode, resp.Message)
	}()

	wg.Wait()
}

// ─── Tray setup ───────────────────────────────────────────────────────────────

func onReady() {
	defer logging.CapturePanic()
	logging.SetFile(logFile)

	embedded.ReadAssets(&assets)

	m := &Menu{}

	// ── Genshin Impact section ─────────────────────────────────────────────
	systray.AddMenuItem("── Genshin Impact ──", "").Disable()
	m.Resin = ui.CreateMenuItem("Resin: ?/?", assets.ResinNotFull)
	m.Commission = ui.CreateMenuItem("Commissions: ?/?", assets.Commission)
	m.Expedition = ui.CreateMenuItem("Expeditions: ?/?", assets.Expedition)
	m.Realm = ui.CreateMenuItem("Realm: ?/?", assets.Realm)
	m.Domain = ui.CreateMenuItem("Weekly Bosses: ?/?", assets.WeeklyBoss)
	m.GenshinCI = ui.CreateMenuItem("Check In (Genshin)", assets.GenshinCI)
	m.GenshinWeb = systray.AddMenuItem("Open HoyoLAB (Genshin)", "")

	systray.AddSeparator()

	// ── Honkai: Star Rail section ───────────────────────────────────────────
	systray.AddMenuItem("── Honkai: Star Rail ──", "").Disable()
	m.Stamina = ui.CreateMenuItem("Stamina: ?/?", assets.StaminaNotFull)
	m.Training = ui.CreateMenuItem("Training: ?/?", assets.Training)
	m.HsrExp = ui.CreateMenuItem("Expeditions: ?/?", assets.HsrExpedition)
	m.Reserve = ui.CreateMenuItem("Reserve: ?/2400", assets.StaminaFull)
	m.EchoOfWar = ui.CreateMenuItem("Echo of War: ?/?", assets.EchoOfWar)
	m.HsrCI = ui.CreateMenuItem("Check In (HSR)", assets.HsrCI)
	m.HsrWeb = systray.AddMenuItem("Open HoyoLAB (HSR)", "")

	systray.AddSeparator()

	// ── Zenless Zone Zero section ───────────────────────────────────────────
	systray.AddMenuItem("── Zenless Zone Zero ──", "").Disable()
	m.Charge = ui.CreateMenuItem("Charge: ?/?", assets.ChargeNotFull)
	m.Engagement = ui.CreateMenuItem("Engagement: ?/?", assets.Engagement)
	m.ScratchCard = ui.CreateMenuItem("Scratch Card: ???", assets.Ticket)
	m.VideoStore = ui.CreateMenuItem("Video Store: ???", assets.Tape)
	m.ZzzCI = ui.CreateMenuItem("Check In (ZZZ)", assets.ZzzCI)
	m.ZzzWeb = systray.AddMenuItem("Open HoyoLAB (ZZZ)", "")

	systray.AddSeparator()
	m.SettingsNoti = systray.AddMenuItem("Notification Settings", "Adjust thresholds for alerts")
	m.TestNoti = systray.AddMenuItem("Send Test Notification", "Verify that Windows Toasts are working")

	systray.AddSeparator()

	// ── Init app (login, refresh loop, common menu controls) ────────────────
	rand.Seed(time.Now().UnixNano())

	mgr := ui.InitApp(
		"HoyoLAB Real-Time Notes",
		"Genshin · HSR · ZZZ",
		assets.ResinNotFull,
		logFile,
		configFile,
		m,
		"hoyo",
		func(cfg *config.Config, m *Menu) {
			refreshAll(cfg, m)
		},
	)

	// Single source of refresh: handled by fresh loop inside InitApp.
	// No extra startup refresh here to prevent race conditions.
	m.GenshinCI.Click(func() {
		logging.Info("Clicked Genshin check-in")
		resp, err := hoyo.GetDailyData[genshin.GenshinDailyResponse](genshin.DailyURL, mgr.Get().Ltoken, mgr.Get().Ltuid, genshin.ActID, "genshin")
		if err != nil {
			logging.Fail("Genshin check-in: %s", err)
			ui.Notify("Genshin Check-In", fmt.Sprintf("Failed: %v", err), "genshin", assets.GenshinCI)
			return
		}
		logging.Info("Genshin check-in: %d %s", resp.Retcode, resp.Message)
		ui.Notify("Genshin Check-In", resp.Message, "genshin", assets.GenshinCI)
	})

	m.GenshinWeb.Click(func() {
		open.Start("https://act.hoyolab.com/app/community-game-records-sea/index.html?#/ys")
	})

	m.HsrCI.Click(func() {
		logging.Info("Clicked HSR check-in")
		resp, err := hoyo.GetDailyData[hsr.HsrDailyResponse](hsr.DailyURL, mgr.Get().Ltoken, mgr.Get().Ltuid, hsr.ActID, "hsr")
		if err != nil {
			logging.Fail("HSR check-in: %s", err)
			ui.Notify("HSR Check-In", fmt.Sprintf("Failed: %v", err), "hsr", assets.HsrCI)
			return
		}
		logging.Info("HSR check-in: %d %s", resp.Retcode, resp.Message)
		ui.Notify("HSR Check-In", resp.Message, "hsr", assets.HsrCI)
	})

	m.HsrWeb.Click(func() {
		open.Start("https://act.hoyolab.com/app/community-game-records-sea/rpg/index.html?bbs_presentation_style=fullscreen&gid=6&bbs_theme=dark#/hsr")
	})

	m.ZzzCI.Click(func() {
		logging.Info("Clicked ZZZ check-in")
		resp, err := hoyo.GetDailyData[zzz.ZzzDailyResponse](zzz.DailyURL, mgr.Get().Ltoken, mgr.Get().Ltuid, zzz.ActID, "zzz")
		if err != nil {
			logging.Fail("ZZZ check-in: %s", err)
			ui.Notify("ZZZ Check-In", fmt.Sprintf("Failed: %v", err), "zzz", assets.ZzzCI)
			return
		}
		logging.Info("ZZZ check-in: %d %s", resp.Retcode, resp.Message)
		ui.Notify("ZZZ Check-In", resp.Message, "zzz", assets.ZzzCI)
	})

	m.ZzzWeb.Click(func() {
		open.Start("https://act.hoyolab.com/app/zzz-game-record/index.html?hyl_presentation_style=fullscreen&lang=en-us&bbs_theme=dark#/zzz")
	})

	m.SettingsNoti.Click(func() {
		logging.Info("Clicked Notification Settings")
		wd, _ := os.Getwd()
		exeName := fmt.Sprintf("WebViewLogin-%s.exe", config.VERSION)
		exe := filepath.Join(wd, "login", exeName)
		exec.Command(exe, "--settings").Start()
	})

	m.TestNoti.Click(func() {
		logging.Info("Clicked Test Notification")
		ui.Notify("Test: Genshin Impact", "This is a sample notification with the Resin icon!", "genshin", assets.ResinFull)
		time.Sleep(500 * time.Millisecond) // Slight delay so they don't overlap too much
		ui.Notify("Test: Honkai: Star Rail", "This is a sample notification with the Stamina icon!", "hsr", assets.StaminaFull)
		time.Sleep(500 * time.Millisecond)
		ui.Notify("Test: Zenless Zone Zero", "This is a sample notification with the Charge icon!", "zzz", assets.ChargeFull)
	})
}

// ─── Entry point ─────────────────────────────────────────────────────────────

func main() {
	// Single Instance Logic: Warn user if already running
	mutexName := "Global\\HoyoLABMonitorMutex"
	_, err := windows.CreateMutex(nil, false, windows.StringToUTF16Ptr(mutexName))
	if err != nil && err == windows.ERROR_ALREADY_EXISTS {
		windows.MessageBox(0, windows.StringToUTF16Ptr("HoyoLAB Monitor is already running. Please close the existing instance before launching a new one."), windows.StringToUTF16Ptr("HoyoLAB Monitor"), windows.MB_OK|windows.MB_ICONINFORMATION)
		os.Exit(0)
	}

	embedded.ExtractEmbeddedFiles()
	cmd.ReadArgs(configFile, ".\\daily_hoyo.log", checkIn)
	defer logging.CapturePanic()
	systray.Run(onReady, cmd.OnExit)
}
