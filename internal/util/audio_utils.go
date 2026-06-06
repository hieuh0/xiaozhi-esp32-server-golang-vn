package util

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"time"

	log "xiaozhi-esp32-server-golang/logger"

	"github.com/go-audio/audio"
	"github.com/go-audio/wav"
	"github.com/gopxl/beep"
	"github.com/gopxl/beep/mp3"
	"gopkg.in/hraban/opus.v2"
)

// min returns the smaller of x or y.
func min(x, y int) int {
	if x < y {
		return x
	}
	return y
}

// readCloserWrapper provides a Close method for bytes.Reader to implement the ReadCloser interface
type readCloserWrapper struct {
	*bytes.Reader
}

// Close implements the io.Closer interface
func (r *readCloserWrapper) Close() error {
	return nil
}

// newReadCloserWrapper creates a new ReadCloser wrapper
func newReadCloserWrapper(data []byte) *readCloserWrapper {
	return &readCloserWrapper{bytes.NewReader(data)}
}

// WavToOpus converts WAV audio data to standard Opus format.
// Returns a slice of Opus frames, where each element is one encoded Opus frame.
func WavToOpus(wavData []byte, sampleRate int, channels int, bitRate int) ([][]byte, error) {
	// Create WAV decoder
	wavReader := bytes.NewReader(wavData)
	wavDecoder := wav.NewDecoder(wavReader)
	if !wavDecoder.IsValidFile() {
		return nil, fmt.Errorf("invalid WAV file")
	}

	// Read WAV file info
	wavDecoder.ReadInfo()
	format := wavDecoder.Format()
	wavSampleRate := int(format.SampleRate)
	wavChannels := int(format.NumChannels)

	// If provided parameters differ from file parameters, use the file parameters
	if sampleRate == 0 {
		sampleRate = wavSampleRate
	}
	if channels == 0 {
		channels = wavChannels
	}

	// Print wavDecoder info
	fmt.Println("WAV format:", format)

	enc, err := opus.NewEncoder(sampleRate, channels, opus.AppAudio)
	if err != nil {
		return nil, fmt.Errorf("failed to create Opus encoder: %v", err)
	}

	// Set bit rate
	if bitRate > 0 {
		if err := enc.SetBitrate(bitRate); err != nil {
			return nil, fmt.Errorf("failed to set bit rate: %v", err)
		}
	}

	// Create output frame slice
	opusFrames := make([][]byte, 0)

	perFrameDuration := 20
	// PCM buffer - Opus frame size (60ms)
	frameSize := sampleRate * perFrameDuration / 1000
	pcmBuffer := make([]int16, frameSize*channels)
	opusBuffer := make([]byte, 1000) // Buffer large enough to hold encoded data

	// Read audio buffer
	audioBuf := &audio.IntBuffer{Data: make([]int, frameSize*channels), Format: format}

	fmt.Println("Starting conversion...")
	for {
		// Read WAV data
		n, err := wavDecoder.PCMBuffer(audioBuf)
		if err == io.EOF || n == 0 {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read WAV data: %v", err)
		}

		// Convert int to int16
		for i := 0; i < len(audioBuf.Data); i++ {
			if i < len(pcmBuffer) {
				pcmBuffer[i] = int16(audioBuf.Data[i])
			}
		}

		// Encode to Opus format
		n, err = enc.Encode(pcmBuffer, opusBuffer)
		if err != nil {
			return nil, fmt.Errorf("encoding failed: %v", err)
		}

		// Copy current frame to a new slice and append to frame array
		frameData := make([]byte, n)
		copy(frameData, opusBuffer[:n])
		opusFrames = append(opusFrames, frameData)
	}

	return opusFrames, nil
}

type AudioDecoder struct {
	streamer           beep.StreamSeekCloser
	format             beep.Format
	enc                *opus.Encoder
	pipeReader         io.ReadCloser
	perFrameDurationMs int
	AudioFormat        string
	targetSampleRate   int
	TargetAudioFormat  string

	outputOpusChan chan []byte     // opus output frame by frame
	ctx            context.Context // context for cancellation
}

// CreateAudioDecoder creates an audio decoder controlled via context.
// Retained for backward compatibility.
func CreateAudioDecoder(ctx context.Context, pipeReader io.ReadCloser, outputOpusChan chan []byte, perFrameDurationMs int, AudioFormat string) (*AudioDecoder, error) {
	return &AudioDecoder{
		pipeReader:         pipeReader,
		outputOpusChan:     outputOpusChan,
		perFrameDurationMs: perFrameDurationMs,
		AudioFormat:        AudioFormat,
		ctx:                ctx,
		TargetAudioFormat:  "opus",
	}, nil
}

// CreateAudioDecoderWithSampleRate creates an audio decoder with a target sample rate.
// Retained for backward compatibility.
func CreateAudioDecoderWithSampleRate(ctx context.Context, pipeReader io.ReadCloser, outputOpusChan chan []byte, perFrameDurationMs int, AudioFormat string, targetSampleRate int) (*AudioDecoder, error) {
	return &AudioDecoder{
		pipeReader:         pipeReader,
		outputOpusChan:     outputOpusChan,
		perFrameDurationMs: perFrameDurationMs,
		AudioFormat:        AudioFormat,
		targetSampleRate:   targetSampleRate,
		ctx:                ctx,
		TargetAudioFormat:  "opus",
	}, nil
}

func (d *AudioDecoder) WithFormat(format beep.Format) *AudioDecoder {
	d.format = format
	return d
}

func (d *AudioDecoder) WithTargetAudioFormat(targetAudioFormat string) *AudioDecoder {
	d.TargetAudioFormat = targetAudioFormat
	return d
}

func (d *AudioDecoder) Run(startTs int64) error {
	if d.AudioFormat == "wav" {
		d.RunWavDecoder(startTs, false)
	} else if d.AudioFormat == "pcm" {
		d.RunWavDecoder(startTs, true)
	} else if d.AudioFormat == "mp3" {
		return d.RunMp3Decoder(startTs)
	} else if d.AudioFormat == "opus" {
		return d.RunOpusDecoder(startTs)
	} else if d.AudioFormat == "ogg_opus" {
		return d.RunOggOpusDecoder(startTs)
	}
	return nil
}

