package main

import (
	"fmt"
	"regexp"
	"strings"
	"sync"
)

var (
	//Define a collection of punctuation marks
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

	//Object pool for reuse
	builderPool = sync.Pool{
		New: func() interface{} {
			return &strings.Builder{}
		},
	}

	//Slice pool used to store results
	runeSlicePool = sync.Pool{
		New: func() interface{} {
			slice := make([]rune, 0, 1024)
			return &slice
		},
	}

	//Precompiled regular expressions
	numberPrefixRegex = regexp.MustCompile(`(?m)^[\s]*\d{1,3}\.$`)
)

// Use fast character checking alternative regex
func isNumberPrefix(text []rune, pos int) bool {
	if pos <= 0 || text[pos] != '.' {
		return false
	}

	//Look forward to start of line or newline character
	start := pos - 1
	digitCount := 0
	foundDigit := false

	//Skip whitespace characters before dot
	for start >= 0 && (text[start] == ' ' || text[start] == '\t') {
		start--
	}

	//statistics
	for start >= 0 && text[start] >= '0' && text[start] <= '9' {
		digitCount++
		foundDigit = true
		if digitCount > 3 { //Numbers exceeding 3 digits are not legal serial numbers
			return false
		}
		start--
	}

	//Check if a number is preceded by a whitespace character or the beginning of a line
	if start >= 0 && text[start] != ' ' && text[start] != '\t' && text[start] != '\n' {
		return false
	}

	return foundDigit
}

// Remove leading and trailing whitespace characters
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

func findLastPunctuation(text []rune) int {
	//Find the last punctuation mark from back to front
	lastPos := -1
	for i := len(text) - 1; i >= 0; i-- {
		//Check if it is a punctuation mark
		if punctuationMap[text[i]] {
			//If it is a dot number, check whether it is part of the sequence number
			if text[i] == '.' && isNumberPrefix(text, i) {
				continue
			}
			return i
		}
	}
	return lastPos
}

func findNextSplitPoint(text []rune, startPos int, maxLen int) int {
	//Calculate the end position of the search
	endPos := startPos + maxLen
	if endPos > len(text) {
		endPos = len(text)
	}

	//Search from front to back
	for i := startPos; i < endPos; i++ {
		//Check whether it is a newline character, and also check whether the next line is a sequence number
		if text[i] == '\n' {
			nextPos := i + 1
			//Skip whitespace characters
			for nextPos < endPos && (text[nextPos] == ' ' || text[nextPos] == '\t') {
				nextPos++
			}
			//Check if it is the beginning of sequence number
			if nextPos < endPos-2 && text[nextPos] >= '0' && text[nextPos] <= '9' {
				return i
			}
			continue
		}

		//Use map to check if it is a punctuation mark
		if punctuationMap[text[i]] {
			return i
		}
	}

	//If not found within the maxLen range, try to find within a larger range
	if endPos < len(text) {
		for i := endPos; i < len(text); i++ {
			if text[i] == '\n' || punctuationMap[text[i]] {
				return i
			}
		}
	}

	return -1
}

func extractSmartSentences(text string, minLen, maxLen int) (sentences []string, remaining string) {
	//Pre-allocate a reasonable slice capacity
	estimatedCount := len(text) / 50
	if estimatedCount < 10 {
		estimatedCount = 10
	}
	sentences = make([]string, 0, estimatedCount)

	//Convert to rune slice in one go
	currentRunes := []rune(text)
	startPos := 0

	//Get reused objects from the object pool
	builder := builderPool.Get().(*strings.Builder)
	defer builderPool.Put(builder)
	builder.Grow(maxLen * 2)

	//Get temporary rune slice
	tempRunesPtr := runeSlicePool.Get().(*[]rune)
	tempRunes := (*tempRunesPtr)[:0]
	defer runeSlicePool.Put(tempRunesPtr)

	for startPos < len(currentRunes) {
		//Skip leading whitespace characters
		for startPos < len(currentRunes) && (currentRunes[startPos] == ' ' || currentRunes[startPos] == '\t' || currentRunes[startPos] == '\n') {
			startPos++
		}

		if startPos >= len(currentRunes) {
			break
		}

		//Find next split point
		splitPos := findNextSplitPoint(currentRunes, startPos, maxLen)
		if splitPos == -1 {
			//No split point is found, the remaining text is treated as remaining
			segment := trimSpaceRunes(currentRunes[startPos:])
			if len(segment) > 0 {
				remaining = string(segment)
			}
			break
		}

		//Extract current paragraph
		builder.Reset()
		tempRunes = tempRunes[:0]

		//Collect and process the current paragraph
		segment := trimSpaceRunes(currentRunes[startPos : splitPos+1])

		//Check if the paragraph meets the minimum length requirements and ends with punctuation
		if len(segment) >= minLen && punctuationMap[segment[len(segment)-1]] {
			sentences = append(sentences, string(segment))
		} else {
			//If the conditions are not met, add it to remaining
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

func main() {
	text := `厚,人家就晓得你又在敷衍我!每次问你都没有,你是不是不喜欢我了啦?哼,人家要生气喽!不跟你好了!除非...你答应我,等下带人家去夜市吃豆花啦~还要牵人家手手逛大街,一路上都要逗人家笑,逗得人家开心到飞上天!不然人家真的会不理你哦~`
	sentences, remaining := extractSmartSentences(text, 3, 200)
	for i, sentence := range sentences {
		fmt.Printf("\n sentence %d:\n%s\n", i+1, sentence)
	}
	if remaining != "" {
		fmt.Printf("\n remaining:\n%s\n", remaining)
	}
}
