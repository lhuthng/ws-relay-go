package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGenerateTokenSkipsExistingRooms(t *testing.T) {
	hub := newHub(4, false)
	hub.rooms["123"] = &Room{token: "123"}

	for i := 0; i < 25; i++ {
		token := hub.generateToken()
		if token == "123" {
			t.Fatalf("expected a different token, got %q", token)
		}
		if len(token) != 3 {
			t.Fatalf("expected a 3-digit token, got %q", token)
		}
	}
}

func TestJoinRoomRejectsFullRoom(t *testing.T) {
	hub := newHub(2, false)
	hostConn := &safeConn{}
	room, _ := hub.hostRoom(hostConn, "cfg")

	guestConn := &safeConn{}
	if _, _, err := hub.joinRoom(guestConn, room.token); err != nil {
		t.Fatalf("expected first guest to join, got %v", err)
	}

	lateConn := &safeConn{}
	if _, _, err := hub.joinRoom(lateConn, room.token); err == nil || err.Error() != "room_full" {
		t.Fatalf("expected room_full error, got %v", err)
	}
}

func TestUpdateConfigRequiresHost(t *testing.T) {
	hub := newHub(3, false)
	hostConn := &safeConn{}
	room, host := hub.hostRoom(hostConn, "old")

	guestConn := &safeConn{}
	_, guest, err := hub.joinRoom(guestConn, room.token)
	if err != nil {
		t.Fatalf("join failed: %v", err)
	}

	if _, err := hub.updateConfig(room, guest, "new"); err == nil || err.Error() != "not_host" {
		t.Fatalf("expected not_host error, got %v", err)
	}

	others, err := hub.updateConfig(room, host, "new")
	if err != nil {
		t.Fatalf("host update failed: %v", err)
	}
	if room.config != "new" {
		t.Fatalf("expected config to be updated, got %q", room.config)
	}
	if len(others) != 1 || others[0] != guest {
		t.Fatalf("expected exactly the guest to be notified")
	}
}

func TestLeaveClosesRoomWhenHostDisconnects(t *testing.T) {
	hub := newHub(3, false)
	hostConn := &safeConn{}
	room, _ := hub.hostRoom(hostConn, "cfg")

	guestConn := &safeConn{}
	_, guest, err := hub.joinRoom(guestConn, room.token)
	if err != nil {
		t.Fatalf("join failed: %v", err)
	}

	res := hub.leave(hostConn)
	if res == nil {
		t.Fatal("expected leave result")
	}
	if !res.roomClosed {
		t.Fatal("expected room to close when host leaves")
	}
	if len(res.remaining) != 1 || res.remaining[0] != guest {
		t.Fatal("expected guest to remain in notification snapshot")
	}
	if _, ok := hub.rooms[room.token]; ok {
		t.Fatal("expected room to be removed from hub")
	}
	if _, _, ok := hub.lookup(guestConn); ok {
		t.Fatal("expected guest connection to be evicted when room closes")
	}
}

func TestLeaveKeepsRoomUsableWhenMemberLeavesAndCloseIsDisabled(t *testing.T) {
	hub := newHub(4, false)
	hostConn := &safeConn{}
	room, host := hub.hostRoom(hostConn, "cfg")

	guestConn := &safeConn{}
	guestRoom, _, err := hub.joinRoom(guestConn, room.token)
	if err != nil {
		t.Fatalf("join failed: %v", err)
	}
	if guestRoom != room {
		t.Fatal("expected guest to join the hosted room")
	}

	res := hub.leave(guestConn)
	if res == nil {
		t.Fatal("expected leave result")
	}
	if res.roomClosed {
		t.Fatal("expected room to remain open")
	}
	if len(room.members) != 1 || room.members[0] != host {
		t.Fatal("expected host to remain in room")
	}
	if _, ok := hub.rooms[room.token]; !ok {
		t.Fatal("expected room to remain registered")
	}
	others, err := hub.updateConfig(room, host, "next")
	if err != nil {
		t.Fatalf("expected host to keep managing room, got %v", err)
	}
	if len(others) != 0 {
		t.Fatal("expected no other members after guest left")
	}
}

func TestLeaveClosesRoomWhenMemberLeavesAndCloseIsEnabled(t *testing.T) {
	hub := newHub(4, true)
	hostConn := &safeConn{}
	room, _ := hub.hostRoom(hostConn, "cfg")

	guestConn := &safeConn{}
	if _, _, err := hub.joinRoom(guestConn, room.token); err != nil {
		t.Fatalf("join failed: %v", err)
	}

	res := hub.leave(guestConn)
	if res == nil {
		t.Fatal("expected leave result")
	}
	if !res.roomClosed {
		t.Fatal("expected room to close when member leaves")
	}
	if res.closeReason != "member_left" {
		t.Fatalf("expected member_left close reason, got %q", res.closeReason)
	}
	if _, ok := hub.rooms[room.token]; ok {
		t.Fatal("expected room to be removed")
	}
}

func TestStatusEndpointRespondsWithHubStats(t *testing.T) {
	hub := newHub(4, true)
	hostConn := &safeConn{}
	room, _ := hub.hostRoom(hostConn, "cfg")
	guestConn := &safeConn{}
	if _, _, err := hub.joinRoom(guestConn, room.token); err != nil {
		t.Fatalf("join failed: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/status", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, hub.status())
	})

	req := httptest.NewRequest(http.MethodGet, "/status", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rec.Code)
	}

	var got StatusResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("expected valid JSON response, got %v", err)
	}

	if got.Status != "ok" {
		t.Fatalf("expected status ok, got %q", got.Status)
	}
	if got.Rooms != 1 {
		t.Fatalf("expected 1 room, got %d", got.Rooms)
	}
	if got.ActiveConnections != 2 {
		t.Fatalf("expected 2 active connections, got %d", got.ActiveConnections)
	}
	if got.MaxRoomSize != 4 {
		t.Fatalf("expected max room size 4, got %d", got.MaxRoomSize)
	}
	if got.TotalCapacity != 4 {
		t.Fatalf("expected total capacity 4, got %d", got.TotalCapacity)
	}
	if !got.CloseRoomOnLeave {
		t.Fatal("expected close_room_on_leave to be true")
	}
}
