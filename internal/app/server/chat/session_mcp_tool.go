package chat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	user_config "xiaozhi-esp32-server-golang/internal/domain/config"
	config_types "xiaozhi-esp32-server-golang/internal/domain/config/types"
	llm_memory "xiaozhi-esp32-server-golang/internal/domain/memory/llm_memory"
	"xiaozhi-esp32-server-golang/internal/domain/rag"
	log "xiaozhi-esp32-server-golang/logger"

	"github.com/spf13/viper"
)

// This file handles tool calls for local mcp tool and session binding

// Music search API response structure
type MusicSearchResponse struct {
	Data  []MusicItem `json:"data"`
	Code  int         `json:"code"`
	Error string      `json:"error"`
}

type MusicItem struct {
	Type   string `json:"type"`
	Link   string `json:"link"`
	SongID string `json:"songid"`
	Title  string `json:"title"`
	Author string `json:"author"`
	LRC    bool   `json:"lrc"`
	URL    string `json:"url"`
	Pic    string `json:"pic"`
}

// Global HTTP client
var (
	httpClient     *http.Client
	httpClientOnce sync.Once
)

// Get an HTTP client configured with a connection pool
func getHTTPClient() *http.Client {
	httpClientOnce.Do(func() {
		transport := &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			MaxIdleConns:          100,
			MaxIdleConnsPerHost:   10,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		}
		httpClient = &http.Client{
			Transport: transport,
			Timeout:   30 * time.Second,
		}
	})
	return httpClient
}

// Close session
func (c *ChatManager) LocalMcpCloseChat() error {
	return c.ExitChat()
}

// Clear conversation history
func (c *ChatManager) LocalMcpClearHistory() error {
	llm_memory.Get().ResetMemory(c.ctx, c.DeviceID)
	return nil
}

type PlayMusicParams struct {
	Name string `json:"name,omitempty" description:"Name of the music"`
	// Welcome string `json:"welcome" description:"Searching for music takes too long, a reminder to appease the user" required:"true"`
}

type MusicPlaybackControlParams struct {
	Action string `json:"action" description:"Control action: resume (continue playing/resume/keep listening), pause, stop, prev, next, play_playlist (play playlist/play songs in playlist), enqueue_current; play and continue are also normalized to resume" required:"true"`
}

type MusicPlaybackControlResult struct {
	Action          string `json:"action"`
	Status          string `json:"status"`
	CurrentTitle    string `json:"current_title,omitempty"`
	CurrentIndex    int    `json:"current_index"`
	PlaylistLength  int    `json:"playlist_length"`
	CurrentSource   string `json:"current_source,omitempty"`
	PositionMs      int64  `json:"position_ms"`
	AddedTitle      string `json:"added_title,omitempty"`
	SilenceResponse bool   `json:"silence_response"`
}

// play music
func (c *ChatManager) LocalMcpPlayMusic(ctx context.Context, musicParams *PlayMusicParams) error {
	musicName := musicParams.Name
	// welcome := musicParams.Welcome
	welcome := ""
	log.Infof("Search music: %s, welcome: %s", musicName, welcome)
	var musicURL, realMusicName string
	var wg sync.WaitGroup
	var ierr error
	wg.Add(2)
	go func() {
		defer wg.Done()
		// Here you can get the music URL based on the music name
		// The implementation is currently simplified, assuming that musicName is the URL or obtained from the configuration
		musicURL, realMusicName, ierr = getMusicURL(musicName)
		if ierr != nil {
			log.Errorf("Failed to get music URL: %v", ierr)
			return
		}

		return
	}()
	go func() {
		defer wg.Done()
		// c.session.ttsManager.handleTts(ctx, common.LLMResponseStruct{Text: welcome, IsStart: true})
	}()

	wg.Wait()

	if musicURL == "" {
		log.Errorf("Music not found: %s", musicName)
		return fmt.Errorf("Music not found: %s", musicName)
	}

	log.Infof("Music found: %s, URL: %s", realMusicName, musicURL)

	return nil
}

