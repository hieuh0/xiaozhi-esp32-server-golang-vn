package util

import (
	"bytes"
	"strings"
	"sync"
	"unicode"
)

var (
	// punctuationMap maps sentence-ending and pause punctuation
	punctuationMap = map[rune]bool{
		'。':  true,
		'？':  true,
		'！':  true,
		'；':  true,
		'：':  true,
		'\n': true,
		'.':  true,
		'?':  true,
		'!':  true,
		';':  true,
		':':  true,
	}

	// firstPunctuation maps punctuation used during first-pass processing (includes comma)
	firstPunctuation = map[rune]bool{
		'，':  true,
		',':  true,
		'。':  true,
		'？':  true,
		'！':  true,
		'；':  true,
		'：':  true,
		'\n': true,
		'.':  true,
		'?':  true,
		'!':  true,
		';':  true,
		':':  true,
	}

	// sentence-ending punctuation
	sentenceEndPunctuation = []rune{'.', '。', '!', '！', '?', '？', '\n'}

	// sentence-pause punctuation (can serve as break points for long sentences)
	sentencePausePunctuation = []rune{',', '，', ';', '；', ':', '：'}

	// object pool for builder reuse
	builderPool = sync.Pool{
		New: func() interface{} {
			return &strings.Builder{}
		},
	}

	// slice pool for storing results
	runeSlicePool = sync.Pool{
		New: func() interface{} {
			slice := make([]rune, 0, 1024)
			return &slice
		},
	}
)

// IsSentenceEndPunctuation reports whether a rune is a sentence-ending punctuation mark
func IsSentenceEndPunctuation(r rune) bool {
	for _, p := range sentenceEndPunctuation {
		if r == p {
			return true
		}
	}
	return false
}

// IsSentencePausePunctuation reports whether a rune is a sentence-pause punctuation mark
func IsSentencePausePunctuation(r rune) bool {
	for _, p := range sentencePausePunctuation {
		if r == p {
			return true
		}
	}
	return false
}

// IsNumberWithDot reports whether the string is a number-dot format (e.g. "1.", "2.")
func IsNumberWithDot(s string) bool {
	trimmed := strings.TrimSpace(s)
	if len(trimmed) < 2 || trimmed[len(trimmed)-1] != '.' {
		return false
	}

	for i := 0; i < len(trimmed)-1; i++ {
		if !unicode.IsDigit(rune(trimmed[i])) {
			return false
		}
	}
	return true
}

// ExtractCompleteSentences extracts complete sentences from text.
// Returns a slice of complete sentences and the remaining incomplete content.
func ExtractCompleteSentences(text string) ([]string, string) {
	if text == "" {
		return []string{}, ""
	}

	var sentences []string
	var currentSentence bytes.Buffer

	runes := []rune(text)
	lastIndex := len(runes) - 1

	for i, r := range runes {
		currentSentence.WriteRune(r)

		// Check whether the sentence has ended
		if IsSentenceEndPunctuation(r) {
			// Sentence-ending punctuation found
			sentence := strings.TrimSpace(currentSentence.String())
			if sentence != "" {
				sentences = append(sentences, sentence)
			}
			currentSentence.Reset()
		} else if i == lastIndex {
			// Last character but not a sentence-ending punctuation; keep in remaining
			break
		}
	}

	// Return the current incomplete sentence as remaining
	remaining := currentSentence.String()
	return sentences, strings.TrimSpace(remaining)
}

// isNumberPrefix uses fast character checks instead of regex to detect a numeric list prefix
func isNumberPrefix(text []rune, pos int) bool {
	if pos <= 0 || text[pos] != '.' {
		return false
	}

	// Scan backward to find line start or newline
	start := pos - 1
	digitCount := 0
	foundDigit := false

	// Skip whitespace before the dot
	for start >= 0 && (text[start] == ' ' || text[start] == '\t') {
		start--
	}

	// Count digits
	for start >= 0 && text[start] >= '0' && text[start] <= '9' {
		digitCount++
		foundDigit = true
		if digitCount > 3 { // More than 3 digits is not a valid list number
			return false
		}
		start--
	}

	// Check that what precedes the digits is whitespace or the start of a line
	if start >= 0 && text[start] != ' ' && text[start] != '\t' && text[start] != '\n' {
		return false
	}

	return foundDigit
}

// trimSpaceRunes strips leading and trailing whitespace from a rune slice
func trimSpaceRunes(text []rune) []rune {
	start, end := 0, len(text)-1

	for start <= end && (text[start] == ' ' || text[start] == '\t' || text[start] == '\n') {
		start++
	}

	for end >= start && (text[end] == ' ' || text[end] == '\t' || text[end] == '\n') {
		end--
	}

	if start > end {
		return nil
	}
	return text[start : end+1]
}

func isDigitAdjacentColon(text []rune, pos int) bool {
	if pos < 0 || pos >= len(text) {
		return false
	}

	colon := text[pos]
	if colon != ':' && colon != '：' {
		return false
	}

	if pos == 0 || !unicode.IsDigit(text[pos-1]) {
		return false
	}

	if pos == len(text)-1 {
		return true
	}

	return unicode.IsDigit(text[pos+1])
}