// WriteLengthPrefixedFrame writes a single audio frame as "4-byte length header + payload"
// for streaming to a generic decoder.
func WriteLengthPrefixedFrame(writer io.Writer, frame []byte) error {
	if writer == nil {
		return fmt.Errorf("writer cannot be nil")
	}
	if len(frame) == 0 {
		return fmt.Errorf("frame cannot be empty")
	}

	var header [4]byte
	binary.LittleEndian.PutUint32(header[:], uint32(len(frame)))
	if _, err := writer.Write(header[:]); err != nil {
		return fmt.Errorf("failed to write frame length: %v", err)
	}
	if _, err := writer.Write(frame); err != nil {
		return fmt.Errorf("failed to write frame data: %v", err)
	}
	return nil
}

func readLengthPrefixedFrame(reader io.Reader) ([]byte, error) {
	var header [4]byte
	if _, err := io.ReadFull(reader, header[:]); err != nil {
		return nil, err
	}

	frameLen := binary.LittleEndian.Uint32(header[:])
	if frameLen == 0 {
		return nil, fmt.Errorf("frame length cannot be 0")
	}
	if frameLen > 64*1024 {
		return nil, fmt.Errorf("frame length too large: %d", frameLen)
	}

	frame := make([]byte, int(frameLen))
	if _, err := io.ReadFull(reader, frame); err != nil {
		return nil, err
	}
	return frame, nil
}

func (d *AudioDecoder) RunOpusDecoder(startTs int64) error {
	defer func() {
		close(d.outputOpusChan)
		if d.pipeReader != nil {
			d.pipeReader.Close()
		}
	}()

	sourceSampleRate := int(d.format.SampleRate)
	if sourceSampleRate < 1 {
		sourceSampleRate = 16000
		log.Warnf("Opus input sample rate is 0, defaulting to 16000 Hz")
	}

	channels := d.format.NumChannels
	if channels < 1 {
		channels = 1
		log.Warnf("Opus input channel count is 0, defaulting to mono")
	}

	return d.runOpusPacketStream(startTs, sourceSampleRate, channels, func() ([]byte, error) {
		packet, err := readLengthPrefixedFrame(d.pipeReader)
		if err == io.ErrUnexpectedEOF {
			return nil, fmt.Errorf("failed to read Opus frame: incomplete data")
		}
		if err != nil {
			return nil, err
		}
		return packet, nil
	})
}

func (d *AudioDecoder) RunOggOpusDecoder(startTs int64) error {
	defer func() {
		close(d.outputOpusChan)
		if d.pipeReader != nil {
			d.pipeReader.Close()
		}
	}()

	packetReader := &oggOpusPacketReader{reader: d.pipeReader}
	info, err := packetReader.Prepare()
	if err != nil {
		return fmt.Errorf("failed to parse Ogg Opus header: %v", err)
	}

	log.Debugf("Ogg Opus decoder started, source sample rate: %d, source channels: %d, target sample rate: %d, target format: %s", info.SampleRate, info.Channels, d.getTargetSampleRate(info.SampleRate), d.TargetAudioFormat)

	return d.runOpusPacketStream(startTs, info.SampleRate, info.Channels, packetReader.NextPacket)
}

func (d *AudioDecoder) runOpusPacketStream(startTs int64, sourceSampleRate int, channels int, nextPacket func() ([]byte, error)) error {
	firstPacket, err := nextPacket()
	if err == io.EOF {
		return nil
	}
	if err != nil {
		return err
	}

	if d.canPassthroughOpusPacket(sourceSampleRate, channels, firstPacket) {
		return d.passThroughOpusPackets(startTs, firstPacket, nextPacket)
	}
	if d.canRepacketizeOpusPacket(sourceSampleRate, channels, firstPacket) {
		return d.repacketizeOpusPackets(startTs, sourceSampleRate, firstPacket, nextPacket)
	}

	return d.transcodeOpusPackets(startTs, sourceSampleRate, channels, firstPacket, nextPacket)
}

func (d *AudioDecoder) passThroughOpusPackets(startTs int64, firstPacket []byte, nextPacket func() ([]byte, error)) error {
	var firstFrame bool
	emitPacket := func(packet []byte) error {
		if len(packet) == 0 {
			return nil
		}
		if !firstFrame {
			firstFrame = true
			log.Infof("tts cloud->first frame passthrough latency: %d ms", time.Now().UnixMilli()-startTs)
		}
		frameData := make([]byte, len(packet))
		copy(frameData, packet)
		select {
		case <-d.ctx.Done():
			log.Debugf("opus passthrough context done, exit")
			return nil
		case d.outputOpusChan <- frameData:
		}
		return nil
	}

	if err := emitPacket(firstPacket); err != nil {
		return err
	}

	for {
		select {
		case <-d.ctx.Done():
			log.Debugf("opus passthrough context done, exit")
			return nil
		default:
		}

		packet, err := nextPacket()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		if err := emitPacket(packet); err != nil {
			return err
		}
	}
}

