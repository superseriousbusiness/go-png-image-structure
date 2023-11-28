package pngstructure

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path"
	"reflect"
	"testing"

	"github.com/dsoprea/go-exif/v3"
	exifcommon "github.com/dsoprea/go-exif/v3/common"
)

func TestChunk_Bytes(t *testing.T) {
	c := Chunk{
		Offset: 0,
		Length: 5,
		Type:   "ABCD",
		Data:   []byte{0x11, 0x22, 0x33, 0x44, 0x55},
		Crc:    0x5678,
	}

	actual, err := c.Bytes()
	if err != nil {
		t.Fatal(err)
	}

	expected := []byte{
		0x00, 0x00, 0x00, 0x05,
		0x41, 0x42, 0x43, 0x44,
		0x11, 0x22, 0x33, 0x44, 0x55,
		0x00, 0x00, 0x56, 0x78,
	}

	if !bytes.Equal(actual, expected) {
		t.Fatalf("bytes not correct")
	}
}

func TestChunk_Write(t *testing.T) {
	c := Chunk{
		Offset: 0,
		Length: 5,
		Type:   "ABCD",
		Data:   []byte{0x11, 0x22, 0x33, 0x44, 0x55},
		Crc:    0x5678,
	}

	b := new(bytes.Buffer)
	_, err := c.WriteTo(b)
	if err != nil {
		t.Fatal(err)
	}

	expected, err := c.Bytes()
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(b.Bytes(), expected) {
		t.Fatalf("bytes not correct")
	}
}

func TestChunkSlice_Index(t *testing.T) {
	filepath := path.Join(assetsPath, "Selection_058.png")

	pmp := NewPngMediaParser()

	intfc, err := pmp.ParseFile(filepath)
	if err != nil {
		t.Fatal(err)
	}

	cs := intfc.(*ChunkSlice)
	index := cs.Index()

	tallies := make(map[string]int)
	for key, chunks := range index {
		tallies[key] = len(chunks)
	}

	expected := map[string]int{
		"IDAT": 222,
		"IEND": 1,
		"IHDR": 1,
		"pHYs": 1,
		"tIME": 1,
	}

	if reflect.DeepEqual(tallies, expected) != true {
		t.Fatalf("index not correct")
	}
}

func TestChunkSlice_FindExif_Miss(t *testing.T) {
	filepath := path.Join(assetsPath, "Selection_058.png")

	pmp := NewPngMediaParser()

	intfc, err := pmp.ParseFile(filepath)
	if err != nil {
		t.Fatal(err)
	}

	cs := intfc.(*ChunkSlice)
	_, err = cs.FindExif()

	if err == nil {
		t.Fatalf("expected error for missing EXIF")
	} else if !errors.Is(err, exif.ErrNoExif) {
		t.Fatal(err)
	}
}

func TestChunkSlice_FindExif_Hit(t *testing.T) {
	testBasicFilepath, err := getTestBasicImageFilepath()
	if err != nil {
		t.Fatal(err)
	}

	pmp := NewPngMediaParser()

	intfc, err := pmp.ParseFile(testBasicFilepath)
	if err != nil {
		t.Fatal(err)
	}

	cs := intfc.(*ChunkSlice)

	exifChunk, err := cs.FindExif()
	if err != nil {
		t.Fatal(err)
	}

	exifFilepath := fmt.Sprintf("%s.exif", testBasicFilepath)

	expectedExifData, err := os.ReadFile(exifFilepath)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(exifChunk.Data, expectedExifData) {
		t.Fatalf("Exif not extract correctly.")
	}
}

