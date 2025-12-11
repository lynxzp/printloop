package webserver

import (
	"embed"
	"encoding/json"
	"net/http"
	"strings"
)

//go:embed translations/*.json
var translationFiles embed.FS

// Translation holds translations for a specific language
type Translation map[string]string

// Translations holds all loaded translations
type Translations map[string]Translation

var translations Translations

// LoadTranslations loads all translation files
func LoadTranslations() error {
	translations = make(Translations)

	// Load English translations
	enData, err := translationFiles.ReadFile("translations/en.json")
	if err != nil {
		return err
	}

	var enTrans Translation

	err = json.Unmarshal(enData, &enTrans)
	if err != nil {
		return err
	}

	translations["en"] = enTrans

	// Load Ukrainian translations
	ukData, err := translationFiles.ReadFile("translations/uk.json")
	if err != nil {
		return err
	}

	var ukTrans Translation

	err = json.Unmarshal(ukData, &ukTrans)
	if err != nil {
		return err
	}

	translations["uk"] = ukTrans

	return nil
}

// GetLanguageFromRequest determines the language from URL param or Accept-Language header
func GetLanguageFromRequest(r *http.Request) string {
	// First, check URL parameter
	if lang := r.URL.Query().Get("lang"); lang != "" {
		if isValidLanguage(lang) {
			return lang
		}
	}

	// Second, check Accept-Language header
	acceptLang := r.Header.Get("Accept-Language")
	if acceptLang != "" {
		// Parse Accept-Language header (simplified version)
		// Format: "en-US,en;q=0.9,uk;q=0.8"
		for lang := range strings.SplitSeq(acceptLang, ",") {
			// Remove quality values and extra parameters
			lang = strings.TrimSpace(strings.Split(lang, ";")[0])
			// Extract main language code
			lang = strings.Split(lang, "-")[0]
			if lang == "ru" {
				return "uk"
			}

			if isValidLanguage(lang) {
				return lang
			}
		}
	}

	// Default to English
	return "en"
}

// isValidLanguage checks if the language is supported
func isValidLanguage(lang string) bool {
	_, exists := translations[lang]
	return exists
}

// GetTranslation returns the translation for a given key and language
func GetTranslation(lang, key string) string {
	if trans, exists := translations[lang]; exists {
		if text, exists := trans[key]; exists {
			return text
		}
	}

	// Fallback to English
	if trans, exists := translations["en"]; exists {
		if text, exists := trans[key]; exists {
			return text
		}
	}

	// Fallback to key if translation not found
	return key
}

// GetTranslations returns all translations for a given language
func GetTranslations(lang string) Translation {
	if trans, exists := translations[lang]; exists {
		return trans
	}

	// Fallback to English
	if trans, exists := translations["en"]; exists {
		return trans
	}

	// Return empty translation if nothing found
	return make(Translation)
}
