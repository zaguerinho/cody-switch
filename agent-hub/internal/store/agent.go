package store

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/zaguerinho/claude-switch/agent-hub/internal/model"
)

// JoinRoom adds an agent to a room.
func (fs *FileStore) JoinRoom(room, alias, role string) (*model.Agent, error) {
	mu := fs.mu.get(room)
	mu.Lock()
	defer mu.Unlock()

	if _, err := fs.getRoomUnlocked(room); err != nil {
		return nil, err
	}

	agents, err := fs.getAgentsUnlocked(room)
	if err != nil {
		return nil, err
	}

	for _, a := range agents {
		if a.Alias == alias {
			return nil, fmt.Errorf("agent %q already joined room %q", alias, room)
		}
	}

	agent := model.Agent{
		Alias:    alias,
		Role:     role,
		JoinedAt: time.Now().UTC(),
	}
	agents = append(agents, agent)

	if err := fs.writeAgentsUnlocked(room, agents); err != nil {
		return nil, err
	}

	// Initialize cursor for the new agent at 0 (nothing read)
	_ = fs.setCursorUnlocked(room, alias, 0)

	return &agent, nil
}

// LeaveRoom removes an agent from a room.
func (fs *FileStore) LeaveRoom(room, alias string) error {
	mu := fs.mu.get(room)
	mu.Lock()
	defer mu.Unlock()

	agents, err := fs.getAgentsUnlocked(room)
	if err != nil {
		return err
	}

	found := false
	var updated []model.Agent
	for _, a := range agents {
		if a.Alias == alias {
			found = true
			continue
		}
		updated = append(updated, a)
	}
	if !found {
		return fmt.Errorf("agent %q is not a member of room %q", alias, room)
	}

	return fs.writeAgentsUnlocked(room, updated)
}

// GetAgents returns all agents in a room.
func (fs *FileStore) GetAgents(room string) ([]model.Agent, error) {
	mu := fs.mu.get(room)
	mu.Lock()
	defer mu.Unlock()

	return fs.getAgentsUnlocked(room)
}

func (fs *FileStore) getAgentsUnlocked(room string) ([]model.Agent, error) {
	path := filepath.Join(fs.roomDir(room), "agents.json")
	var list model.AgentList
	if err := readJSON(path, &list); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	return list.Agents, nil
}

func (fs *FileStore) writeAgentsUnlocked(room string, agents []model.Agent) error {
	path := filepath.Join(fs.roomDir(room), "agents.json")
	return atomicWriteJSON(path, &model.AgentList{Agents: agents})
}

// IsAgentMember checks if an agent belongs to a room.
func (fs *FileStore) IsAgentMember(room, alias string) (bool, error) {
	agents, err := fs.GetAgents(room)
	if err != nil {
		return false, err
	}
	for _, a := range agents {
		if a.Alias == alias {
			return true, nil
		}
	}
	return false, nil
}

// GetAgentRooms returns all rooms an agent belongs to.
func (fs *FileStore) GetAgentRooms(alias string) ([]string, error) {
	rooms, err := fs.ListRooms()
	if err != nil {
		return nil, err
	}

	var result []string
	for _, s := range rooms {
		agents, err := fs.GetAgents(s.Name)
		if err != nil {
			continue
		}
		for _, a := range agents {
			if a.Alias == alias {
				result = append(result, s.Name)
				break
			}
		}
	}
	return result, nil
}
