package store

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/zaguerinho/claude-switch/agent-hub/internal/model"
)

func tempStore(t *testing.T) *FileStore {
	t.Helper()
	dir := t.TempDir()
	fs, err := New(dir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if err := fs.Init(); err != nil {
		t.Fatalf("Init: %v", err)
	}
	return fs
}

func TestRoomLifecycle(t *testing.T) {
	fs := tempStore(t)

	// Create
	meta, err := fs.CreateRoom("test", "A test room")
	if err != nil {
		t.Fatalf("CreateRoom: %v", err)
	}
	if meta.Name != "test" || meta.Description != "A test room" {
		t.Fatalf("unexpected meta: %+v", meta)
	}

	// Duplicate
	_, err = fs.CreateRoom("test", "")
	if err == nil {
		t.Fatal("expected error on duplicate create")
	}

	// Get
	got, err := fs.GetRoom("test")
	if err != nil {
		t.Fatalf("GetRoom: %v", err)
	}
	if got.Name != "test" {
		t.Fatalf("wrong name: %s", got.Name)
	}

	// List
	rooms, err := fs.ListRooms()
	if err != nil {
		t.Fatalf("ListRooms: %v", err)
	}
	if len(rooms) != 1 {
		t.Fatalf("expected 1 room, got %d", len(rooms))
	}

	// Archive
	if err := fs.ArchiveRoom("test"); err != nil {
		t.Fatalf("ArchiveRoom: %v", err)
	}
	rooms, err = fs.ListRooms()
	if err != nil {
		t.Fatalf("ListRooms after archive: %v", err)
	}
	if len(rooms) != 0 {
		t.Fatalf("expected 0 rooms after archive, got %d", len(rooms))
	}

	// List all includes archived
	all, err := fs.ListAllRooms()
	if err != nil {
		t.Fatalf("ListAllRooms: %v", err)
	}
	if len(all) != 1 || !all[0].Archived {
		t.Fatalf("expected 1 archived room, got %+v", all)
	}
}

func TestRoomNotFound(t *testing.T) {
	fs := tempStore(t)
	_, err := fs.GetRoom("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent room")
	}
}

func TestAgentJoinLeave(t *testing.T) {
	fs := tempStore(t)
	fs.CreateRoom("sess", "")

	// Join
	agent, err := fs.JoinRoom("sess", "alice", "architect")
	if err != nil {
		t.Fatalf("JoinRoom: %v", err)
	}
	if agent.Alias != "alice" || agent.Role != "architect" {
		t.Fatalf("wrong agent: %+v", agent)
	}

	// Duplicate join
	_, err = fs.JoinRoom("sess", "alice", "")
	if err == nil {
		t.Fatal("expected error on duplicate join")
	}

	// Second agent
	_, err = fs.JoinRoom("sess", "bob", "reviewer")
	if err != nil {
		t.Fatalf("JoinRoom bob: %v", err)
	}

	// List
	agents, err := fs.GetAgents("sess")
	if err != nil {
		t.Fatalf("GetAgents: %v", err)
	}
	if len(agents) != 2 {
		t.Fatalf("expected 2 agents, got %d", len(agents))
	}

	// Leave
	if err := fs.LeaveRoom("sess", "alice"); err != nil {
		t.Fatalf("LeaveRoom: %v", err)
	}
	agents, err = fs.GetAgents("sess")
	if err != nil {
		t.Fatalf("GetAgents after leave: %v", err)
	}
	if len(agents) != 1 || agents[0].Alias != "bob" {
		t.Fatalf("unexpected agents after leave: %+v", agents)
	}

	// Leave nonexistent
	err = fs.LeaveRoom("sess", "carol")
	if err == nil {
		t.Fatal("expected error leaving non-member")
	}
}

