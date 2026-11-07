package application

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math/big"
	"strconv"
	"strings"

	"mars-admin-go/internal/modules/identity/domain"

	"github.com/google/uuid"
)

var glyphs = map[rune][]string{
	'0': {"01110", "10001", "10011", "10101", "11001", "10001", "01110"},
	'1': {"00100", "01100", "00100", "00100", "00100", "00100", "01110"},
	'2': {"01110", "10001", "00001", "00010", "00100", "01000", "11111"},
	'3': {"11110", "00001", "00001", "01110", "00001", "00001", "11110"},
	'4': {"00010", "00110", "01010", "10010", "11111", "00010", "00010"},
	'5': {"11111", "10000", "10000", "11110", "00001", "00001", "11110"},
	'6': {"00110", "01000", "10000", "11110", "10001", "10001", "01110"},
	'7': {"11111", "00001", "00010", "00100", "01000", "01000", "01000"},
	'8': {"01110", "10001", "10001", "01110", "10001", "10001", "01110"},
	'9': {"01110", "10001", "10001", "01111", "00001", "00010", "01100"},
	'+': {"00000", "00100", "00100", "11111", "00100", "00100", "00000"},
	'-': {"00000", "00000", "00000", "11111", "00000", "00000", "00000"},
	'=': {"00000", "11111", "00000", "11111", "00000", "00000", "00000"},
	'?': {"01110", "10001", "00001", "00010", "00100", "00000", "00100"},
}

func generateCaptcha(kind string) (domain.Captcha, error) {
	if kind == "slider" {
		return generateSliderCaptcha()
	}
	left, err := randomInt(9)
	if err != nil {
		return domain.Captcha{}, err
	}
	right, err := randomInt(8)
	if err != nil {
		return domain.Captcha{}, err
	}
	left++
	right++
	operator := "+"
	answer := left + right
	if left > right {
		operator = "-"
		answer = left - right
	}
	text := fmt.Sprintf("%d%s%d=?", left, operator, right)
	img := randomCanvas(150, 50)
	drawText(img, text, 12, 12, color.RGBA{R: 24, G: 34, B: 48, A: 255}, 3)
	encoded, err := encodePNG(img)
	if err != nil {
		return domain.Captcha{}, err
	}
	return domain.Captcha{
		ID: mustID(), Image: "data:image/png;base64," + encoded,
		Answer: answer, AnswerCode: strconv.Itoa(answer),
	}, nil
}

func generateSliderCaptcha() (domain.Captcha, error) {
	const width, height, pieceSize = 320, 120, 50
	targetX, err := boundedInt(70, width-pieceSize-10)
	if err != nil {
		return domain.Captcha{}, err
	}
	targetY, err := boundedInt(15, height-pieceSize-10)
	if err != nil {
		return domain.Captcha{}, err
	}
	base := randomCanvas(width, height)
	pieceBounds := image.Rect(targetX, targetY, targetX+pieceSize, targetY+pieceSize)
	piece := image.NewRGBA(image.Rect(0, 0, pieceSize, pieceSize))
	draw.Draw(piece, piece.Bounds(), base, pieceBounds.Min, draw.Src)
	hole := color.RGBA{R: 20, G: 26, B: 38, A: 210}
	draw.Draw(base, pieceBounds, &image.Uniform{C: hole}, image.Point{}, draw.Over)
	baseEncoded, err := encodePNG(base)
	if err != nil {
		return domain.Captcha{}, err
	}
	pieceEncoded, err := encodePNG(piece)
	if err != nil {
		return domain.Captcha{}, err
	}
	return domain.Captcha{
		ID: mustID(), Image: "data:image/png;base64," + baseEncoded,
		Piece: "data:image/png;base64," + pieceEncoded, TargetY: targetY,
		TargetX: targetX, Answer: targetX, AnswerCode: strconv.Itoa(targetX),
		Tolerance: 4, Slider: true,
	}, nil
}

func randomCanvas(width, height int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.SetRGBA(x, y, color.RGBA{
				R: uint8(220 + (x+y)%26),
				G: uint8(225 + (x*2+y)%20),
				B: uint8(230 + (x+y*2)%18),
				A: 255,
			})
		}
	}
	for i := 0; i < 22; i++ {
		x1, _ := randomInt(width)
		y1, _ := randomInt(height)
		x2, _ := randomInt(width)
		y2, _ := randomInt(height)
		lineColor := color.RGBA{R: uint8(110 + i*5), G: uint8(130 + i*3), B: uint8(150 + i*2), A: 110}
		drawLine(img, x1, y1, x2, y2, lineColor)
	}
	return img
}

func drawText(img *image.RGBA, text string, x, y int, ink color.RGBA, scale int) {
	for _, r := range strings.ToUpper(text) {
		glyph, ok := glyphs[r]
		if !ok {
			continue
		}
		for row, line := range glyph {
			for col, bit := range line {
				if bit != '1' {
					continue
				}
				for dy := 0; dy < scale; dy++ {
					for dx := 0; dx < scale; dx++ {
						img.SetRGBA(x+col*scale+dx, y+row*scale+dy, ink)
					}
				}
			}
		}
		x += 6*scale + 2
	}
}

func drawLine(img *image.RGBA, x0, y0, x1, y1 int, c color.RGBA) {
	dx := abs(x1 - x0)
	dy := -abs(y1 - y0)
	sx := -1
	if x0 < x1 {
		sx = 1
	}
	sy := -1
	if y0 < y1 {
		sy = 1
	}
	err := dx + dy
	for {
		if image.Pt(x0, y0).In(img.Bounds()) {
			img.SetRGBA(x0, y0, c)
		}
		if x0 == x1 && y0 == y1 {
			break
		}
		e2 := 2 * err
		if e2 >= dy {
			err += dy
			x0 += sx
		}
		if e2 <= dx {
			err += dx
			y0 += sy
		}
	}
}

func encodePNG(img image.Image) (string, error) {
	var buffer bytes.Buffer
	if err := png.Encode(&buffer, img); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(buffer.Bytes()), nil
}

func randomInt(max int) (int, error) {
	if max <= 0 {
		return 0, nil
	}
	value, err := rand.Int(rand.Reader, big.NewInt(int64(max)))
	if err != nil {
		return 0, err
	}
	return int(value.Int64()), nil
}

func boundedInt(minimum, maximum int) (int, error) {
	value, err := randomInt(maximum - minimum + 1)
	if err != nil {
		return 0, err
	}
	return minimum + value, nil
}

func abs(value int) int {
	if value < 0 {
		return -value
	}
	return value
}

func mustID() uuid.UUID {
	id, err := uuid.NewV7()
	if err != nil {
		return uuid.New()
	}
	return id
}