func TestChunkSlice_Exif(t *testing.T) {
	testExifFilepath, err := getTestExifImageFilepath()
	if err != nil {
		t.Fatal(err)
	}

	pmp := NewPngMediaParser()

	intfc, err := pmp.ParseFile(testExifFilepath)
	if err != nil {
		t.Fatal(err)
	}

	cs := intfc.(*ChunkSlice)

	rootIfd, _, err := cs.Exif()
	if err != nil {
		t.Fatal(err)
	}

	tags := rootIfd.Entries()

	if rootIfd.IfdIdentity().Equals(exifcommon.IfdStandardIfdIdentity) != true {
		t.Fatalf("root-IFD not parsed correctly")
	} else if len(tags) != 2 {
		t.Fatalf("incorrect number of encoded tags")
	} else if tags[0].TagId() != 0x0100 {
		t.Fatalf("first tag is not correct")
	} else if tags[1].TagId() != 0x0101 {
		t.Fatalf("second tag is not correct")
	}
}

func TestChunkSlice_SetExif_Existing(t *testing.T) {
	testBasicFilepath, err := getTestBasicImageFilepath()
	if err != nil {
		t.Fatal(err)
	}

	// Build EXIF.

	im, err := exifcommon.NewIfdMappingWithStandard()
	if err != nil {
		t.Fatal(err)
	}

	ti := exif.NewTagIndex()

	ib := exif.NewIfdBuilder(im, ti, exifcommon.IfdStandardIfdIdentity, exifcommon.TestDefaultByteOrder)

	err = ib.AddStandardWithName("ImageWidth", []uint32{11})
	if err != nil {
		t.Fatal(err)
	}

	err = ib.AddStandardWithName("ImageLength", []uint32{22})
	if err != nil {
		t.Fatal(err)
	}

	// Replace into PNG.

	pmp := NewPngMediaParser()

	intfc, err := pmp.ParseFile(testBasicFilepath)
	if err != nil {
		t.Fatal(err)
	}

	cs := intfc.(*ChunkSlice)

	err = cs.SetExif(ib)
	if err != nil {
		t.Fatal(err)
	}

	b := new(bytes.Buffer)

	err = cs.WriteTo(b)
	if err != nil {
		t.Fatal(err)
	}

	updatedImageData := b.Bytes()

	// Re-parse.

	intfc, err = pmp.ParseBytes(updatedImageData)
	if err != nil {
		t.Fatal(err)
	}

	cs = intfc.(*ChunkSlice)

	exifChunk, err := cs.FindExif()
	if err != nil {
		t.Fatal(err)
	}

	chunkData, err := exifChunk.Bytes()
	if err != nil {
		t.Fatal(err)
	}

	// Chunk data length minus length, type, and CRC data.
	expectedExifLen := len(chunkData) - 4 - 4 - 4

	if int(exifChunk.Length) != expectedExifLen {
		t.Fatalf("actual chunk data length does not match prescribed chunk data length: (%d) != (%d)", exifChunk.Length, len(exifChunk.Data))
	} else if len(exifChunk.Data) != expectedExifLen {
		t.Fatalf("chunk data length not correct")
	}

	// The first eight bytes belong to the PNG chunk structure.
	offset := 8
	_, index, err := exif.Collect(im, ti, chunkData[offset:offset+expectedExifLen])
	if err != nil {
		t.Fatal(err)
	}

	tags := index.RootIfd.Entries()

	if len(tags) != 2 {
		t.Fatalf("incorrect number of encoded tags")
	} else if tags[0].TagId() != 0x0100 {
		t.Fatalf("first tag is not correct")
	} else if tags[1].TagId() != 0x0101 {
		t.Fatalf("second tag is not correct")
	}
}