func TestMessagePostAndRead(t *testing.T) {
	fs := tempStore(t)
	fs.CreateRoom("sess", "")

	// Post 3 messages
	for i := 1; i <= 3; i++ {
		msg, err := fs.PostMessage("sess", "alice", model.MessageQuestion, fmt.Sprintf("Q%d", i), fmt.Sprintf("Body %d", i))
		if err != nil {
			t.Fatalf("PostMessage %d: %v", i, err)
		}
		if msg.ID != i {
			t.Fatalf("expected ID %d, got %d", i, msg.ID)
		}
	}

	// Read all
	msgs, err := fs.ReadMessages("sess", 0)
	if err != nil {
		t.Fatalf("ReadMessages: %v", err)
	}
	if len(msgs) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(msgs))
	}

	// Read last 2
	msgs, err = fs.ReadMessages("sess", 2)
	if err != nil {
		t.Fatalf("ReadMessages last 2: %v", err)
	}
	if len(msgs) != 2 || msgs[0].ID != 2 {
		t.Fatalf("unexpected last-2 messages: %+v", msgs)
	}
}

func TestCursorAndUnread(t *testing.T) {
	fs := tempStore(t)
	fs.CreateRoom("sess", "")
	fs.JoinRoom("sess", "alice", "")
	fs.JoinRoom("sess", "bob", "")

	// Post 5 messages
	for i := 1; i <= 5; i++ {
		fs.PostMessage("sess", "alice", model.MessageNote, "", fmt.Sprintf("msg %d", i))
	}

	// Bob has read nothing → 5 unread
	count, latest, err := fs.UnreadCount("sess", "bob")
	if err != nil {
		t.Fatalf("UnreadCount: %v", err)
	}
	if count != 5 {
		t.Fatalf("expected 5 unread, got %d", count)
	}
	if latest == nil || latest.ID != 5 {
		t.Fatalf("expected latest ID 5, got %+v", latest)
	}

	// Ack up to 3
	if err := fs.Acknowledge("sess", "bob", 3); err != nil {
		t.Fatalf("Acknowledge: %v", err)
	}

	count, _, err = fs.UnreadCount("sess", "bob")
	if err != nil {
		t.Fatalf("UnreadCount after ack: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected 2 unread, got %d", count)
	}

	// Read unread
	unread, err := fs.ReadUnread("sess", "bob")
	if err != nil {
		t.Fatalf("ReadUnread: %v", err)
	}
	if len(unread) != 2 || unread[0].ID != 4 {
		t.Fatalf("unexpected unread: %+v", unread)
	}

	// Ack all (0 = latest)
	if err := fs.Acknowledge("sess", "bob", 0); err != nil {
		t.Fatalf("Acknowledge all: %v", err)
	}
	count, _, _ = fs.UnreadCount("sess", "bob")
	if count != 0 {
		t.Fatalf("expected 0 unread after ack-all, got %d", count)
	}

	// Ack beyond max
	err = fs.Acknowledge("sess", "bob", 99)
	if err == nil {
		t.Fatal("expected error acking beyond max")
	}
}

func TestConcurrentMessagePosting(t *testing.T) {
	fs := tempStore(t)
	fs.CreateRoom("sess", "")

	n := 50
	var wg sync.WaitGroup
	errs := make(chan error, n)
	ids := make(chan int, n)

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			msg, err := fs.PostMessage("sess", "agent", model.MessageNote, "", fmt.Sprintf("concurrent %d", idx))
			if err != nil {
				errs <- err
				return
			}
			ids <- msg.ID
		}(i)
	}
	wg.Wait()
	close(errs)
	close(ids)

	for err := range errs {
		t.Fatalf("concurrent post error: %v", err)
	}

	// Verify all IDs are unique and sequential
	seen := map[int]bool{}
	for id := range ids {
		if seen[id] {
			t.Fatalf("duplicate ID: %d", id)
		}
		seen[id] = true
	}
	if len(seen) != n {
		t.Fatalf("expected %d unique IDs, got %d", n, len(seen))
	}

	// Verify sequential (1..n)
	for i := 1; i <= n; i++ {
		if !seen[i] {
			t.Fatalf("missing ID %d", i)
		}
	}
}

