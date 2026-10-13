package data

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	stdDraw "image/draw"
	_ "image/jpeg"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/chai2010/webp"
	xdraw "golang.org/x/image/draw"
	"golang.org/x/image/math/f64"
)

const (
	sheetWidth  = 330 * 2
	sheetHeight = 288 * 2

	// A small penalty for extremely wide or tall tiles
	// Increase this to favor square tiles more strongly
	tileShapeWeight = 0.12
)

type sheetSource struct {
	path   string
	aspect float64
}

type sheetTile struct {
	imageIndex int
	bounds     image.Rectangle
}

type sheetLayout struct {
	sources   []sheetSource
	prefixLog []float64
	prefixSq  []float64
}

func generateCollectionSheet(
	collection *Collection,
	cards map[string]*Card,
) (string, error) {
	if len(collection.Cards) == 0 {
		return "", errors.New("cannot generate a sheet for an empty collection")
	}

	path := "data/dist/collection-sheet-" + collection.Name + ".webp"
	if _, err := os.Stat(path); err == nil {
		return path, nil
	}

	// First pass: get image dimensions without fully decoding every image
	sources := make([]sheetSource, len(collection.Cards))

	for i, cardName := range collection.Cards {
		card, ok := cards[cardName.Name]
		if !ok {
			return "", fmt.Errorf("card %q not found", cardName.Name)
		}

		path := card.ArtworkPath
		if strings.TrimSpace(path) == "" {
			return "", fmt.Errorf("card %q has no artwork path", cardName.Name)
		}

		cfg, err := inspectImage(path)
		if err != nil {
			return "", fmt.Errorf("inspect artwork %q: %w", path, err)
		}

		if cfg.Width <= 0 || cfg.Height <= 0 {
			return "", fmt.Errorf("invalid image dimensions for %q", path)
		}

		sources[i] = sheetSource{
			path:   path,
			aspect: float64(cfg.Width) / float64(cfg.Height),
		}
	}

	// Calculate the crop-aware layout
	canvas := image.NewRGBA(image.Rect(0, 0, sheetWidth, sheetHeight))
	stdDraw.Draw(
		canvas,
		canvas.Bounds(),
		image.NewUniform(color.White),
		image.Point{},
		stdDraw.Src,
	)

	layout := newSheetLayout(sources)

	tiles, err := layout.generate(canvas.Bounds())
	if err != nil {
		return "", fmt.Errorf("generate tile layout: %w", err)
	}

	// Each source has exactly one assigned rectangle
	boundsByImage := make([]image.Rectangle, len(sources))
	for _, tile := range tiles {
		boundsByImage[tile.imageIndex] = tile.bounds
	}

	// Second pass: decode and render one image at a time to limit memory use
	for i, source := range sources {
		img, err := decodeImage(source.path)
		if err != nil {
			return "", fmt.Errorf("decode artwork %q: %w", source.path, err)
		}

		drawCover(canvas, img, boundsByImage[i])
	}

	// Return the path of a generated PNG
	file, err := os.Create(path)
	if err != nil {
		return "", fmt.Errorf("create output file: %w", err)
	}

	if err := webp.Encode(file, canvas, &webp.Options{Quality: 80}); err != nil {
		_ = file.Close()
		_ = os.Remove(path)
		return "", fmt.Errorf("encode collection sheet: %w", err)
	}

	if err := file.Close(); err != nil {
		_ = os.Remove(path)
		return "", fmt.Errorf("close collection sheet: %w", err)
	}

	return path, nil
}

// inspectImage reads dimensions without decoding all image pixels
func inspectImage(path string) (image.Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return image.Config{}, err
	}
	defer file.Close()

	if strings.EqualFold(filepath.Ext(path), ".webp") {
		return webp.DecodeConfig(file)
	}

	cfg, _, err := image.DecodeConfig(file)
	return cfg, err
}

// decodeImage supports WebP, PNG, and JPEG
func decodeImage(path string) (image.Image, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	if strings.EqualFold(filepath.Ext(path), ".webp") {
		return webp.Decode(file)
	}

	img, _, err := image.Decode(file)
	return img, err
}

func newSheetLayout(sources []sheetSource) *sheetLayout {
	l := &sheetLayout{
		sources:   sources,
		prefixLog: make([]float64, len(sources)+1),
		prefixSq:  make([]float64, len(sources)+1),
	}

	// Prefix sums allow us to score any contiguous group of images quickly
	for i, src := range sources {
		logAspect := math.Log(src.aspect)

		l.prefixLog[i+1] = l.prefixLog[i] + logAspect
		l.prefixSq[i+1] = l.prefixSq[i] + logAspect*logAspect
	}

	return l
}

func (l *sheetLayout) generate(
	bounds image.Rectangle,
) ([]sheetTile, error) {
	n := len(l.sources)
	if n == 0 {
		return nil, nil
	}

	if bounds.Empty() ||
		int64(bounds.Dx())*int64(bounds.Dy()) < int64(n) {
		return nil, errors.New("output dimensions are too small for the image count")
	}

	tiles := make([]sheetTile, 0, n)

	if err := l.split(bounds, 0, n, &tiles); err != nil {
		return nil, err
	}

	return tiles, nil
}