// LocalMcpSwitchDeviceRole switches device roles by role name (supports fuzzy matching)
func (c *ChatManager) LocalMcpSwitchDeviceRole(ctx context.Context, roleName string) (string, error) {
	roleName = strings.TrimSpace(roleName)
	if roleName == "" {
		return "", fmt.Errorf("role_name cannot be empty")
	}

	configProvider, err := user_config.GetProvider(viper.GetString("config_provider.type"))
	if err != nil {
		return "", fmt.Errorf("Failed to get configuration provider: %w", err)
	}

	matchedRoleName, err := configProvider.SwitchDeviceRoleByName(ctx, c.DeviceID, roleName)
	if err != nil {
		return "", err
	}

	if err := c.ReloadDeviceConfig(ctx); err != nil {
		return "", fmt.Errorf("Role switched, but refresh session configuration failed: %w", err)
	}

	log.Infof("Device %s switched roles successfully, request=%s, match=%s", c.DeviceID, roleName, matchedRoleName)
	return matchedRoleName, nil
}

// LocalMcpRestoreDeviceDefaultRole restores the device default role
func (c *ChatManager) LocalMcpRestoreDeviceDefaultRole(ctx context.Context) error {
	configProvider, err := user_config.GetProvider(viper.GetString("config_provider.type"))
	if err != nil {
		return fmt.Errorf("Failed to get configuration provider: %w", err)
	}

	if err := configProvider.RestoreDeviceDefaultRole(ctx, c.DeviceID); err != nil {
		return err
	}

	if err := c.ReloadDeviceConfig(ctx); err != nil {
		return fmt.Errorf("Default role restored, but refresh session configuration failed: %w", err)
	}

	log.Infof("Device %s restored to default role successfully", c.DeviceID)
	return nil
}

// LocalMcpSearchKnowledge retrieves the knowledge base bound to the current agent
func (c *ChatManager) LocalMcpSearchKnowledge(ctx context.Context, query string, topK int, knowledgeBaseIDs []uint) ([]config_types.KnowledgeSearchHit, error) {
	if c == nil || c.clientState == nil {
		return nil, fmt.Errorf("Session state is unavailable")
	}
	return rag.Search(ctx, query, topK, c.clientState.DeviceConfig.KnowledgeBases, knowledgeBaseIDs)
}

func (c *ChatManager) LocalMcpControlMusicPlayback(ctx context.Context, params *MusicPlaybackControlParams) (*MusicPlaybackControlResult, error) {
	if c == nil {
		return nil, fmt.Errorf("chat manager is not available")
	}
	return controlMusicPlayback(ctx, c.GetSession(), params)
}

func controlMusicPlayback(ctx context.Context, session *ChatSession, params *MusicPlaybackControlParams) (*MusicPlaybackControlResult, error) {
	if session == nil || session.mediaPlayer == nil {
		return nil, fmt.Errorf("Media player is not available")
	}
	if params == nil {
		return nil, fmt.Errorf("Control parameters cannot be empty")
	}

	action := normalizeMusicPlaybackAction(params.Action)
	if action == "" {
		return nil, fmt.Errorf("Unsupported control action: %s", params.Action)
	}

	result := &MusicPlaybackControlResult{
		Action:          action,
		SilenceResponse: true,
	}

	switch action {
	case "resume":
		if err := session.mediaPlayer.Play(ctx); err != nil {
			return nil, err
		}
	case "pause":
		if err := session.mediaPlayer.Pause(); err != nil {
			return nil, err
		}
		flushQueuedMediaAudio(session, action)
	case "stop":
		if err := session.mediaPlayer.Stop(ctx); err != nil {
			return nil, err
		}
		flushQueuedMediaAudio(session, action)
	case "prev":
		if err := session.mediaPlayer.Prev(ctx); err != nil {
			return nil, err
		}
	case "next":
		if err := session.mediaPlayer.Next(ctx); err != nil {
			return nil, err
		}
	case "play_playlist":
		if err := session.mediaPlayer.PlayAgentPlaylist(ctx); err != nil {
			return nil, err
		}
	case "enqueue_current":
		appendResult, err := session.mediaPlayer.AppendCurrentToPlaylist()
		if err != nil {
			return nil, err
		}
		result.AddedTitle = appendResult.AddedTitle
		if _, err := session.mediaPlayer.ResumeIfInterruptedPause(); err != nil {
			log.Warnf("enqueue_current automatic resume playback failed: %v", err)
		}
	}

	state := session.mediaPlayer.GetState()
	result.Status = state.Status.String()
	result.CurrentTitle = state.CurrentTitle
	result.CurrentIndex = state.CurrentIndex
	result.PlaylistLength = len(state.Playlist)
	result.CurrentSource = string(state.CurrentSourceType)
	result.PositionMs = state.PositionMs
	return result, nil
}

