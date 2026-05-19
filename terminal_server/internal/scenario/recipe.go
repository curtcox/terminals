package scenario

import (
	"context"
	"strings"
	"sync"

	iorouter "github.com/curtcox/terminals/terminal_server/internal/io"
)

// ScenarioRecipe captures the common scenario skeleton:
// resolve targets -> claim resources -> apply media plan -> render UI -> cleanup.
type ScenarioRecipe struct { //nolint:revive
	ActivationID string
	Resolve      func(ctx context.Context, env *Environment) []string
	Claims       func(targets []string) []iorouter.Claim
	MediaPlan    func(targets []string) *iorouter.MediaPlan
	OnStart      func(ctx context.Context, env *Environment, targets []string) error
	OnStop       func(ctx context.Context, env *Environment) error
}

type scenarioRecipeState struct {
	mu sync.Mutex

	env        *Environment
	planHandle iorouter.PlanHandle
}

func (s *scenarioRecipeState) start(ctx context.Context, env *Environment, recipe ScenarioRecipe) error {
	if env == nil {
		return nil
	}
	targets := resolveRecipeTargets(ctx, env, recipe)
	if err := requestRecipeClaims(ctx, env, recipe, targets); err != nil {
		return err
	}
	planHandle, err := applyRecipeMediaPlan(ctx, env, recipe, targets)
	if err != nil {
		return err
	}

	s.mu.Lock()
	s.env = env
	s.planHandle = planHandle
	s.mu.Unlock()

	if recipe.OnStart != nil {
		return recipe.OnStart(ctx, env, targets)
	}
	return nil
}

func (s *scenarioRecipeState) stop(ctx context.Context, recipe ScenarioRecipe) error {
	s.mu.Lock()
	env := s.env
	planHandle := s.planHandle
	s.env = nil
	s.planHandle = ""
	s.mu.Unlock()

	if env == nil {
		return nil
	}
	if err := tearRecipeMediaPlan(ctx, env, planHandle); err != nil {
		return err
	}
	if err := releaseRecipeClaims(ctx, env, recipe.ActivationID); err != nil {
		return err
	}
	if recipe.OnStop != nil {
		return recipe.OnStop(ctx, env)
	}
	return nil
}

func tearRecipeMediaPlan(ctx context.Context, env *Environment, planHandle iorouter.PlanHandle) error {
	routeIO, ok := env.IO.(interface{ MediaPlanner() *iorouter.MediaPlanner })
	if !ok {
		return nil
	}
	planner := routeIO.MediaPlanner()
	if planner == nil || planHandle == "" {
		return nil
	}
	return planner.Tear(ctx, planHandle)
}

func releaseRecipeClaims(ctx context.Context, env *Environment, activationID string) error {
	activationID = strings.TrimSpace(activationID)
	if activationID == "" {
		return nil
	}
	routeIO, ok := env.IO.(interface{ Claims() *iorouter.ClaimManager })
	if !ok {
		return nil
	}
	claims := routeIO.Claims()
	if claims == nil {
		return nil
	}
	return claims.Release(ctx, activationID)
}
