//go:build supertonic

package supertonic

import (
	"fmt"
	"math"
	"math/rand"
	"time"

	ort "github.com/yalue/onnxruntime_go"
)

func (tts *TextToSpeech) sampleNoisyLatent(durOnnx []float32) ([][][]float64, [][][]float64) {
	bsz := len(durOnnx)
	maxDur := float64(0)
	for _, d := range durOnnx {
		if float64(d) > maxDur {
			maxDur = float64(d)
		}
	}

	wavLengths := make([]int64, bsz)
	for i, d := range durOnnx {
		wavLengths[i] = int64(float64(d) * float64(tts.SampleRate))
	}

	chunkSize := tts.baseChunkSize * tts.chunkCompress
	latentLen := int((maxDur*float64(tts.SampleRate) + float64(chunkSize) - 1) / float64(chunkSize))
	latentDim := tts.ldim * tts.chunkCompress

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	noisyLatent := make([][][]float64, bsz)
	for b := 0; b < bsz; b++ {
		batch := make([][]float64, latentDim)
		for d := 0; d < latentDim; d++ {
			row := make([]float64, latentLen)
			for t := 0; t < latentLen; t++ {
				const eps = 1e-10
				u1 := math.Max(eps, rng.Float64())
				u2 := rng.Float64()
				row[t] = math.Sqrt(-2.0*math.Log(u1)) * math.Cos(2.0*math.Pi*u2)
			}
			batch[d] = row
		}
		noisyLatent[b] = batch
	}

	latentMask := getLatentMask(wavLengths, tts.cfg)
	for b := 0; b < bsz; b++ {
		for d := 0; d < latentDim; d++ {
			for t := 0; t < latentLen; t++ {
				noisyLatent[b][d][t] *= latentMask[b][0][t]
			}
		}
	}

	return noisyLatent, latentMask
}