func (d *AudioDecoder) transcodeOpusPackets(startTs int64, sourceSampleRate int, channels int, firstPacket []byte, nextPacket func() ([]byte, error)) error {
	targetSampleRate := d.getTargetSampleRate(sourceSampleRate)
	frameDurationMs := d.perFrameDurationMs
	if frameDurationMs <= 0 {
		frameDurationMs = 60
	}
	sourceFrameSize := sourceSampleRate * frameDurationMs / 1000
	if sourceFrameSize <= 0 {
		return fmt.Errorf("invalid Opus frame duration: %d ms", frameDurationMs)
	}

	outputChannels := 1
	var enc *opus.Encoder
	var err error
	if d.TargetAudioFormat == "opus" {
		enc, err = opus.NewEncoder(targetSampleRate, outputChannels, opus.AppAudio)
		if err != nil {
			return fmt.Errorf("failed to create Opus encoder: %v", err)
		}
		d.enc = enc
	}

	opusDecoder, err := opus.NewDecoder(sourceSampleRate, channels)
	if err != nil {
		return fmt.Errorf("failed to create Opus decoder: %v", err)
	}

	maxDecodeSamples := channels * sourceSampleRate * 120 / 1000
	if maxDecodeSamples < channels*sourceSampleRate/50 {
		maxDecodeSamples = channels * sourceSampleRate / 50
	}
	decodedBuffer := make([]int16, maxDecodeSamples)
	pcmBuffer := make([]int16, 0, sourceFrameSize*2)
	opusBuffer := make([]byte, 1000)
	var firstFrame bool

	log.Debugf("Opus transcode started, source sample rate: %d, target sample rate: %d, source channels: %d, frame size: %d, target format: %s", sourceSampleRate, targetSampleRate, channels, sourceFrameSize, d.TargetAudioFormat)

	emitFrame := func(frame []int16) error {
		if len(frame) == 0 {
			return nil
		}

		outputPCM := append([]int16(nil), frame...)
		if targetSampleRate > 0 && targetSampleRate != sourceSampleRate {
			pcmBytes := Int16SliceToBytes(outputPCM)
			pcmFloat32 := PCM16BytesToFloat32(pcmBytes)
			pcmFloat32 = ResampleLinearFloat32(pcmFloat32, sourceSampleRate, targetSampleRate)
			outputPCM = Float32SliceToInt16Slice(pcmFloat32)
		}

		if !firstFrame {
			firstFrame = true
			log.Infof("tts cloud->first frame decode latency: %d ms", time.Now().UnixMilli()-startTs)
		}

		switch d.TargetAudioFormat {
		case "opus":
			n, encodeErr := enc.Encode(outputPCM, opusBuffer)
			if encodeErr != nil {
				return fmt.Errorf("Opus re-encode failed: %v", encodeErr)
			}
			frameData := make([]byte, n)
			copy(frameData, opusBuffer[:n])
			select {
			case <-d.ctx.Done():
				log.Debugf("opusDecoder context done, exit")
				return nil
			case d.outputOpusChan <- frameData:
			}
		case "pcm":
			pcmData := Int16SliceToBytes(outputPCM)
			select {
			case <-d.ctx.Done():
				log.Debugf("opusDecoder context done, exit")
				return nil
			case d.outputOpusChan <- pcmData:
			}
		default:
			return fmt.Errorf("unsupported target audio format: %s", d.TargetAudioFormat)
		}

		return nil
	}

	flushFrames := func(flushLast bool) error {
		for len(pcmBuffer) >= sourceFrameSize {
			frame := append([]int16(nil), pcmBuffer[:sourceFrameSize]...)
			if err := emitFrame(frame); err != nil {
				return err
			}
			pcmBuffer = pcmBuffer[sourceFrameSize:]
		}
		if flushLast && len(pcmBuffer) > 0 {
			padded := make([]int16, sourceFrameSize)
			copy(padded, pcmBuffer)
			if err := emitFrame(padded); err != nil {
				return err
			}
			pcmBuffer = pcmBuffer[:0]
		}
		return nil
	}

	processPacket := func(packet []byte) error {
		n, err := opusDecoder.Decode(packet, decodedBuffer)
		if err != nil {
			return fmt.Errorf("failed to decode Opus frame: %v", err)
		}
		if n <= 0 {
			return nil
		}

		if channels == 1 {
			pcmBuffer = append(pcmBuffer, decodedBuffer[:n]...)
		} else {
			for i := 0; i < n; i++ {
				base := i * channels
				var sampleSum int32
				for ch := 0; ch < channels; ch++ {
					sampleSum += int32(decodedBuffer[base+ch])
				}
				pcmBuffer = append(pcmBuffer, int16(sampleSum/int32(channels)))
			}
		}
		return flushFrames(false)
	}

	if err := processPacket(firstPacket); err != nil {
		return err
	}

	for {
		select {
		case <-d.ctx.Done():
			log.Debugf("opusDecoder context done, exit")
			return nil
		default:
		}

		packet, err := nextPacket()
		if err == io.EOF {
			log.Debugf("Opus stream read complete, processing remaining data")
			return flushFrames(true)
		}
		if err != nil {
			return err
		}
		if err := processPacket(packet); err != nil {
			return err
		}
	}
}

func (d *AudioDecoder) repacketizeOpusPackets(startTs int64, sourceSampleRate int, firstPacket []byte, nextPacket func() ([]byte, error)) error {
	targetDurationMs := d.perFrameDurationMs
	if targetDurationMs <= 0 {
		return fmt.Errorf("invalid target Opus frame duration: %d ms", targetDurationMs)
	}

	rp, err := newOpusRepacketizer()
	if err != nil {
		return err
	}
	defer rp.close()

	currentDurationMs := 0
	prevTOC := byte(0)
	var firstFrame bool

	emitCurrent := func() error {
		if rp.nbFrames() == 0 {
			return nil
		}
		packet, err := rp.out()
		if err != nil {
			return fmt.Errorf("failed to output repacketized Opus packet: %v", err)
		}
		if len(packet) == 0 {
			rp.reset()
			currentDurationMs = 0
			prevTOC = 0
			return nil
		}
		if !firstFrame {
			firstFrame = true
			log.Infof("tts cloud->first frame repacketize latency: %d ms", time.Now().UnixMilli()-startTs)
		}
		frameData := make([]byte, len(packet))
		copy(frameData, packet)
		select {
		case <-d.ctx.Done():
			log.Debugf("opus repacketize context done, exit")
			return nil
		case d.outputOpusChan <- frameData:
		}
		rp.reset()
		currentDurationMs = 0
		prevTOC = 0
		return nil
	}

	appendPacket := func(packet []byte) error {
		if len(packet) == 0 {
			return nil
		}
		packetDurationMs, err := opusPacketDurationMs(packet, sourceSampleRate)
		if err != nil {
			return err
		}
		if packetDurationMs <= 0 {
			return fmt.Errorf("invalid Opus packet duration: %d ms", packetDurationMs)
		}
		if packetDurationMs > targetDurationMs {
			return fmt.Errorf("Opus packet duration %d ms exceeds target frame duration %d ms, cannot handle by repacketizing alone", packetDurationMs, targetDurationMs)
		}

		needFlush := rp.nbFrames() > 0 && (((prevTOC & 0xFC) != (packet[0] & 0xFC)) || currentDurationMs+packetDurationMs > targetDurationMs)
		if needFlush {
			if err := emitCurrent(); err != nil {
				return err
			}
		}

		if err := rp.cat(packet); err != nil {
			return fmt.Errorf("failed to submit Opus packet to repacketizer: %v", err)
		}
		prevTOC = packet[0]
		currentDurationMs += packetDurationMs
		if currentDurationMs == targetDurationMs {
			return emitCurrent()
		}
		return nil
	}

	if err := appendPacket(firstPacket); err != nil {
		return err
	}

	for {
		select {
		case <-d.ctx.Done():
			log.Debugf("opus repacketize context done, exit")
			return nil
		default:
		}

		packet, err := nextPacket()
		if err == io.EOF {
			return emitCurrent()
		}
		if err != nil {
			return err
		}
		if err := appendPacket(packet); err != nil {
			return err
		}
	}
}