func TestChunkSlice_SetExif_Chunk(t *testing.T) {
	// Build EXIF.

	im, err := exifcommon.NewIfdMappingWithStandard()
	if err != nil {
		t.Fatal(err)
	}

	ti := exif.NewTagIndex()

	ib := exif.NewIfdBuilder(im, ti, exifcommon.IfdStandardIfdIdentity, exifcommon.TestDefaultByteOrder)

	err = ib.AddStandardWithName("ImageWidth", []uint32{11})
	if err != nil {
		t.Fatal(err)
	}

	err = ib.AddStandardWithName("ImageLength", []uint32{22})
	if err != nil {
		t.Fatal(err)
	}

	// Create PNG.

	cs, err := NewPngChunkSlice()
	if err != nil {
		t.Fatal(err)
	}

	err = cs.SetExif(ib)
	if err != nil {
		t.Fatal(err)
	}

	exifChunk, err := cs.FindExif()
	if err != nil {
		t.Fatal(err)
	}

	chunkData, err := exifChunk.Bytes()
	if err != nil {
		t.Fatal(err)
	}

	// Chunk data length minus length, type, and CRC data.
	expectedExifLen := len(chunkData) - 4 - 4 - 4

	if int(exifChunk.Length) != expectedExifLen {
		t.Fatalf("actual chunk data length does not match prescribed chunk data length: (%d) != (%d)", exifChunk.Length, len(exifChunk.Data))
	} else if len(exifChunk.Data) != expectedExifLen {
		t.Fatalf("chunk data length not correct")
	}

	// The first eight bytes belong to the PNG chunk structure.
	offset := 8
	_, index, err := exif.Collect(im, ti, chunkData[offset:offset+expectedExifLen])
	if err != nil {
		t.Fatal(err)
	}

	tags := index.RootIfd.Entries()

	if len(tags) != 2 {
		t.Fatalf("incorrect number of encoded tags")
	} else if tags[0].TagId() != 0x0100 {
		t.Fatalf("first tag is not correct")
	} else if tags[1].TagId() != 0x0101 {
		t.Fatalf("second tag is not correct")
	}
}

func TestChunk_Crc32_Cycle(t *testing.T) {
	c := &Chunk{
		Type: "pHYs",
		Data: []byte{0x00, 0x00, 0x0b, 0x13, 0x00, 0x00, 0x0b, 0x13, 0x01},
	}

	c.UpdateCrc32()

	if c.Crc != calculateCrc32(c) {
		t.Fatalf("CRC value not consistently calculated")
	} else if c.Crc != 0x9a9c18 {
		t.Fatalf("CRC (1) not correct")
	} else if c.CheckCrc32() != true {
		t.Fatalf("CRC (1) check failed")
	}

	c.Type = "tIME"
	c.Data = []byte{0x07, 0xcc, 0x06, 0x07, 0x11, 0x3a, 0x08}

	c.UpdateCrc32()

	if c.Crc != 0x8eff267a {
		t.Fatalf("CRC (2) not correct")
	} else if c.CheckCrc32() != true {
		t.Fatalf("CRC (2) check failed")
	}

	c.Data = []byte{0x99, 0x99, 0x99, 0x99}

	if c.CheckCrc32() != false {
		t.Fatalf("CRC check didn't fail but should've")
	}
}

