package chat

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	types_audio "xiaozhi-esp32-server-golang/internal/data/audio"
	mcp_domain "xiaozhi-esp32-server-golang/internal/domain/mcp"
	"xiaozhi-esp32-server-golang/internal/domain/play_music"
	"xiaozhi-esp32-server-golang/internal/util"
	log "xiaozhi-esp32-server-golang/logger"

	mcpclient "github.com/mark3labs/mcp-go/client"
	mcp_go "github.com/mark3labs/mcp-go/mcp"
)

const defaultMediaPlayerPageSize = 16 * 1024

type MediaSourceType string

const (
	MediaSourceTypeMCPResource MediaSourceType = "mcp_resource"
	MediaSourceTypeHTTPURL     MediaSourceType = "http_url"
	MediaSourceTypeLocalFile   MediaSourceType = "local_file"
	MediaSourceTypeInlineAudio MediaSourceType = "inline_audio"
)

type mediaPlaybackMode string

const (
	mediaPlaybackModeStandalone    mediaPlaybackMode = "standalone"
	mediaPlaybackModeAgentPlaylist mediaPlaybackMode = "agent_playlist"
)

type mediaPauseReason string

const (
	mediaPauseReasonNone      mediaPauseReason = ""
	mediaPauseReasonUser      mediaPauseReason = "user"
	mediaPauseReasonInterrupt mediaPauseReason = "interrupt"
)

type mediaRecoveryTrigger string

const (
	mediaRecoveryTriggerUser   mediaRecoveryTrigger = "user"
	mediaRecoveryTriggerAttach mediaRecoveryTrigger = "attach"
)

type MediaSourceDescriptor struct {
	ID         string
	Title      string
	MIMEType   string
	SourceType MediaSourceType
	Meta       map[string]string

	MCP    *MCPMediaSource
	HTTP   *HTTPMediaSource
	Local  *LocalMediaSource
	Inline *InlineAudioSource
}

type MCPMediaSource struct {
	ServerName       string
	EndpointSnapshot string
	ToolName         string
	ResourceURI      string
	DirectAudioURL   string
	Description      string
	ReadArgs         map[string]any
	PageSize         int
	Client           *mcpclient.Client
}

type HTTPMediaSource struct {
	URL string
}

type LocalMediaSource struct {
	Path string
}

type InlineAudioSource struct {
	Data []byte
}

type MediaPlaylistItem struct {
	ID      string
	Source  MediaSourceDescriptor
	AddedAt int64
}

type MediaPlayerState struct {
	Status            play_music.PlaybackStatus
	Playlist          []MediaPlaylistItem
	CurrentIndex      int
	CurrentTitle      string
	CurrentSourceType MediaSourceType
	PositionMs        int64
	UpdatedAt         int64
	ErrMsg            string
}

type PlaylistAppendResult struct {
	AddedTitle      string
	PlaylistLength  int
	CurrentIndex    int
	CurrentStatus   play_music.PlaybackStatus
	CurrentSource   MediaSourceType
	CurrentPosition int64
}

type MediaPlaybackHandle struct {
	done <-chan struct{}
}

type mediaPlaybackAudioConfig struct {
	SampleRate    int
	FrameDuration int
}

type activeMediaPlayback struct {
	ctx      context.Context
	cancel   context.CancelFunc
	done     chan struct{}
	pauseMu  sync.Mutex
	paused   bool
	resumeCh chan struct{}

	exclusiveHeld bool
}

type mediaOutputBridge interface {
	BeginExclusiveMediaPlayback(ctx context.Context) error
	EndExclusiveMediaPlayback()
	SendSentenceStart(ctx context.Context, text string, onError func(error)) error
	SendSentenceEnd(ctx context.Context, text string, onError func(error), onEnd func(error)) error
	SendAudioFrame(ctx context.Context, frame []byte, onError func(error)) error
}

type mediaSessionAttachment struct {
	session *ChatSession
	bridge  mediaOutputBridge
}

type sessionMediaOutputBridge struct {
	session *ChatSession
}

type agentMediaPlaylist struct {
	mu        sync.RWMutex
	items     []MediaPlaylistItem
	updatedAt int64
}

type mediaPlaybackCoordinator struct {
	ctx context.Context

	mu             sync.Mutex
	agentPlaylists map[string]*agentMediaPlaylist
	deviceRuntimes map[string]*deviceMediaRuntime
}

type deviceMediaRuntime struct {
	coordinator *mediaPlaybackCoordinator
	deviceID    string

	mu                  sync.RWMutex
	state               MediaPlayerState
	mode                mediaPlaybackMode
	playlistAgentID     string
	standaloneItems     []MediaPlaylistItem
	playbackIndex       int
	audioConfig         mediaPlaybackAudioConfig
	currentSource       *MediaSourceDescriptor
	active              *activeMediaPlayback
	pauseReason         mediaPauseReason
	attachment          *mediaSessionAttachment
	attachmentWaitCh    chan struct{}
	resumeOnAttach      bool
	exclusiveAttachment *mediaSessionAttachment
}

type SessionMediaPlayer struct {
	session     *ChatSession
	coordinator *mediaPlaybackCoordinator
	runtime     *deviceMediaRuntime
}

var sharedMediaPlaybackCoordinator = newMediaPlaybackCoordinator()
var closedMediaPlaybackDoneCh = func() <-chan struct{} {
	ch := make(chan struct{})
	close(ch)
	return ch
}()

func newMediaPlaybackHandle(active *activeMediaPlayback) *MediaPlaybackHandle {
	if active == nil || active.done == nil {
		return &MediaPlaybackHandle{done: closedMediaPlaybackDoneCh}
	}
	return &MediaPlaybackHandle{done: active.done}
}

func (h *MediaPlaybackHandle) Done() <-chan struct{} {
	if h == nil || h.done == nil {
		return closedMediaPlaybackDoneCh
	}
	return h.done
}