func (d *AudioDecoder) getTargetSampleRate(sourceSampleRate int) int {
	targetSampleRate := sourceSampleRate
	if d.targetSampleRate > 0 {
		targetSampleRate = d.targetSampleRate
	}
	return targetSampleRate
}

func (d *AudioDecoder) canPassthroughOpusPacket(sourceSampleRate int, channels int, firstPacket []byte) bool {
	if d.TargetAudioFormat != "opus" {
		return false
	}
	if channels != 1 {
		return false
	}
	if d.getTargetSampleRate(sourceSampleRate) != sourceSampleRate {
		return false
	}
	if d.perFrameDurationMs <= 0 {
		return true
	}

	packetDurationMs, err := opusPacketDurationMs(firstPacket, sourceSampleRate)
	if err != nil {
		log.Debugf("failed to parse Opus packet duration, falling back to transcode: %v", err)
		return false
	}
	if packetDurationMs != d.perFrameDurationMs {
		log.Debugf("Opus packet duration mismatch, falling back to transcode: packet=%dms target=%dms", packetDurationMs, d.perFrameDurationMs)
		return false
	}
	return true
}

func (d *AudioDecoder) canRepacketizeOpusPacket(sourceSampleRate int, channels int, firstPacket []byte) bool {
	if d.TargetAudioFormat != "opus" {
		return false
	}
	if channels != 1 {
		return false
	}
	if d.getTargetSampleRate(sourceSampleRate) != sourceSampleRate {
		return false
	}
	targetDurationMs := d.perFrameDurationMs
	if targetDurationMs <= 0 || targetDurationMs > 120 {
		return false
	}

	packetDurationMs, err := opusPacketDurationMs(firstPacket, sourceSampleRate)
	if err != nil {
		log.Debugf("failed to parse Opus packet duration, falling back to transcode: %v", err)
		return false
	}
	if packetDurationMs <= 0 || packetDurationMs >= targetDurationMs {
		return false
	}
	return true
}

func opusPacketDurationMs(packet []byte, sampleRate int) (int, error) {
	if len(packet) == 0 {
		return 0, fmt.Errorf("empty Opus packet")
	}
	if sampleRate <= 0 {
		sampleRate = 48000
	}

	samplesPerFrame := opusPacketSamplesPerFrame(packet[0], sampleRate)
	frameCount, err := opusPacketFrameCount(packet)
	if err != nil {
		return 0, err
	}
	totalSamples := samplesPerFrame * frameCount
	return totalSamples * 1000 / sampleRate, nil
}

func opusPacketSamplesPerFrame(toc byte, sampleRate int) int {
	if toc&0x80 != 0 {
		return (sampleRate << ((toc >> 3) & 0x03)) / 400
	}
	if toc&0x60 == 0x60 {
		if toc&0x08 != 0 {
			return sampleRate / 50
		}
		return sampleRate / 100
	}

	audioSize := (toc >> 3) & 0x03
	if audioSize == 3 {
		return sampleRate * 60 / 1000
	}
	return (sampleRate << audioSize) / 100
}

func opusPacketFrameCount(packet []byte) (int, error) {
	if len(packet) == 0 {
		return 0, fmt.Errorf("empty Opus packet")
	}

	switch packet[0] & 0x03 {
	case 0:
		return 1, nil
	case 1, 2:
		return 2, nil
	default:
		if len(packet) < 2 {
			return 0, fmt.Errorf("Opus packet too short to parse frame count")
		}
		return int(packet[1] & 0x3F), nil
	}
}

type opusStreamInfo struct {
	SampleRate int
	Channels   int
}

type oggPage struct {
	HeaderType byte
	Segments   []byte
	Body       []byte
}

type oggOpusPacketReader struct {
	reader   io.Reader
	queue    [][]byte
	carry    []byte
	info     opusStreamInfo
	headSeen bool
	tagsSeen bool
}

func (r *oggOpusPacketReader) Prepare() (opusStreamInfo, error) {
	for !r.headSeen || !r.tagsSeen {
		if err := r.readNextPage(); err != nil {
			if err == io.EOF {
				return opusStreamInfo{}, fmt.Errorf("Ogg Opus stream is missing required headers")
			}
			return opusStreamInfo{}, err
		}
	}

	if r.info.SampleRate <= 0 {
		r.info.SampleRate = 48000
	}
	if r.info.Channels <= 0 {
		r.info.Channels = 1
	}
	return r.info, nil
}

func (r *oggOpusPacketReader) NextPacket() ([]byte, error) {
	for len(r.queue) == 0 {
		if err := r.readNextPage(); err != nil {
			if err == io.EOF {
				if len(r.carry) > 0 {
					return nil, io.ErrUnexpectedEOF
				}
				return nil, io.EOF
			}
			return nil, err
		}
	}

	packet := r.queue[0]
	r.queue = r.queue[1:]
	return packet, nil
}

func (r *oggOpusPacketReader) readNextPage() error {
	page, err := readOggPage(r.reader)
	if err != nil {
		return err
	}

	packet := r.carry
	if len(packet) == 0 && page.HeaderType&0x01 != 0 {
		return fmt.Errorf("received Ogg continuation page with no preceding data")
	}

	offset := 0
	for _, segmentLen := range page.Segments {
		end := offset + int(segmentLen)
		if end > len(page.Body) {
			return fmt.Errorf("Ogg page body length is incomplete")
		}
		packet = append(packet, page.Body[offset:end]...)
		offset = end
		if segmentLen < 255 {
			completePacket := append([]byte(nil), packet...)
			if err := r.handlePacket(completePacket); err != nil {
				return err
			}
			packet = nil
		}
	}

	if offset != len(page.Body) {
		return fmt.Errorf("Ogg page body has unconsumed trailing data: offset=%d total=%d", offset, len(page.Body))
	}

	r.carry = packet
	return nil
}

func (r *oggOpusPacketReader) handlePacket(packet []byte) error {
	switch {
	case !r.headSeen:
		info, err := parseOpusHeadPacket(packet)
		if err != nil {
			return err
		}
		r.info = info
		r.headSeen = true
	case !r.tagsSeen:
		if !bytes.HasPrefix(packet, []byte("OpusTags")) {
			return fmt.Errorf("missing OpusTags packet")
		}
		r.tagsSeen = true
	default:
		if len(packet) > 0 {
			r.queue = append(r.queue, packet)
		}
	}
	return nil
}

