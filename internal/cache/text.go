package cache

import (
	"image/color"
	"time"

	"fyne.io/fyne/v2"
)

var fontSizeCache = make(map[fontSizeEntry]*fontMetric)

type fontMetric struct {
	expiringCache
	size     fyne.Size
	baseLine float32
}

type fontSizeEntry struct {
	Text   string
	Size   float32
	Style  fyne.TextStyle
	Source string
}

type FontCacheEntry struct {
	fontSizeEntry

	Canvas fyne.Canvas
	Color  color.Color
}

// GetFontMetrics looks up a calculated size and baseline required for the specified text parameters.
func GetFontMetrics(text string, fontSize float32, style fyne.TextStyle, source fyne.Resource) (size fyne.Size, base float32) {
	name := ""
	if source != nil {
		name = source.Name()
	}
	ent := fontSizeEntry{text, fontSize, style, name}
	ret, ok := fontSizeCache[ent]
	if !ok {
		return fyne.Size{Width: 0, Height: 0}, 0
	}
	ret.setAlive()
	return ret.size, ret.baseLine
}

// SetFontMetrics stores a calculated font size and baseline for parameters that were missing from the cache.
func SetFontMetrics(text string, fontSize float32, style fyne.TextStyle, source fyne.Resource, size fyne.Size, base float32) {
	name := ""
	if source != nil {
		name = source.Name()
	}
	ent := fontSizeEntry{text, fontSize, style, name}
	metric := &fontMetric{size: size, baseLine: base}
	metric.setAlive()
	fontSizeCache[ent] = metric
}

// destroyExpiredFontMetrics destroys expired fontSizeCache entries
func destroyExpiredFontMetrics(now time.Time) {
	for key, metric := range fontSizeCache {
		if metric.isExpired(now) {
			delete(fontSizeCache, key)
		}
	}
}
