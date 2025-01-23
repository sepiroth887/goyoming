package handler

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os/exec"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

func New(config Configuration) *Handler {
	h := &Handler{
		config:     config,
		httpClient: http.DefaultClient,
	}
	h.httpClient.Timeout = time.Second * 5
	return h
}

func (h *Handler) ListenAndServe() error {
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", h.config.Listen, h.config.Port))
	if err != nil {
		return err
	}

	go h.monitorSatellite()

	defer listener.Close()

	for {
		log.Info("listening for new clients")
		conn, err := listener.Accept()
		if err != nil {
			log.Errorf("failed to accept connection: %v", err)
		}

		client := Client{
			conn: conn,
		}
		go client.Handle(h)
	}
}

func (h *Handler) monitorSatellite() {
	for range time.NewTicker(time.Second * 2).C {
		if h.isAwake && !h.hasResponded {
			if time.Now().After(h.timeOnWake.Add(time.Second * 15)) {
				log.Warn("recognition started, but no response. restarting satellite")
				err := exec.Command("systemctl", "restart", "satellite").Run()
				if err != nil {
					log.Errorf("failed to restart satellite: %v", err)
				}
			}
		}
	}
}

func (c *Client) Handle(h *Handler) {
	reader := bufio.NewReader(c.conn)
	log.Infof("new client connected: %s", c.conn.RemoteAddr().String())
	for {
		var event Event
		var eventData EventData
		m, err := reader.ReadBytes('\n')

		if err != nil {
			c.conn.Close()
			log.Errorf("failed to read from conn: %v", err)
			return
		}
		log.Debugf("received event: %s", string(m))
		err = json.Unmarshal(m, &event)
		if err != nil {
			log.Errorf("failed to decode event: %s\n%v", string(m), err)
			continue
		}

		if event.Length <= 0 {
			// log.Println("received no addl data in event.")
			continue
		}
		log.Debugf("reading %d bytes", event.Length)
		dataBuf := make([]byte, event.Length)

		n, err := io.ReadFull(reader, dataBuf)
		if err != nil {
			log.Errorf("failed to read event data: %v\n", err)
			continue
		}

		log.Debugf("read %d bytes", n)

		err = json.Unmarshal(dataBuf, &eventData)
		if err != nil {
			fmt.Printf("failed to decode event data: %v\n", err)
			continue
		}
		event.Data = eventData
		log.Debug(event)

		switch event.Type {
		case "detection":
			h.isAwake = true
			h.timeOnWake = time.Now()
			go h.playSound(h.config.ActivitySettings.RecognitionStart)
		case "voice-started":
			h.timeOnWake = time.Now()
		case "voice-stopped":
			h.timeOnWake = time.Now()
			go h.playSound(h.config.ActivitySettings.RecognitionStop)
		case "transcript":
			h.timeOnWake = time.Now()
		case "synthesize":
			go h.syntesize(event.Data["text"].(string))
			h.hasResponded = true
			h.isAwake = false
		}
	}
}

func (h *Handler) playSound(mediaFile string) {
	source := fmt.Sprintf("media-source://media_source/local/%s/%s", h.config.ActivitySettings.MediaFolder, mediaFile)
	payload := PlayMediaPayload{
		EntityID: h.config.Homeassistant.TargetMediaPlayer,
		Content:  source,
		Type:     "audio/mpeg",
	}
	data, err := json.Marshal(&payload)
	if err != nil {
		log.Errorf("failed to create voice notification payload: %v", err)
		return
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/api/services/media_player/play_media", h.config.Homeassistant.Host), bytes.NewBuffer(data))
	if err != nil {
		log.Errorf("failed to create voice notification request: %v", err)
		return
	}

	req.Header.Set("authorization", fmt.Sprintf("Bearer %s", h.config.Homeassistant.Token))
	req.Header.Set("content-type", "application/json")

	res, err := h.httpClient.Do(req)
	if err != nil {
		log.Errorf("failed to make voice notification request: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode > 299 {
		out, _ := io.ReadAll(res.Body)
		log.Errorf("unexpected response from hass [%d]: %s", res.StatusCode, string(out))
	}
}

func trimRawText(rawText string) string {
	if len(rawText) <= 500 {
		return rawText
	}
	cut := 500
	for i := len(rawText) - 1; i >= 400; i-- {
		if i > 500 {
			continue
		}
		if rawText[i] == '.' || rawText[i] == '?' || rawText[i] == '!' {
			cut = i
			break
		}
	}

	trimString := rawText[0 : cut+1]
	return trimString + " This response was trimmed."
}

func inTimeSpan(start, end, check time.Time) bool {
	if start.Before(end) {
		return !check.Before(start) && !check.After(end)
	}
	if start.Equal(end) {
		return check.Equal(start)
	}
	return !start.After(check) || !end.Before(check)
}

func (h *Handler) syntesize(rawText string) {
	text := strings.NewReplacer("*", "", "&", "", "[", "", "]", "", "(", "", ")", "", "{", "", "}", "", "\n", "", "\r", "").Replace(rawText)

	text = trimRawText(text)
	log.Debugf("sending payload to tts engine: %s", text)

	adjustedVolume := h.config.Tts.VolumeLevel
	start, _ := time.Parse("15:04", "20:00")
	end, _ := time.Parse("15:04", "08:00")
	if inTimeSpan(start, end, time.Now()) {
		adjustedVolume = adjustedVolume * 0.85
	}

	payload := TTSPayload{
		EntityID:    h.config.Homeassistant.TargetMediaPlayer,
		Platform:    h.config.Tts.TtsPlatform,
		Voice:       h.config.Tts.Voice,
		Announce:    h.config.Tts.Announce,
		VolumeLevel: adjustedVolume,
		Message:     text,
	}

	data, err := json.Marshal(&payload)
	if err != nil {
		log.Errorf("failed to create tts notification payload: %v", err)
		return
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/api/services/chime_tts/say", h.config.Homeassistant.Host), bytes.NewBuffer(data))
	if err != nil {
		log.Errorf("failed to create tts notification request: %v", err)
		return
	}

	req.Header.Set("authorization", fmt.Sprintf("Bearer %s", h.config.Homeassistant.Token))
	req.Header.Set("content-type", "application/json")

	h.httpClient.Timeout = 10 * time.Second
	defer func() {
		h.httpClient.Timeout = 5 * time.Second
	}()
	res, err := h.httpClient.Do(req)

	if err != nil {
		log.Errorf("failed to make tts notification request: %v", err)
		return
	}

	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode > 299 {
		data, _ := io.ReadAll(res.Body)
		log.Errorf("unexpected response from HA %d: %s", res.StatusCode, string(data))
	}
}