func parseOpusHeadPacket(packet []byte) (opusStreamInfo, error) {
	if len(packet) < 19 {
		return opusStreamInfo{}, fmt.Errorf("OpusHead packet too short: %d", len(packet))
	}
	if !bytes.HasPrefix(packet, []byte("OpusHead")) {
		return opusStreamInfo{}, fmt.Errorf("missing OpusHead packet")
	}

	channels := int(packet[9])
	if channels < 1 {
		channels = 1
	}
	sampleRate := int(binary.LittleEndian.Uint32(packet[12:16]))
	if sampleRate <= 0 {
		sampleRate = 48000
	}

	return opusStreamInfo{
		SampleRate: NormalizeOpusSampleRate(sampleRate),
		Channels:   channels,
	}, nil
}

func readOggPage(reader io.Reader) (oggPage, error) {
	header := make([]byte, 27)
	n, err := io.ReadFull(reader, header)
	if err != nil {
		if err == io.EOF && n == 0 {
			return oggPage{}, io.EOF
		}
		if err == io.ErrUnexpectedEOF || err == io.EOF {
			return oggPage{}, io.ErrUnexpectedEOF
		}
		return oggPage{}, err
	}

	if !bytes.Equal(header[:4], []byte("OggS")) {
		return oggPage{}, fmt.Errorf("invalid OggS header")
	}
	if header[4] != 0 {
		return oggPage{}, fmt.Errorf("unsupported Ogg version: %d", header[4])
	}

	segmentCount := int(header[26])
	segments := make([]byte, segmentCount)
	if _, err := io.ReadFull(reader, segments); err != nil {
		if err == io.ErrUnexpectedEOF || err == io.EOF {
			return oggPage{}, io.ErrUnexpectedEOF
		}
		return oggPage{}, err
	}

	bodyLen := 0
	for _, segmentLen := range segments {
		bodyLen += int(segmentLen)
	}

	body := make([]byte, bodyLen)
	if _, err := io.ReadFull(reader, body); err != nil {
		if err == io.ErrUnexpectedEOF || err == io.EOF {
			return oggPage{}, io.ErrUnexpectedEOF
		}
		return oggPage{}, err
	}

	return oggPage{
		HeaderType: header[5],
		Segments:   segments,
		Body:       body,
	}, nil
}

