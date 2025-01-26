package cache

import (
	"os"
	"time"

	"fyne.io/fyne/v2"
)

var (
	CacheDuration     = 1 * time.Minute
	cleanTaskInterval = CacheDuration / 2

	lastClean                     time.Time
	skippedCleanWithCanvasRefresh = false

	// testing purpose only
	timeNow = time.Now
)

func init() {
	if t, err := time.ParseDuration(os.Getenv("FYNE_CACHE")); err == nil {
		CacheDuration = t
		cleanTaskInterval = CacheDuration / 2
	}
}

// Clean run cache clean task, it should be called on paint events.
func Clean(canvasRefreshed bool) {
	now := timeNow()
	// do not run clean task too fast
	if now.Sub(lastClean) < 10*time.Second {
		if canvasRefreshed {
			skippedCleanWithCanvasRefresh = true
		}
		return
	}
	if skippedCleanWithCanvasRefresh {
		skippedCleanWithCanvasRefresh = false
		canvasRefreshed = true
	}
	if !canvasRefreshed && now.Sub(lastClean) < cleanTaskInterval {
		return
	}
	destroyExpiredSvgs(now)
	destroyExpiredFontMetrics(now)
	if canvasRefreshed {
		// Destroy renderers on canvas refresh to avoid flickering screen.
		destroyExpiredRenderers(now)
		// canvases cache should be invalidated only on canvas refresh, otherwise there wouldn't
		// be a way to recover them later
		destroyExpiredCanvases(now)
	}
	lastClean = timeNow()
}

// CleanCanvas performs a complete remove of all the objects that belong to the specified
// canvas. Usually used to free all objects from a closing windows.
func CleanCanvas(canvas fyne.Canvas) {
	canvases.Range(func(obj fyne.CanvasObject, cinfo *canvasInfo) bool {
		if cinfo.canvas != canvas {
			return true
		}

		canvases.Delete(obj)

		wid, ok := obj.(fyne.Widget)
		if !ok {
			return true
		}
		rinfo, ok := renderers.LoadAndDelete(wid)
		if !ok {
			return true
		}
		rinfo.renderer.Destroy()
		overrides.Delete(wid)
		return true
	})
}

// CleanCanvases runs cache clean tasks for canvases that are being refreshed. This is called on paint events.
func CleanCanvases(refreshingCanvases []fyne.Canvas) {
	now := timeNow()
	delta := now.Sub(lastClean)

	if delta < 10*time.Second || delta < cleanTaskInterval {
		return // Do not clean too fast.
	}

	destroyExpiredSvgs(now)
	destroyExpiredFontMetrics(now)

	canvases.Range(func(obj fyne.CanvasObject, cinfo *canvasInfo) bool {
		if !cinfo.isExpired(now) || !matchesACanvas(cinfo, refreshingCanvases) {
			return true
		}

		canvases.Delete(obj)

		wid, ok := obj.(fyne.Widget)
		if !ok {
			return true
		}

		rinfo, ok := renderers.LoadAndDelete(wid)
		if !ok || !rinfo.isExpired(now) {
			return true
		}

		rinfo.renderer.Destroy()
		overrides.Delete(wid)
		return true
	})

	lastClean = timeNow()
}

// ResetThemeCaches clears all the svg and text size cache maps
func ResetThemeCaches() {
	svgs.Clear()
	fontSizeCache.Clear()
}

// destroyExpiredCanvases deletes objects from the canvases cache.
func destroyExpiredCanvases(now time.Time) {
	canvases.Range(func(obj fyne.CanvasObject, cinfo *canvasInfo) bool {
		if cinfo.isExpired(now) {
			canvases.Delete(obj)
		}
		return true
	})
}

// destroyExpiredRenderers deletes the renderer from the cache and calls
// renderer.Destroy()
func destroyExpiredRenderers(now time.Time) {
	renderers.Range(func(wid fyne.Widget, rinfo *rendererInfo) bool {
		if rinfo.isExpired(now) {
			rinfo.renderer.Destroy()
			overrides.Delete(wid)
			renderers.Delete(wid)
		}
		return true
	})
}

// matchesACanvas returns true if the canvas represented by the canvasInfo object matches one of
// the canvases passed in 'canvases', otherwise false is returned.
func matchesACanvas(cinfo *canvasInfo, canvases []fyne.Canvas) bool {
	canvas := cinfo.canvas

	for _, obj := range canvases {
		if obj == canvas {
			return true
		}
	}
	return false
}

type expiringCache struct {
	expires time.Time
}

// isExpired check if the cache data is expired.
func (c *expiringCache) isExpired(now time.Time) bool {
	return c.expires.Before(now)
}

// setAlive updates expiration time.
func (c *expiringCache) setAlive() {
	c.expires = timeNow().Add(CacheDuration)
}
