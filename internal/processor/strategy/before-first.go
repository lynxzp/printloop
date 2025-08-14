package strategy

// BeforeCommandStrategy finds markers that appear before specific commands
type BeforeCommandStrategy struct{}

func (s *BeforeCommandStrategy) FindInitSectionPosition(filePath string, markers []string) (int64, int64, error) {
	scanner, err := NewFileScanner(filePath)
	if err != nil {
		return 0, 0, err
	}
	return scanner.FindFirstMarkerFromStart(markers)
}

func (s *BeforeCommandStrategy) FindPrintSectionPosition(filePath string, markers []string, searchFromLine int64) (int64, int64, error) {
	scanner, err := NewFileScanner(filePath)
	if err != nil {
		return 0, 0, err
	}
	return scanner.FindFirstMarkerFromLine(markers, searchFromLine+1)
}
