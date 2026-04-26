// Package util provides utility functions for the TUI.
package util

import (
	"strings"
	"unicode"
)

// Wrap wraps text at the specified width, preserving word boundaries.
// Returns a slice of wrapped lines.
func Wrap(text string, width int) []string {
	if width <= 0 {
		return []string{text}
	}

	var lines []string
	var currentLine strings.Builder
	var currentWidth int

	words := splitIntoWords(text)

	for _, word := range words {
		wordWidth := StringWidth(word)

		// Handle words that are longer than the width
		if wordWidth > width {
			// Flush current line if it has content
			if currentWidth > 0 {
				lines = append(lines, currentLine.String())
				currentLine.Reset()
				currentWidth = 0
			}
			// Break long word
			for i := 0; i < len(word); {
				runeWidth := 0
				end := i
				for end < len(word) && runeWidth < width {
					r, size := decodeRune(word, end)
					rw := runeWidthFunc(r)
					if runeWidth+rw > width {
						break
					}
					runeWidth += rw
					end += size
				}
				if end > i {
					lines = append(lines, word[i:end])
				}
				i = end
			}
			continue
		}

		// Check if word fits on current line
		spaceNeeded := 0
		if currentWidth > 0 {
			spaceNeeded = 1 // Space before word
		}

		if currentWidth+spaceNeeded+wordWidth <= width {
			// Word fits on current line
			if currentWidth > 0 {
				currentLine.WriteByte(' ')
				currentWidth++
			}
			currentLine.WriteString(word)
			currentWidth += wordWidth
		} else {
			// Word doesn't fit, start new line
			if currentWidth > 0 {
				lines = append(lines, currentLine.String())
				currentLine.Reset()
			}
			currentLine.WriteString(word)
			currentWidth = wordWidth
		}
	}

	// Don't forget the last line
	if currentWidth > 0 {
		lines = append(lines, currentLine.String())
	}

	return lines
}

// WrapPreserveIndent wraps text while preserving leading indentation.
// Useful for code blocks and quoted text.
func WrapPreserveIndent(text string, width int) []string {
	lines := strings.Split(text, "\n")
	var result []string

	for _, line := range lines {
		// Get leading whitespace
		trimmed := strings.TrimLeftFunc(line, unicode.IsSpace)
		indent := line[:len(line)-len(trimmed)]
		indentWidth := StringWidth(indent)

		if trimmed == "" {
			result = append(result, line)
			continue
		}

		// Calculate available width for content
		contentWidth := width - indentWidth
		if contentWidth < 10 {
			// Not enough space, just add as-is
			result = append(result, line)
			continue
		}

		// Wrap the content
		wrappedContent := Wrap(trimmed, contentWidth)

		// Add indentation back to each wrapped line
		for i, content := range wrappedContent {
			if i == 0 {
				result = append(result, indent+content)
			} else {
				// Continuation lines get the same indentation
				result = append(result, indent+content)
			}
		}
	}

	return result
}

// WrapPreserveNewlines wraps text but preserves explicit newlines.
func WrapPreserveNewlines(text string, width int) []string {
	lines := strings.Split(text, "\n")
	var result []string

	for _, line := range lines {
		if line == "" {
			// Preserve empty lines
			result = append(result, "")
		} else {
			wrapped := Wrap(line, width)
			result = append(result, wrapped...)
		}
	}

	return result
}

// Truncate truncates text to the specified width, adding ellipsis if truncated.
func Truncate(text string, width int, ellipsis string) string {
	if width <= 0 {
		return ""
	}

	textWidth := StringWidth(text)
	if textWidth <= width {
		return text
	}

	ellipsisWidth := StringWidth(ellipsis)
	if width <= ellipsisWidth {
		return ellipsis[:width]
	}

	// Find the point where we should truncate
	targetWidth := width - ellipsisWidth
	result := ""
	currentWidth := 0

	for i := 0; i < len(text); {
		r, size := decodeRune(text, i)
		rw := runeWidthFunc(r)
		if currentWidth+rw > targetWidth {
			break
		}
		result += string(r)
		currentWidth += rw
		i += size
	}

	return result + ellipsis
}

// StringWidth returns the display width of a string.
// Accounts for wide characters (CJK, emoji) and ANSI escape codes.
func StringWidth(s string) int {
	width := 0
	inEscape := false

	for i := 0; i < len(s); {
		// Check for ANSI escape sequence
		if s[i] == '\x1b' && i+1 < len(s) && s[i+1] == '[' {
			inEscape = true
			i += 2
			continue
		}
		if inEscape {
			if s[i] == 'm' {
				inEscape = false
			}
			i++
			continue
		}

		r, size := decodeRune(s, i)
		width += runeWidthFunc(r)
		i += size
	}

	return width
}