func TestStatusBoard(t *testing.T) {
	fs := tempStore(t)
	fs.CreateRoom("sess", "")

	// Update
	if err := fs.UpdateStatus("sess", "migration", "in-progress", "alice"); err != nil {
		t.Fatalf("UpdateStatus: %v", err)
	}

	board, err := fs.GetStatus("sess")
	if err != nil {
		t.Fatalf("GetStatus: %v", err)
	}
	if len(board.Entries) != 1 || board.Entries[0].Value != "in-progress" {
		t.Fatalf("unexpected board: %+v", board)
	}

	// Update same key
	if err := fs.UpdateStatus("sess", "migration", "done", "bob"); err != nil {
		t.Fatalf("UpdateStatus 2: %v", err)
	}
	board, _ = fs.GetStatus("sess")
	if len(board.Entries) != 1 || board.Entries[0].Value != "done" || board.Entries[0].UpdatedBy != "bob" {
		t.Fatalf("update didn't overwrite: %+v", board)
	}

	// Add second key
	fs.UpdateStatus("sess", "tests", "passing", "alice")
	board, _ = fs.GetStatus("sess")
	if len(board.Entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(board.Entries))
	}
}

func TestOrphanTmpCleanup(t *testing.T) {
	fs := tempStore(t)
	fs.CreateRoom("sess", "")

	// Create orphaned .tmp files
	dir := fs.roomDir("sess")
	os.WriteFile(filepath.Join(dir, "meta.json.tmp"), []byte("junk"), 0o644)
	os.WriteFile(filepath.Join(dir, "messages", "0001.json.tmp"), []byte("junk"), 0o644)

	// Init should clean them up
	if err := fs.Init(); err != nil {
		t.Fatalf("Init: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, "meta.json.tmp")); !os.IsNotExist(err) {
		t.Fatal("orphan meta.json.tmp not removed")
	}
	if _, err := os.Stat(filepath.Join(dir, "messages", "0001.json.tmp")); !os.IsNotExist(err) {
		t.Fatal("orphan 0001.json.tmp not removed")
	}

	// Original meta should still be intact
	_, err := fs.GetRoom("sess")
	if err != nil {
		t.Fatalf("room broken after cleanup: %v", err)
	}
}

func TestAgentRooms(t *testing.T) {
	fs := tempStore(t)
	fs.CreateRoom("sess1", "")
	fs.CreateRoom("sess2", "")
	fs.CreateRoom("sess3", "")

	fs.JoinRoom("sess1", "alice", "")
	fs.JoinRoom("sess2", "alice", "")
	fs.JoinRoom("sess3", "bob", "")

	rooms, err := fs.GetAgentRooms("alice")
	if err != nil {
		t.Fatalf("GetAgentRooms: %v", err)
	}
	if len(rooms) != 2 {
		t.Fatalf("expected 2 rooms for alice, got %d", len(rooms))
	}
}

func TestRoomInfo(t *testing.T) {
	fs := tempStore(t)
	fs.CreateRoom("sess", "desc")
	fs.JoinRoom("sess", "alice", "dev")
	fs.JoinRoom("sess", "bob", "reviewer")
	fs.PostMessage("sess", "alice", model.MessageQuestion, "Q1", "body")
	fs.PostMessage("sess", "bob", model.MessageAnswer, "A1", "reply")

	info, err := fs.GetRoomInfo("sess")
	if err != nil {
		t.Fatalf("GetRoomInfo: %v", err)
	}
	if info.AgentCount != 2 {
		t.Fatalf("expected 2 agents, got %d", info.AgentCount)
	}
	if info.MessageCount != 2 {
		t.Fatalf("expected 2 messages, got %d", info.MessageCount)
	}
	if info.LastActivity == nil {
		t.Fatal("expected non-nil last activity")
	}
}