func TestChunkSlice_ConstructExifBuilder(t *testing.T) {
	testExifFilepath, err := getTestExifImageFilepath()
	if err != nil {
		t.Fatal(err)
	}

	pmp := NewPngMediaParser()

	intfc, err := pmp.ParseFile(testExifFilepath)
	if err != nil {
		t.Fatal(err)
	}

	cs := intfc.(*ChunkSlice)

	// Add a new tag to the additional EXIF.

	rootIb, err := cs.ConstructExifBuilder()
	if err != nil {
		t.Fatal(err)
	}

	err = rootIb.SetStandardWithName("ImageLength", []uint32{44})
	if err != nil {
		t.Fatal(err)
	}

	err = rootIb.AddStandardWithName("BitsPerSample", []uint16{33})
	if err != nil {
		t.Fatal(err)
	}

	// Update the image.

	err = cs.SetExif(rootIb)
	if err != nil {
		t.Fatal(err)
	}

	b := new(bytes.Buffer)

	err = cs.WriteTo(b)
	if err != nil {
		t.Fatal(err)
	}

	updatedImageData := b.Bytes()

	// Re-parse.

	pmp = NewPngMediaParser()

	intfc, err = pmp.ParseBytes(updatedImageData)
	if err != nil {
		t.Fatal(err)
	}

	cs = intfc.(*ChunkSlice)

	rootIfd, _, err := cs.Exif()
	if err != nil {
		t.Fatal(err)
	}

	tags := rootIfd.Entries()

	v1, err := tags[0].Value()
	if err != nil {
		t.Fatal(err)
	}

	v2, err := tags[1].Value()
	if err != nil {
		t.Fatal(err)
	}

	v3, err := tags[2].Value()
	if err != nil {
		t.Fatal(err)
	}

	if rootIfd.IfdIdentity().Equals(exifcommon.IfdStandardIfdIdentity) != true {
		t.Fatalf("root-IFD not parsed correctly")
	} else if len(tags) != 3 {
		t.Fatalf("incorrect number of encoded tags")
	} else if tags[0].TagId() != 0x0100 || reflect.DeepEqual(v1.([]uint32), []uint32{11}) != true {
		t.Fatalf("first tag is not correct")
	} else if tags[1].TagId() != 0x0101 || reflect.DeepEqual(v2.([]uint32), []uint32{44}) != true {
		t.Fatalf("second tag is not correct")
	} else if tags[2].TagId() != 0x0102 || reflect.DeepEqual(v3.([]uint16), []uint16{33}) != true {
		t.Fatalf("third tag is not correct")
	}
}

func TestPngSplitter_Write(t *testing.T) {
	filepath := path.Join(assetsPath, "Selection_058.png")

	original, err := os.ReadFile(filepath)
	if err != nil {
		t.Fatal(err)
	}

	pmp := NewPngMediaParser()

	intfc, err := pmp.ParseBytes(original)
	if err != nil {
		t.Fatal(err)
	}

	cs := intfc.(*ChunkSlice)

	b := new(bytes.Buffer)

	err = cs.WriteTo(b)
	if err != nil {
		t.Fatal(err)
	}

	written := b.Bytes()

	if !bytes.Equal(written, original) {
		t.Fatalf("written bytes (%d) do not equal read bytes (%d)", len(written), len(original))
	}
}

func TestChunkSlice_Write(t *testing.T) {
	chunkData := []byte{
		0x00, 0x00, 0x00, 0x0d,
		0x49, 0x48, 0x44, 0x52,
		0x00, 0x00, 0x05, 0xc0, 0x00, 0x00, 0x02, 0x56, 0x08, 0x02, 0x00, 0x00, 0x00,
		0xf0, 0x49, 0xb3, 0x65,

		0x00, 0x00, 0x00, 0x09,
		0x70, 0x48, 0x59, 0x73,
		0x00, 0x00, 0x0b, 0x13, 0x00, 0x00, 0x0b, 0x13, 0x01,
		0x00, 0x9a, 0x9c, 0x18,
	}

	b := new(bytes.Buffer)

	_, err := b.Write(PngSignature[:])
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.Write(chunkData)
	if err != nil {
		t.Fatal(err)
	}

	originalFull := make([]byte, len(b.Bytes()))
	copy(originalFull, b.Bytes())

	br := bytes.NewReader(b.Bytes())

	pmp := NewPngMediaParser()

	intfc, err := pmp.Parse(br, len(b.Bytes()))
	if err != nil {
		t.Fatal(err)
	}

	cs := intfc.(*ChunkSlice)

	chunks := cs.Chunks()
	if len(chunks) != 2 {
		t.Fatalf("number of chunks not correct")
	}

	b2 := new(bytes.Buffer)

	err = cs.WriteTo(b2)
	if err != nil {
		t.Fatal(err)
	}

	actual := b2.Bytes()

	if !bytes.Equal(actual, originalFull) {
		fmt.Printf("ACTUAL:\n")
		DumpBytesClause(actual)

		fmt.Printf("EXPECTED:\n")
		DumpBytesClause(originalFull)

		t.Fatalf("did not write correctly")
	}
}