func (tts *TextToSpeech) _infer(textList []string, langList []string, style *Style, totalStep int, speed float32) ([]float32, []float32, error) {
	bsz := len(textList)

	textIDs, textMask := tts.textProcessor.Call(textList, langList)
	textIDsShape := []int64{int64(bsz), int64(len(textIDs[0]))}
	textMaskShape := []int64{int64(bsz), 1, int64(len(textMask[0][0]))}

	textIDsTensor := IntArrayToTensor(textIDs, textIDsShape)
	defer textIDsTensor.Destroy()
	textMaskTensor := ArrayToTensor(textMask, textMaskShape)
	defer textMaskTensor.Destroy()

	dpOutputs := []ort.Value{nil}
	if err := tts.dpOrt.Run([]ort.Value{textIDsTensor, style.DpTensor, textMaskTensor}, dpOutputs); err != nil {
		return nil, nil, fmt.Errorf("failed to run duration predictor: %w", err)
	}
	durTensor := dpOutputs[0].(*ort.Tensor[float32])
	defer durTensor.Destroy()
	durOnnx := durTensor.GetData()
	for i := range durOnnx {
		durOnnx[i] /= speed
	}

	textIDsTensor2 := IntArrayToTensor(textIDs, textIDsShape)
	defer textIDsTensor2.Destroy()
	textEncOutputs := []ort.Value{nil}
	if err := tts.textEncOrt.Run([]ort.Value{textIDsTensor2, style.TtlTensor, textMaskTensor}, textEncOutputs); err != nil {
		return nil, nil, fmt.Errorf("failed to run text encoder: %w", err)
	}
	textEmbTensor := textEncOutputs[0].(*ort.Tensor[float32])
	defer textEmbTensor.Destroy()

	xt, latentMask := tts.sampleNoisyLatent(durOnnx)
	latentShape := []int64{int64(bsz), int64(len(xt[0])), int64(len(xt[0][0]))}
	latentMaskShape := []int64{int64(bsz), 1, int64(len(latentMask[0][0]))}
	scalarShape := []int64{int64(bsz)}

	totalStepArr := make([]float32, bsz)
	for b := range totalStepArr {
		totalStepArr[b] = float32(totalStep)
	}
	totalStepTensor, _ := ort.NewTensor(scalarShape, totalStepArr)
	defer totalStepTensor.Destroy()

	for step := 0; step < totalStep; step++ {
		currentStepArr := make([]float32, bsz)
		for b := range currentStepArr {
			currentStepArr[b] = float32(step)
		}
		currentStepTensor, _ := ort.NewTensor(scalarShape, currentStepArr)
		noisyLatentTensor := ArrayToTensor(xt, latentShape)
		latentMaskTensor := ArrayToTensor(latentMask, latentMaskShape)
		textMaskTensor2 := ArrayToTensor(textMask, textMaskShape)

		vectorEstOutputs := []ort.Value{nil}
		err := tts.vectorEstOrt.Run(
			[]ort.Value{noisyLatentTensor, textEmbTensor, style.TtlTensor, latentMaskTensor, textMaskTensor2, currentStepTensor, totalStepTensor},
			vectorEstOutputs,
		)
		if err != nil {
			noisyLatentTensor.Destroy()
			latentMaskTensor.Destroy()
			textMaskTensor2.Destroy()
			currentStepTensor.Destroy()
			return nil, nil, fmt.Errorf("failed to run vector estimator at step %d: %w", step, err)
		}

		denoisedTensor := vectorEstOutputs[0].(*ort.Tensor[float32])
		denoisedData := denoisedTensor.GetData()
		idx := 0
		for b := 0; b < bsz; b++ {
			for d := range xt[b] {
				for t := range xt[b][d] {
					xt[b][d][t] = float64(denoisedData[idx])
					idx++
				}
			}
		}

		noisyLatentTensor.Destroy()
		latentMaskTensor.Destroy()
		textMaskTensor2.Destroy()
		currentStepTensor.Destroy()
		denoisedTensor.Destroy()
	}

	finalLatentTensor := ArrayToTensor(xt, latentShape)
	defer finalLatentTensor.Destroy()

	vocoderOutputs := []ort.Value{nil}
	if err := tts.vocoderOrt.Run([]ort.Value{finalLatentTensor}, vocoderOutputs); err != nil {
		return nil, nil, fmt.Errorf("failed to run vocoder: %w", err)
	}
	wavBatchTensor := vocoderOutputs[0].(*ort.Tensor[float32])
	defer wavBatchTensor.Destroy()

	return wavBatchTensor.GetData(), durOnnx, nil
}

func (tts *TextToSpeech) Call(text string, lang string, style *Style, totalStep int, speed float32, silenceDuration float32) ([]float32, float32, error) {
	maxLen := 300
	if lang == "ko" || lang == "ja" {
		maxLen = 120
	}
	chunks := chunkText(text, maxLen)

	var wavCat []float32
	var durCat float32

	for i, chunk := range chunks {
		wavData, duration, err := tts._infer([]string{chunk}, []string{lang}, style, totalStep, speed)
		if err != nil {
			return nil, 0, err
		}
		dur := duration[0]
		wavLen := int(float32(tts.SampleRate) * dur)
		wavChunk := wavData[:wavLen]

		if i == 0 {
			wavCat = wavChunk
			durCat = dur
		} else {
			silence := make([]float32, int(silenceDuration*float32(tts.SampleRate)))
			wavCat = append(wavCat, silence...)
			wavCat = append(wavCat, wavChunk...)
			durCat += silenceDuration + dur
		}
	}
	return wavCat, durCat, nil
}

func (tts *TextToSpeech) Destroy() {
	if tts.dpOrt != nil {
		tts.dpOrt.Destroy()
	}
	if tts.textEncOrt != nil {
		tts.textEncOrt.Destroy()
	}
	if tts.vectorEstOrt != nil {
		tts.vectorEstOrt.Destroy()
	}
	if tts.vocoderOrt != nil {
		tts.vocoderOrt.Destroy()
	}
}