func (d *AudioDecoder) RunWavDecoder(startTs int64, isRaw bool) error {
	defer func() {
		close(d.outputOpusChan)
		if d.pipeReader != nil {
			d.pipeReader.Close()
		}
	}()
	var sampleRate int
	var channels int

	if !isRaw {
		// WAV file header is fixed at 44 bytes
		headerSize := 44
		header := make([]byte, headerSize)
		_, err := io.ReadFull(d.pipeReader, header)
		if err != nil {
			return fmt.Errorf("failed to read WAV header: %v", err)
		}

		// Extract basic parameters from WAV header
		// Sample rate: bytes 24-27
		sampleRate = int(uint32(header[24]) | uint32(header[25])<<8 | uint32(header[26])<<16 | uint32(header[27])<<24)
		// Channel count: bytes 22-23
		channels = int(uint16(header[22]) | uint16(header[23])<<8)
		if channels < 1 {
			channels = 1
			log.Warnf("WAV header channel count is 0, defaulting to mono")
		}
		if sampleRate < 1 {
			sampleRate = 24000
			log.Warnf("WAV header sample rate is 0, defaulting to 24000 Hz")
		}
		log.Debugf("WAV format: %d Hz, %d channels", sampleRate, channels)
	} else {
		// For raw PCM data, use the parameters from format
		sampleRate = int(d.format.SampleRate)
		channels = d.format.NumChannels
		if channels < 1 {
			channels = 1
			log.Warnf("PCM channel count is 0, defaulting to mono")
		}
		if sampleRate < 1 {
			sampleRate = 24000
			log.Warnf("PCM sample rate is 0, defaulting to 24000 Hz")
		}
		log.Debugf("Raw PCM format: %d Hz, %d channels", sampleRate, channels)
	}

	// Always output mono
	outputChannels := 1
	if channels > 1 {
		log.Debugf("Converting multi-channel audio to mono output")
	}

	opusSampleRate := int(sampleRate)
	if d.targetSampleRate > 0 {
		opusSampleRate = d.targetSampleRate
	}

	// Create Opus encoder only if target format requires it
	var enc *opus.Encoder
	var err error
	if d.TargetAudioFormat == "opus" {
		enc, err = opus.NewEncoder(opusSampleRate, outputChannels, opus.AppAudio)
		if err != nil {
			return fmt.Errorf("failed to create Opus encoder: %v", err)
		}
		d.enc = enc
	}

	// Opus config and buffers
	frameDurationMs := d.perFrameDurationMs               // frame duration in ms
	frameSize := int(sampleRate) * frameDurationMs / 1000 // samples per frame (based on source sample rate)
	pcmBuffer := make([]int16, frameSize*outputChannels)  // PCM buffer
	opusBuffer := make([]byte, 1000)                      // Opus output buffer

	log.Debugf("WAV/PCM decoder started, source sample rate: %d, target sample rate: %d, frame size: %d, target format: %s", sampleRate, opusSampleRate, frameSize, d.TargetAudioFormat)

	// Buffer for reading raw PCM data
	bytesPerPoint := 2 * channels // 16-bit samples = 2 bytes; aggregate per sample point for multi-channel
	rawBuffer := make([]byte, frameSize*bytesPerPoint)
	remainderBytes := make([]byte, 0, bytesPerPoint*4) // Save misaligned leftover bytes to avoid disrupting subsequent sample boundaries
	currentFramePos := 0
	var firstFrame bool

	flushLastFrame := func() error {
		if currentFramePos <= 0 {
			return nil
		}

		// Create a full frame buffer, padding the remainder with zeros
		paddedFrame := make([]int16, len(pcmBuffer))
		copy(paddedFrame, pcmBuffer[:currentFramePos]) // Copy valid data to the start; remaining defaults to 0

		var opusPcmBuffer []int16 = paddedFrame
		if d.targetSampleRate > 0 && d.targetSampleRate != sampleRate {
			pcmBytes := Int16SliceToBytes(opusPcmBuffer)
			pcmFloat32 := PCM16BytesToFloat32(pcmBytes)
			pcmFloat32 = ResampleLinearFloat32(pcmFloat32, sampleRate, d.targetSampleRate)
			opusPcmBuffer = Float32SliceToInt16Slice(pcmFloat32)
		}

		// Output data based on target format
		if d.TargetAudioFormat == "opus" {
			// Encode the last frame
			n, encodeErr := enc.Encode(opusPcmBuffer, opusBuffer)
			if encodeErr != nil {
				log.Errorf("failed to encode remaining data: %v", encodeErr)
				return fmt.Errorf("failed to encode remaining data: %v", encodeErr)
			}
			frameData := make([]byte, n)
			copy(frameData, opusBuffer[:n])
			select {
			case <-d.ctx.Done():
				log.Debugf("wavDecoder context done, exit")
				return nil
			case d.outputOpusChan <- frameData:
			}
			return nil
		}
		if d.TargetAudioFormat == "pcm" {
			// Output PCM data directly
			pcmData := Int16SliceToBytes(opusPcmBuffer)
			select {
			case <-d.ctx.Done():
				log.Debugf("wavDecoder context done, exit")
				return nil
			case d.outputOpusChan <- pcmData:
			}
		}
		return nil
	}

	for {
		select {
		case <-d.ctx.Done():
			log.Debugf("wavDecoder context done, exit")
			return nil
		default:
			// Read PCM data
			n, readErr := d.pipeReader.Read(rawBuffer)
			if n <= 0 && readErr == nil {
				continue
			}

			var chunk []byte
			if n > 0 {
				chunk = rawBuffer[:n]
				if len(remainderBytes) > 0 {
					combined := make([]byte, 0, len(remainderBytes)+len(chunk))
					combined = append(combined, remainderBytes...)
					combined = append(combined, chunk...)
					chunk = combined
					remainderBytes = remainderBytes[:0]
				}

				alignedBytes := (len(chunk) / bytesPerPoint) * bytesPerPoint
				if alignedBytes < len(chunk) {
					remainderBytes = append(remainderBytes[:0], chunk[alignedBytes:]...)
					chunk = chunk[:alignedBytes]
				}
			}

			// Convert byte data to int16 samples (guaranteed to be aligned on sample boundaries)
			samplesRead := len(chunk) / bytesPerPoint
			for i := 0; i < samplesRead; i++ {
				// For multi-channel, compute the average
				var sampleSum int32
				for ch := 0; ch < channels; ch++ {
					pos := i*bytesPerPoint + ch*2
					sample := int16(uint16(chunk[pos]) | uint16(chunk[pos+1])<<8)
					sampleSum += int32(sample)
				}

				// Compute multi-channel average
				avgSample := int16(sampleSum / int32(channels))
				pcmBuffer[currentFramePos] = avgSample
				currentFramePos++

				// If the buffer is full, encode or output
				if currentFramePos == len(pcmBuffer) {
					if !firstFrame {
						firstFrame = true
						log.Infof("tts cloud->first frame decode latency: %d ms", time.Now().UnixMilli()-startTs)
					}

					var opusPcmBuffer []int16 = pcmBuffer
					if d.targetSampleRate > 0 && d.targetSampleRate != sampleRate {
						pcmBytes := Int16SliceToBytes(opusPcmBuffer)
						pcmFloat32 := PCM16BytesToFloat32(pcmBytes)
						pcmFloat32 = ResampleLinearFloat32(pcmFloat32, sampleRate, d.targetSampleRate)
						opusPcmBuffer = Float32SliceToInt16Slice(pcmFloat32)
					}

					if d.TargetAudioFormat == "opus" {
						// Opus encode and output
						opusLen, err := enc.Encode(opusPcmBuffer, opusBuffer)
						if err != nil {
							log.Errorf("WAV/PCM decode encode failed: %v", err)
							// Skip this frame on encode failure but continue processing
							currentFramePos = 0 // Reset frame position
							continue
						}

						// Copy the current frame to a new slice
						frameData := make([]byte, opusLen)
						copy(frameData, opusBuffer[:opusLen])
						select {
						case <-d.ctx.Done():
							log.Debugf("wavDecoder context done, exit")
							return nil
						case d.outputOpusChan <- frameData:
						}
					} else if d.TargetAudioFormat == "pcm" {
						// Output PCM data directly
						pcmData := Int16SliceToBytes(opusPcmBuffer)
						select {
						case <-d.ctx.Done():
							log.Debugf("wavDecoder context done, exit")
							return nil
						case d.outputOpusChan <- pcmData:
						}
					}
					currentFramePos = 0
				}
			}

			if readErr == io.EOF {
				log.Debugf("WAV/PCM stream read complete, processing remaining data")
				if len(remainderBytes) > 0 {
					log.Warnf("WAV/PCM has misaligned leftover bytes, discarding: %d", len(remainderBytes))
				}
				return flushLastFrame()
			}
			if readErr != nil {
				return fmt.Errorf("failed to read PCM data: %v", readErr)
			}
		}
	}
}

