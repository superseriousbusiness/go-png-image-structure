package pngstructure

import (
	"os"
	"path"
	"testing"
)

func TestPngMediaParser_ParseFile(t *testing.T) {
	filepath := path.Join(assetsPath, "Selection_058.png")

	pmp := NewPngMediaParser()

	_, err := pmp.ParseFile(filepath)
	if err != nil {
		t.Fatal(err)
	}
}

func TestPngMediaParser_LooksLikeFormat(t *testing.T) {
	filepath := path.Join(assetsPath, "libpng.png")

	data, err := os.ReadFile(filepath)
	if err != nil {
		t.Fatal(err)
	}

	pmp := NewPngMediaParser()
	if pmp.LooksLikeFormat(data) != true {
		t.Fatalf("not detected as png")
	}
}