func (h *MediaPlaybackHandle) Wait(ctx context.Context) error {
	if h == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}

	select {
	case <-h.Done():
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func newActiveMediaPlayback(parent context.Context) *activeMediaPlayback {
	if parent == nil {
		parent = context.Background()
	}
	ctx, cancel := context.WithCancel(parent)
	resumeCh := make(chan struct{})
	close(resumeCh)
	return &activeMediaPlayback{
		ctx:      ctx,
		cancel:   cancel,
		done:     make(chan struct{}),
		resumeCh: resumeCh,
	}
}

func (a *activeMediaPlayback) closeDone() {
	select {
	case <-a.done:
	default:
		close(a.done)
	}
}

func (a *activeMediaPlayback) setPaused(paused bool) bool {
	a.pauseMu.Lock()
	defer a.pauseMu.Unlock()

	if a.paused == paused {
		return false
	}

	if paused {
		a.paused = true
		a.resumeCh = make(chan struct{})
		return true
	}

	a.paused = false
	close(a.resumeCh)
	return true
}

func (a *activeMediaPlayback) isPaused() bool {
	a.pauseMu.Lock()
	defer a.pauseMu.Unlock()
	return a.paused
}

func (a *activeMediaPlayback) hasExclusiveHeld() bool {
	a.pauseMu.Lock()
	defer a.pauseMu.Unlock()
	return a.exclusiveHeld
}

func (a *activeMediaPlayback) setExclusiveHeld(held bool) bool {
	a.pauseMu.Lock()
	defer a.pauseMu.Unlock()

	if a.exclusiveHeld == held {
		return false
	}
	a.exclusiveHeld = held
	return true
}

func (a *activeMediaPlayback) waitIfPaused(ctx context.Context) error {
	for {
		a.pauseMu.Lock()
		paused := a.paused
		resumeCh := a.resumeCh
		a.pauseMu.Unlock()

		if !paused {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-resumeCh:
		}
	}
}

func newMediaPlaybackCoordinator() *mediaPlaybackCoordinator {
	return &mediaPlaybackCoordinator{
		ctx:            context.Background(),
		agentPlaylists: make(map[string]*agentMediaPlaylist),
		deviceRuntimes: make(map[string]*deviceMediaRuntime),
	}
}

func newDeviceMediaRuntime(coordinator *mediaPlaybackCoordinator, deviceID string) *deviceMediaRuntime {
	return &deviceMediaRuntime{
		coordinator:      coordinator,
		deviceID:         deviceID,
		playbackIndex:    -1,
		attachmentWaitCh: make(chan struct{}),
		state: MediaPlayerState{
			Status:       play_music.StatusIdle,
			CurrentIndex: -1,
			UpdatedAt:    time.Now().UnixMilli(),
		},
	}
}

func (c *mediaPlaybackCoordinator) getOrCreateAgentPlaylist(agentID string) *agentMediaPlaylist {
	c.mu.Lock()
	defer c.mu.Unlock()

	playlist, ok := c.agentPlaylists[agentID]
	if ok {
		return playlist
	}

	playlist = &agentMediaPlaylist{}
	c.agentPlaylists[agentID] = playlist
	return playlist
}

func (c *mediaPlaybackCoordinator) getOrCreateRuntime(deviceID string) *deviceMediaRuntime {
	c.mu.Lock()
	defer c.mu.Unlock()

	runtime, ok := c.deviceRuntimes[deviceID]
	if ok {
		return runtime
	}

	runtime = newDeviceMediaRuntime(c, deviceID)
	c.deviceRuntimes[deviceID] = runtime
	return runtime
}

func (c *mediaPlaybackCoordinator) snapshotAgentPlaylist(agentID string) []MediaPlaylistItem {
	agentID = strings.TrimSpace(agentID)
	if agentID == "" {
		return nil
	}

	playlist := c.getOrCreateAgentPlaylist(agentID)
	playlist.mu.RLock()
	defer playlist.mu.RUnlock()
	return cloneMediaPlaylistItems(playlist.items)
}

func (c *mediaPlaybackCoordinator) appendToAgentPlaylist(agentID string, source MediaSourceDescriptor) (MediaPlaylistItem, int, []MediaPlaylistItem, error) {
	agentID = strings.TrimSpace(agentID)
	if agentID == "" {
		return MediaPlaylistItem{}, -1, nil, fmt.Errorf("agentID cannot be empty")
	}

	playlist := c.getOrCreateAgentPlaylist(agentID)
	nowMs := time.Now().UnixMilli()
	cloned := cloneMediaSourceDescriptor(source)
	if cloned.ID == "" {
		cloned.ID = fmt.Sprintf("media_%d", time.Now().UnixNano())
	}
	if cloned.Title == "" {
		cloned.Title = deriveMediaTitle(cloned)
	}

	item := MediaPlaylistItem{
		ID:      cloned.ID,
		Source:  cloned,
		AddedAt: nowMs,
	}

	playlist.mu.Lock()
	defer playlist.mu.Unlock()

	playlist.items = append(playlist.items, item)
	playlist.updatedAt = nowMs

	return cloneMediaPlaylistItem(item), len(playlist.items) - 1, cloneMediaPlaylistItems(playlist.items), nil
}

func (b *sessionMediaOutputBridge) BeginExclusiveMediaPlayback(ctx context.Context) error {
	if b == nil || b.session == nil || b.session.ttsManager == nil {
		return fmt.Errorf("Media output bridge not initialized")
	}
	return b.session.ttsManager.BeginExclusiveMediaPlayback(ctx)
}

func (b *sessionMediaOutputBridge) EndExclusiveMediaPlayback() {
	if b == nil || b.session == nil || b.session.ttsManager == nil {
		return
	}
	b.session.ttsManager.EndExclusiveMediaPlayback()
}

func (b *sessionMediaOutputBridge) SendSentenceStart(ctx context.Context, text string, onError func(error)) error {
	if b == nil || b.session == nil || b.session.ttsManager == nil {
		return fmt.Errorf("Media output bridge not initialized")
	}
	return b.session.ttsManager.EnqueueMediaSentenceStart(ctx, text, onError)
}

func (b *sessionMediaOutputBridge) SendSentenceEnd(ctx context.Context, text string, onError func(error), onEnd func(error)) error {
	if b == nil || b.session == nil || b.session.ttsManager == nil {
		return fmt.Errorf("Media output bridge not initialized")
	}
	return b.session.ttsManager.EnqueueMediaSentenceEnd(ctx, text, onError, onEnd)
}

func (b *sessionMediaOutputBridge) SendAudioFrame(ctx context.Context, frame []byte, onError func(error)) error {
	if b == nil || b.session == nil || b.session.ttsManager == nil {
		return fmt.Errorf("Media output bridge not initialized")
	}
	return b.session.ttsManager.EnqueueMediaFrame(ctx, frame, onError)
}

func NewSessionMediaPlayer(session *ChatSession) *SessionMediaPlayer {
	player := &SessionMediaPlayer{
		session:     session,
		coordinator: sharedMediaPlaybackCoordinator,
	}

	if session != nil && session.clientState != nil {
		player.runtime = player.coordinator.getOrCreateRuntime(session.clientState.DeviceID)
	}

	return player
}

func (p *SessionMediaPlayer) runtimeOrErr() (*deviceMediaRuntime, error) {
	if p == nil || p.runtime == nil {
		return nil, fmt.Errorf("media player not initialized")
	}
	return p.runtime, nil
}

func (p *SessionMediaPlayer) audioConfig() mediaPlaybackAudioConfig {
	cfg := mediaPlaybackAudioConfig{
		SampleRate:    types_audio.SampleRate,
		FrameDuration: types_audio.FrameDuration,
	}
	if p == nil || p.session == nil || p.session.clientState == nil {
		return cfg
	}
	if p.session.clientState.OutputAudioFormat.SampleRate > 0 {
		cfg.SampleRate = p.session.clientState.OutputAudioFormat.SampleRate
	}
	if p.session.clientState.OutputAudioFormat.FrameDuration > 0 {
		cfg.FrameDuration = p.session.clientState.OutputAudioFormat.FrameDuration
	}
	return cfg
}

func (p *SessionMediaPlayer) currentAgentID() string {
	if p == nil || p.session == nil || p.session.clientState == nil {
		return ""
	}
	return strings.TrimSpace(p.session.clientState.AgentID)
}

func (p *SessionMediaPlayer) AttachSession() {
	runtime, err := p.runtimeOrErr()
	if err != nil {
		return
	}
	runtime.attachSession(p.session)
}

func (p *SessionMediaPlayer) DetachSession(preserve bool) {
	runtime, err := p.runtimeOrErr()
	if err != nil {
		return
	}
	runtime.detachSession(p.session, preserve)
}

func (p *SessionMediaPlayer) PlaySource(ctx context.Context, source MediaSourceDescriptor) error {
	_, err := p.PlaySourceWithHandle(ctx, source)
	return err
}

func (p *SessionMediaPlayer) ReplacePlaylistAndPlay(ctx context.Context, sources []MediaSourceDescriptor, startIndex int) error {
	_, err := p.ReplacePlaylistAndPlayWithHandle(ctx, sources, startIndex)
	return err
}

func (p *SessionMediaPlayer) PlaySourceWithHandle(ctx context.Context, source MediaSourceDescriptor) (*MediaPlaybackHandle, error) {
	return p.ReplacePlaylistAndPlayWithHandle(ctx, []MediaSourceDescriptor{source}, 0)
}

func (p *SessionMediaPlayer) ReplacePlaylistAndPlayWithHandle(ctx context.Context, sources []MediaSourceDescriptor, startIndex int) (*MediaPlaybackHandle, error) {
	runtime, err := p.runtimeOrErr()
	if err != nil {
		return nil, err
	}
	return runtime.ReplaceStandaloneQueueAndPlayWithHandle(ctx, sources, startIndex, p.audioConfig())
}

func (p *SessionMediaPlayer) Pause() error {
	runtime, err := p.runtimeOrErr()
	if err != nil {
		return err
	}
	return runtime.Pause()
}

func (p *SessionMediaPlayer) Play(ctx context.Context) error {
	runtime, err := p.runtimeOrErr()
	if err != nil {
		return err
	}
	return runtime.RecoverPlayback(ctx, p.audioConfig(), mediaRecoveryTriggerUser)
}

func (p *SessionMediaPlayer) PlayAgentPlaylist(ctx context.Context) error {
	runtime, err := p.runtimeOrErr()
	if err != nil {
		return err
	}
	return runtime.PlayAgentPlaylist(ctx, p.currentAgentID(), p.audioConfig())
}

func (p *SessionMediaPlayer) Resume() error {
	runtime, err := p.runtimeOrErr()
	if err != nil {
		return err
	}
	return runtime.Resume()
}

func (p *SessionMediaPlayer) ResumeIfInterruptedPause() (bool, error) {
	runtime, err := p.runtimeOrErr()
	if err != nil {
		return false, err
	}
	return runtime.ResumeIfInterruptedPause()
}

func (p *SessionMediaPlayer) Stop(ctx context.Context) error {
	runtime, err := p.runtimeOrErr()
	if err != nil {
		return err
	}
	return runtime.Stop(ctx)
}

func (p *SessionMediaPlayer) Suspend() error {
	runtime, err := p.runtimeOrErr()
	if err != nil {
		return err
	}
	return runtime.Suspend()
}

func (p *SessionMediaPlayer) Next(ctx context.Context) error {
	runtime, err := p.runtimeOrErr()
	if err != nil {
		return err
	}
	return runtime.Next(ctx, p.audioConfig())
}

func (p *SessionMediaPlayer) Prev(ctx context.Context) error {
	runtime, err := p.runtimeOrErr()
	if err != nil {
		return err
	}
	return runtime.Prev(ctx, p.audioConfig())
}

func (p *SessionMediaPlayer) GetState() MediaPlayerState {
	runtime, err := p.runtimeOrErr()
	if err != nil {
		return MediaPlayerState{
			Status:       play_music.StatusIdle,
			CurrentIndex: -1,
			UpdatedAt:    time.Now().UnixMilli(),
		}
	}
	return runtime.GetState()
}

func (p *SessionMediaPlayer) HasRealtimeMcpAudioControlContext() bool {
	runtime, err := p.runtimeOrErr()
	if err != nil {
		return false
	}
	return runtime.hasRealtimeMcpAudioControlContext()
}

func (p *SessionMediaPlayer) ShouldGateRealtimeMcpAudioASR() bool {
	runtime, err := p.runtimeOrErr()
	if err != nil {
		return false
	}
	return runtime.shouldGateRealtimeMcpAudioASR()
}

func (p *SessionMediaPlayer) AppendCurrentToPlaylist() (*PlaylistAppendResult, error) {
	runtime, err := p.runtimeOrErr()
	if err != nil {
		return nil, err
	}
	return runtime.AppendCurrentToAgentPlaylist(p.currentAgentID())
}

func (r *deviceMediaRuntime) attachSession(session *ChatSession) {
	if r == nil || session == nil || session.ttsManager == nil || session.serverTransport == nil {
		return
	}

	attachment := &mediaSessionAttachment{
		session: session,
		bridge:  &sessionMediaOutputBridge{session: session},
	}

	var shouldResume bool

	r.mu.Lock()
	if r.attachment != nil && r.attachment.session == session {
		r.mu.Unlock()
		return
	}
	r.attachment = attachment
	r.notifyAttachmentChangedLocked()
	shouldResume = r.resumeOnAttach
	r.mu.Unlock()

	log.Infof("Device %s media playback attachment is bound", r.deviceID)

	if shouldResume {
		go func() {
			if err := r.RecoverPlayback(context.Background(), r.currentAudioConfig(), mediaRecoveryTriggerAttach); err != nil && !errors.Is(err, context.Canceled) {
				log.Warnf("Failed to resume media playback on device %s: %v", r.deviceID, err)
			}
		}()
	}
}

func (r *deviceMediaRuntime) detachSession(session *ChatSession, preserve bool) {
	if r == nil || session == nil {
		return
	}

	var releaseAttachment *mediaSessionAttachment

	r.mu.Lock()
	currentAttachment := r.attachment
	if currentAttachment == nil || currentAttachment.session != session {
		r.mu.Unlock()
		return
	}

	wasPlaying := r.state.Status == play_music.StatusPlaying
	if preserve && r.active != nil && wasPlaying {
		r.active.setPaused(true)
		r.state.Status = play_music.StatusPaused
		r.state.UpdatedAt = time.Now().UnixMilli()
		r.resumeOnAttach = true
	}

	r.attachment = nil
	if r.exclusiveAttachment == currentAttachment {
		releaseAttachment = currentAttachment
		r.exclusiveAttachment = nil
		if r.active != nil {
			r.active.setExclusiveHeld(false)
		}
	}
	r.notifyAttachmentChangedLocked()
	r.mu.Unlock()

	log.Infof("Device %s media playback attachment has been unbound, preserve=%v", r.deviceID, preserve)

	if releaseAttachment != nil {
		releaseAttachment.bridge.EndExclusiveMediaPlayback()
	}
}

func (r *deviceMediaRuntime) notifyAttachmentChangedLocked() {
	if r.attachmentWaitCh != nil {
		close(r.attachmentWaitCh)
	}
	r.attachmentWaitCh = make(chan struct{})
}

func (r *deviceMediaRuntime) ReplaceStandaloneQueueAndPlay(ctx context.Context, sources []MediaSourceDescriptor, startIndex int, cfg mediaPlaybackAudioConfig) error {
	_, err := r.ReplaceStandaloneQueueAndPlayWithHandle(ctx, sources, startIndex, cfg)
	return err
}

func (r *deviceMediaRuntime) ReplaceStandaloneQueueAndPlayWithHandle(ctx context.Context, sources []MediaSourceDescriptor, startIndex int, cfg mediaPlaybackAudioConfig) (*MediaPlaybackHandle, error) {
	if r == nil {
		return nil, fmt.Errorf("Media player not initialized")
	}
	if len(sources) == 0 {
		return nil, r.Stop(ctx)
	}
	if startIndex < 0 || startIndex >= len(sources) {
		return nil, fmt.Errorf("Invalid playback start index: %d", startIndex)
	}

	items := buildMediaPlaylistItems(sources)
	active := newActiveMediaPlayback(r.coordinator.ctx)

	r.mu.Lock()
	oldActive, oldExclusive := r.replaceActivePlaybackLocked(active)
	r.mode = mediaPlaybackModeStandalone
	r.playlistAgentID = ""
	r.standaloneItems = cloneMediaPlaylistItems(items)
	r.playbackIndex = startIndex
	r.audioConfig = normalizeMediaPlaybackAudioConfig(cfg, r.audioConfig)
	currentSource := cloneMediaSourceDescriptor(items[startIndex].Source)
	r.currentSource = &currentSource
	r.pauseReason = mediaPauseReasonNone
	r.resumeOnAttach = false
	r.state.Status = play_music.StatusPlaying
	r.state.Playlist = nil
	r.state.CurrentIndex = -1
	r.state.CurrentTitle = items[startIndex].Source.Title
	r.state.CurrentSourceType = items[startIndex].Source.SourceType
	r.state.PositionMs = 0
	r.state.ErrMsg = ""
	r.state.UpdatedAt = time.Now().UnixMilli()
	r.mu.Unlock()

	log.Infof("Device %s starts standalone media playback, source=%s, title=%s", r.deviceID, items[startIndex].Source.SourceType, items[startIndex].Source.Title)

	releaseMediaAttachment(oldExclusive)
	stopOldPlayback(oldActive)

	go r.runPlayback(active)
	return newMediaPlaybackHandle(active), nil
}

func (r *deviceMediaRuntime) PlayAgentPlaylistIndex(ctx context.Context, agentID string, startIndex int, cfg mediaPlaybackAudioConfig) error {
	if r == nil {
		return fmt.Errorf("Media player not initialized")
	}
	agentID = strings.TrimSpace(agentID)
	if agentID == "" {
		return fmt.Errorf("agentID cannot be empty")
	}

	snapshot := r.coordinator.snapshotAgentPlaylist(agentID)
	if len(snapshot) == 0 {
		return fmt.Errorf("Playlist is empty")
	}
	if startIndex < 0 || startIndex >= len(snapshot) {
		return fmt.Errorf("Invalid playback start index: %d", startIndex)
	}

	active := newActiveMediaPlayback(r.coordinator.ctx)

	r.mu.Lock()
	oldActive, oldExclusive := r.replaceActivePlaybackLocked(active)
	r.mode = mediaPlaybackModeAgentPlaylist
	r.playlistAgentID = agentID
	r.standaloneItems = nil
	r.playbackIndex = startIndex
	r.audioConfig = normalizeMediaPlaybackAudioConfig(cfg, r.audioConfig)
	currentSource := cloneMediaSourceDescriptor(snapshot[startIndex].Source)
	r.currentSource = &currentSource
	r.pauseReason = mediaPauseReasonNone
	r.resumeOnAttach = false
	r.state.Status = play_music.StatusPlaying
	r.state.Playlist = cloneMediaPlaylistItems(snapshot)
	r.state.CurrentIndex = startIndex
	r.state.CurrentTitle = snapshot[startIndex].Source.Title
	r.state.CurrentSourceType = snapshot[startIndex].Source.SourceType
	r.state.PositionMs = 0
	r.state.ErrMsg = ""
	r.state.UpdatedAt = time.Now().UnixMilli()
	r.mu.Unlock()

	log.Infof("Device %s starts intelligent playlist playback, agent=%s, index=%d, title=%s", r.deviceID, agentID, startIndex, snapshot[startIndex].Source.Title)

	releaseMediaAttachment(oldExclusive)
	stopOldPlayback(oldActive)

	go r.runPlayback(active)
	return nil
}

func (r *deviceMediaRuntime) PlayAgentPlaylist(ctx context.Context, agentID string, cfg mediaPlaybackAudioConfig) error {
	if r == nil {
		return fmt.Errorf("Media player not initialized")
	}

	agentID = strings.TrimSpace(agentID)
	if agentID == "" {
		return fmt.Errorf("agentID cannot be empty")
	}

	snapshot := r.coordinator.snapshotAgentPlaylist(agentID)
	if len(snapshot) == 0 {
		return fmt.Errorf("Playlist is empty")
	}

	startIndex := 0

	r.mu.RLock()
	if r.mode == mediaPlaybackModeAgentPlaylist && r.playlistAgentID == agentID && r.playbackIndex >= 0 && r.playbackIndex < len(snapshot) {
		startIndex = r.playbackIndex
	}
	r.mu.RUnlock()

	return r.PlayAgentPlaylistIndex(ctx, agentID, startIndex, cfg)
}

func (r *deviceMediaRuntime) Pause() error {
	return r.pausePlayback(true, mediaPauseReasonUser)
}

func (r *deviceMediaRuntime) Suspend() error {
	return r.pausePlayback(false, mediaPauseReasonInterrupt)
}

func (r *deviceMediaRuntime) Play(ctx context.Context, cfg mediaPlaybackAudioConfig) error {
	return r.RecoverPlayback(ctx, cfg, mediaRecoveryTriggerUser)
}

func (r *deviceMediaRuntime) RecoverPlayback(ctx context.Context, cfg mediaPlaybackAudioConfig, trigger mediaRecoveryTrigger) error {
	if r == nil {
		return fmt.Errorf("Media player not initialized")
	}

	if trigger == mediaRecoveryTriggerAttach && !r.shouldResumeOnAttach() {
		return nil
	}

	r.mu.RLock()
	active := r.active
	mode := r.mode
	agentID := r.playlistAgentID
	playbackIndex := r.playbackIndex
	currentSource := cloneCurrentSource(r.currentSource)
	r.mu.RUnlock()

	if active != nil {
		if active.isPaused() {
			return r.recoverActivePlayback(trigger)
		}
		if trigger == mediaRecoveryTriggerAttach {
			r.clearResumeOnAttach()
		}
		log.Infof("Device %s skipped media recovery, trigger=%s, reason=already_playing", r.deviceID, trigger)
		return nil
	}

	switch mode {
	case mediaPlaybackModeAgentPlaylist:
		snapshot := r.coordinator.snapshotAgentPlaylist(agentID)
		if len(snapshot) == 0 {
			return fmt.Errorf("Playlist is empty")
		}
		if playbackIndex < 0 || playbackIndex >= len(snapshot) {
			playbackIndex = 0
		}
		log.Infof("Device %s performing media recovery, trigger=%s, mode=agent_playlist, index=%d", r.deviceID, trigger, playbackIndex)
		return r.PlayAgentPlaylistIndex(ctx, agentID, playbackIndex, cfg)
	default:
		if currentSource == nil {
			return fmt.Errorf("There is currently no playable media")
		}
		log.Infof("Device %s performing media recovery, trigger=%s, mode=standalone, source=%s, title=%s", r.deviceID, trigger, currentSource.SourceType, currentSource.Title)
		return r.ReplaceStandaloneQueueAndPlay(ctx, []MediaSourceDescriptor{*currentSource}, 0, cfg)
	}
}

func (r *deviceMediaRuntime) Resume() error {
	return r.recoverActivePlayback(mediaRecoveryTriggerUser)
}

func (r *deviceMediaRuntime) recoverActivePlayback(trigger mediaRecoveryTrigger) error {
	if r == nil {
		return fmt.Errorf("Media player not initialized")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.active == nil {
		return fmt.Errorf("There is no media currently playing")
	}
	if r.attachment == nil {
		return fmt.Errorf("There are currently no playback channels available.")
	}
	if !r.active.setPaused(false) {
		r.resumeOnAttach = false
		log.Infof("Device %s skipped media recovery, trigger=%s, reason=already_playing", r.deviceID, trigger)
		return nil
	}
	r.pauseReason = mediaPauseReasonNone
	r.resumeOnAttach = false
	r.state.Status = play_music.StatusPlaying
	r.state.UpdatedAt = time.Now().UnixMilli()
	log.Infof("Device %s performs media recovery, trigger=%s, mode=resume_active", r.deviceID, trigger)
	return nil
}

func (r *deviceMediaRuntime) ResumeIfInterruptedPause() (bool, error) {
	if r == nil {
		return false, fmt.Errorf("Media player not initialized")
	}

	r.mu.RLock()
	shouldResume := r.active != nil && r.pauseReason == mediaPauseReasonInterrupt
	r.mu.RUnlock()

	if !shouldResume {
		return false, nil
	}

	if err := r.recoverActivePlayback(mediaRecoveryTriggerUser); err != nil {
		return false, err
	}
	return true, nil
}

func (r *deviceMediaRuntime) shouldResumeOnAttach() bool {
	if r == nil {
		return false
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.resumeOnAttach
}

func (r *deviceMediaRuntime) clearResumeOnAttach() {
	if r == nil {
		return
	}

	r.mu.Lock()
	r.resumeOnAttach = false
	r.mu.Unlock()
}

func (r *deviceMediaRuntime) Stop(ctx context.Context) error {
	if r == nil {
		return fmt.Errorf("Media player not initialized")
	}

	var (
		active          *activeMediaPlayback
		exclusiveAttach *mediaSessionAttachment
	)

	r.mu.Lock()
	active = r.active
	exclusiveAttach = r.exclusiveAttachment
	if active != nil {
		active.setExclusiveHeld(false)
	}
	r.active = nil
	r.pauseReason = mediaPauseReasonNone
	r.exclusiveAttachment = nil
	r.resumeOnAttach = false
	r.state.Status = play_music.StatusStopped
	r.state.PositionMs = 0
	r.state.ErrMsg = ""
	r.state.UpdatedAt = time.Now().UnixMilli()
	r.mu.Unlock()

	releaseMediaAttachment(exclusiveAttach)
	stopOldPlayback(active)
	return nil
}

func (r *deviceMediaRuntime) Next(ctx context.Context, cfg mediaPlaybackAudioConfig) error {
	return r.jumpToRelative(ctx, 1, cfg)
}

func (r *deviceMediaRuntime) Prev(ctx context.Context, cfg mediaPlaybackAudioConfig) error {
	return r.jumpToRelative(ctx, -1, cfg)
}

func (r *deviceMediaRuntime) GetState() MediaPlayerState {
	if r == nil {
		return MediaPlayerState{
			Status:       play_music.StatusIdle,
			CurrentIndex: -1,
			UpdatedAt:    time.Now().UnixMilli(),
		}
	}

	r.mu.Lock()
	r.refreshPublicStateLocked()
	state := r.state
	r.mu.Unlock()

	state.Playlist = cloneMediaPlaylistItems(state.Playlist)
	return state
}

func (r *deviceMediaRuntime) hasRealtimeMcpAudioControlContext() bool {
	canControl, _ := r.realtimeMcpAudioGateStatus()
	return canControl
}

func (r *deviceMediaRuntime) shouldGateRealtimeMcpAudioASR() bool {
	_, shouldGate := r.realtimeMcpAudioGateStatus()
	return shouldGate
}

func (r *deviceMediaRuntime) realtimeMcpAudioGateStatus() (bool, bool) {
	if r == nil {
		return false, false
	}

	r.mu.RLock()
	sourceType := r.state.CurrentSourceType
	if sourceType == "" && r.currentSource != nil {
		sourceType = r.currentSource.SourceType
	}
	status := r.state.Status
	active := r.active
	attachment := r.attachment
	resumeOnAttach := r.resumeOnAttach
	r.mu.RUnlock()

	if !isRealtimeMcpAudioSourceType(sourceType) {
		return false, false
	}

	canControl := active != nil || resumeOnAttach || status == play_music.StatusPaused
	if active == nil || attachment == nil || status != play_music.StatusPlaying {
		return canControl, false
	}

	return true, !active.isPaused()
}

func (r *deviceMediaRuntime) AppendCurrentToAgentPlaylist(agentID string) (*PlaylistAppendResult, error) {
	if r == nil {
		return nil, fmt.Errorf("Media player not initialized")
	}

	agentID = strings.TrimSpace(agentID)

	r.mu.RLock()
	currentSource := cloneCurrentSource(r.currentSource)
	currentStatus := r.state.Status
	currentPosition := r.state.PositionMs
	currentIndex := r.playbackIndex
	currentMode := r.mode
	currentPlaylistAgentID := r.playlistAgentID
	r.mu.RUnlock()

	if currentSource == nil {
		return nil, fmt.Errorf("There is currently no media available to add to the playlist")
	}
	if currentSource.SourceType == MediaSourceTypeInlineAudio {
		return nil, fmt.Errorf("The current audio source does not support adding to the playlist")
	}

	item, index, snapshot, err := r.coordinator.appendToAgentPlaylist(agentID, *currentSource)
	if err != nil {
		return nil, err
	}

	targetPlaybackIndex := index
	preserveCurrentPlayback := currentMode == mediaPlaybackModeAgentPlaylist &&
		currentPlaylistAgentID == agentID &&
		currentIndex >= 0 &&
		currentIndex < len(snapshot)
	if preserveCurrentPlayback {
		targetPlaybackIndex = currentIndex
	}

	r.mu.Lock()
	r.mode = mediaPlaybackModeAgentPlaylist
	r.playlistAgentID = agentID
	r.playbackIndex = targetPlaybackIndex
	currentCopy := cloneMediaSourceDescriptor(*currentSource)
	r.currentSource = &currentCopy
	r.state.Playlist = cloneMediaPlaylistItems(snapshot)
	r.state.CurrentIndex = targetPlaybackIndex
	r.state.CurrentTitle = currentCopy.Title
	r.state.CurrentSourceType = currentCopy.SourceType
	r.state.UpdatedAt = time.Now().UnixMilli()
	r.mu.Unlock()

	log.Infof("Device %s adds the current media to the smart playlist, agent=%s, title=%s, index=%d", r.deviceID, strings.TrimSpace(agentID), item.Source.Title, index)

	return &PlaylistAppendResult{
		AddedTitle:      item.Source.Title,
		PlaylistLength:  len(snapshot),
		CurrentIndex:    targetPlaybackIndex,
		CurrentStatus:   currentStatus,
		CurrentSource:   item.Source.SourceType,
		CurrentPosition: currentPosition,
	}, nil
}

func (r *deviceMediaRuntime) pausePlayback(requireActive bool, reason mediaPauseReason) error {
	if r == nil {
		return fmt.Errorf("Media player not initialized")
	}

	var exclusiveAttach *mediaSessionAttachment

	r.mu.Lock()
	active := r.active
	if active == nil {
		if requireActive {
			r.mu.Unlock()
			return fmt.Errorf("There is no media currently playing")
		}
		r.mu.Unlock()
		return nil
	}

	active.setPaused(true)
	exclusiveAttach = r.exclusiveAttachment
	r.exclusiveAttachment = nil
	active.setExclusiveHeld(false)
	r.pauseReason = reason
	r.resumeOnAttach = false
	r.state.Status = play_music.StatusPaused
	r.state.UpdatedAt = time.Now().UnixMilli()
	r.mu.Unlock()

	releaseMediaAttachment(exclusiveAttach)
	return nil
}

func (r *deviceMediaRuntime) jumpToRelative(ctx context.Context, delta int, cfg mediaPlaybackAudioConfig) error {
	if r == nil {
		return fmt.Errorf("Media player not initialized")
	}

	r.mu.RLock()
	mode := r.mode
	agentID := r.playlistAgentID
	currentIndex := r.playbackIndex
	r.mu.RUnlock()

	if mode != mediaPlaybackModeAgentPlaylist {
		return fmt.Errorf("The current playback is not added to the agent playlist, please execute enqueue_current first")
	}

	snapshot := r.coordinator.snapshotAgentPlaylist(agentID)
	if len(snapshot) == 0 {
		return fmt.Errorf("Playlist is empty")
	}

	if currentIndex < 0 || currentIndex >= len(snapshot) {
		if delta >= 0 {
			currentIndex = 0
		} else {
			currentIndex = len(snapshot) - 1
		}
	}

	nextIndex := (currentIndex + delta) % len(snapshot)
	if nextIndex < 0 {
		nextIndex += len(snapshot)
	}

	return r.PlayAgentPlaylistIndex(ctx, agentID, nextIndex, cfg)
}

func (r *deviceMediaRuntime) replaceActivePlaybackLocked(active *activeMediaPlayback) (*activeMediaPlayback, *mediaSessionAttachment) {
	oldActive := r.active
	oldExclusive := r.exclusiveAttachment
	if oldActive != nil {
		oldActive.setExclusiveHeld(false)
	}
	r.active = active
	r.exclusiveAttachment = nil
	return oldActive, oldExclusive
}

func (r *deviceMediaRuntime) refreshPublicStateLocked() {
	switch r.mode {
	case mediaPlaybackModeAgentPlaylist:
		snapshot := r.coordinator.snapshotAgentPlaylist(r.playlistAgentID)
		r.state.Playlist = snapshot
		if len(snapshot) == 0 {
			r.state.CurrentIndex = -1
		} else if r.playbackIndex >= 0 && r.playbackIndex < len(snapshot) {
			r.state.CurrentIndex = r.playbackIndex
		} else {
			r.state.CurrentIndex = -1
		}
	default:
		r.state.Playlist = nil
		r.state.CurrentIndex = -1
	}
}

func (r *deviceMediaRuntime) currentPlaybackQueueLocked() []MediaPlaylistItem {
	switch r.mode {
	case mediaPlaybackModeAgentPlaylist:
		return r.coordinator.snapshotAgentPlaylist(r.playlistAgentID)
	default:
		return cloneMediaPlaylistItems(r.standaloneItems)
	}
}

func (r *deviceMediaRuntime) currentFrameDurationMs() int64 {
	r.mu.RLock()
	frameDuration := r.audioConfig.FrameDuration
	r.mu.RUnlock()
	if frameDuration <= 0 {
		return int64(types_audio.FrameDuration)
	}
	return int64(frameDuration)
}

func (r *deviceMediaRuntime) currentAudioConfig() mediaPlaybackAudioConfig {
	r.mu.RLock()
	cfg := r.audioConfig
	r.mu.RUnlock()
	return normalizeMediaPlaybackAudioConfig(cfg, mediaPlaybackAudioConfig{
		SampleRate:    types_audio.SampleRate,
		FrameDuration: types_audio.FrameDuration,
	})
}

func (r *deviceMediaRuntime) currentIndexForActive(active *activeMediaPlayback) int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.active != active {
		return -1
	}
	return r.playbackIndex
}

func (r *deviceMediaRuntime) prepareTrack(active *activeMediaPlayback, index int) (MediaPlaylistItem, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.active != active {
		return MediaPlaylistItem{}, false
	}

	queue := r.currentPlaybackQueueLocked()
	if index < 0 || index >= len(queue) {
		return MediaPlaylistItem{}, false
	}

	item := cloneMediaPlaylistItem(queue[index])
	r.playbackIndex = index
	currentSource := cloneMediaSourceDescriptor(item.Source)
	r.currentSource = &currentSource
	r.state.CurrentTitle = item.Source.Title
	r.state.CurrentSourceType = item.Source.SourceType
	r.state.PositionMs = 0
	r.state.ErrMsg = ""
	if active.isPaused() {
		r.state.Status = play_music.StatusPaused
	} else {
		r.state.Status = play_music.StatusPlaying
	}
	r.state.UpdatedAt = time.Now().UnixMilli()
	r.refreshPublicStateLocked()
	return item, true
}

func (r *deviceMediaRuntime) advanceAfterTrack(active *activeMediaPlayback, index int) (int, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.active != active {
		return -1, false
	}

	queue := r.currentPlaybackQueueLocked()
	currentIndex := r.playbackIndex
	if currentIndex < 0 || currentIndex >= len(queue) {
		currentIndex = index
	}
	nextIndex := currentIndex + 1
	if nextIndex >= len(queue) {
		r.active = nil
		r.pauseReason = mediaPauseReasonNone
		r.resumeOnAttach = false
		r.state.Status = play_music.StatusStopped
		r.state.PositionMs = 0
		r.state.UpdatedAt = time.Now().UnixMilli()
		r.refreshPublicStateLocked()
		return -1, false
	}

	r.playbackIndex = nextIndex
	currentSource := cloneMediaSourceDescriptor(queue[nextIndex].Source)
	r.currentSource = &currentSource
	r.state.CurrentTitle = queue[nextIndex].Source.Title
	r.state.CurrentSourceType = queue[nextIndex].Source.SourceType
	r.state.PositionMs = 0
	r.state.Status = play_music.StatusPlaying
	r.state.UpdatedAt = time.Now().UnixMilli()
	r.refreshPublicStateLocked()
	return nextIndex, true
}

func (r *deviceMediaRuntime) markPlaybackError(active *activeMediaPlayback, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.active != active {
		return
	}

	r.active = nil
	r.pauseReason = mediaPauseReasonNone
	r.resumeOnAttach = false
	r.state.Status = play_music.StatusError
	r.state.ErrMsg = err.Error()
	r.state.UpdatedAt = time.Now().UnixMilli()
	r.refreshPublicStateLocked()
}

func (r *deviceMediaRuntime) markPlaybackStopped(active *activeMediaPlayback, status play_music.PlaybackStatus, errMsg string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.active != active {
		return
	}

	r.active = nil
	r.pauseReason = mediaPauseReasonNone
	r.resumeOnAttach = false
	r.state.Status = status
	r.state.ErrMsg = errMsg
	r.state.UpdatedAt = time.Now().UnixMilli()
	r.refreshPublicStateLocked()
}

func (r *deviceMediaRuntime) runPlayback(active *activeMediaPlayback) {
	defer active.closeDone()
	defer r.releaseExclusivePlayback(active)

	index := r.currentIndexForActive(active)
	for index >= 0 {
		if err := active.waitIfPaused(active.ctx); err != nil {
			if errors.Is(err, context.Canceled) {
				r.markPlaybackStopped(active, play_music.StatusStopped, "")
			} else {
				r.markPlaybackError(active, err)
			}
			return
		}

		item, ok := r.prepareTrack(active, index)
		if !ok {
			return
		}

		if err := r.playItem(active, item); err != nil {
			if errors.Is(err, context.Canceled) {
				r.markPlaybackStopped(active, play_music.StatusStopped, "")
				return
			}
			r.markPlaybackError(active, err)
			return
		}

		nextIndex, hasNext := r.advanceAfterTrack(active, index)
		if !hasNext {
			return
		}
		index = nextIndex
	}
}

func (r *deviceMediaRuntime) waitForAttachment(active *activeMediaPlayback, ctx context.Context) (*mediaSessionAttachment, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	for {
		r.mu.RLock()
		if r.active != active {
			r.mu.RUnlock()
			return nil, context.Canceled
		}
		attachment := r.attachment
		waitCh := r.attachmentWaitCh
		r.mu.RUnlock()

		if attachment != nil {
			return attachment, nil
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-waitCh:
		}
	}
}

func (r *deviceMediaRuntime) ensureExclusivePlayback(active *activeMediaPlayback, attachment *mediaSessionAttachment, ctx context.Context) error {
	if active == nil {
		return fmt.Errorf("There is no media currently playing")
	}
	if attachment == nil {
		return fmt.Errorf("There are currently no playback channels available.")
	}

	r.mu.RLock()
	sameAttachment := r.active == active && r.exclusiveAttachment == attachment && active.hasExclusiveHeld()
	r.mu.RUnlock()
	if sameAttachment {
		return nil
	}

	if err := attachment.bridge.BeginExclusiveMediaPlayback(ctx); err != nil {
		return err
	}

	var releaseAttachment *mediaSessionAttachment

	r.mu.Lock()
	if r.active != active || r.attachment != attachment {
		r.mu.Unlock()
		attachment.bridge.EndExclusiveMediaPlayback()
		return context.Canceled
	}

	if r.exclusiveAttachment != nil && r.exclusiveAttachment != attachment {
		releaseAttachment = r.exclusiveAttachment
	}
	r.exclusiveAttachment = attachment
	active.setExclusiveHeld(true)
	r.mu.Unlock()

	releaseMediaAttachment(releaseAttachment)
	return nil
}

func (r *deviceMediaRuntime) releaseExclusivePlayback(active *activeMediaPlayback) {
	if r == nil || active == nil {
		return
	}

	var attachment *mediaSessionAttachment

	r.mu.Lock()
	if !active.hasExclusiveHeld() {
		r.mu.Unlock()
		return
	}
	attachment = r.exclusiveAttachment
	r.exclusiveAttachment = nil
	active.setExclusiveHeld(false)
	r.mu.Unlock()

	releaseMediaAttachment(attachment)
}

func (r *deviceMediaRuntime) handleAttachmentFailure(active *activeMediaPlayback, attachment *mediaSessionAttachment, err error) {
	if r == nil || attachment == nil {
		return
	}

	var releaseAttachment *mediaSessionAttachment

	r.mu.Lock()
	if r.attachment != attachment || r.active != active {
		r.mu.Unlock()
		return
	}

	if r.exclusiveAttachment == attachment {
		releaseAttachment = attachment
		r.exclusiveAttachment = nil
		active.setExclusiveHeld(false)
	}

	if r.state.Status == play_music.StatusPlaying {
		active.setPaused(true)
		r.state.Status = play_music.StatusPaused
		r.state.UpdatedAt = time.Now().UnixMilli()
		r.resumeOnAttach = true
	}

	r.attachment = nil
	r.notifyAttachmentChangedLocked()
	r.mu.Unlock()

	releaseMediaAttachment(releaseAttachment)
	log.Warnf("The media output channel of device %s is invalid, waiting for reconnection recovery: %v", r.deviceID, err)
}

func (r *deviceMediaRuntime) playItem(active *activeMediaPlayback, item MediaPlaylistItem) error {
	source := item.Source
	title := deriveMediaTitle(source)
	playText := ""
	if title != "" {
		playText = fmt.Sprintf("Playing music: %s", title)
	}

	audioChan, err := r.openSourceAudioStream(active.ctx, source, active)
	if err != nil {
		return err
	}

	gatedChan := r.wrapMediaAudioStream(active.ctx, audioChan, active)
	if err := r.streamMediaAudio(active.ctx, gatedChan, active, playText); err != nil {
		return err
	}

	log.Infof("Media playback completed: %s", title)
	return nil
}

func (r *deviceMediaRuntime) streamMediaAudio(ctx context.Context, audioChan <-chan []byte, active *activeMediaPlayback, playText string) error {
	var (
		startedOn      *mediaSessionAttachment
		enqueuedFrames bool
	)
	frameDurationMs := r.currentFrameDurationMs()

	for {
		if err := active.waitIfPaused(ctx); err != nil {
			return err
		}

		var (
			frame []byte
			ok    bool
		)

		select {
		case <-ctx.Done():
			return ctx.Err()
		case frame, ok = <-audioChan:
		}

		if !ok {
			if !enqueuedFrames {
				return nil
			}
			return r.waitMediaSentenceEnd(ctx, active, playText)
		}
		if len(frame) == 0 {
			continue
		}

		for {
			if err := active.waitIfPaused(ctx); err != nil {
				return err
			}

			attachment, err := r.waitForAttachment(active, ctx)
			if err != nil {
				return err
			}
			if err := r.ensureExclusivePlayback(active, attachment, ctx); err != nil {
				if errors.Is(err, context.Canceled) {
					return err
				}
				r.handleAttachmentFailure(active, attachment, err)
				continue
			}

			sendError := func(err error) {
				r.handleAttachmentFailure(active, attachment, err)
			}

			if startedOn != attachment && playText != "" {
				if err := attachment.bridge.SendSentenceStart(ctx, playText, sendError); err != nil {
					r.handleAttachmentFailure(active, attachment, err)
					continue
				}
				startedOn = attachment
			}

			if err := attachment.bridge.SendAudioFrame(ctx, frame, sendError); err != nil {
				r.handleAttachmentFailure(active, attachment, err)
				continue
			}

			enqueuedFrames = true
			r.incrementPlaybackPosition(active, frameDurationMs)
			break
		}
	}
}

func (r *deviceMediaRuntime) waitMediaSentenceEnd(ctx context.Context, active *activeMediaPlayback, playText string) error {
	if active == nil {
		return nil
	}

	r.mu.RLock()
	attachment := r.attachment
	isActive := r.active == active
	r.mu.RUnlock()
	if !isActive || attachment == nil {
		return nil
	}

	doneCh := make(chan error, 1)
	onEnd := func(err error) {
		select {
		case doneCh <- err:
		default:
		}
	}
	onError := func(err error) {
		r.handleAttachmentFailure(active, attachment, err)
	}

	if err := attachment.bridge.SendSentenceEnd(ctx, playText, onError, onEnd); err != nil {
		r.handleAttachmentFailure(active, attachment, err)
		return err
	}

	select {
	case err := <-doneCh:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (r *deviceMediaRuntime) openSourceAudioStream(ctx context.Context, source MediaSourceDescriptor, active *activeMediaPlayback) (<-chan []byte, error) {
	cfg := r.currentAudioConfig()
	audioFormat := util.GetAudioFormatByMimeType(source.MIMEType)

	switch source.SourceType {
	case MediaSourceTypeInlineAudio:
		if source.Inline == nil || len(source.Inline.Data) == 0 {
			return nil, fmt.Errorf("inline audio data is empty")
		}
		return play_music.PlayMusicFromAudioData(ctx, source.Inline.Data, cfg.SampleRate, cfg.FrameDuration, audioFormat)
	case MediaSourceTypeHTTPURL:
		if source.HTTP == nil || strings.TrimSpace(source.HTTP.URL) == "" {
			return nil, fmt.Errorf("HTTP audio address is empty")
		}
		return play_music.PlayMusicStream(ctx, source.HTTP.URL, cfg.SampleRate, cfg.FrameDuration, audioFormat)
	case MediaSourceTypeLocalFile:
		if source.Local == nil || strings.TrimSpace(source.Local.Path) == "" {
			return nil, fmt.Errorf("Local audio path is empty")
		}
		return openLocalMediaFileStream(ctx, source.Local.Path, cfg.SampleRate, cfg.FrameDuration, audioFormat)
	case MediaSourceTypeMCPResource:
		if source.MCP == nil {
			return nil, fmt.Errorf("MCP audio source is empty")
		}
		return r.openMCPResourceAudioStream(ctx, source.MCP, active, cfg.SampleRate, cfg.FrameDuration, audioFormat)
	default:
		return nil, fmt.Errorf("Unsupported media source type: %s", source.SourceType)
	}
}

func openLocalMediaFileStream(ctx context.Context, path string, sampleRate int, frameDuration int, audioFormat string) (<-chan []byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("Failed to open local audio file: %v", err)
	}

	outputChan := make(chan []byte, 100)
	go func() {
		defer file.Close()
		decoder, err := util.CreateAudioDecoderWithSampleRate(ctx, file, outputChan, frameDuration, audioFormat, sampleRate)
		if err != nil {
			log.Errorf("Failed to create local audio decoder: %v", err)
			close(outputChan)
			return
		}
		if err := decoder.Run(time.Now().UnixMilli()); err != nil && !errors.Is(err, context.Canceled) {
			log.Errorf("Local audio decoding failed: %v", err)
		}
	}()
	return outputChan, nil
}

func (r *deviceMediaRuntime) openMCPResourceAudioStream(ctx context.Context, source *MCPMediaSource, active *activeMediaPlayback, sampleRate int, frameDuration int, audioFormat string) (<-chan []byte, error) {
	if source == nil {
		return nil, fmt.Errorf("MCP audio source is empty")
	}
	if strings.TrimSpace(source.ResourceURI) == "" {
		return nil, fmt.Errorf("MCP Resource URI is empty")
	}

	pipeReader, pipeWriter := io.Pipe()
	audioChan, err := play_music.PlayMusicFromPipe(ctx, pipeReader, sampleRate, frameDuration, audioFormat)
	if err != nil {
		pipeWriter.CloseWithError(err)
		return nil, fmt.Errorf("Failed to create MCP audio decoding stream: %v", err)
	}

	go r.streamMCPResourceToPipe(ctx, source, active, pipeWriter)
	return audioChan, nil
}

func (r *deviceMediaRuntime) streamMCPResourceToPipe(ctx context.Context, source *MCPMediaSource, active *activeMediaPlayback, pipeWriter *io.PipeWriter) {
	defer pipeWriter.Close()

	client := source.Client
	start := 0
	pageSize := source.PageSize
	if pageSize <= 0 {
		pageSize = defaultMediaPlayerPageSize
	}

	for {
		if err := active.waitIfPaused(ctx); err != nil {
			pipeWriter.CloseWithError(err)
			return
		}

		readArgs := cloneMediaSourceAnyMap(source.ReadArgs)
		readArgs["start"] = start
		readArgs["end"] = start + pageSize

		resourceResult, err := r.readMCPResourcePage(ctx, source, client, readArgs)
		if err != nil {
			pipeWriter.CloseWithError(err)
			return
		}
		if len(resourceResult.Contents) == 0 {
			return
		}

		hasData := false
		for _, content := range resourceResult.Contents {
			audioContent, ok := content.(mcp_go.BlobResourceContents)
			if !ok {
				continue
			}
			if len(audioContent.Blob) == 0 {
				continue
			}

			rawAudioData, err := base64.StdEncoding.DecodeString(audioContent.Blob)
			if err != nil {
				pipeWriter.CloseWithError(fmt.Errorf("Failed to decode MCP audio data: %v", err))
				return
			}
			if string(rawAudioData) == McpReadResourceStreamDoneFlag {
				return
			}

			if err := active.waitIfPaused(ctx); err != nil {
				pipeWriter.CloseWithError(err)
				return
			}

			if _, err := pipeWriter.Write(rawAudioData); err != nil {
				pipeWriter.CloseWithError(fmt.Errorf("Failed to write to MCP audio stream: %v", err))
				return
			}
			hasData = true
			if len(rawAudioData) < pageSize {
				return
			}
		}

		if !hasData {
			return
		}
		start += pageSize
	}
}

func (r *deviceMediaRuntime) readMCPResourcePage(ctx context.Context, source *MCPMediaSource, client *mcpclient.Client, readArgs map[string]any) (mcp_go.ReadResourceResult, error) {
	if client == nil && source.ServerName != "" {
		client = mcp_domain.GetServerClientByName(source.ServerName)
	}
	if client == nil {
		return mcp_go.ReadResourceResult{}, fmt.Errorf("MCP client unavailable: %s", source.ServerName)
	}

	readCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	resourceResult, err := client.ReadResource(readCtx, mcp_go.ReadResourceRequest{
		Params: mcp_go.ReadResourceParams{
			URI:       source.ResourceURI,
			Arguments: readArgs,
		},
	})
	if err == nil {
		if resourceResult == nil {
			return mcp_go.ReadResourceResult{}, nil
		}
		return *resourceResult, nil
	}

	if source.ServerName == "" || !strings.Contains(err.Error(), "session closed") {
		return mcp_go.ReadResourceResult{}, err
	}

	newClient, reconnErr := mcp_domain.ReconnectServerByName(source.ServerName)
	if reconnErr != nil {
		return mcp_go.ReadResourceResult{}, fmt.Errorf("MCP resource read failed and reconnection failed: %v", err)
	}
	source.Client = newClient

	readCtx, cancel = context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	resourceResult, err = newClient.ReadResource(readCtx, mcp_go.ReadResourceRequest{
		Params: mcp_go.ReadResourceParams{
			URI:       source.ResourceURI,
			Arguments: readArgs,
		},
	})
	if err != nil {
		return mcp_go.ReadResourceResult{}, err
	}
	if resourceResult == nil {
		return mcp_go.ReadResourceResult{}, nil
	}
	return *resourceResult, nil
}

func (r *deviceMediaRuntime) wrapMediaAudioStream(ctx context.Context, input <-chan []byte, active *activeMediaPlayback) <-chan []byte {
	output := make(chan []byte)

	go func() {
		defer close(output)

		for {
			select {
			case <-ctx.Done():
				return
			case frame, ok := <-input:
				if !ok {
					return
				}
				if err := active.waitIfPaused(ctx); err != nil {
					return
				}
				select {
				case <-ctx.Done():
					return
				case output <- frame:
				}
			}
		}
	}()

	return output
}

func (r *deviceMediaRuntime) incrementPlaybackPosition(active *activeMediaPlayback, deltaMs int64) {
	if deltaMs <= 0 {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.active != active {
		return
	}

	r.state.PositionMs += deltaMs
	r.state.UpdatedAt = time.Now().UnixMilli()
}

func buildMediaPlaylistItems(sources []MediaSourceDescriptor) []MediaPlaylistItem {
	items := make([]MediaPlaylistItem, 0, len(sources))
	nowMs := time.Now().UnixMilli()
	for i, source := range sources {
		cloned := cloneMediaSourceDescriptor(source)
		if cloned.ID == "" {
			cloned.ID = fmt.Sprintf("media_%d_%d", time.Now().UnixNano(), i)
		}
		if cloned.Title == "" {
			cloned.Title = deriveMediaTitle(cloned)
		}
		items = append(items, MediaPlaylistItem{
			ID:      cloned.ID,
			Source:  cloned,
			AddedAt: nowMs,
		})
	}
	return items
}

func normalizeMediaPlaybackAudioConfig(primary mediaPlaybackAudioConfig, fallback mediaPlaybackAudioConfig) mediaPlaybackAudioConfig {
	cfg := fallback
	if primary.SampleRate > 0 {
		cfg.SampleRate = primary.SampleRate
	}
	if primary.FrameDuration > 0 {
		cfg.FrameDuration = primary.FrameDuration
	}
	if cfg.SampleRate <= 0 {
		cfg.SampleRate = types_audio.SampleRate
	}
	if cfg.FrameDuration <= 0 {
		cfg.FrameDuration = types_audio.FrameDuration
	}
	return cfg
}

func cloneCurrentSource(source *MediaSourceDescriptor) *MediaSourceDescriptor {
	if source == nil {
		return nil
	}
	cloned := cloneMediaSourceDescriptor(*source)
	return &cloned
}

func releaseMediaAttachment(attachment *mediaSessionAttachment) {
	if attachment == nil {
		return
	}
	attachment.bridge.EndExclusiveMediaPlayback()
}

func stopOldPlayback(active *activeMediaPlayback) {
	if active == nil {
		return
	}
	active.cancel()
	waitForActivePlaybackDone(active, 2*time.Second)
}

func deriveMediaTitle(source MediaSourceDescriptor) string {
	if strings.TrimSpace(source.Title) != "" {
		return strings.TrimSpace(source.Title)
	}

	switch source.SourceType {
	case MediaSourceTypeHTTPURL:
		if source.HTTP != nil {
			if parsed, err := url.Parse(source.HTTP.URL); err == nil {
				if base := strings.TrimSpace(parsed.Path); base != "" {
					parts := strings.Split(base, "/")
					return strings.TrimSpace(parts[len(parts)-1])
				}
			}
			return strings.TrimSpace(source.HTTP.URL)
		}
	case MediaSourceTypeLocalFile:
		if source.Local != nil {
			return strings.TrimSpace(source.Local.Path)
		}
	case MediaSourceTypeMCPResource:
		if source.MCP != nil {
			if title := strings.TrimSpace(source.MCP.Description); title != "" {
				return title
			}
			if title := strings.TrimSpace(source.MCP.DirectAudioURL); title != "" {
				return title
			}
			return strings.TrimSpace(source.MCP.ResourceURI)
		}
	}

	return "Unknown audio"
}

func cloneMediaSourceDescriptor(source MediaSourceDescriptor) MediaSourceDescriptor {
	cloned := source
	cloned.Meta = cloneMediaSourceStringMap(source.Meta)
	if source.MCP != nil {
		mcpCopy := *source.MCP
		mcpCopy.ReadArgs = cloneMediaSourceAnyMap(source.MCP.ReadArgs)
		cloned.MCP = &mcpCopy
	}
	if source.HTTP != nil {
		httpCopy := *source.HTTP
		cloned.HTTP = &httpCopy
	}
	if source.Local != nil {
		localCopy := *source.Local
		cloned.Local = &localCopy
	}
	if source.Inline != nil {
		inlineCopy := *source.Inline
		inlineCopy.Data = append([]byte(nil), source.Inline.Data...)
		cloned.Inline = &inlineCopy
	}
	return cloned
}

func cloneMediaPlaylistItems(items []MediaPlaylistItem) []MediaPlaylistItem {
	if len(items) == 0 {
		return nil
	}
	cloned := make([]MediaPlaylistItem, 0, len(items))
	for _, item := range items {
		cloned = append(cloned, cloneMediaPlaylistItem(item))
	}
	return cloned
}

func cloneMediaPlaylistItem(item MediaPlaylistItem) MediaPlaylistItem {
	itemCopy := item
	itemCopy.Source = cloneMediaSourceDescriptor(item.Source)
	return itemCopy
}

func cloneMediaSourceStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func cloneMediaSourceAnyMap(in map[string]any) map[string]any {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func waitForActivePlaybackDone(active *activeMediaPlayback, timeout time.Duration) {
	if active == nil {
		return
	}
	if timeout <= 0 {
		timeout = 2 * time.Second
	}

	select {
	case <-active.done:
	case <-time.After(timeout):
	}
}
