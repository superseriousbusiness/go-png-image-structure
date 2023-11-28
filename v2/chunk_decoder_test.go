package pngstructure

import (
	"path"
	"testing"
)

func TestChunkDecoder_decodeIHDR(t *testing.T) {
	assetsPath, err := getTestAssetsPath()
	if err != nil {
		t.Fatal(err)
	}

	filepath := path.Join(assetsPath, "Selection_058.png")

	pmp := NewPngMediaParser()

	intfc, err := pmp.ParseFile(filepath)
	if err != nil {
		t.Fatal(err)
	}

	cs := intfc.(*ChunkSlice)
	index := cs.Index()

	ihdrRawSlice, found := index["IHDR"]
	if !found {
		t.Fatalf("Could not find IHDR chunk.")
	}

	cd := NewChunkDecoder()

	ihdrRaw, err := cd.Decode(ihdrRawSlice[0])
	if err != nil {
		t.Fatal(err)
	}

	ihdr := ihdrRaw.(*ChunkIHDR)

	expected := &ChunkIHDR{
		Width:             1472,
		Height:            598,
		BitDepth:          8,
		ColorType:         2,
		CompressionMethod: 0,
		FilterMethod:      0,
		InterlaceMethod:   0,
	}

	if *ihdr != *expected {
		t.Fatalf("ihdr not correct")
	}
}
