package main

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
)

type Glyph struct {
	code         int
	width        int
	height       int
	xOffset      int
	bbxY         int
	xAdvance     int
	bitmap       []byte
	bitmapOffset int
	yOffsetTFT   int
}

func main() {
	if len(os.Args) != 3 {
		log.Fatal("Usage: bdf2tft <input.bdf> <output.h>")
	}

	inputFile := os.Args[1]
	outputFile := os.Args[2]

	fontAscent, fontDescent, glyphs := parseBDF(inputFile)
	generateHeader(outputFile, fontAscent, fontDescent, glyphs)
}

func parseBDF(filename string) (int, int, []*Glyph) {
	file, err := os.Open(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	var fontAscent, fontDescent int
	var glyphs []*Glyph
	var currentGlyph *Glyph
	insideGlyph := false
	insideBitmap := false
	var bytesPerRow int

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()

		if insideGlyph && insideBitmap {
			line = strings.TrimSpace(line)
			if line == "ENDCHAR" {
				currentGlyph.yOffsetTFT = -(currentGlyph.bbxY + currentGlyph.height)
				glyphs = append(glyphs, currentGlyph)
				insideGlyph = false
				insideBitmap = false
				continue
			}

			rowBytes, err := hex.DecodeString(line)
			if err != nil {
				log.Fatalf("Hex decode error: %v", err)
			}
			if len(rowBytes) != bytesPerRow {
				log.Fatalf("Expected %d bytes, got %d", bytesPerRow, len(rowBytes))
			}
			currentGlyph.bitmap = append(currentGlyph.bitmap, rowBytes...)
			continue
		}

		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}

		switch fields[0] {
		case "FONT_ASCENT":
			fontAscent, _ = strconv.Atoi(fields[1])
		case "FONT_DESCENT":
			fontDescent, _ = strconv.Atoi(fields[1])
		case "STARTCHAR":
			currentGlyph = &Glyph{}
			insideGlyph = true
		case "ENCODING":
			if insideGlyph {
				code, _ := strconv.Atoi(fields[1])
				currentGlyph.code = code
			}
		case "DWIDTH":
			if insideGlyph {
				xAdvance, _ := strconv.Atoi(fields[1])
				currentGlyph.xAdvance = xAdvance
			}
		case "BBX":
			if insideGlyph {
				width, _ := strconv.Atoi(fields[1])
				height, _ := strconv.Atoi(fields[2])
				xOffset, _ := strconv.Atoi(fields[3])
				bbxY, _ := strconv.Atoi(fields[4])
				currentGlyph.width = width
				currentGlyph.height = height
				currentGlyph.xOffset = xOffset
				currentGlyph.bbxY = bbxY
				bytesPerRow = (width + 7) / 8
			}
		case "BITMAP":
			if insideGlyph {
				currentGlyph.bitmap = []byte{}
				insideBitmap = true
			}
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

	sort.Slice(glyphs, func(i, j int) bool {
		return glyphs[i].code < glyphs[j].code
	})

	return fontAscent, fontDescent, glyphs
}

func generateHeader(filename string, ascent, descent int, glyphs []*Glyph) {
	outFile, err := os.Create(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer outFile.Close()

	var bitmapData []byte
	var offsets []int
	offset := 0
	for _, g := range glyphs {
		offsets = append(offsets, offset)
		bitmapData = append(bitmapData, g.bitmap...)
		offset += len(g.bitmap)
	}

	fmt.Fprintf(outFile, "// typedef struct {\n")
	fmt.Fprintf(outFile, "//   uint16_t bitmapOffset;\n")
	fmt.Fprintf(outFile, "//   uint8_t  width;\n")
	fmt.Fprintf(outFile, "//   uint8_t  height;\n")
	fmt.Fprintf(outFile, "//   uint8_t  xAdvance;\n")
	fmt.Fprintf(outFile, "//   int8_t   xOffset;\n")
	fmt.Fprintf(outFile, "//   int8_t   yOffset;\n} GFXglyph;\n\n")

	fmt.Fprintf(outFile, "// typedef struct {\n")
	fmt.Fprintf(outFile, "//   uint8_t  *bitmap;\n")
	fmt.Fprintf(outFile, "//   GFXglyph *glyph;\n")
	fmt.Fprintf(outFile, "//   uint16_t  first;\n")
	fmt.Fprintf(outFile, "//   uint16_t  last;\n")
	fmt.Fprintf(outFile, "//   uint8_t   yAdvance;\n} GFXfont;\n\n")

	fmt.Fprintf(outFile, "const uint8_t FontBitmaps[] PROGMEM = {\n  ")
	for i, b := range bitmapData {
		if i > 0 && i%(ascent+descent) == 0 {
			fmt.Fprint(outFile, "\n  ")
		}
		fmt.Fprintf(outFile, "0x%02X, ", b)
	}
	fmt.Fprintf(outFile, "\n};\n\n")

	fmt.Fprintf(outFile, "const GFXglyph FontGlyphs[] PROGMEM = {\n")
	for i, g := range glyphs {
		fmt.Fprintf(outFile, "  { %5d, %2d, %2d, %2d, %3d, %3d }, // 0x%04X\n",
			offsets[i], g.width, g.height, g.xAdvance, g.xOffset, g.yOffsetTFT, g.code)
	}
	fmt.Fprint(outFile, "};\n\n")

	first, last := glyphs[0].code, glyphs[len(glyphs)-1].code
	fmt.Fprintf(outFile, "const GFXfont Font PROGMEM = {\n")
	fmt.Fprintf(outFile, "  (uint8_t*)FontBitmaps,\n")
	fmt.Fprintf(outFile, "  (GFXglyph*)FontGlyphs,\n")
	fmt.Fprintf(outFile, "  0x%x, 0x%x, %d\n};\n", first, last, ascent+descent)
}
