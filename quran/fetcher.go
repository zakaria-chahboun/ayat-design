package quran

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// Surah represents a chapter in the Quran
type Surah struct {
	ID        int     `json:"id"`
	Name      string  `json:"name"`
	Translit  string  `json:"transliteration"`
	Type      string  `json:"type"`
	TotalAyah int     `json:"total_verses"`
	Verses    []Verse `json:"verses"`
}

// Verse represents a single Ayah
type Verse struct {
	ID   int    `json:"id"`
	Text string `json:"text"`
}

// Data holds the entire Quran
type Data struct {
	Surahs []Surah
}

var quranData Data

// LoadQuran loads the Quran JSON from the given file path
func LoadQuran(filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("could not read quran file: %w", err)
	}

	var surahs []Surah
	if err := json.Unmarshal(data, &surahs); err != nil {
		return fmt.Errorf("could not parse quran json: %w", err)
	}

	quranData.Surahs = surahs
	return nil
}

// GetSurahByName searches for a Surah by its Arabic name
func GetSurahByName(name string) (int, error) {
	name = strings.TrimSpace(name)

	for _, surah := range quranData.Surahs {
		if surah.Name == name {
			return surah.ID, nil
		}
		// Allow ignoring the difference between 'آ' and 'ا'
		if strings.ReplaceAll(surah.Name, "آ", "ا") == strings.ReplaceAll(name, "آ", "ا") {
			return surah.ID, nil
		}
	}

	return 0, fmt.Errorf("لم يتم العثور على سورة باسم: %s", name)
}

// FetchAyat retrieves a range of verses for a given Surah
func FetchAyat(surahNum, startAyah, endAyah int) ([]Verse, string, error) {
	if surahNum < 1 || surahNum > 114 {
		return nil, "", fmt.Errorf("رقم السورة غير صحيح (يجب أن يكون بين ١ و ١١٤)")
	}

	if startAyah > endAyah {
		return nil, "", fmt.Errorf("نطاق الآيات غير صحيح (الآية الأولى يجب أن تكون قبل أو تساوي الآية الأخيرة)")
	}

	// Surahs are 1-indexed in our input, but 0-indexed in the array
	surah := quranData.Surahs[surahNum-1]

	if startAyah < 1 || endAyah > surah.TotalAyah {
		return nil, "", fmt.Errorf("عذراً، سورة %s تحتوي على %d آية فقط", surah.Name, surah.TotalAyah)
	}

	// Extract verses. Verses are 1-indexed in input, 0-indexed in array.
	var verses []Verse
	for i := startAyah - 1; i < endAyah; i++ {
		verses = append(verses, surah.Verses[i])
	}

	return verses, surah.Name, nil
}

// GetOpeningText determines the correct opening phrase based on the rules
func GetOpeningText(surahNum, startAyah int) string {
	if surahNum == 9 {
		return "أَعُوذُ بِاللَّهِ مِنَ الشَّيْطَانِ الرَّجِيمِ"
	}
	if startAyah == 1 {
		return "بِسْمِ اللَّهِ الرَّحْمَنِ الرَّحِيمِ"
	}
	return "أَعُوذُ بِاللَّهِ مِنَ الشَّيْطَانِ الرَّجِيمِ"
}

// ConvertToArabicIndic converts Western numerals to Arabic-Indic numerals
func ConvertToArabicIndic(s string) string {
	mapping := map[rune]rune{
		'0': '٠', '1': '١', '2': '٢', '3': '٣', '4': '٤',
		'5': '٥', '6': '٦', '7': '٧', '8': '٨', '9': '٩',
	}
	result := []rune{}
	for _, char := range s {
		if val, exists := mapping[char]; exists {
			result = append(result, val)
		} else {
			result = append(result, char)
		}
	}
	return string(result)
}