// StripANSI removes ANSI escape codes from text.
func StripANSI(s string) string {
	var result strings.Builder
	inEscape := false

	for i := 0; i < len(s); {
		if s[i] == '\x1b' && i+1 < len(s) && s[i+1] == '[' {
			inEscape = true
			i += 2
			continue
		}
		if inEscape {
			if s[i] == 'm' {
				inEscape = false
			}
			i++
			continue
		}
		result.WriteByte(s[i])
		i++
	}

	return result.String()
}

// splitIntoWords splits text into words, preserving spaces within reason.
func splitIntoWords(text string) []string {
	return strings.Fields(text)
}

// decodeRune decodes a UTF-8 rune at position i.
func decodeRune(s string, i int) (rune, int) {
	if i >= len(s) {
		return 0, 0
	}

	// Fast path for ASCII
	if s[i] < 0x80 {
		return rune(s[i]), 1
	}

	// Decode UTF-8
	r, size := utf8DecodeRune(s[i:])
	return r, size
}

// utf8DecodeRune decodes a UTF-8 rune from a byte slice.
func utf8DecodeRune(s string) (rune, int) {
	if len(s) == 0 {
		return 0, 0
	}

	b0 := s[0]

	// 1-byte sequence (0xxxxxxx)
	if b0 < 0x80 {
		return rune(b0), 1
	}

	// 2-byte sequence (110xxxxx 10xxxxxx)
	if b0 < 0xE0 && len(s) >= 2 {
		return rune((int(b0&0x1F) << 6) | int(s[1]&0x3F)), 2
	}

	// 3-byte sequence (1110xxxx 10xxxxxx 10xxxxxx)
	if b0 < 0xF0 && len(s) >= 3 {
		return rune((int(b0&0x0F) << 12) | (int(s[1]&0x3F) << 6) | int(s[2]&0x3F)), 3
	}

	// 4-byte sequence (11110xxx 10xxxxxx 10xxxxxx 10xxxxxx)
	if len(s) >= 4 {
		return rune((int(b0&0x07) << 18) | (int(s[1]&0x3F) << 12) | (int(s[2]&0x3F) << 6) | int(s[3]&0x3F)), 4
	}

	// Invalid sequence, return replacement character
	return '\uFFFD', 1
}

// runeWidthFunc returns the display width of a rune.
// Uses East Asian Width for CJK characters.
func runeWidthFunc(r rune) int {
	// Control characters and formatters have 0 width
	if r < 0x20 || (r >= 0x7F && r < 0xA0) {
		return 0
	}

	// Check for zero-width characters
	if isZeroWidth(r) {
		return 0
	}

	// Check for wide characters (CJK, emoji, etc.)
	if isWide(r) {
		return 2
	}

	// Normal ASCII and other characters have width 1
	return 1
}

// isZeroWidth checks if a rune has zero display width.
func isZeroWidth(r rune) bool {
	// Zero-width joiners and modifiers
	if r >= 0x200B && r <= 0x200F {
		return true // Zero-width spaces
	}
	if r >= 0xFE00 && r <= 0xFE0F {
		return true // Variation selectors
	}
	if r == 0xFEFF {
		return true // Byte order mark
	}

	// Combining characters
	if r >= 0x0300 && r <= 0x036F {
		return true // Combining diacritical marks
	}
	if r >= 0x1AB0 && r <= 0x1AFF {
		return true // Combining diacritical marks extended
	}
	if r >= 0x1DC0 && r <= 0x1DFF {
		return true // Combining diacritical marks supplement
	}

	return false
}

// isWide checks if a rune has double display width.
func isWide(r rune) bool {
	// CJK Unified Ideographs
	if r >= 0x4E00 && r <= 0x9FFF {
		return true
	}
	if r >= 0x3400 && r <= 0x4DBF {
		return true // CJK Extension A
	}
	if r >= 0x20000 && r <= 0x2A6DF {
		return true // CJK Extension B
	}

	// Hangul Syllables
	if r >= 0xAC00 && r <= 0xD7AF {
		return true
	}

	// Fullwidth forms
	if r >= 0xFF01 && r <= 0xFF60 {
		return true
	}
	if r >= 0xFFE0 && r <= 0xFFE6 {
		return true
	}

	// Emoji (simplified check)
	if r >= 0x1F300 && r <= 0x1F9FF {
		return true
	}

	return false
}
