package internal

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/democracy-tools/countmein/internal/bq"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

type AnnouncementDB struct {
	Id                string  `bigquery:"id"`
	UserId            string  `bigquery:"user_id"`
	UserDeviceId      string  `bigquery:"user_device_id"`
	UserDeviceType    string  `bigquery:"user_device_type"`
	SeenDeviceId      string  `bigquery:"seen_device_id"`
	SeenDeviceType    string  `bigquery:"seen_device_type"`
	LocationLatitude  float64 `bigquery:"location_latitude"`
	LocationLongitude float64 `bigquery:"location_longitude"`
	UserTime          int64   `bigquery:"user_time"`
	ServerTime        int64   `bigquery:"server_time"`
}

type Announcement struct {
	UserId     string   `json:"user_id"`
	UserDevice Device   `json:"device_id"`
	SeenDevice Device   `json:"seen_device"`
	Location   Location `json:"location"`
	Time       int64    `json:"time"`
}

type Location struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type Device struct {
	Id   string `json:"id"`
	Type string `json:"type"`
}

type Handle struct{ bqClient bq.Client }

func NewHandle(bqClient bq.Client) *Handle {

	return &Handle{bqClient: bqClient}
}

func (h *Handle) Announcements(w http.ResponseWriter, r *http.Request) {

	announcements, code := getAnnouncements(r)
	if code != http.StatusOK {
		w.WriteHeader(code)
		return
	}

	ok := validateAnnouncements(announcements)
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	err := h.bqClient.Insert(bq.TableAnnouncement, toDBAnnouncements(announcements))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func validateAnnouncements(announcements []*Announcement) bool {

	for _, currAnnouncement := range announcements {
		if !validateAnnouncement(currAnnouncement) {
			return false
		}
	}

	return true
}

func validateAnnouncement(announcement *Announcement) bool {

	if announcement.Time < 1678133631 || announcement.Time > time.Now().Add(time.Hour*2).Unix() {
		log.Infof("invalid announcement time '%d' user '%s' user device '%s'",
			announcement.Time, announcement.UserId, announcement.UserDevice.Id)
		return false
	}
	count := len(announcement.UserId)
	if count < 1 || count > 48 {
		log.Debugf("invalid announcement user '%s'", announcement.UserId)
		return false
	}
	count = len(announcement.UserDevice.Id)
	if count > 48 {
		log.Debugf("invalid announcement user device '%s' user '%s'",
			announcement.UserDevice.Id, announcement.UserId)
		return false
	}
	count = len(announcement.UserDevice.Type)
	if count > 48 {
		log.Debugf("invalid announcement user device type '%s' user '%s'",
			announcement.UserDevice.Type, announcement.UserId)
		return false
	}
	count = len(announcement.SeenDevice.Id)
	if count > 48 {
		log.Debugf("invalid announcement seen device '%s' user '%s'",
			announcement.SeenDevice.Id, announcement.UserId)
		return false
	}
	count = len(announcement.SeenDevice.Type)
	if count > 48 {
		log.Debugf("invalid announcement seen device type '%s' user '%s'",
			announcement.SeenDevice.Type, announcement.UserId)
		return false
	}

	return true
}

func getAnnouncements(r *http.Request) ([]*Announcement, int) {

	var announcements map[string][]*Announcement
	err := json.NewDecoder(r.Body).Decode(&announcements)
	if err != nil {
		log.Infof("failed to decode announcements request with %v", err)
		return nil, http.StatusBadRequest
	}

	res, ok := announcements["announcements"]
	if !ok {
		log.Info("no 'announcements' key found in request")
		return nil, http.StatusBadRequest
	}

	return res, http.StatusOK
}

func toDBAnnouncements(announcements []*Announcement) []*AnnouncementDB {

	res := []*AnnouncementDB{}
	for _, currAnnouncement := range announcements {
		res = append(res, toDBAnnouncement(currAnnouncement))
	}

	return res
}

func toDBAnnouncement(announcement *Announcement) *AnnouncementDB {

	return &AnnouncementDB{
		Id:                uuid.NewString(),
		UserId:            announcement.UserId,
		UserDeviceId:      announcement.UserDevice.Id,
		UserDeviceType:    announcement.UserDevice.Type,
		SeenDeviceId:      announcement.SeenDevice.Id,
		SeenDeviceType:    announcement.SeenDevice.Type,
		LocationLatitude:  announcement.Location.Latitude,
		LocationLongitude: announcement.Location.Longitude,
		UserTime:          announcement.Time,
		ServerTime:        time.Now().Unix(),
	}
}