// findLastPunctuation searches backward for the last punctuation mark
func findLastPunctuation(text []rune, separatorMap map[rune]bool) int {
	lastPos := -1
	for i := len(text) - 1; i >= 0; i-- {
		// Check whether it is a punctuation mark
		if separatorMap[text[i]] {
			// If it is a dot, check whether it is part of a list number
			if text[i] == '.' && isNumberPrefix(text, i) {
				continue
			}
			if isDigitAdjacentColon(text, i) {
				continue
			}
			return i
		}
	}
	return lastPos
}

// findNextSplitPoint finds the next split point in text
func findNextSplitPoint(text []rune, startPos int, maxLen int, separatorMap map[rune]bool) int {
	// Calculate the end position to search up to
	endPos := startPos + maxLen
	if endPos > len(text) {
		endPos = len(text)
	}

	// Search forward
	for i := startPos; i < endPos; i++ {
		// Check for newline; also check whether the next line starts with a list number
		if text[i] == '\n' {
			nextPos := i + 1
			// Skip whitespace
			for nextPos < endPos && (text[nextPos] == ' ' || text[nextPos] == '\t') {
				nextPos++
			}
			// Check whether a list number begins here
			if nextPos < endPos-2 && text[nextPos] >= '0' && text[nextPos] <= '9' {
				return i
			}
			continue
		}

		// Use map to check whether it is a punctuation mark
		if separatorMap[text[i]] {
			if isDigitAdjacentColon(text, i) {
				continue
			}
			return i
		}
	}

	// If no split point found within maxLen, search a wider range
	if endPos < len(text) {
		for i := endPos; i < len(text); i++ {
			if text[i] == '\n' {
				return i
			}
			if separatorMap[text[i]] {
				if isDigitAdjacentColon(text, i) {
					continue
				}
				return i
			}
		}
	}

	return -1
}

// ExtractSmartSentences intelligently extracts sentences from text.
// text: text to process
// minLen: minimum sentence length
// maxLen: maximum sentence length
// isFirst: whether this is the first pass (first pass allows comma as separator)
func ExtractSmartSentences(text string, minLen, maxLen int, isFirst bool) (sentences []string, remaining string) {
	// When isFirst is true, relax to allow comma as separator
	separatorMap := punctuationMap
	if isFirst {
		separatorMap = firstPunctuation
	}
	// Pre-allocate a reasonable slice capacity
	estimatedCount := len(text) / 50
	if estimatedCount < 10 {
		estimatedCount = 10
	}
	sentences = make([]string, 0, estimatedCount)

	// Convert to rune slice once
	currentRunes := []rune(text)
	startPos := 0

	// Get reusable objects from the pool
	builder := builderPool.Get().(*strings.Builder)
	defer builderPool.Put(builder)
	builder.Grow(maxLen * 2)

	// Get a temporary rune slice from the pool
	tempRunesPtr := runeSlicePool.Get().(*[]rune)
	tempRunes := (*tempRunesPtr)[:0]
	defer runeSlicePool.Put(tempRunesPtr)

	for startPos < len(currentRunes) {
		// Skip leading whitespace
		for startPos < len(currentRunes) && (currentRunes[startPos] == ' ' || currentRunes[startPos] == '\t' || currentRunes[startPos] == '\n') {
			startPos++
		}

		if startPos >= len(currentRunes) {
			break
		}

		// Find the next split point
		splitPos := findNextSplitPoint(currentRunes, startPos, maxLen, separatorMap)
		if splitPos == -1 {
			// No split point found; treat remaining text as remaining
			segment := trimSpaceRunes(currentRunes[startPos:])
			if len(segment) > 0 {
				remaining = string(segment)
			}
			break
		}

		// Extract the current segment
		builder.Reset()
		tempRunes = tempRunes[:0]

		// Collect and process the current segment
		segment := trimSpaceRunes(currentRunes[startPos : splitPos+1])

		// Check whether the segment meets the minimum length and ends with punctuation
		if len(segment) >= minLen && separatorMap[segment[len(segment)-1]] {
			sentences = append(sentences, string(segment))
		} else {
			// Does not meet conditions; add to remaining
			if len(segment) > 0 {
				if len(remaining) > 0 {
					remaining += " "
				}
				remaining += string(segment)
			}
		}

		startPos = splitPos + 1
	}

	return sentences, remaining
}

// ContainsSentenceSeparator reports whether the string contains a separator (sentence-ending or pause punctuation)
func ContainsSentenceSeparator(s string, isFirst bool) bool {
	separatorMap := punctuationMap
	if isFirst {
		separatorMap = firstPunctuation
	}

	runes := []rune(s)
	for i, r := range runes {
		if !separatorMap[r] {
			continue
		}
		if isDigitAdjacentColon(runes, i) {
			continue
		}
		return true
	}

	return false
}
