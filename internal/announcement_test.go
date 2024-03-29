package internal_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/democracy-tools/countmein/internal"
	"github.com/democracy-tools/countmein/internal/bq"
	"github.com/stretchr/testify/require"
)

func TestHandle_Announcement(t *testing.T) {

	var buf bytes.Buffer
	require.NoError(t, json.NewEncoder(&buf).Encode(map[string][]*internal.Announcement{
		"announcements": {{
			UserId:     "test",
			UserDevice: internal.Device{Id: "test-1", Type: "iphone 14"},
			SeenDevice: internal.Device{Id: "test-2", Type: "iphone 15"},
			Location:   internal.Location{Latitude: 32.05766501361105, Longitude: 34.76640727232065},
			Time:       time.Now().Unix(),
		}}}))
	r, err := http.NewRequest(http.MethodPost, "/announcements", bytes.NewReader(buf.Bytes()))
	require.NoError(t, err)
	w := httptest.NewRecorder()

	internal.NewHandle(bq.NewInMemoryClient()).Announcements(w, r)

	require.Equal(t, http.StatusCreated, w.Result().StatusCode)
}