// Split recursively partitions the canvas
//
// Each child receives an area proportional to its number of images
// Candidate cuts are scored by estimating their crop loss and tile shapes
func (l *sheetLayout) split(
	bounds image.Rectangle,
	start, count int,
	out *[]sheetTile,
) error {
	if count == 1 {
		*out = append(*out, sheetTile{
			imageIndex: start,
			bounds:     bounds,
		})
		return nil
	}

	bestScore := math.Inf(1)
	bestK := 0
	var bestA, bestB image.Rectangle

	// Examine every possible split point, in both orientations
	for k := 1; k < count; k++ {
		// Vertical cut: first k images on the left
		x := bounds.Min.X + bounds.Dx()*k/count

		a := image.Rect(bounds.Min.X, bounds.Min.Y, x, bounds.Max.Y)
		b := image.Rect(x, bounds.Min.Y, bounds.Max.X, bounds.Max.Y)

		if canFit(a, k) && canFit(b, count-k) {
			score := l.groupCost(a, start, k) +
				l.groupCost(b, start+k, count-k)

			if score < bestScore {
				bestScore = score
				bestK = k
				bestA, bestB = a, b
			}
		}

		// Horizontal cut: first k images at the top
		y := bounds.Min.Y + bounds.Dy()*k/count

		a = image.Rect(bounds.Min.X, bounds.Min.Y, bounds.Max.X, y)
		b = image.Rect(bounds.Min.X, y, bounds.Max.X, bounds.Max.Y)

		if canFit(a, k) && canFit(b, count-k) {
			score := l.groupCost(a, start, k) +
				l.groupCost(b, start+k, count-k)

			if score < bestScore {
				bestScore = score
				bestK = k
				bestA, bestB = a, b
			}
		}
	}

	if bestK == 0 {
		return fmt.Errorf(
			"cannot partition %d images inside %v",
			count,
			bounds,
		)
	}

	if err := l.split(bestA, start, bestK, out); err != nil {
		return err
	}

	return l.split(
		bestB,
		start+bestK,
		count-bestK,
		out,
	)
}

func canFit(bounds image.Rectangle, count int) bool {
	return bounds.Dx() > 0 &&
		bounds.Dy() > 0 &&
		int64(bounds.Dx())*int64(bounds.Dy()) >= int64(count)
}

// groupCost estimates the best possible cost for a group of images within
// a rectangle, assuming a regular grid of equal-area cells
//
// For count images and columns columns, the estimated cell aspect ratio is:
//
//	cellAR = rectangleAR * count / columns²
//
// The score penalizes cropping and, more lightly, extremely wide or tall cells
func (l *sheetLayout) groupCost(
	bounds image.Rectangle,
	start, count int,
) float64 {
	if bounds.Empty() || count <= 0 {
		return math.Inf(1)
	}

	regionAR := float64(bounds.Dx()) / float64(bounds.Dy())
	sumLog := l.prefixLog[start+count] - l.prefixLog[start]
	sumSq := l.prefixSq[start+count] - l.prefixSq[start]

	n := float64(count)
	best := math.Inf(1)

	// Estimate the cell aspect ratio for each possible column count
	for columns := 1; columns <= count; columns++ {
		c := float64(columns)
		cellAR := regionAR * n / (c * c)
		logAR := math.Log(cellAR)

		// Sum of log(cellAR/sourceAR)^2 for this group
		cropCost := n*logAR*logAR -
			2*logAR*sumLog +
			sumSq

		if cropCost < 0 {
			cropCost = 0
		}

		// Only penalize extreme shapes (wider than 3:1 or taller
		// than 1:3). Ordinary landscape and portrait tiles are fine
		excess := math.Max(0, math.Abs(logAR)-math.Log(3))
		shapeCost := 0.05 * n * excess * excess

		if cost := cropCost + shapeCost; cost < best {
			best = cost
		}
	}

	return best
}

// drawCover applies a uniform scale and centers the image in the target tile
// Excess content is clipped by the tile boundaries; neither axis is scaled
// independently
func drawCover(
	dst *image.RGBA,
	src image.Image,
	bounds image.Rectangle,
) {
	sb := src.Bounds()
	if bounds.Empty() || sb.Empty() {
		return
	}

	w, h := bounds.Dx(), bounds.Dy()

	// Render into a tile-local canvas. This is what prevents one image
	// from spilling over into its neighbors
	tileBounds := image.Rect(0, 0, w, h)
	tile := image.NewRGBA(tileBounds)

	// Make transparent PNG/WebP content display against white
	stdDraw.Draw(
		tile,
		tileBounds,
		image.NewUniform(color.White),
		image.Point{},
		stdDraw.Src,
	)

	// Uniform scaling: cover the tile without distorting the source
	scale := math.Max(
		float64(w)/float64(sb.Dx()),
		float64(h)/float64(sb.Dy()),
	)

	// Map the center of the source image to the center of the tile
	srcCX := float64(sb.Min.X) + float64(sb.Dx())/2
	srcCY := float64(sb.Min.Y) + float64(sb.Dy())/2

	tx := float64(w)/2 - scale*srcCX
	ty := float64(h)/2 - scale*srcCY

	transform := f64.Aff3{
		scale, 0, tx,
		0, scale, ty,
	}

	// The transform is clipped by tile.Bounds(), not the whole sheet
	xdraw.CatmullRom.Transform(
		tile,
		transform,
		src,
		sb,
		xdraw.Over,
		nil,
	)

	// The temporary tile is exactly w×h, so the copy cannot overlap
	// any neighboring tile
	stdDraw.Draw(
		dst,
		bounds,
		tile,
		image.Point{},
		stdDraw.Src,
	)
}
