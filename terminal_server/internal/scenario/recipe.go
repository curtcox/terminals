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
	targets := []string(nil)
	if recipe.Resolve != nil {
		targets = recipe.Resolve(ctx, env)
	}

	if routeIO, ok := env.IO.(interface{ Claims() *iorouter.ClaimManager }); ok && recipe.Claims != nil {
		if claims := routeIO.Claims(); claims != nil {
			if _, err := claims.Request(ctx, recipe.Claims(targets)); err != nil && err != iorouter.ErrClaimConflict {
				return err
			}
		}
	}

	var planHandle iorouter.PlanHandle
	if routeIO, ok := env.IO.(interface{ MediaPlanner() *iorouter.MediaPlanner }); ok && recipe.MediaPlan != nil {
		if planner := routeIO.MediaPlanner(); planner != nil {
			plan := recipe.MediaPlan(targets)
			if plan != nil {
				handle, err := planner.Apply(ctx, *plan)
				if err != nil {
					return err
				}
				planHandle = handle
			}
		}
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
	if routeIO, ok := env.IO.(interface{ MediaPlanner() *iorouter.MediaPlanner }); ok {
		if planner := routeIO.MediaPlanner(); planner != nil && planHandle != "" {
			if err := planner.Tear(ctx, planHandle); err != nil {
				return err
			}
		}
	}
	if routeIO, ok := env.IO.(interface{ Claims() *iorouter.ClaimManager }); ok {
		if claims := routeIO.Claims(); claims != nil && strings.TrimSpace(recipe.ActivationID) != "" {
			if err := claims.Release(ctx, recipe.ActivationID); err != nil {
				return err
			}
		}
	}
	if recipe.OnStop != nil {
		return recipe.OnStop(ctx, env)
	}
	return nil
}
