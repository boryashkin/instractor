package translation

type TranslationMap map[string]map[int]string

func (m TranslationMap) AddTranslation(lang string, code int, translation string) {
	m[lang][code] = translation
}
func (m TranslationMap) InitLangMap(lang string) {
	m[lang] = make(map[int]string)
}
