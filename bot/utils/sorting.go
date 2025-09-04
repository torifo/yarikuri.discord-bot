package utils

import "unicode"

// IsJapanese は文字列に日本語文字が含まれているかチェックする
func IsJapanese(s string) bool {
	for _, r := range s {
		if unicode.In(r, unicode.Hiragana, unicode.Katakana, unicode.Han) {
			return true
		}
	}
	return false
}

// SortJapaneseFirst は日本語を優先してソートするための比較関数
func SortJapaneseFirst(s1, s2 string) bool {
	isJp1, isJp2 := IsJapanese(s1), IsJapanese(s2)
	if isJp1 != isJp2 {
		return isJp1
	}
	return s1 < s2
}