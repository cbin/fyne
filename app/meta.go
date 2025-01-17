package app

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/internal/build"
)

var meta = fyne.AppMetadata{
	ID:         "",
	Name:       "",
	Version:    "0.0.1",
	Build:      1,
	Release:    false,
	Custom:     map[string]string{},
	Migrations: map[string]bool{},
}

// SetMetadata overrides the packaged application metadata.
// This data can be used in many places like notifications and about screens.
func SetMetadata(m fyne.AppMetadata) {
	meta = m

	if meta.Custom == nil {
		meta.Custom = map[string]string{}
	}
	if meta.Migrations == nil {
		meta.Migrations = map[string]bool{}
	} else {
		setupMigrations(m.Migrations)
	}
}

func (a *fyneApp) Metadata() fyne.AppMetadata {
	if meta.ID == "" && meta.Name == "" {
		checkLocalMetadata()
	}

	return meta
}

func setupMigrations(data map[string]bool) {
	if done, ok := data["fyneDo"]; ok && done {
		build.DisableThreadChecks = true
	}
}
