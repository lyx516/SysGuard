package skills

import (
	"context"
	"fmt"
	"sync"
)

// Skill interface defines the contract for all skills
type Skill interface {
	Name() string
	Description() string
	Execute(ctx context.Context, input *SkillInput) (*SkillOutput, error)
}

// SkillInput represents input to a skill
type SkillInput struct {
	Params map[string]interface{}
}

// SkillOutput represents output from a skill
type SkillOutput struct {
	Result    interface{}
	Error      error
	Metadata   map[string]string
	Success    bool
}

// SkillRegistry manages all available skills
type SkillRegistry struct {
	skills map[string]Skill
	mu     sync.RWMutex
}

// NewSkillRegistry creates a new skill registry
func NewSkillRegistry() *SkillRegistry {
	return &SkillRegistry{
		skills: make(map[string]Skill),
	}
}

// Register adds a skill to the registry
func (r *SkillRegistry) Register(skill Skill) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := skill.Name()
	if _, exists := r.skills[name]; exists {
		return fmt.Errorf("skill '%s' already registered", name)
	}

	r.skills[name] = skill
	return nil
}

// Get retrieves a skill by name
func (r *SkillRegistry) Get(name string) (Skill, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	skill, exists := r.skills[name]
	if !exists {
		return nil, fmt.Errorf("skill '%s' not found", name)
	}

	return skill, nil
}

// List returns all registered skill names
func (r *SkillRegistry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.skills))
	for name := range r.skills {
		names = append(names, name)
	}
	return names
}

// Execute executes a skill by name
func (r *SkillRegistry) Execute(ctx context.Context, name string, input *SkillInput) (*SkillOutput, error) {
	skill, err := r.Get(name)
	if err != nil {
		return nil, err
	}

	return skill.Execute(ctx, input)
}