func flushQueuedMediaAudio(session *ChatSession, action string) {
	if session == nil || session.ttsManager == nil {
		return
	}

	interruptCtx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	if err := session.ttsManager.InterruptAndClearQueueSync(interruptCtx); err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			log.Warnf("Timeout for clearing the audio sending queue after media control: action=%s", action)
			return
		}
		if !errors.Is(err, context.Canceled) {
			log.Warnf("Failed to clear audio sending queue after media control: action=%s, err=%v", action, err)
		}
	}
}

func normalizeMusicPlaybackAction(action string) string {
	switch strings.ToLower(strings.TrimSpace(action)) {
	case "play", "resume", "continue":
		return "resume"
	case "pause":
		return "pause"
	case "stop":
		return "stop"
	case "prev", "previous":
		return "prev"
	case "next":
		return "next"
	case "play_playlist", "play_agent_playlist", "play_playlist_songs", "playlist":
		return "play_playlist"
	case "enqueue_current", "append_current", "add_current_to_playlist":
		return "enqueue_current"
	default:
		return ""
	}
}

// searchMusicFromAPI Search music from API
func getMusicURL(musicName string) (string, string, error) {
	client := getHTTPClient()

	// Build request body
	data := fmt.Sprintf("input=%s&filter=name&type=migu&page=1",
		url.QueryEscape(musicName))

	req, err := http.NewRequest("POST", "https://music.txqq.pro/",
		strings.NewReader(data))
	if err != nil {
		return "", "", fmt.Errorf("Create request failed: %v", err)
	}

	// Set request headers to simulate browser requests
	req.Header.Set("Accept", "application/json, text/javascript, */*; q=0.01")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	req.Header.Set("Origin", "https://music.txqq.pro")
	req.Header.Set("Pragma", "no-cache")
	req.Header.Set("Referer", "https://music.txqq.pro/")
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/138.0.0.0 Safari/537.36")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("sec-ch-ua", `"Not)A;Brand";v="8", "Chromium";v="138", "Google Chrome";v="138"`)
	req.Header.Set("sec-ch-ua-mobile", "?0")
	req.Header.Set("sec-ch-ua-platform", `"Windows"`)

	// Set timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	req = req.WithContext(ctx)

	resp, err := client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("API request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("API request failed, status code: %d", resp.StatusCode)
	}

	// Parse response
	var searchResp MusicSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return "", "", fmt.Errorf("Failed to parse response: %v", err)
	}

	if searchResp.Code != 200 {
		return "", "", fmt.Errorf("API returned error: %s", searchResp.Error)
	}

	if len(searchResp.Data) == 0 {
		return "", "", fmt.Errorf("Music not found: %s", musicName)
	}
	musicItem := searchResp.Data[0]
	// Returns the URL of the first search result
	return musicItem.URL, musicItem.Title, nil
}
