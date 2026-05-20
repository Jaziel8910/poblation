package ui

import (
	"testing"

	"github.com/user/poblation/internal/entities"
	uiviews "github.com/user/poblation/internal/ui/views"
)

func TestFinishPobleCreationRequiresTwoPlayerFounders(t *testing.T) {
	model := NewAppModel(nil, nil)
	model = model.resetRuntime(77)

	first := entities.NewPoble("founder_1", "Ari", 28, entities.Female)
	updated, _ := model.finishPobleCreation(uiviews.CreatePobleCompleteMsg{Poble: &first})
	model = updated.(AppModel)
	if got := model.World.GetPopulation(); got != 1 {
		t.Fatalf("first founder should not auto-spawn a companion; population=%d", got)
	}
	if model.CurrentView != VIEW_CREATE_POBLE {
		t.Fatalf("after first founder view=%s, want create flow", model.CurrentView.String())
	}

	second := entities.NewPoble("founder_2", "Ben", 31, entities.Male)
	updated, _ = model.finishPobleCreation(uiviews.CreatePobleCompleteMsg{Poble: &second})
	model = updated.(AppModel)
	if got := model.World.GetPopulation(); got != 2 {
		t.Fatalf("second player founder should make population 2; population=%d", got)
	}
	if model.CurrentView != VIEW_CREATE_POBLE {
		t.Fatalf("after second founder view=%s, want more-founders choice", model.CurrentView.String())
	}

	updated, _ = model.startCreatedCivilization()
	model = updated.(AppModel)
	if model.CurrentView != VIEW_MAIN_MAP {
		t.Fatalf("starting with two founders view=%s, want main map", model.CurrentView.String())
	}
}
