//go:build supertonic

package supertonic

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"

	"golang.org/x/text/unicode/norm"
)

var AvailableLangs = []string{
	"en", "ko", "ja", "ar", "bg", "cs", "da", "de", "el", "es",
	"et", "fi", "fr", "hi", "hr", "hu", "id", "it", "lt", "lv",
	"nl", "pl", "pt", "ro", "ru", "sk", "sl", "sv", "tr", "uk", "vi", "na",
}

const maxChunkLength = 300

var abbreviations = []string{
	"Dr.", "Mr.", "Mrs.", "Ms.", "Prof.", "Sr.", "Jr.",
	"St.", "Ave.", "Rd.", "Blvd.", "Dept.", "Inc.", "Ltd.",
	"Co.", "Corp.", "etc.", "vs.", "i.e.", "e.g.", "Ph.D.",
}

func NewUnicodeProcessor(unicodeIndexerPath string) (*UnicodeProcessor, error) {
	indexer, err := loadJSONInt64(unicodeIndexerPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load unicode indexer: %w", err)
	}
	return &UnicodeProcessor{indexer: indexer}, nil
}

func loadJSONInt64(filePath string) ([]int64, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	var result []int64
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (up *UnicodeProcessor) Call(textList []string, langList []string) ([][]int64, [][][]float64) {
	processedTexts := make([]string, len(textList))
	for i, text := range textList {
		processedTexts[i] = preprocessText(text, langList[i])
	}

	textLengths := make([]int64, len(processedTexts))
	maxLen := 0
	for i, text := range processedTexts {
		textLengths[i] = int64(len([]rune(text)))
		if int(textLengths[i]) > maxLen {
			maxLen = int(textLengths[i])
		}
	}

	textIDs := make([][]int64, len(processedTexts))
	for i, text := range processedTexts {
		row := make([]int64, maxLen)
		runes := []rune(text)
		for j, r := range runes {
			unicodeVal := int(r)
			if unicodeVal < len(up.indexer) {
				row[j] = up.indexer[unicodeVal]
			} else {
				row[j] = -1
			}
		}
		textIDs[i] = row
	}

	textMask := lengthToMask(textLengths, maxLen)
	return textIDs, textMask
}

func isValidLang(lang string) bool {
	for _, l := range AvailableLangs {
		if l == lang {
			return true
		}
	}
	return false
}

func preprocessText(text string, lang string) string {
	text = norm.NFKD.String(text)

	emojiPattern := regexp.MustCompile(`[\x{1F600}-\x{1F64F}\x{1F300}-\x{1F5FF}\x{1F680}-\x{1F6FF}\x{1F700}-\x{1F77F}\x{1F780}-\x{1F7FF}\x{1F800}-\x{1F8FF}\x{1F900}-\x{1F9FF}\x{1FA00}-\x{1FA6F}\x{1FA70}-\x{1FAFF}\x{2600}-\x{26FF}\x{2700}-\x{27BF}\x{1F1E6}-\x{1F1FF}]+`)
	text = emojiPattern.ReplaceAllString(text, "")

	replacements := map[string]string{
		"–": "-", "‑": "-", "—": "-", "_": " ",
		"“": "\"", "”": "\"", "‘": "'", "’": "'",
		"´": "'", "`": "'", "[": " ", "]": " ", "|": " ", "/": " ",
		"#": " ", "→": " ", "←": " ",
	}
	for old, newVal := range replacements {
		text = strings.ReplaceAll(text, old, newVal)
	}

	for _, symbol := range []string{"♥", "☆", "♡", "©", "\\"} {
		text = strings.ReplaceAll(text, symbol, "")
	}

	exprReplacements := map[string]string{
		"@": " at ", "e.g.,": "for example, ", "i.e.,": "that is, ",
	}
	for old, newVal := range exprReplacements {
		text = strings.ReplaceAll(text, old, newVal)
	}

	text = regexp.MustCompile(` ,`).ReplaceAllString(text, ",")
	text = regexp.MustCompile(` \.`).ReplaceAllString(text, ".")
	text = regexp.MustCompile(` !`).ReplaceAllString(text, "!")
	text = regexp.MustCompile(` \?`).ReplaceAllString(text, "?")
	text = regexp.MustCompile(` ;`).ReplaceAllString(text, ";")
	text = regexp.MustCompile(` :`).ReplaceAllString(text, ":")
	text = regexp.MustCompile(` '`).ReplaceAllString(text, "'")

	for strings.Contains(text, `""`) {
		text = strings.ReplaceAll(text, `""`, `"`)
	}
	for strings.Contains(text, "''") {
		text = strings.ReplaceAll(text, "''", "'")
	}

	text = regexp.MustCompile(`\s+`).ReplaceAllString(text, " ")
	text = strings.TrimSpace(text)

	if text != "" {
		endsWithPunct := regexp.MustCompile(`[.!?;:,'"\x{201C}\x{201D}\x{2018}\x{2019})\]}…。」』】〉》›»]$`)
		if !endsWithPunct.MatchString(text) {
			text += "."
		}
	}

	if !isValidLang(lang) {
		panic(fmt.Sprintf("invalid language: %s (available: %v)", lang, AvailableLangs))
	}

	return fmt.Sprintf("<%s>%s</%s>", lang, text, lang)
}

func lengthToMask(lengths []int64, maxLen int) [][][]float64 {
	mask := make([][][]float64, len(lengths))
	for i, l := range lengths {
		row := make([]float64, maxLen)
		for j := 0; j < maxLen; j++ {
			if int64(j) < l {
				row[j] = 1.0
			}
		}
		mask[i] = [][]float64{row}
	}
	return mask
}

func getTextMask(textLengths []int64, maxLen int) [][][]float64 {
	return lengthToMask(textLengths, maxLen)
}

func getLatentMask(wavLengths []int64, cfg Config) [][][]float64 {
	baseChunkSize := int64(cfg.AE.BaseChunkSize)
	chunkCompressFactor := int64(cfg.TTL.ChunkCompressFactor)
	latentSize := baseChunkSize * chunkCompressFactor

	latentLengths := make([]int64, len(wavLengths))
	maxLen := int64(0)
	for i, wavLen := range wavLengths {
		latentLengths[i] = (wavLen + latentSize - 1) / latentSize
		if latentLengths[i] > maxLen {
			maxLen = latentLengths[i]
		}
	}
	return lengthToMask(latentLengths, int(maxLen))
}

func chunkText(text string, maxLen int) []string {
	if maxLen == 0 {
		maxLen = maxChunkLength
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return []string{""}
	}

	paragraphs := regexp.MustCompile(`\n\s*\n`).Split(text, -1)
	var chunks []string

	for _, para := range paragraphs {
		para = strings.TrimSpace(para)
		if para == "" {
			continue
		}
		if len(para) <= maxLen {
			chunks = append(chunks, para)
			continue
		}

		sentences := splitSentences(para)
		var current strings.Builder
		currentLen := 0

		for _, sentence := range sentences {
			sentence = strings.TrimSpace(sentence)
			if sentence == "" {
				continue
			}
			sentenceLen := len(sentence)

			if sentenceLen > maxLen {
				if current.Len() > 0 {
					chunks = append(chunks, strings.TrimSpace(current.String()))
					current.Reset()
					currentLen = 0
				}
				parts := strings.Split(sentence, ",")
				for _, part := range parts {
					part = strings.TrimSpace(part)
					if part == "" {
						continue
					}
					partLen := len(part)
					if partLen > maxLen {
						words := strings.Fields(part)
						var wordChunk strings.Builder
						wordChunkLen := 0
						for _, word := range words {
							wordLen := len(word)
							if wordChunkLen+wordLen+1 > maxLen && wordChunk.Len() > 0 {
								chunks = append(chunks, strings.TrimSpace(wordChunk.String()))
								wordChunk.Reset()
								wordChunkLen = 0
							}
							if wordChunk.Len() > 0 {
								wordChunk.WriteString(" ")
								wordChunkLen++
							}
							wordChunk.WriteString(word)
							wordChunkLen += wordLen
						}
						if wordChunk.Len() > 0 {
							chunks = append(chunks, strings.TrimSpace(wordChunk.String()))
						}
					} else {
						if currentLen+partLen+1 > maxLen && current.Len() > 0 {
							chunks = append(chunks, strings.TrimSpace(current.String()))
							current.Reset()
							currentLen = 0
						}
						if current.Len() > 0 {
							current.WriteString(", ")
							currentLen += 2
						}
						current.WriteString(part)
						currentLen += partLen
					}
				}
				continue
			}

			if currentLen+sentenceLen+1 > maxLen && current.Len() > 0 {
				chunks = append(chunks, strings.TrimSpace(current.String()))
				current.Reset()
				currentLen = 0
			}
			if current.Len() > 0 {
				current.WriteString(" ")
				currentLen++
			}
			current.WriteString(sentence)
			currentLen += sentenceLen
		}
		if current.Len() > 0 {
			chunks = append(chunks, strings.TrimSpace(current.String()))
		}
	}

	if len(chunks) == 0 {
		return []string{""}
	}
	return chunks
}

func splitSentences(text string) []string {
	re := regexp.MustCompile(`([.!?])\s+`)
	matches := re.FindAllStringIndex(text, -1)
	if len(matches) == 0 {
		return []string{text}
	}

	var sentences []string
	lastEnd := 0
	for _, match := range matches {
		beforePunc := text[lastEnd:match[0]]
		isAbbrev := false
		for _, abbrev := range abbreviations {
			if strings.HasSuffix(strings.TrimSpace(beforePunc+text[match[0]:match[0]+1]), abbrev) {
				isAbbrev = true
				break
			}
		}
		if !isAbbrev {
			sentences = append(sentences, text[lastEnd:match[1]])
			lastEnd = match[1]
		}
	}
	if lastEnd < len(text) {
		sentences = append(sentences, text[lastEnd:])
	}
	if len(sentences) == 0 {
		return []string{text}
	}
	return sentences
}
