package strategy

// AfterFirstAppearStrategy finds the first appearance of markers
type AfterFirstAppearStrategy struct{}

func (s *AfterFirstAppearStrategy) FindInitSectionPosition(filePath string, markers []string) (int64, int64, error) {
	scanner, err := NewFileScanner(filePath)
	if err != nil {
		return 0, 0, err
	}
	return scanner.FindFirstMarkerFromStart(markers)
}

func (s *AfterFirstAppearStrategy) FindPrintSectionPosition(filePath string, markers []string, searchFromLine int64) (int64, int64, error) {
	scanner, err := NewFileScanner(filePath)
	if err != nil {
		return 0, 0, err
	}
	return scanner.FindFirstMarkerFromLine(markers, searchFromLine+1)
}