func (d *AudioDecoder) RunMp3Decoder(startTs int64) error {
	defer func() {
		close(d.outputOpusChan)
		if d.pipeReader != nil {
			d.pipeReader.Close()
		}
	}()

	decoder, format, err := mp3.Decode(d.pipeReader)
	if err != nil {
		return fmt.Errorf("failed to create MP3 decoder: %v", err)
	}
	log.Debugf("MP3 format: %d Hz, %d channels", format.SampleRate, format.NumChannels)
	d.streamer = decoder
	d.format = format

	// Stream-decode MP3
	defer func() {
		d.streamer.Close()
	}()

	// Get MP3 audio info
	sampleRate := format.SampleRate
	channels := format.NumChannels

	// Always output mono
	outputChannels := 1
	if channels > 1 {
		log.Debugf("Converting stereo audio to mono output")
	}

	opusSampleRate := int(sampleRate)
	if d.targetSampleRate > 0 {
		opusSampleRate = d.targetSampleRate
	}

	// Create Opus encoder only if target format requires it
	var enc *opus.Encoder
	if d.TargetAudioFormat == "opus" {
		enc, err = opus.NewEncoder(opusSampleRate, outputChannels, opus.AppAudio)
		if err != nil {
			return fmt.Errorf("failed to create Opus encoder: %v", err)
		}
		d.enc = enc
	}

	// Opus config and buffers; create buffer to receive audio samples
	frameDurationMs := d.perFrameDurationMs               // 60ms
	frameSize := int(sampleRate) * frameDurationMs / 1000 // 60ms frame size
	// Temporary PCM storage for converting audio to PCM format
	pcmBuffer := make([]int16, frameSize*outputChannels)

	// MP3 read buffer
	mp3Buffer := make([][2]float64, 2048)

	// Opus output buffer
	opusBuffer := make([]byte, 1000)

	currentFramePos := 0 // Current fill position in pcmBuffer
	var firstFrame bool
	frameCount := 0

	log.Debugf("MP3 decoder started, source sample rate: %d, target sample rate: %d, frame size: %d, target format: %s", int(sampleRate), opusSampleRate, frameSize, d.TargetAudioFormat)

	for {
		select {
		case <-d.ctx.Done():
			log.Debugf("mp3Decoder context done, exit")
			return nil
		default:
			// Read PCM data from MP3
			n, ok := d.streamer.Stream(mp3Buffer)

			if !ok {
				log.Debugf("MP3 stream read complete, processing remaining data")
				// Handle remaining data that does not fill a full frame
				if currentFramePos > 0 {
					// Create a full frame buffer, padding the remainder with zeros
					paddedFrame := make([]int16, len(pcmBuffer))
					copy(paddedFrame, pcmBuffer[:currentFramePos]) // Copy valid data; remaining defaults to 0

					var opusPcmBuffer []int16 = paddedFrame
					if d.targetSampleRate > 0 && d.targetSampleRate != int(sampleRate) {
						pcmBytes := Int16SliceToBytes(opusPcmBuffer)
						pcmFloat32 := PCM16BytesToFloat32(pcmBytes)
						pcmFloat32 = ResampleLinearFloat32(pcmFloat32, int(sampleRate), d.targetSampleRate)
						opusPcmBuffer = Float32SliceToInt16Slice(pcmFloat32)
					}

					// Output data based on target format
					if d.TargetAudioFormat == "opus" {
						// Encode the zero-padded complete frame
						n, err := enc.Encode(opusPcmBuffer, opusBuffer)
						if err != nil {
							log.Errorf("failed to encode remaining data: %v", err)
							return fmt.Errorf("failed to encode remaining data: %v", err)
						} else {
							frameData := make([]byte, n)
							copy(frameData, opusBuffer[:n])

							select {
							case <-d.ctx.Done():
								log.Debugf("mp3Decoder context done, exit")
								return nil
							case d.outputOpusChan <- frameData:
								frameCount++
								log.Debugf("MP3 decode complete, total frames processed: %d", frameCount)
							}
						}
					} else if d.TargetAudioFormat == "pcm" {
						// Output PCM data directly
						pcmData := Int16SliceToBytes(opusPcmBuffer)
						select {
						case <-d.ctx.Done():
							log.Debugf("mp3Decoder context done, exit")
							return nil
						case d.outputOpusChan <- pcmData:
							frameCount++
							log.Debugf("MP3 decode complete, total frames processed: %d", frameCount)
						}
					}
				}
				return nil
			}

			if n == 0 {
				continue
			}

			// Convert floating-point audio data to PCM format (16-bit integer)
			for i := 0; i < n; i++ {
				// Compute mono average in float to avoid integer overflow
				monoSampleFloat := (mp3Buffer[i][0] + mp3Buffer[i][1]) * 0.5

				// Clamp to valid range
				if monoSampleFloat > 1.0 {
					monoSampleFloat = 1.0
				} else if monoSampleFloat < -1.0 {
					monoSampleFloat = -1.0
				}

				// Convert float average to 16-bit integer
				monoSample := int16(monoSampleFloat * 32767.0)
				pcmBuffer[currentFramePos] = monoSample
				currentFramePos++

				// If pcmBuffer has a full frame, encode or output
				if currentFramePos == len(pcmBuffer) {
					if !firstFrame {
						firstFrame = true
						log.Infof("tts cloud->first frame decode latency: %d ms", time.Now().UnixMilli()-startTs)
					}

					var opusPcmBuffer []int16 = pcmBuffer
					if d.targetSampleRate > 0 && d.targetSampleRate != int(sampleRate) {
						pcmBytes := Int16SliceToBytes(opusPcmBuffer)
						pcmFloat32 := PCM16BytesToFloat32(pcmBytes)
						pcmFloat32 = ResampleLinearFloat32(pcmFloat32, int(sampleRate), d.targetSampleRate)
						opusPcmBuffer = Float32SliceToInt16Slice(pcmFloat32)
					}

					if d.TargetAudioFormat == "opus" {
						// Opus encode and output
						opusLen, err := enc.Encode(opusPcmBuffer, opusBuffer)
						if err != nil {
							log.Errorf("MP3 decode encode failed: %v", err)
							// Skip this frame on encode failure but continue processing
							currentFramePos = 0 // Reset frame position
							continue
						}

						// Copy the current frame to a new slice and append to frame array
						frameData := make([]byte, opusLen)
						copy(frameData, opusBuffer[:opusLen])

						select {
						case <-d.ctx.Done():
							log.Debugf("mp3Decoder context done, exit")
							return nil
						case d.outputOpusChan <- frameData:
							frameCount++
							if frameCount%100 == 0 {
								log.Debugf("MP3 decode processed %d frames", frameCount)
							}
						}
					} else if d.TargetAudioFormat == "pcm" {
						// Output PCM data directly
						pcmData := Int16SliceToBytes(opusPcmBuffer)
						select {
						case <-d.ctx.Done():
							log.Debugf("mp3Decoder context done, exit")
							return nil
						case d.outputOpusChan <- pcmData:
							frameCount++
							if frameCount%100 == 0 {
								log.Debugf("MP3 decode processed %d frames", frameCount)
							}
						}
					}

					currentFramePos = 0 // Reset frame position
				}
			}
		}
	}
}

// GetAudioFormatByMimeType returns the audio format string for the given MIME type
func GetAudioFormatByMimeType(mimeType string) string {
	switch mimeType {
	case "audio/mpeg", "audio/mp3", "audio/mpeg3", "audio/x-mpeg-3":
		return "mp3"
	case "audio/wav", "audio/wave", "audio/x-wav":
		return "wav"
	case "audio/pcm", "audio/x-pcm":
		return "pcm"
	case "audio/ogg", "application/ogg":
		return "ogg_opus"
	case "audio/opus":
		return "opus"
	default:
		// Default to mp3 format
		return "mp3"
	}
}

// writeSeekerBuffer implements io.WriteSeeker, wrapping bytes.Buffer
type writeSeekerBuffer struct {
	*bytes.Buffer
	pos int64
}

