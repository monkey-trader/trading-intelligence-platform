package httpapi

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/journal"
	"github.com/monkey-trader/trading-intelligence-platform/backend/internal/domain/position"
)

// Vertrag: GET /positions/{positionId}/note (getPositionNote) in docs/openapi.yaml.
// Neue Settings-Eigenschaft (optional, nil = Route nicht registriert):
//   GetPositionNote func(ctx context.Context, positionID string) (journal.JournalNote, error)

const notePositionID = "9100000000000000001"

func noteSettings(gotID *string, calls *int, note journal.JournalNote, err error) Settings {
	settings := settingsWith(notConfiguredOpenPositions)
	settings.GetPositionNote = func(_ context.Context, positionID string) (journal.JournalNote, error) {
		*calls++
		*gotID = positionID
		return note, err
	}
	return settings
}

func getNote(settings Settings, target string) (int, http.Header, string) {
	recorder := serve(NewHandler(settings, discardLogger()), newTestRequest(http.MethodGet, target, allowedTestHost))
	return recorder.Code, recorder.Header(), recorder.Body.String()
}

func TestPositionNote_ErfolgLiefertAlleFelderLautSpec(t *testing.T) {
	setup, mistakes, notes, rating := "Breakout", "zu früh", "ok", 4
	note := journal.JournalNote{PositionID: notePositionID, Setup: &setup, Rating: &rating, Mistakes: &mistakes, Notes: &notes, UpdatedAt: 1791600000000}
	var gotID string
	var calls int

	status, header, body := getNote(noteSettings(&gotID, &calls, note, nil), "/positions/"+notePositionID+"/note")

	want := `{"positionId":"9100000000000000001","setup":"Breakout","rating":4,"mistakes":"zu früh","notes":"ok","updatedAt":1791600000000}`
	if status != http.StatusOK || header.Get("Content-Type") != "application/json; charset=utf-8" || body != want {
		t.Errorf("status=%d ct=%q body=%s", status, header.Get("Content-Type"), body)
	}
	if gotID != notePositionID {
		t.Errorf("positionId = %q", gotID)
	}
}

func TestPositionNote_NichtGesetzteFelderSindNull(t *testing.T) {
	var gotID string
	var calls int
	note := journal.JournalNote{PositionID: notePositionID, UpdatedAt: 5}

	_, _, body := getNote(noteSettings(&gotID, &calls, note, nil), "/positions/"+notePositionID+"/note")

	want := `{"positionId":"9100000000000000001","setup":null,"rating":null,"mistakes":null,"notes":null,"updatedAt":5}`
	if body != want {
		t.Errorf("Body = %s", body)
	}
}

func TestPositionNote_OhneNotizIst404NotFound(t *testing.T) {
	var gotID string
	var calls int
	settings := noteSettings(&gotID, &calls, journal.JournalNote{}, &journal.NoteNotFoundError{PositionID: notePositionID})
	recorder := serve(NewHandler(settings, discardLogger()), newTestRequest(http.MethodGet, "/positions/"+notePositionID+"/note", allowedTestHost))
	assertProblem(t, recorder, http.StatusNotFound, notFoundBody)
}

func TestPositionNote_OhneDatenbankIst503NotConfigured(t *testing.T) {
	var gotID string
	var calls int
	unavailable := &position.OpenPositionsUnavailableError{Reason: position.ReasonNotConfigured}
	settings := noteSettings(&gotID, &calls, journal.JournalNote{}, unavailable)
	recorder := serve(NewHandler(settings, discardLogger()), newTestRequest(http.MethodGet, "/positions/"+notePositionID+"/note", allowedTestHost))
	assertProblem(t, recorder, http.StatusServiceUnavailable, notConfiguredBody)
}

func TestPositionNote_UnerwarteterFehlerIst500OhneDetails(t *testing.T) {
	var gotID string
	var calls int
	settings := noteSettings(&gotID, &calls, journal.JournalNote{}, errors.New("geheim"))
	recorder := serve(NewHandler(settings, discardLogger()), newTestRequest(http.MethodGet, "/positions/"+notePositionID+"/note", allowedTestHost))
	assertProblem(t, recorder, http.StatusInternalServerError, internalBody)
}

func TestPositionNote_UngueltigePositionIdIst400OhneAufrufDesUseCase(t *testing.T) {
	for _, id := range []string{"abc", "12x", "123456789012345678901"} {
		var gotID string
		var calls int
		settings := noteSettings(&gotID, &calls, journal.JournalNote{}, nil)
		recorder := serve(NewHandler(settings, discardLogger()), newTestRequest(http.MethodGet, "/positions/"+id+"/note", allowedTestHost))
		assertProblem(t, recorder, http.StatusBadRequest, badRequestBody)
		if calls != 0 {
			t.Errorf("id %q: Use Case wurde aufgerufen", id)
		}
	}
}