func newWriteSeekerBuffer() *writeSeekerBuffer {
	return &writeSeekerBuffer{
		Buffer: bytes.NewBuffer(nil),
		pos:    0,
	}
}

func (w *writeSeekerBuffer) Write(p []byte) (n int, err error) {
	// If current position is at the end of the buffer, append directly
	if w.pos == int64(w.Buffer.Len()) {
		n, err = w.Buffer.Write(p)
		w.pos += int64(n)
		return n, err
	}

	// If current position is in the middle of the buffer, write at that position.
	// Get a copy of the current buffer data (avoid modifying the underlying buffer directly)
	data := make([]byte, w.Buffer.Len())
	copy(data, w.Buffer.Bytes())

	// If the write would exceed the current buffer, extend it
	endPos := w.pos + int64(len(p))
	if endPos > int64(len(data)) {
		// Extend the buffer
		extra := int(endPos - int64(len(data)))
		data = append(data, make([]byte, extra)...)
	}

	// Write data at the specified position
	copy(data[w.pos:], p)

	// Update the buffer
	w.Buffer.Reset()
	w.Buffer.Write(data)

	n = len(p)
	w.pos += int64(n)
	return n, nil
}

func (w *writeSeekerBuffer) Seek(offset int64, whence int) (int64, error) {
	var newPos int64
	switch whence {
	case io.SeekStart:
		newPos = offset
	case io.SeekCurrent:
		newPos = w.pos + offset
	case io.SeekEnd:
		newPos = int64(w.Buffer.Len()) + offset
	default:
		return 0, fmt.Errorf("invalid whence: %d", whence)
	}

	if newPos < 0 {
		return 0, fmt.Errorf("negative position")
	}

	// If new position exceeds the current buffer length, extend it
	if newPos > int64(w.Buffer.Len()) {
		// Extend the buffer
		extra := int(newPos - int64(w.Buffer.Len()))
		w.Buffer.Write(make([]byte, extra))
	}

	w.pos = newPos
	return w.pos, nil
}

// PCMFloat32BytesToWav converts a PCM float32 byte array to WAV format.
// audioData: byte array in PCM float32 format (each float32 is 4 bytes, little-endian)
// sampleRate: sample rate
// channels: number of channels
// Returns: byte array in WAV format
func PCMFloat32BytesToWav(audioData []byte, sampleRate, channels int) ([]byte, error) {
	if len(audioData) == 0 {
		return nil, fmt.Errorf("audio data is empty")
	}

	// Convert byte array to float32 slice (little-endian, 4 bytes per float32)
	if len(audioData)%4 != 0 {
		// If not a multiple of 4, truncate to the nearest multiple of 4
		audioData = audioData[:len(audioData)-len(audioData)%4]
	}
	float32Data := make([]float32, len(audioData)/4)
	for i := 0; i < len(float32Data); i++ {
		bits := uint32(audioData[i*4]) | uint32(audioData[i*4+1])<<8 | uint32(audioData[i*4+2])<<16 | uint32(audioData[i*4+3])<<24
		float32Data[i] = math.Float32frombits(bits)
	}

	// Convert float32 to int16
	int16Data := Float32SliceToInt16Slice(float32Data)

	// Create WAV encoder (using writeSeekerBuffer as output)
	wavBuffer := newWriteSeekerBuffer()
	wavEncoder := wav.NewEncoder(wavBuffer, sampleRate, 16, channels, 1)

	// Create audio buffer
	audioBuf := &audio.IntBuffer{
		Format: &audio.Format{
			NumChannels: channels,
			SampleRate:  sampleRate,
		},
		SourceBitDepth: 16,
		Data:           make([]int, len(int16Data)),
	}

	// Convert int16 data to int slice
	for i, sample := range int16Data {
		audioBuf.Data[i] = int(sample)
	}

	// Write WAV file
	if err := wavEncoder.Write(audioBuf); err != nil {
		return nil, fmt.Errorf("failed to write WAV data: %v", err)
	}

	if err := wavEncoder.Close(); err != nil {
		return nil, fmt.Errorf("failed to close WAV encoder: %v", err)
	}

	return wavBuffer.Buffer.Bytes(), nil
}

// OpusFramesToWav converts an array of Opus frames to WAV format.
// opusFrames: array of Opus audio frames (each element is one Opus frame)
// sampleRate: sample rate
// channels: number of channels
// Returns: byte array in WAV format
// Reference: OpusToWav implementation in test/test_audio/audio_utils.go
func OpusFramesToWav(opusFrames [][]byte, sampleRate, channels int) ([]byte, error) {
	if len(opusFrames) == 0 {
		return nil, fmt.Errorf("audio data is empty")
	}

	// Create Opus decoder
	opusDecoder, err := opus.NewDecoder(sampleRate, channels)
	if err != nil {
		return nil, fmt.Errorf("failed to create Opus decoder: %v", err)
	}

	// Create WAV encoder (using writeSeekerBuffer as output)
	wavBuffer := newWriteSeekerBuffer()
	wavEncoder := wav.NewEncoder(wavBuffer, sampleRate, 16, channels, 1)

	// Create audio buffer
	audioBuf := &audio.IntBuffer{
		Format: &audio.Format{
			NumChannels: channels,
			SampleRate:  sampleRate,
		},
		SourceBitDepth: 16,
		Data:           make([]int, 0),
	}

	// PCM buffer for decoding (use 60ms as estimate, large enough to hold one frame)
	perFrameDuration := 60
	pcmBuffer := make([]int16, channels*sampleRate*perFrameDuration/1000)

	// Iterate over all Opus frames and decode
	for _, opusFrame := range opusFrames {
		if len(opusFrame) == 0 {
			continue
		}

		// Decode Opus frame
		n, err := opusDecoder.Decode(opusFrame, pcmBuffer)
		if err != nil {
			return nil, fmt.Errorf("failed to decode Opus frame: %v", err)
		}

		// Convert PCM data to int format and append to buffer
		for i := 0; i < n; i++ {
			audioBuf.Data = append(audioBuf.Data, int(pcmBuffer[i]))
		}
	}

	// Write WAV file
	if len(audioBuf.Data) > 0 {
		if err := wavEncoder.Write(audioBuf); err != nil {
			return nil, fmt.Errorf("failed to write WAV data: %v", err)
		}
	}

	if err := wavEncoder.Close(); err != nil {
		return nil, fmt.Errorf("failed to close WAV encoder: %v", err)
	}

	return wavBuffer.Buffer.Bytes(), nil
}
